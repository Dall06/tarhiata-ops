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

type toolHandler struct {
	repo ports.ConfigRepository
}

// NewToolHandler crea el caso de uso para herramientas auxiliares y diagnósticos en CLI.
func NewToolHandler(repo ports.ConfigRepository) ports.ToolHandler {
	return &toolHandler{repo: repo}
}

func (h *toolHandler) Execute(config sysdomain.ServerConfig) {
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
		sshExec := sysrepositories.NewCryptoSSHExecutor()
		if errConn := sshExec.Connect(config); errConn != nil {
			fmt.Printf("❌ Error al conectar por SSH: %v\n", errConn)
			return
		}
		defer func() {
			if errClose := sshExec.Close(); errClose != nil {
				slog.Warn("cli: error cerrando ssh en tool test_ssh", "error", errClose)
			}
		}()

		res, errCmd := sshExec.RunCommand("uname -a && docker --version")
		if errCmd != nil || res.ExitCode != 0 {
			fmt.Printf("⚠️  SSH conectó pero ocurrió un error ejecutando comandos: %v\n", errCmd)
			return
		}
		fmt.Printf("✅ ¡Conexión SSH y Docker verificados con éxito!\n💻 Servidor: %s\n", res.Output)
		return
	}

	if action == "update_os" {
		var confirm bool
		if errPrompt := huh.NewForm(huh.NewGroup(
			huh.NewConfirm().
				Title("⚠️ ¡PELIGRO! ¿Estás seguro de actualizar el SO?\nEsto descargará nuevas dependencias sin contexto y podría romper Docker o contenedores en ejecución.\n¿Continuar bajo tu propio riesgo?").
				Value(&confirm),
		)).Run(); errPrompt != nil {
			slog.Warn("cli: cancelado prompt update_os", "error", errPrompt)
			return
		}
		if !confirm {
			return
		}

		fmt.Println("\n⏳ Conectando al servidor...")
		sshExec := sysrepositories.NewCryptoSSHExecutor()
		if errConn := sshExec.Connect(config); errConn != nil {
			fmt.Printf("❌ Error SSH: %v\n", errConn)
			return
		}
		defer func() {
			if errClose := sshExec.Close(); errClose != nil {
				slog.Warn("cli: error cerrando ssh en tool update_os", "error", errClose)
			}
		}()

		updateUC := sysusecases.NewUpdateServerUseCase(sshExec)
		if errUpd := updateUC.Execute(); errUpd != nil {
			fmt.Printf("❌ Error: %v\n", errUpd)
		}
	}
}
