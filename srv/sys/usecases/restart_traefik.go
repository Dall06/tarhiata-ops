package usecases

import (
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type RestartTraefikUseCase struct {
	sshExec ports.SSHExecutor
}

func NewRestartTraefikUseCase(sshExec ports.SSHExecutor) *RestartTraefikUseCase {
	return &RestartTraefikUseCase{sshExec: sshExec}
}

func (uc *RestartTraefikUseCase) Execute() (string, error) {
	res, err := uc.sshExec.RunCommand("docker service update --force traefik_traefik")
	if err != nil {
		return "", fmt.Errorf("error reiniciando proxy Traefik: %w", err)
	}
	output := strings.TrimSpace(res.Output)
	if output == "" {
		output = "✔ Proxy Ingress Traefik reiniciado y recargado exitosamente."
	}
	return output, nil
}
