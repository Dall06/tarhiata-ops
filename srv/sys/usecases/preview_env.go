package usecases

import (
	"fmt"
	"strings"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type ManagePreviewEnvUseCaseImpl struct {
	repo    ports.ConfigRepository
	sshExec ports.SSHExecutor
}

func NewManagePreviewEnvUseCase(repo ports.ConfigRepository, sshExec ports.SSHExecutor) *ManagePreviewEnvUseCaseImpl {
	return &ManagePreviewEnvUseCaseImpl{
		repo:    repo,
		sshExec: sshExec,
	}
}

func (uc *ManagePreviewEnvUseCaseImpl) Create(input ports.CreatePreviewEnvInput, config domain.ServerConfig) (*domain.SavedPreviewEnv, error) {
	if input.Image == "" && input.ImageSource != "" {
		input.Image = input.ImageSource
	}
	if input.LinkDBName == "" && input.LinkDbName != "" {
		input.LinkDBName = input.LinkDbName
	}
	if input.TargetNode == "" && input.NodeTarget != "" {
		input.TargetNode = input.NodeTarget
	}

	if strings.TrimSpace(input.Name) == "" {
		return nil, fmt.Errorf("el nombre del entorno preview es obligatorio")
	}
	if strings.TrimSpace(input.Image) == "" {
		return nil, fmt.Errorf("la imagen docker es obligatoria")
	}
	if input.Port <= 0 {
		input.Port = 80
	}

	serviceName := fmt.Sprintf("prev-%s", strings.TrimSpace(input.Name))

	// 1. Guardar en SQLite
	env := domain.SavedPreviewEnv{
		Name:        input.Name,
		ImageSource: input.Image,
		Port:        input.Port,
		Domain:      input.Domain,
		LinkDBName:  input.LinkDBName,
		Status:      "active",
		CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
		TargetNode:  input.TargetNode,
	}

	if uc.repo != nil {
		if err := uc.repo.SavePreviewEnv(env); err != nil {
			return nil, fmt.Errorf("error guardando entorno preview en SQLite: %w", err)
		}
	}

	// 2. Si hay ejecutor SSH activo, desplegar contenedor temporal en Docker Swarm
	if uc.sshExec != nil {
		var envVarFlags string
		if input.LinkDBName != "" && uc.repo != nil {
			// Buscar la BD para inyectar su URL
			if db, err := uc.repo.GetDatabase(input.LinkDBName); err == nil && db != nil {
				envURL := fmt.Sprintf("postgres://admin:secret@tarhiata-db-%s:%d/db", db.Name, db.InternalPort)
				envVarFlags = fmt.Sprintf(" --env DATABASE_URL='%s'", envURL)
			}
		}

		var traefikLabels string
		var portPublish string
		if input.Domain != "" {
			routerName := fmt.Sprintf("prev-%s", input.Name)
			traefikLabels = fmt.Sprintf(
				" --label traefik.enable=true"+
					" --label traefik.http.routers.%s.rule='Host(`%s`)'"+
					" --label traefik.http.routers.%s.entrypoints=web,websecure"+
					" --label traefik.http.services.%s.loadbalancer.server.port=%d",
				routerName, input.Domain, routerName, routerName, input.Port,
			)
		} else {
			portPublish = fmt.Sprintf(" --publish published=%d,target=%d", input.Port, input.Port)
		}

		constraint := formatNodeConstraint(input.TargetNode)
		nodeFlag := fmt.Sprintf(" --constraint '%s'", constraint)

		cmd := fmt.Sprintf(
			"docker service create --name %s --network tarhiata-overlay --replicas 1%s%s%s%s %s",
			serviceName, envVarFlags, traefikLabels, portPublish, nodeFlag, input.Image,
		)

		if _, err := uc.sshExec.RunCommand(cmd); err != nil {
			if uc.repo != nil {
				_ = uc.repo.DeletePreviewEnv(input.Name)
			}
			return nil, fmt.Errorf("falló el despliegue del entorno preview en Swarm: %w", err)
		}
	}

	return &env, nil
}

func (uc *ManagePreviewEnvUseCaseImpl) List() ([]domain.SavedPreviewEnv, error) {
	if uc.repo == nil {
		return []domain.SavedPreviewEnv{}, nil
	}
	envs, err := uc.repo.GetPreviewEnvs()
	if err != nil {
		return nil, err
	}
	if envs == nil {
		envs = []domain.SavedPreviewEnv{}
	}
	return envs, nil
}

func (uc *ManagePreviewEnvUseCaseImpl) Destroy(name string, config domain.ServerConfig) error {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return fmt.Errorf("el nombre del entorno preview a destruir es requerido")
	}

	serviceName := fmt.Sprintf("prev-%s", cleanName)

	// 1. Si hay ejecutor SSH, remover servicio de Docker Swarm
	if uc.sshExec != nil {
		_, _ = uc.sshExec.RunCommand(fmt.Sprintf("docker service rm %s", serviceName))
	}

	// 2. Eliminar registro de SQLite
	if uc.repo != nil {
		if err := uc.repo.DeletePreviewEnv(cleanName); err != nil {
			return fmt.Errorf("error borrando entorno preview de SQLite: %w", err)
		}
	}

	return nil
}
