package usecases

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Dall06/tarhiata-ops/pkg/validator"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

// generateDBPassword crea una contraseña aleatoria de 32 caracteres hexadecimales,
// única por despliegue (a diferencia de un valor estático compartido entre clientes).
func generateDBPassword() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("error generando contraseña de base de datos: %w", err)
	}
	return hex.EncodeToString(b), nil
}

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
	if !validator.IsIdentifier(strings.TrimSpace(input.AppName)) {
		return &result, fmt.Errorf("el nombre del servicio (AppName) solo puede contener letras, números, puntos, guiones y guiones bajos")
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
		existingLinks, err := uc.repo.GetServiceLinks(config.Name)
		if err == nil {
			for _, l := range existingLinks {
				if l.SourceSvc == input.AppName {
					if uc.unlinkUC != nil {
						if errUnlink := uc.unlinkUC.Execute(l.SourceSvc, l.TargetSvc, config.Name); errUnlink != nil {
							slog.Warn("Fallo al desvincular enlace previo", "source", l.SourceSvc, "target", l.TargetSvc, "error", errUnlink)
						}
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
		dbPassword, errPwd := generateDBPassword()
		if errPwd != nil {
			return &result, errPwd
		}
		db := domain.SavedDatabase{
			Name:         dbName,
			Engine:       strings.ToLower(input.DBEngine),
			Password:     dbPassword,
			InternalPort: 5432,
			DeployType:   "manager",
			ServerName:   config.Name,
		}

		if uc.dbUC != nil {
			if err := uc.dbUC.Execute(db, config); err != nil {
				return &result, fmt.Errorf("error desplegando base de datos %s: %w", dbName, err)
			}
		}
		if uc.repo != nil {
			if errSave := uc.repo.SaveDatabase(db); errSave != nil {
				return &result, fmt.Errorf("error guardando base de datos en repositorio: %w", errSave)
			}
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
		ServerName:  config.Name,
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
		if errSave := uc.repo.SaveService(svc); errSave != nil {
			return &result, fmt.Errorf("error guardando servicio en repositorio: %w", errSave)
		}
	}
	result.App = svc

	// 4. Auto-Interconectar (Link A -> B) e inyectar la variable de entorno
	if createdDB != nil && uc.linkUC != nil {
		link, err := uc.linkUC.Execute(svc.Name, createdDB.Name, input.EnvVarName, config.Name)
		if err == nil {
			result.Link = &link
		}
	}

	return &result, nil
}
