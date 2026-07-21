package usecases

import (
	"fmt"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
)

// UpdateServerUseCase se encarga de actualizar los paquetes del SO del servidor bajo demanda
type UpdateServerUseCase struct {
	ssh ports.SSHExecutor
}

// NewUpdateServerUseCase crea una nueva instancia
func NewUpdateServerUseCase(ssh ports.SSHExecutor) ports.UpdateServerUseCase {
	return &UpdateServerUseCase{
		ssh: ssh,
	}
}

func (uc *UpdateServerUseCase) Execute() error {
	fmt.Println("\n📦 [Update Server] Descargando y aplicando parches de seguridad (esto puede tomar unos minutos)...")

	cmd := "export DEBIAN_FRONTEND=noninteractive; apt-get update && apt-get upgrade -y && apt-get autoremove -y"
	res, err := uc.ssh.RunCommand(cmd)
	if err != nil || res.ExitCode != 0 {
		return fmt.Errorf("falló la actualización del sistema: %s", res.Output)
	}

	fmt.Println("✅ Servidor actualizado exitosamente.")
	return nil
}
