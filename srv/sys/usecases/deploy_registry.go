package usecases

import (
	"fmt"

	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

// RegistryStackName es el nombre del stack Swarm del registry Docker privado interno.
const RegistryStackName = "tarhiata_registry"

// RegistryInternalAddress es la dirección (sin esquema) a la que los nodos del Swarm
// deben poder hacer pull/push dentro de la red overlay interna. No lleva TLS porque
// nunca se expone fuera de esa red.
const RegistryInternalAddress = "tarhiata-registry:5000"

type DeployRegistryUseCase struct {
	ssh ports.SSHExecutor
}

func NewDeployRegistryUseCase(ssh ports.SSHExecutor) *DeployRegistryUseCase {
	return &DeployRegistryUseCase{ssh: ssh}
}

// Execute despliega (o actualiza, es idempotente) un registry Docker privado propio
// (registry:2) en la red interna del Swarm, para que build-from-source pueda publicar
// imágenes que cualquier nodo del cluster pueda luego hacer pull.
func (uc *DeployRegistryUseCase) Execute() error {
	compose := `version: '3.8'
services:
  registry:
    image: registry:2
    networks:
      tarhiata_internal:
        aliases:
          - tarhiata-registry
    volumes:
      - tarhiata_registry_data:/var/lib/registry
    deploy:
      placement:
        constraints: [node.role == manager]

networks:
  tarhiata_internal:
    external: true

volumes:
  tarhiata_registry_data:
`

	writeCmd := fmt.Sprintf("cat << 'EOF' > /tmp/registry-stack.yml\n%s\nEOF", compose)
	if _, err := uc.ssh.RunCommand(writeCmd); err != nil {
		return fmt.Errorf("falló al escribir compose del registry: %w", err)
	}

	res, err := uc.ssh.RunCommand("docker stack deploy -c /tmp/registry-stack.yml " + RegistryStackName)
	if err != nil || res == nil || res.ExitCode != 0 {
		out := ""
		if res != nil {
			out = res.Output
		}
		if out == "" && err != nil {
			out = err.Error()
		}
		return fmt.Errorf("falló al desplegar el registry privado: %s", out)
	}

	return nil
}
