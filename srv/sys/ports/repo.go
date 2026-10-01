package ports

import "github.com/Dall06/tarhiata-ops/srv/sys/domain"

// ConfigRepository define los métodos para persistir configuraciones locales del CLI.
type ConfigRepository interface {
	// SaveServerConfig guarda o actualiza la configuración de un servidor en el catálogo.
	SaveServerConfig(config domain.ServerConfig) error

	// GetServerConfig obtiene la configuración del servidor activo actualmente. Retorna nil si no existe.
	GetServerConfig() (*domain.ServerConfig, error)

	// --- Catálogo de Conexiones / Servidores Multi-Host ---
	GetAllServerConfigs() ([]domain.ServerConfig, error)
	GetServerConfigByName(name string) (*domain.ServerConfig, error)
	SetActiveServerConfig(name string) error
	DeleteServerConfig(name string) error

	// --- Catálogo de Servicios ---
	// serverName identifica al servidor del fleet (server_configs.name) dueño de la fila:
	// services/databases/service_links son multi-servidor, nunca globales.
	SaveService(svc domain.SavedService) error
	GetServices(serverName string) ([]domain.SavedService, error)
	GetService(name, serverName string) (*domain.SavedService, error)
	DeleteService(name, serverName string) error

	// --- Catálogo de Bases de Datos ---
	SaveDatabase(db domain.SavedDatabase) error
	GetDatabases(serverName string) ([]domain.SavedDatabase, error)
	GetDatabase(name, serverName string) (*domain.SavedDatabase, error)
	DeleteDatabase(name, serverName string) error

	// --- Observabilidad ---
	SaveObservability(obs domain.SavedObservability) error
	GetObservability() (*domain.SavedObservability, error)
	DeleteObservability() error

	// --- Interconexión de Servicios ---
	SaveServiceLink(link domain.ServiceLink) error
	GetServiceLinks(serverName string) ([]domain.ServiceLink, error)
	DeleteServiceLink(sourceSvc, targetSvc, serverName string) error

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
	GetMigrationFiles(dbName, serverName string) ([]domain.MigrationFile, error)
	DeleteMigrationFile(dbName, filename, serverName string) error
	RecordMigrationExecution(dbName, filename, serverName, status, logs string) error

	// --- Backups & Snapshots ---
	SaveBackup(backup domain.SavedBackup) error
	GetBackups(serverName string) ([]domain.SavedBackup, error)
	GetBackupByID(id int, serverName string) (*domain.SavedBackup, error)
	DeleteBackup(id int, serverName string) error

	// --- Logs de Auditoría Inmutables ---
	SaveAuditLog(log domain.AuditLog) error
	GetAuditLogs(limit int) ([]domain.AuditLog, error)

	// --- Configuración de Alertas Salientes ---
	SaveAlertSettings(settings domain.AlertSettings) error
	GetAlertSettings() (*domain.AlertSettings, error)

	// --- Historial de Despliegues y Rollback Arbitrario ---
	SaveDeploymentRecord(record domain.DeploymentRecord) error
	GetDeploymentHistory(serviceName string, limit int) ([]domain.DeploymentRecord, error)
	GetDeploymentRecordByID(id int) (*domain.DeploymentRecord, error)

	// Close cierra la conexión a la base de datos local.
	Close() error
}

// SSHExecutor define el contrato para interactuar con el servidor remoto.
type SSHExecutor interface {
	// Connect establece la conexión SSH con el servidor.
	Connect(config domain.ServerConfig) error

	// RunCommand ejecuta un comando de forma síncrona y devuelve el resultado.
	RunCommand(cmd string) (*domain.CommandResult, error)

	// RunCommandStreaming ejecuta un comando y entrega cada línea de salida a onLine a
	// medida que se produce (no al final, como RunCommand). Pensado para comandos
	// largos (ej. "docker build") donde se quiere mostrar progreso en vivo.
	RunCommandStreaming(cmd string, onLine func(line string)) (*domain.CommandResult, error)

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
	ProvisionNode(token string, nodeName string, region string, plan string) (domain.NodeProvisionResult, error)

	// DestroyNode destruye la infraestructura de un nodo por su nombre
	DestroyNode(token string, nodeName string) error
}

// DNSResolver resuelve un nombre de dominio a sus direcciones IP. Existe para desacoplar
// CheckDomainDNSUseCase de la resolución de red real y poder testearlo con datos falsos.
type DNSResolver interface {
	LookupHost(host string) ([]string, error)
}

// VultrClient consulta la API pública de Vultr (planes, regiones, llaves SSH). Existe
// para desacoplar los usecases de un *http.Client concreto y poder testearlos sin red real.
type VultrClient interface {
	GetPlans(apiKey string) ([]domain.VultrPlan, error)
	GetRegions(apiKey string) ([]domain.VultrRegion, error)
	GetSSHKeys(apiKey string) ([]domain.VultrSSHKey, error)
}
