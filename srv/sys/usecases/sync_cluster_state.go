package usecases

import (
	"encoding/json"
	"fmt"

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
		svcs, _ = uc.repo.GetServices()
		dbs, _ = uc.repo.GetDatabases()
		links, _ = uc.repo.GetServiceLinks()
		obs, _ = uc.repo.GetObservability()
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

	_, _ = uc.sshExec.RunCommand("mkdir -p /opt/tarhiata")
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
			_ = uc.repo.SaveService(s)
		}
		for _, d := range dump.Databases {
			_ = uc.repo.SaveDatabase(d)
		}
		for _, l := range dump.ServiceLinks {
			_ = uc.repo.SaveServiceLink(l)
		}
		if dump.Observability != nil {
			_ = uc.repo.SaveObservability(*dump.Observability)
		}
	}

	return &dump, nil
}
