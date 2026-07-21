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
					huh.NewOption("📈 Instalar Observabilidad PÚBLICA (Portainer/Dozzle sin VPN - ⚠️ Inseguro)", "obs_public"),
					huh.NewOption("📦 Actualizar dependencias del OS (⚠️ Peligro)", "update_os"),
					huh.NewOption("🔙 Volver", "back"),
				).
				Value(&action),
		),
	).Run()
	if err != nil || action == "back" {
		return
	}

	if action == "obs_public" {
		var confirm bool
		huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("⚠️ ¿Estás completamente seguro? Cualquiera con tu IP podrá ver el Login de Portainer y Dozzle").Value(&confirm))).Run()
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

		obsUC := usecases.NewDeployObservabilityUseCase(sshExec)
		fmt.Println("🚀 Desplegando Observabilidad Pública...")
		if err := obsUC.Execute(true); err != nil {
			fmt.Println("❌ Error:", err)
		} else {
			fmt.Printf("✅ Observabilidad Pública Instalada exitosamente.\n")
			fmt.Printf("📊 Portainer: http://%s:9000\n", config.Host)
			fmt.Printf("📝 Dozzle: http://%s:8888\n", config.Host)
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
