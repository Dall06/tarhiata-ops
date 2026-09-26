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
	cmd := "docker service update --force tarhiata_proxy_traefik 2>&1 || docker service update --force traefik_traefik 2>&1 || docker service update --force tarhiata_traefik 2>&1"
	res, err := uc.sshExec.RunCommand(cmd)
	if err != nil || res == nil || res.ExitCode != 0 {
		out := ""
		if res != nil {
			out = res.Output
		}
		if out == "" && err != nil {
			out = err.Error()
		}
		return "", fmt.Errorf("error reiniciando proxy Traefik: %s", out)
	}
	output := strings.TrimSpace(res.Output)
	if output == "" {
		output = "✔ Proxy Ingress Traefik reiniciado y recargado exitosamente."
	}
	return output, nil
}
