package usecases

import (
	"fmt"
	"log/slog"

	"github.com/Dall06/tarhiata-ops/srv/cli/ports"
	sysdomain "github.com/Dall06/tarhiata-ops/srv/sys/domain"
	sysrepositories "github.com/Dall06/tarhiata-ops/srv/sys/repositories"
)

type shellHandler struct {
	repo ports.ConfigRepository
}

// NewShellHandler inicializa el caso de uso para abrir una shell interactiva por SSH.
func NewShellHandler(repo ports.ConfigRepository) ports.ShellHandler {
	return &shellHandler{repo: repo}
}

func (h *shellHandler) Execute(config sysdomain.ServerConfig) {
	fmt.Println("\n💻 Abriendo túnel seguro interactivo (Escribe 'exit' para salir)...")
	sshExec := sysrepositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(config); err != nil {
		fmt.Printf("❌ Error conectando por SSH: %v\n", err)
		return
	}
	defer func() {
		if errClose := sshExec.Close(); errClose != nil {
			slog.Warn("cli: error cerrando ssh en shell handler", "error", errClose)
		}
	}()

	if err := sshExec.InteractiveShell(); err != nil {
		fmt.Printf("\nSesión terminada: %v\n", err)
	}
}
