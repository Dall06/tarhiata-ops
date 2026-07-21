package ports

import "github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"

// ConfigRepository define los métodos para persistir configuraciones locales del CLI.
type ConfigRepository interface {
	// SaveServerConfig guarda la configuración principal del servidor activo.
	SaveServerConfig(config domain.ServerConfig) error

	// GetServerConfig obtiene la configuración del servidor activo. Retorna nil si no existe.
	GetServerConfig() (*domain.ServerConfig, error)

	// --- Catálogo de Servicios ---
	SaveService(svc domain.SavedService) error
	GetServices() ([]domain.SavedService, error)
	GetService(name string) (*domain.SavedService, error)
	DeleteService(name string) error

	// --- Catálogo de Bases de Datos ---
	SaveDatabase(db domain.SavedDatabase) error
	GetDatabases() ([]domain.SavedDatabase, error)
	GetDatabase(name string) (*domain.SavedDatabase, error)
	DeleteDatabase(name string) error

	// --- Observabilidad ---
	SaveObservability(obs domain.SavedObservability) error
	GetObservability() (*domain.SavedObservability, error)
	DeleteObservability() error

	// Close cierra la conexión a la base de datos local.
	Close() error
}

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

// Provisioner define el contrato para herramientas de IaC (Terraform)
type Provisioner interface {
	// ProvisionNode crea o actualiza un nodo (Droplet/EC2) y retorna su IP Pública y la llave privada SSH generada
	ProvisionNode(token string, nodeName string, region string) (string, string, error)

	// DestroyNode destruye la infraestructura de un nodo por su nombre
	DestroyNode(token string, nodeName string) error
}
