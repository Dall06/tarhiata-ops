package repositories

import (
	"github.com/Dall06/tarhiata-ops/pkg/sshclient"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
)

// CryptoSSHExecutor implementa la interfaz ports.SSHExecutor utilizando nuestro paquete genérico sshclient.
type CryptoSSHExecutor struct {
	client *sshclient.Client
}

// NewCryptoSSHExecutor crea una nueva instancia del adaptador
func NewCryptoSSHExecutor() *CryptoSSHExecutor {
	return &CryptoSSHExecutor{
		client: sshclient.New(),
	}
}

// Connect establece la conexión SSH segura.
func (e *CryptoSSHExecutor) Connect(config domain.ServerConfig) error {
	return e.client.Connect(config.Host, config.User, config.PrivateKey, config.Port)
}

// RunCommand ejecuta un comando de forma síncrona y captura la salida.
func (e *CryptoSSHExecutor) RunCommand(cmd string) (*domain.CommandResult, error) {
	out, exitCode, err := e.client.RunCommand(cmd)

	result := &domain.CommandResult{
		Output:   out,
		ExitCode: exitCode,
		Error:    err,
	}

	return result, nil
}

// InteractiveShell abre una consola PTY interactiva conectada a la terminal local del usuario.
func (e *CryptoSSHExecutor) InteractiveShell() error {
	return e.client.InteractiveShell()
}

// InteractiveCommand ejecuta un comando específico pero manteniendo la terminal PTY interactiva conectada
func (e *CryptoSSHExecutor) InteractiveCommand(cmd string) error {
	return e.client.InteractiveCommand(cmd)
}

// CheckConnection hace un "ping" silencioso para la goroutine de estatus.
func (e *CryptoSSHExecutor) CheckConnection() bool {
	return e.client.CheckConnection()
}

// Close finaliza la conexión.
func (e *CryptoSSHExecutor) Close() error {
	return e.client.Close()
}
