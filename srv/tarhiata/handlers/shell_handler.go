package handlers

import (
	"fmt"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/repositories"
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
