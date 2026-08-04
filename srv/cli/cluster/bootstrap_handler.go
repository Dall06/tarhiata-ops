package cluster

import (
	"fmt"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
	"github.com/Dall06/tarhiata-ops/srv/sys/repositories"
	"github.com/Dall06/tarhiata-ops/srv/sys/usecases"
	"github.com/charmbracelet/huh"
)

type bootstrapHandler struct {
	repo ports.ConfigRepository
}

func NewBootstrapHandler(repo ports.ConfigRepository) ports.BootstrapHandler {
	return &bootstrapHandler{repo: repo}
}

func (h *bootstrapHandler) Execute(config domain.ServerConfig) {
	var installObs bool
	var acmeEmail string
	huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("¿Deseas desplegar el Stack de Observabilidad (Portainer / Dozzle)?").
				Value(&installObs),
			huh.NewInput().
				Title("Correo para Let's Encrypt (SSL Automático). Déjalo vacío si no usarás dominios públicos.").
				Value(&acmeEmail),
		),
	).Run()

	fmt.Println("\n⏳ Conectando al servidor para inicializar Bootstrapper...")
	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(config); err != nil {
		fmt.Printf("❌ Error conectando por SSH: %v\n", err)
		return
	}
	defer sshExec.Close()

	initServerUC := usecases.NewInitServerUseCase(sshExec)
	fmt.Println("🚀 Ejecutando inicialización (Docker, Swarm, Firewall, Traefik)...")

	if err := initServerUC.Execute(acmeEmail); err != nil {
		fmt.Printf("❌ Falló la inicialización base: %v\n", err)
		return
	}

	if installObs {
		fmt.Println("🚀 Desplegando stack de Observabilidad...")
		obsUC := usecases.NewDeployObservabilityUseCase(sshExec)
		if err := obsUC.Execute(true); err != nil {
			fmt.Printf("❌ Falló Observabilidad: %v\n", err)
		}
	}

	fmt.Println("✅ ¡Servidor inicializado con éxito!")
}
