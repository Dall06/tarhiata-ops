package usecases

import (
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type linkServicesUseCase struct {
	repo    ports.ConfigRepository
	sshExec ports.SSHExecutor
}

func NewLinkServicesUseCase(repo ports.ConfigRepository, sshExec ports.SSHExecutor) ports.LinkServicesUseCase {
	return &linkServicesUseCase{
		repo:    repo,
		sshExec: sshExec,
	}
}

func (u *linkServicesUseCase) Execute(sourceSvc string, targetSvc string, envVarName string) (domain.ServiceLink, error) {
	if sourceSvc == "" || targetSvc == "" {
		return domain.ServiceLink{}, fmt.Errorf("sourceSvc y targetSvc son requeridos")
	}

	if envVarName == "" {
		envVarName = "DATABASE_URL"
	}
	envVarName = strings.ToUpper(envVarName)

	var targetURL string

	// 1. Buscar si targetSvc es una Base de Datos en SQLite
	if u.repo != nil {
		db, err := u.repo.GetDatabase(targetSvc)
		if err == nil && db != nil {
			switch strings.ToLower(db.Engine) {
			case "postgres":
				targetURL = fmt.Sprintf("postgres://admin:%s@tarhiata-db-%s:5432/db", db.Password, db.Name)
			case "mongodb":
				targetURL = fmt.Sprintf("mongodb://admin:%s@tarhiata-db-%s:27017/?authSource=admin", db.Password, db.Name)
			case "mysql":
				targetURL = fmt.Sprintf("mysql://root:%s@tarhiata-db-%s:3306/db", db.Password, db.Name)
			case "redis":
				targetURL = fmt.Sprintf("redis://tarhiata-db-%s:6379", db.Name)
			default:
				targetURL = fmt.Sprintf("http://tarhiata-db-%s:%d", db.Name, db.InternalPort)
			}
		} else {
			// 2. Si no es BD, buscar si es un Servicio App
			svc, err := u.repo.GetService(targetSvc)
			if err == nil && svc != nil {
				targetURL = fmt.Sprintf("http://tarhiata-app-%s:%d", svc.Name, svc.Port)
			} else {
				// Fallback de DNS interno Docker Swarm
				targetURL = fmt.Sprintf("http://tarhiata-app-%s:80", targetSvc)
			}
		}
	}

	link := domain.ServiceLink{
		SourceSvc:  sourceSvc,
		TargetSvc:  targetSvc,
		EnvVarName: envVarName,
		TargetURL:  targetURL,
	}

	// 3. Persistir interconexión en SQLite
	if u.repo != nil {
		if err := u.repo.SaveServiceLink(link); err != nil {
			return link, fmt.Errorf("error guardando enlace en SQLite: %w", err)
		}
	}

	// 4. Si hay SSHExecutor activo, inyectar dinámicamente la ENV VAR en Docker Swarm
	if u.sshExec != nil {
		cmd := fmt.Sprintf("docker service update --env-add %s=\"%s\" %s 2>/dev/null || docker service update --env-add %s=\"%s\" %s_%s 2>/dev/null || docker service update --env-add %s=\"%s\" tarhiata-app-%s 2>/dev/null || true",
			envVarName, targetURL, sourceSvc,
			envVarName, targetURL, sourceSvc, sourceSvc,
			envVarName, targetURL, sourceSvc)
		_, _ = u.sshExec.RunCommand(cmd)
		syncUC := NewSyncClusterStateUseCase(u.repo, u.sshExec)
		_ = syncUC.ExportStateToRemote()
	}

	return link, nil
}
