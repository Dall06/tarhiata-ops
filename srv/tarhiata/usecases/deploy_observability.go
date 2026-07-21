package usecases

import (
	"fmt"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
)

type DeployObservabilityUseCase struct {
	ssh ports.SSHExecutor
}

func NewDeployObservabilityUseCase(ssh ports.SSHExecutor) ports.DeployObservabilityUseCase {
	return &DeployObservabilityUseCase{ssh: ssh}
}

// Execute despliega Portainer (Dashboard) y Dozzle (Logs web ultra-ligeros)
func (uc *DeployObservabilityUseCase) Execute(exposePublic bool) error {
	if exposePublic {
		// El usuario decidió exponerlo públicamente bajo su propio riesgo
		uc.ssh.RunCommand("ufw allow 9000/tcp && ufw allow 8888/tcp")
	} else {
		// Bloqueamos explícitamente el acceso público a estos puertos usando la cadena DOCKER-USER
		// ya que Docker bypassea UFW. Solo se podrá acceder por Tailscale (tailscale0) o localhost (lo).
		blockCmd := `EXT_IF=$(ip route get 8.8.8.8 | awk '{print $5; exit}') && iptables -I DOCKER-USER -i $EXT_IF -p tcp -m multiport --dports 9000,8888 -j DROP`
		uc.ssh.RunCommand(blockCmd)
	}

	compose := `version: '3.8'
services:
  portainer:
    image: portainer/portainer-ce:latest
    ports:
      - "9000:9000"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - portainer_data:/data
    deploy:
      placement:
        constraints: [node.role == manager]

  dozzle:
    image: amir20/dozzle:latest
    ports:
      - "8888:8080"
    environment:
      - DOZZLE_LEVEL=info
      - DOZZLE_TAIL_SIZE=300
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    deploy:
      placement:
        constraints: [node.role == manager]

volumes:
  portainer_data:
`
	
	writeCmd := fmt.Sprintf("cat << 'EOF' > /tmp/observability-stack.yml\n%s\nEOF", compose)
	if _, err := uc.ssh.RunCommand(writeCmd); err != nil {
		return fmt.Errorf("falló al escribir observability compose: %w", err)
	}

	res, err := uc.ssh.RunCommand("docker stack deploy -c /tmp/observability-stack.yml tarhiata_obs")
	if err != nil || res.ExitCode != 0 {
		return fmt.Errorf("falló al desplegar observabilidad: %s", res.Output)
	}

	return nil
}
