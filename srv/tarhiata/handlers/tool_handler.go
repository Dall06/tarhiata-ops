package handlers

import (
	"fmt"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/repositories"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/usecases"
	"github.com/charmbracelet/huh"
)

type toolHandler struct {
	repo ports.ConfigRepository
}

func NewToolHandler(repo ports.ConfigRepository) ports.ToolHandler {
	return &toolHandler{repo: repo}
}

func (h *toolHandler) Execute(config domain.ServerConfig) {
	var action string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("🛠️ Herramientas Adicionales").
				Options(
					huh.NewOption("📈 Instalar Observabilidad Ligera (Portainer + Dozzle)", "obs_light"),
					huh.NewOption("📊 Instalar Observabilidad Persistente (Portainer + Grafana + Loki + Promtail)", "obs_persist"),
					huh.NewOption("📦 Actualizar dependencias del OS (⚠️ Peligro)", "update_os"),
					huh.NewOption("🔙 Volver", "back"),
				).
				Value(&action),
		),
	).Run()
	if err != nil || action == "back" {
		return
	}

	if action == "obs_light" || action == "obs_persist" {
		var exposePublic bool
		huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("⚠️ ¿Exponer los paneles al internet público? (Inseguro, se recomienda mantener privado y usar Tailscale)").Value(&exposePublic))).Run()

		fmt.Println("\n⏳ Conectando al servidor...")
		sshExec := repositories.NewCryptoSSHExecutor()
		if err := sshExec.Connect(config); err != nil {
			fmt.Println("❌ Error SSH:", err)
			return
		}
		defer sshExec.Close()

		obsUC := usecases.NewDeployObservabilityUseCase(sshExec)
		fmt.Println("🚀 Desplegando Stack de Observabilidad...")

		var err error
		if action == "obs_persist" {
			err = obsUC.ExecutePersistent(exposePublic)
		} else {
			err = obsUC.Execute(exposePublic)
		}

		if err != nil {
			fmt.Println("❌ Error:", err)
		} else {
			fmt.Printf("✅ Observabilidad Instalada exitosamente.\n")
			fmt.Printf("👉 Portainer: http://%s:9000\n", config.Host)
			if action == "obs_persist" {
				fmt.Printf("👉 Grafana: http://%s:3001 (User: admin / Pass: admin)\n", config.Host)
			} else {
				fmt.Printf("👉 Dozzle: http://%s:8888\n", config.Host)
			}
		}
	} else if action == "update_os" {
		var confirm bool
		huh.NewForm(huh.NewGroup(
			huh.NewConfirm().
				Title("⚠️ ¡PELIGRO! ¿Estás seguro de actualizar el SO?\nEsto descargará nuevas dependencias sin contexto y podría romper Docker o contenedores en ejecución.\n¿Continuar bajo tu propio riesgo?").
				Value(&confirm),
		)).Run()
		if !confirm {
			return
		}

		fmt.Println("\n⏳ Conectando al servidor...")
		sshExec := repositories.NewCryptoSSHExecutor()
		if err := sshExec.Connect(config); err != nil {
			fmt.Println("❌ Error SSH:", err)
			return
		}
		defer sshExec.Close()

		updateUC := usecases.NewUpdateServerUseCase(sshExec)
		if err := updateUC.Execute(); err != nil {
			fmt.Println("❌ Error:", err)
		}
	}
}
