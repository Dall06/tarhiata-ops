package usecases

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

// ManageDeploymentHistoryUseCase gestiona el registro y rollback determinista de versiones de un servicio.
type ManageDeploymentHistoryUseCase struct {
	repo     ports.ConfigRepository
	executor ports.SSHExecutor
}

func NewManageDeploymentHistoryUseCase(repo ports.ConfigRepository, executor ports.SSHExecutor) *ManageDeploymentHistoryUseCase {
	return &ManageDeploymentHistoryUseCase{
		repo:     repo,
		executor: executor,
	}
}

// RecordDeployment registra un nuevo despliegue exitoso o fallido en el historial.
func (uc *ManageDeploymentHistoryUseCase) RecordDeployment(rec domain.DeploymentRecord) error {
	if strings.TrimSpace(rec.ServiceName) == "" {
		return errors.New("nombre de servicio requerido")
	}
	if rec.DeployedAt.IsZero() {
		rec.DeployedAt = time.Now()
	}
	return uc.repo.SaveDeploymentRecord(rec)
}

// GetHistory recupera las versiones registradas para un servicio específico.
func (uc *ManageDeploymentHistoryUseCase) GetHistory(serviceName string, limit int) ([]domain.DeploymentRecord, error) {
	if strings.TrimSpace(serviceName) == "" {
		return nil, errors.New("nombre de servicio requerido")
	}
	return uc.repo.GetDeploymentHistory(serviceName, limit)
}

// RollbackToVersion revierte el servicio en Docker Swarm al estado exacto de una versión histórica.
func (uc *ManageDeploymentHistoryUseCase) RollbackToVersion(serviceName string, historyID int, config domain.ServerConfig) (*domain.DeploymentRecord, error) {
	rec, err := uc.repo.GetDeploymentRecordByID(historyID)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo registro de historial: %w", err)
	}
	if rec == nil {
		return nil, fmt.Errorf("versión con ID %d no encontrada", historyID)
	}
	if rec.ServiceName != serviceName {
		return nil, fmt.Errorf("el registro ID %d pertenece a %s, no a %s", historyID, rec.ServiceName, serviceName)
	}

	// 1. Conectar SSH si es necesario
	if err := uc.executor.Connect(config); err != nil {
		return nil, fmt.Errorf("error de conexión SSH: %w", err)
	}
	defer func() {
		if clErr := uc.executor.Close(); clErr != nil {
			slog.Warn("manage_deployment_history: error cerrando conexión SSH", "error", clErr)
		}
	}()

	// 2. Ejecutar actualización en Swarm a la imagen histórica
	cmd := fmt.Sprintf("docker service update --image %s %s || docker service update --image %s tarhiata-app-%s || docker service update --image %s %s_%s",
		rec.ImageTag, rec.ServiceName,
		rec.ImageTag, rec.ServiceName,
		rec.ImageTag, rec.ServiceName, rec.ServiceName,
	)

	cmdRes, err := uc.executor.RunCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("falló rollback en Docker Swarm: %w", err)
	}
	if cmdRes != nil && cmdRes.ExitCode != 0 {
		return nil, fmt.Errorf("comando docker service update falló con código %d: %s", cmdRes.ExitCode, cmdRes.Output)
	}

	// 3. Registrar el nuevo evento de rollback en el historial
	rollbackRecord := domain.DeploymentRecord{
		ServiceName: rec.ServiceName,
		ImageTag:    rec.ImageTag,
		EnvVars:     rec.EnvVars,
		Port:        rec.Port,
		Domain:      rec.Domain,
		Expose:      rec.Expose,
		DeployedAt:  time.Now(),
		Status:      "rolled_back",
	}
	if errSave := uc.repo.SaveDeploymentRecord(rollbackRecord); errSave != nil {
		// Loguear o continuar
	}

	return rec, nil
}
