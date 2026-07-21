package usecases

import (
	"fmt"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
)

type InstallTailscaleUseCase struct {
	ssh ports.SSHExecutor
}

func NewInstallTailscaleUseCase(ssh ports.SSHExecutor) ports.InstallTailscaleUseCase {
	return &InstallTailscaleUseCase{ssh: ssh}
}

// Execute instala y levanta Tailscale en el nodo
func (uc *InstallTailscaleUseCase) Execute(authKey string) error {
	res, _ := uc.ssh.RunCommand("command -v tailscale")
	if res.ExitCode != 0 {
		_, err := uc.ssh.RunCommand("curl -fsSL https://tailscale.com/install.sh | sh")
		if err != nil {
			return err
		}
	}
	
	if authKey != "" {
		res, err := uc.ssh.RunCommand("tailscale up --authkey=" + authKey)
		if err != nil || res.ExitCode != 0 {
			return fmt.Errorf("falló al levantar tailscale: %s", res.Output)
		}
	}
	return nil
}
