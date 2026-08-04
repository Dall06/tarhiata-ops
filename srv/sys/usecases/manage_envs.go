package usecases

import (
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

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

func (uc *ManageEnvVarsUseCase) GetEnvVars(serviceName string) (string, map[string]string, error) {
	svc, err := uc.repo.GetService(serviceName)
	if err != nil || svc == nil {
		return "", nil, fmt.Errorf("servicio '%s' no encontrado en el catálogo", serviceName)
	}
	envMap := ParseEnvContent(svc.EnvVars)
	return svc.EnvVars, envMap, nil
}

func (uc *ManageEnvVarsUseCase) UpdateEnvVars(serviceName string, rawEnvContent string, config domain.ServerConfig) error {
	svc, err := uc.repo.GetService(serviceName)
	if err != nil || svc == nil {
		return fmt.Errorf("servicio '%s' no encontrado", serviceName)
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
					// Escapar comillas para la terminal bash
					escapedVal := strings.ReplaceAll(v, `"`, `\"`)
					envFlags = append(envFlags, fmt.Sprintf("--env-add %s=\"%s\"", k, escapedVal))
				}
				flagsStr := strings.Join(envFlags, " ")
				cmd := fmt.Sprintf("docker service update %s %s 2>/dev/null || docker service update %s %s_%s 2>/dev/null || docker service update %s tarhiata-app-%s 2>/dev/null || true",
					flagsStr, svc.Name, flagsStr, svc.Name, svc.Name, flagsStr, svc.Name)
				uc.ssh.RunCommand(cmd)
			}
		}
	}

	return nil
}
