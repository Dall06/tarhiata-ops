package ports

import "github.com/Dall06/tarhiata-ops/srv/sys/domain"

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

	// --- Interconexión de Servicios ---
	SaveServiceLink(link domain.ServiceLink) error
	GetServiceLinks() ([]domain.ServiceLink, error)
	DeleteServiceLink(sourceSvc, targetSvc string) error

	// --- Entornos de Preview Temporales ---
	SavePreviewEnv(env domain.SavedPreviewEnv) error
	GetPreviewEnvs() ([]domain.SavedPreviewEnv, error)
	GetPreviewEnv(name string) (*domain.SavedPreviewEnv, error)
	DeletePreviewEnv(name string) error

	// --- Credenciales de Docker Registries Privados ---
	SaveRegistryCredential(cred domain.SavedRegistryCredential) error
	GetRegistryCredentials() ([]domain.SavedRegistryCredential, error)
	GetRegistryCredential(server string) (*domain.SavedRegistryCredential, error)
	DeleteRegistryCredential(server string) error

	// --- Gestor de Migraciones de BD ---
	SaveMigrationFile(file domain.MigrationFile) error
	GetMigrationFiles(dbName string) ([]domain.MigrationFile, error)
	DeleteMigrationFile(dbName, filename string) error
	RecordMigrationExecution(dbName, filename, status, logs string) error

	// --- Backups & Snapshots ---
	SaveBackup(backup domain.SavedBackup) error
	GetBackups() ([]domain.SavedBackup, error)
	GetBackupByID(id int) (*domain.SavedBackup, error)
	DeleteBackup(id int) error

	// --- Logs de Auditoría Inmutables ---
	SaveAuditLog(log domain.AuditLog) error
	GetAuditLogs(limit int) ([]domain.AuditLog, error)

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

	// WriteRemoteFile escribe un archivo en el servidor remoto de forma segura usando Base64.
	WriteRemoteFile(remotePath, content string) error

	// CheckConnection verifica si la conexión sigue viva. Útil para monitoreo asíncrono.
	CheckConnection() bool

	// Close cierra la conexión SSH de forma segura.
	Close() error
}

// Provisioner define el contrato para herramientas de IaC (Terraform)
type Provisioner interface {
	// ProvisionNode crea o actualiza un nodo (Droplet/EC2) y retorna su IP Pública y la llave privada SSH generada
	ProvisionNode(token string, nodeName string, region string, plan string) (string, string, error)

	// DestroyNode destruye la infraestructura de un nodo por su nombre
	DestroyNode(token string, nodeName string) error
}
