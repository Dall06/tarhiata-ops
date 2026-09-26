package usecases

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type ClusterStateDump struct {
	Services      []domain.SavedService      `json:"services"`
	Databases     []domain.SavedDatabase     `json:"databases"`
	ServiceLinks  []domain.ServiceLink       `json:"service_links"`
	Observability *domain.SavedObservability `json:"observability,omitempty"`
}

type SyncClusterStateUseCase struct {
	repo    ports.ConfigRepository
	sshExec ports.SSHExecutor
}

func NewSyncClusterStateUseCase(repo ports.ConfigRepository, sshExec ports.SSHExecutor) *SyncClusterStateUseCase {
	return &SyncClusterStateUseCase{
		repo:    repo,
		sshExec: sshExec,
	}
}

// ExportStateToRemote escribe /opt/tarhiata/state.json en el VPS host para asegurar que el VPS sea la fuente de verdad.
func (uc *SyncClusterStateUseCase) ExportStateToRemote() error {
	if uc.sshExec == nil {
		return fmt.Errorf("ssh executor no disponible")
	}

	var svcs []domain.SavedService
	var dbs []domain.SavedDatabase
	var links []domain.ServiceLink
	var obs *domain.SavedObservability

	if uc.repo != nil {
		var err error
		svcs, err = uc.repo.GetServices()
		if err != nil {
			slog.Warn("Error obteniendo servicios para exportar", "error", err)
		}
		dbs, err = uc.repo.GetDatabases()
		if err != nil {
			slog.Warn("Error obteniendo bases de datos para exportar", "error", err)
		}
		links, err = uc.repo.GetServiceLinks()
		if err != nil {
			slog.Warn("Error obteniendo enlaces para exportar", "error", err)
		}
		obs, err = uc.repo.GetObservability()
		if err != nil {
			slog.Warn("Error obteniendo observabilidad para exportar", "error", err)
		}
	}

	dump := ClusterStateDump{
		Services:      svcs,
		Databases:     dbs,
		ServiceLinks:  links,
		Observability: obs,
	}

	data, err := json.MarshalIndent(dump, "", "  ")
	if err != nil {
		return err
	}

	if resMkdir, errMkdir := uc.sshExec.RunCommand("mkdir -p /opt/tarhiata"); errMkdir != nil || (resMkdir != nil && resMkdir.ExitCode != 0) {
		return fmt.Errorf("error creando directorio /opt/tarhiata: %w", errMkdir)
	}
	return uc.sshExec.WriteRemoteFile("/opt/tarhiata/state.json", string(data))
}

// ImportStateFromRemote lee /opt/tarhiata/state.json del VPS Host y sincroniza el catálogo en la BD local de la nueva PC.
func (uc *SyncClusterStateUseCase) ImportStateFromRemote() (*ClusterStateDump, error) {
	if uc.sshExec == nil {
		return nil, fmt.Errorf("ssh executor no disponible")
	}

	res, err := uc.sshExec.RunCommand("cat /opt/tarhiata/state.json 2>/dev/null")
	if err != nil || res.ExitCode != 0 || res.Output == "" {
		return nil, fmt.Errorf("no se encontró el archivo de estado remoto /opt/tarhiata/state.json en el VPS")
	}

	var dump ClusterStateDump
	if err := json.Unmarshal([]byte(res.Output), &dump); err != nil {
		return nil, fmt.Errorf("error decodificando estado remoto del VPS: %w", err)
	}

	// Sincronizar Servicios y Datos en repositorio local si está disponible
	if uc.repo != nil {
		for _, s := range dump.Services {
			if errSave := uc.repo.SaveService(s); errSave != nil {
				slog.Warn("Fallo al guardar servicio sincronizado", "service", s.Name, "error", errSave)
			}
		}
		for _, d := range dump.Databases {
			if errSave := uc.repo.SaveDatabase(d); errSave != nil {
				slog.Warn("Fallo al guardar base de datos sincronizada", "database", d.Name, "error", errSave)
			}
		}
		for _, l := range dump.ServiceLinks {
			if errSave := uc.repo.SaveServiceLink(l); errSave != nil {
				slog.Warn("Fallo al guardar enlace sincronizado", "source", l.SourceSvc, "target", l.TargetSvc, "error", errSave)
			}
		}
		if dump.Observability != nil {
			if errSave := uc.repo.SaveObservability(*dump.Observability); errSave != nil {
				slog.Warn("Fallo al guardar observabilidad sincronizada", "error", errSave)
			}
		}
	}

	return &dump, nil
}
