package usecases

import (
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type bootstrapMasterServiceUseCase struct {
	repo     ports.ConfigRepository
	sshExec  ports.SSHExecutor
	linkUC   ports.LinkServicesUseCase
	unlinkUC ports.UnlinkServicesUseCase
	dbUC     ports.DeployDatabaseUseCase
	svcUC    ports.DeployServiceUseCase
}

func NewBootstrapMasterServiceUseCase(
	repo ports.ConfigRepository,
	sshExec ports.SSHExecutor,
	linkUC ports.LinkServicesUseCase,
	unlinkUC ports.UnlinkServicesUseCase,
	dbUC ports.DeployDatabaseUseCase,
	svcUC ports.DeployServiceUseCase,
) ports.BootstrapMasterServiceUseCase {
	return &bootstrapMasterServiceUseCase{
		repo:     repo,
		sshExec:  sshExec,
		linkUC:   linkUC,
		unlinkUC: unlinkUC,
		dbUC:     dbUC,
		svcUC:    svcUC,
	}
}

func (uc *bootstrapMasterServiceUseCase) Execute(input ports.BootstrapMasterInput, config domain.ServerConfig) (*ports.BootstrapMasterResult, error) {
	result := ports.BootstrapMasterResult{
		UnlinkedOld: []string{},
	}

	if strings.TrimSpace(input.AppName) == "" {
		return &result, fmt.Errorf("el nombre del servicio (AppName) es requerido")
	}

	if strings.TrimSpace(input.Image) == "" {
		input.Image = "node:18-alpine"
	}
	if input.Port <= 0 {
		input.Port = 80
	}
	if strings.TrimSpace(input.EnvVarName) == "" {
		input.EnvVarName = "DATABASE_URL"
	}

	// 1. Auto-desconectar (Unlink) de servicios o bases de datos antiguas si el servicio ya tenía enlaces
	if uc.repo != nil {
		existingLinks, err := uc.repo.GetServiceLinks()
		if err == nil {
			for _, l := range existingLinks {
				if l.SourceSvc == input.AppName {
					if uc.unlinkUC != nil {
						_ = uc.unlinkUC.Execute(l.SourceSvc, l.TargetSvc)
					}
					result.UnlinkedOld = append(result.UnlinkedOld, l.TargetSvc)
				}
			}
		}
	}

	// 2. Crear e Inicializar la Base de Datos si DBEngine != "none"
	var createdDB *domain.SavedDatabase
	if strings.ToLower(input.DBEngine) != "none" && strings.TrimSpace(input.DBEngine) != "" {
		dbName := fmt.Sprintf("%s-%s", strings.ToLower(input.DBEngine), input.AppName)
		db := domain.SavedDatabase{
			Name:         dbName,
			Engine:       strings.ToLower(input.DBEngine),
			Password:     "secretpass123",
			InternalPort: 5432,
			DeployType:   "manager",
		}

		if uc.dbUC != nil {
			if err := uc.dbUC.Execute(db, config); err != nil {
				return &result, fmt.Errorf("error desplegando base de datos %s: %w", dbName, err)
			}
		}
		if uc.repo != nil {
			_ = uc.repo.SaveDatabase(db)
		}
		createdDB = &db
		result.Database = createdDB
	}

	// 3. Crear e Inicializar el Servicio App
	svc := domain.SavedService{
		Name:        input.AppName,
		ImageSource: input.Image,
		Port:        input.Port,
		Domain:      input.Domain,
		Expose:      input.ExposePublic,
		EnableSSL:   input.ExposePublic,
	}

	deployCfg := domain.DeployConfig{
		ImageSource: input.Image,
		Port:        input.Port,
		Domain:      input.Domain,
		Expose:      input.ExposePublic,
		EnableSSL:   input.ExposePublic,
	}

	customSvc := domain.CustomService{
		Name:    svc.Name,
		EnvVars: make(map[string]string),
	}

	if uc.svcUC != nil {
		if err := uc.svcUC.Execute(customSvc, deployCfg); err != nil {
			return &result, fmt.Errorf("error desplegando app %s: %w", svc.Name, err)
		}
	}
	if uc.repo != nil {
		_ = uc.repo.SaveService(svc)
	}
	result.App = svc

	// 4. Auto-Interconectar (Link A -> B) e inyectar la variable de entorno
	if createdDB != nil && uc.linkUC != nil {
		link, err := uc.linkUC.Execute(svc.Name, createdDB.Name, input.EnvVarName)
		if err == nil {
			result.Link = &link
		}
	}

	return &result, nil
}
