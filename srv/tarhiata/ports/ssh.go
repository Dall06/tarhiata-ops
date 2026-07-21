package ports

import "github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"

// SSHExecutor define el contrato para interactuar con el servidor remoto.
type SSHExecutor interface {
	// Connect establece la conexión SSH con el servidor.
	Connect(config domain.ServerConfig) error

	// RunCommand ejecuta un comando de forma síncrona y devuelve el resultado.
	RunCommand(cmd string) (*domain.CommandResult, error)

	// InteractiveShell abre una consola interactiva en el servidor.
	InteractiveShell() error
	
	// InteractiveCommand ejecuta un comando específico con PTY interactivo (ej. para seguir logs)
	InteractiveCommand(cmd string) error

	// CheckConnection verifica si la conexión sigue viva. Útil para monitoreo asíncrono.
	CheckConnection() bool

	// Close cierra la conexión SSH de forma segura.
	Close() error
}
