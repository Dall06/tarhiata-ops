package usecases

import (
	"fmt"

	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type DefaultUnlinkServicesUseCase struct {
	repo ports.ConfigRepository
	ssh  ports.SSHExecutor
}

func NewUnlinkServicesUseCase(repo ports.ConfigRepository, ssh ports.SSHExecutor) *DefaultUnlinkServicesUseCase {
	return &DefaultUnlinkServicesUseCase{
		repo: repo,
		ssh:  ssh,
	}
}

func (u *DefaultUnlinkServicesUseCase) Execute(sourceSvc string, targetSvc string) error {
	// 1. Consultar links existentes para encontrar la variable de entorno a remover
	if u.repo != nil {
		links, err := u.repo.GetServiceLinks()
		if err != nil {
			return fmt.Errorf("error obteniendo enlaces de servicios: %w", err)
		}
		for _, l := range links {
			if l.SourceSvc == sourceSvc && l.TargetSvc == targetSvc {
				// 2. Remover variable de entorno de Docker Swarm
				cmd := fmt.Sprintf("docker service update --env-rm %s %s 2>/dev/null || docker service update --env-rm %s %s_%s 2>/dev/null || docker service update --env-rm %s tarhiata-app-%s 2>/dev/null || true",
					l.EnvVarName, sourceSvc, l.EnvVarName, sourceSvc, sourceSvc, l.EnvVarName, sourceSvc)
				if u.ssh != nil {
					_, _ = u.ssh.RunCommand(cmd)
				}
			}
		}
		// 3. Eliminar de SQLite
		if err := u.repo.DeleteServiceLink(sourceSvc, targetSvc); err != nil {
			return fmt.Errorf("error al eliminar enlace en base de datos: %w", err)
		}
	}

	if u.ssh != nil {
		syncUC := NewSyncClusterStateUseCase(u.repo, u.ssh)
		_ = syncUC.ExportStateToRemote()
	}

	return nil
}
