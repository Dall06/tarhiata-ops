package sys

import (
	"fmt"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
	"github.com/Dall06/tarhiata-ops/srv/sys/repositories"
)

type shellHandler struct {
	repo ports.ConfigRepository
}

func NewShellHandler(repo ports.ConfigRepository) ports.ShellHandler {
	return &shellHandler{repo: repo}
}

func (h *shellHandler) Execute(config domain.ServerConfig) {
	fmt.Println("\n💻 Abriendo túnel seguro interactivo (Escribe 'exit' para salir)...")
	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(config); err != nil {
		fmt.Printf("❌ Error conectando por SSH: %v\n", err)
		return
	}
	defer sshExec.Close()

	if err := sshExec.InteractiveShell(); err != nil {
		fmt.Printf("\nSesión terminada: %v\n", err)
	}
}
