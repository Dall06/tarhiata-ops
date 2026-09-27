package usecases

import (
	"fmt"
	"log/slog"

	"github.com/Dall06/tarhiata-ops/srv/cli/ports"
	sysdomain "github.com/Dall06/tarhiata-ops/srv/sys/domain"
	sysrepositories "github.com/Dall06/tarhiata-ops/srv/sys/repositories"
	sysusecases "github.com/Dall06/tarhiata-ops/srv/sys/usecases"
	"github.com/charmbracelet/huh"
)

type bootstrapHandler struct {
	repo ports.ConfigRepository
}

// NewBootstrapHandler inicializa el caso de uso de inicialización de clúster para CLI.
func NewBootstrapHandler(repo ports.ConfigRepository) ports.BootstrapHandler {
	return &bootstrapHandler{repo: repo}
}

func (h *bootstrapHandler) Execute(config sysdomain.ServerConfig) {
	var installObs bool
	var acmeEmail string
	errForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("¿Deseas desplegar el Stack de Observabilidad (Portainer / Dozzle)?").
				Value(&installObs),
			huh.NewInput().
				Title("Correo para Let's Encrypt (SSL Automático). Déjalo vacío si no usarás dominios públicos.").
				Value(&acmeEmail),
		),
	).Run()
	if errForm != nil {
		slog.Warn("cli: formulario bootstrap cancelado o con error", "error", errForm)
		return
	}

	fmt.Println("\n⏳ Conectando al servidor para inicializar Bootstrapper...")
	sshExec := sysrepositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(config); err != nil {
		fmt.Printf("❌ Error conectando por SSH: %v\n", err)
		return
	}
	defer func() {
		if errClose := sshExec.Close(); errClose != nil {
			slog.Warn("cli: error cerrando ssh en bootstrap", "error", errClose)
		}
	}()

	initServerUC := sysusecases.NewInitServerUseCase(sshExec)
	fmt.Println("🚀 Ejecutando inicialización (Docker, Swarm, Firewall, Traefik)...")

	if err := initServerUC.Execute(acmeEmail); err != nil {
		fmt.Printf("❌ Falló la inicialización base: %v\n", err)
		return
	}

	if installObs {
		fmt.Println("🚀 Desplegando stack de Observabilidad...")
		obsUC := sysusecases.NewDeployObservabilityUseCase(sshExec)
		if err := obsUC.Execute(true); err != nil {
			fmt.Printf("❌ Falló Observabilidad: %v\n", err)
		}
	}

	fmt.Println("✅ ¡Servidor inicializado con éxito!")
}
