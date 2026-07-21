package handlers

import (
	"fmt"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/repositories"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/usecases"
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
	var installTS bool
	var tsAuthKey string
	var exposeObs bool

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("¿Deseas instalar Tailscale (VPN) para gestionar el servidor de forma privada?").
				Value(&installTS),
		),
	).Run()
	if err != nil {
		return
	}

	if installTS {
		exposeObs = false
		huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Auth Key de Tailscale (Opcional, déjalo vacío para loguearte mediante URL en los logs)").
					Value(&tsAuthKey),
				huh.NewConfirm().
					Title("¿Deseas instalar el Stack de Observabilidad? (Estará protegido y oculto dentro de la VPN)").
					Value(&installObs),
			),
		).Run()
	} else {
		// Regla estricta: Si no hay VPN, NO instalamos observabilidad en el Bootstrapper.
		// El usuario debe ir explícitamente a la sección de herramientas si la quiere pública.
		installObs = false
		exposeObs = false
	}

	huh.NewForm(
		huh.NewGroup(
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

	if installTS {
		fmt.Println("🚀 Instalando Tailscale...")
		tsUC := usecases.NewInstallTailscaleUseCase(sshExec)
		if err := tsUC.Execute(tsAuthKey); err != nil {
			fmt.Printf("❌ Falló Tailscale: %v\n", err)
		}
	}

	if installObs {
		fmt.Println("🚀 Desplegando stack de Observabilidad...")
		obsUC := usecases.NewDeployObservabilityUseCase(sshExec)
		if err := obsUC.Execute(exposeObs); err != nil {
			fmt.Printf("❌ Falló Observabilidad: %v\n", err)
		}
	}

	fmt.Println("✅ ¡Servidor inicializado y protegido con éxito!")
	if installTS {
		fmt.Println("🌐 NOTA: Tailscale fue instalado. Si no proveíste AuthKey o la conexión falló, abre la 'Consola SSH' en el menú y corre 'tailscale up' para autenticarte.")
	}
	if installObs {
		fmt.Printf("📊 Portainer (Dashboard) disponible en: http://[IP_DE_TAILSCALE]:9000\n")
		fmt.Printf("📝 Dozzle (Logs en vivo) disponible en: http://[IP_DE_TAILSCALE]:8888\n")
	}
}
