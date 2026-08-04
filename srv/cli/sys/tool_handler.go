package sys

import (
	"fmt"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
	"github.com/Dall06/tarhiata-ops/srv/sys/repositories"
	"github.com/Dall06/tarhiata-ops/srv/sys/usecases"
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

					huh.NewOption("🔑 Probar / Validar Conexión SSH", "test_ssh"),
					huh.NewOption("📦 Actualizar dependencias del OS (⚠️ Peligro)", "update_os"),
					huh.NewOption("🔙 Volver", "back"),
				).
				Value(&action),
		),
	).Run()
	if err != nil || action == "back" {
		return
	}

	if action == "test_ssh" {
		fmt.Printf("\n⏳ Probando conexión SSH hacia %s:%d (Llave: %s, User: %s)...\n", config.Host, config.Port, config.PrivateKey, config.User)
		sshExec := repositories.NewCryptoSSHExecutor()
		if err := sshExec.Connect(config); err != nil {
			fmt.Printf("❌ Error al conectar por SSH: %v\n", err)
			return
		}
		defer sshExec.Close()

		res, err := sshExec.RunCommand("uname -a && docker --version")
		if err != nil || res.ExitCode != 0 {
			fmt.Printf("⚠️  SSH conectó pero ocurrió un error ejecutando comandos: %v\n", err)
			return
		}
		fmt.Printf("✅ ¡Conexión SSH y Docker verificados con éxito!\n💻 Servidor: %s\n", res.Output)
	}

	if action == "update_os" {
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
