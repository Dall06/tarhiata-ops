package usecases

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/Dall06/tarhiata-ops/pkg/validator"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

// envVarNameRegex exige un nombre de variable de entorno POSIX válido, rechazando
// espacios/`;`/etc. que permitirían escapar de la posición de nombre en el comando remoto.
var envVarNameRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type ManageEnvVarsUseCase struct {
	repo ports.ConfigRepository
	ssh  ports.SSHExecutor
}

func NewManageEnvVarsUseCase(repo ports.ConfigRepository, ssh ports.SSHExecutor) *ManageEnvVarsUseCase {
	return &ManageEnvVarsUseCase{repo: repo, ssh: ssh}
}

// ParseEnvContent convierte una cadena multilínea estilo .env a un mapa clave-valor.
func ParseEnvContent(content string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			// Quitar comillas si las tiene
			v = strings.Trim(v, `"'`)
			if k != "" {
				result[k] = v
			}
		}
	}
	return result
}

// FormatEnvMap convierte un mapa clave-valor a formato de texto multilínea .env.
func FormatEnvMap(envMap map[string]string) string {
	var builder strings.Builder
	for k, v := range envMap {
		builder.WriteString(fmt.Sprintf("%s=%s\n", k, v))
	}
	return builder.String()
}

// EnvVarsData encapsula el contenido crudo y el mapa parseado de variables de entorno.
type EnvVarsData struct {
	Raw string
	Map map[string]string
}

func (uc *ManageEnvVarsUseCase) GetEnvVars(serviceName, serverName string) (*EnvVarsData, error) {
	svc, err := uc.repo.GetService(serviceName, serverName)
	if err != nil || svc == nil {
		return &EnvVarsData{
			Raw: "",
			Map: make(map[string]string),
		}, nil
	}
	envMap := ParseEnvContent(svc.EnvVars)
	return &EnvVarsData{
		Raw: svc.EnvVars,
		Map: envMap,
	}, nil
}

func (uc *ManageEnvVarsUseCase) UpdateEnvVars(serviceName string, rawEnvContent string, config domain.ServerConfig) error {
	svc, err := uc.repo.GetService(serviceName, config.Name)
	if err != nil || svc == nil {
		svc = &domain.SavedService{
			Name: serviceName,
		}
	}

	// 1. Guardar en SQLite
	svc.EnvVars = rawEnvContent
	if err := uc.repo.SaveService(*svc); err != nil {
		return fmt.Errorf("error al actualizar variables de entorno en sqlite: %w", err)
	}

	// 2. Si SSH está disponible y hay un host configurado, actualizar el servicio Swarm en vivo
	if config.Host != "" && uc.ssh != nil {
		if err := uc.ssh.Connect(config); err == nil {
			defer uc.ssh.Close()

			envMap := ParseEnvContent(rawEnvContent)
			if len(envMap) > 0 {
				var envFlags []string
				for k, v := range envMap {
					if !envVarNameRegex.MatchString(k) {
						slog.Warn("manage_envs: nombre de variable de entorno inválido, se omite", "key", k)
						continue
					}
					envFlags = append(envFlags, "--env-add "+validator.ShellQuote(k+"="+v))
				}
				flagsStr := strings.Join(envFlags, " ")
				cmd := fmt.Sprintf("docker service update %s %s 2>/dev/null || docker service update %s %s_%s 2>/dev/null || docker service update %s tarhiata-app-%s 2>/dev/null || true",
					flagsStr, svc.Name, flagsStr, svc.Name, svc.Name, flagsStr, svc.Name)
				if _, errRun := uc.ssh.RunCommand(cmd); errRun != nil {
					slog.Warn("falló ejecución de actualización de env vars en swarm", "error", errRun)
				}
			}
		}
	}

	return nil
}
