package domain

import (
	"time"
)

// ServerConfig contiene los datos y credenciales necesarios para conectar a un nodo o servidor.
type ServerConfig struct {
	ID            int    `json:"id,omitempty"`
	Name          string `json:"name"`          // Alias identificador (ej: "local", "vps-prod")
	Host          string `json:"host"`
	Port          int    `json:"port"`
	User          string `json:"user"`
	PrivateKey    string `json:"privateKey"`    // Ruta a la llave SSH o contenido raw (ej: ~/.ssh/id_rsa)
	DOAPIToken    string `json:"doApiToken"`   // Token de API Vultr/Cloud (Para Terraform)
	VultrAPIToken string `json:"vultrApiToken"` // Vultr API Key (Para Terraform)
	CloudProvider string `json:"cloudProvider"` // "vultr", "custom" o "local"
	IsActive      bool   `json:"isActive"`      // Indica si es la conexión predeterminada/activa
}

// SavedService representa la configuración y estado persistido de un servicio en el catálogo local.
type SavedService struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	ImageSource    string `json:"imageSource"`
	IsURL          bool   `json:"isUrl"`
	Port           int    `json:"port"`
	Domain         string `json:"domain"`
	Expose         bool   `json:"expose"`
	EnvFilePath    string `json:"envFilePath"`   // Ruta local al archivo .env asociado (si aplica)
	EnableSSL      bool   `json:"enableSSL"`
	HealthcheckCmd string `json:"healthcheckCmd"` // Comando para matar zombies (ej: curl -f http://localhost/ || exit 1)
	MountsJSON     string `json:"mountsJson"`     // Archivos extra a inyectar (JSON array de ServiceMount)
	EnvVars        string `json:"envVars"`        // Contenido de variables de entorno (.env)
	CustomDomains  string `json:"customDomains"`  // Dominios adicionales y alias (JSON o coma separados)
	TargetNode     string `json:"targetNode"`     // Restricción de nodo Swarm (manager/worker/hostname)
	PreDeployHook  string `json:"preDeployHook"`  // Pre-deploy migration hook (ej. "npx prisma db push")
}

// SavedDatabase representa una base de datos gestionada en el catálogo local.
type SavedDatabase struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Engine            string `json:"engine"`              // "postgres", "mongo", "mysql", "redis"
	DeployType        string `json:"deployType"`          // "external", "single-node", "multi-node"
	ExternalURL       string `json:"externalUrl"`         // Si es externa
	InternalPort      int    `json:"internalPort"`        // Puerto interno en el cluster (ej 5432)
	VolumeHostPath    string `json:"volumeHostPath"`     // Ruta en el servidor para persistencia local
	NodeIP            string `json:"nodeIp"`              // Si es multi-node
	Password          string `json:"password"`            // Contraseña auto-generada
	TargetNode        string `json:"targetNode"`          // Restricción de afinidad (ej: manager, worker-1)
	ReuseExistingData bool   `json:"reuseExistingData"`  // Reutiliza /opt/data/db-<name> en modo Recovery
	CleanExistingData bool   `json:"cleanExistingData"`  // Limpia el directorio host antes de desplegar
}

// ServiceLink representa la interconexión entre dos servicios o bases de datos mediante una variable de entorno.
type ServiceLink struct {
	ID         int    `json:"id"`
	SourceSvc  string `json:"sourceSvc"`  // Nombre del servicio origen (ej: "frontend-app")
	TargetSvc  string `json:"targetSvc"`  // Nombre del servicio o BD destino (ej: "db-postgres")
	EnvVarName string `json:"envVarName"` // Variable inyectada (ej: "DATABASE_URL")
	TargetURL  string `json:"targetUrl"`  // URL interna resuelta (ej: "postgres://admin:***@tarhiata-db-postgres:5432/db")
}

// SavedBackup representa un snapshot / backup realizado de una BD o Volumen persistente.
type SavedBackup struct {
	ID         int    `json:"id"`
	TargetName string `json:"targetName"` // Nombre de la BD o App (ej: "postgres-main", "volume-api")
	TargetType string `json:"targetType"` // "database" o "volume"
	Engine     string `json:"engine"`     // "postgres", "mongo", "mysql", "redis", "volume"
	Filename   string `json:"filename"`   // ej: "backup_postgres-main_20260726_1000.sql.gz"
	FilePath   string `json:"filePath"`   // Ruta remota: "/opt/tarhiata/backups/..."
	SizeBytes  int64  `json:"sizeBytes"`  // Tamaño del archivo en bytes
	Status     string `json:"status"`     // "completed", "failed"
	S3Location string `json:"s3Location,omitempty"` // Ruta S3 en MinIO/S3 si fue subido
	CreatedAt  string `json:"createdAt"`
}

// MigrationFile representa un archivo de migración SQL registrado para una base de datos.
type MigrationFile struct {
	ID          int    `json:"id"`
	DBName      string `json:"dbName"`
	Filename    string `json:"filename"`
	Content     string `json:"content"`     // Sentencias SQL de aplicación (UP)
	DownContent string `json:"downContent"` // Sentencias SQL de regresión (DOWN/Rollback)
	Status      string `json:"status"`      // "pending", "applied", "failed", "reverted"
	ExecutedAt  string `json:"executedAt"`
	LogOutput   string `json:"logOutput"`
}

// SavedPreviewEnv representa un entorno efímero temporal de pruebas desplegado.
type SavedPreviewEnv struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`        // ej: "prev-feat-auth"
	ImageSource string `json:"imageSource"` // ej: "myorg/api:pr-12"
	Port        int    `json:"port"`        // ej: 8080
	Domain      string `json:"domain"`      // ej: "prev-auth.tarhiata.local"
	LinkDBName  string `json:"linkDbName"`  // ej: "postgres-main"
	Status      string `json:"status"`      // "active", "stopped"
	CreatedAt   string `json:"createdAt"`   // Timestamp
	TargetNode  string `json:"targetNode"`  // Restricción de afinidad
}

// SavedObservability representa la configuración y persistencia del stack de métricas y logs.
type SavedObservability struct {
	ID              int    `json:"id"`
	DeployType      string `json:"deploy_type"`  // "external", "single-node", "multi-node"
	ExternalURL     string `json:"external_url"` // ej: URL de Datadog o Grafana Cloud
	NodeIP          string `json:"node_ip"`      // Si es multi-node
	GrafanaPassword string `json:"grafana_password"`
}

// SavedRegistryCredential representa credenciales guardadas para un Docker Registry privado.
type SavedRegistryCredential struct {
	ID        int    `json:"id"`
	Server    string `json:"server"`    // ej: "docker.io", "ghcr.io", "666666.dkr.ecr.us-east-1.amazonaws.com"
	Username  string `json:"username"`  // ej: "myorg"
	Password  string `json:"password"`  // Cifrada/Almacenada
	CreatedAt string `json:"createdAt"`
}

// AuditLog representa una entrada inmutable en el registro de auditoría del sistema.
type AuditLog struct {
	ID           int       `json:"id"`
	Action       string    `json:"action"`       // "DEPLOY", "EDIT", "DELETE", "PROVISION", "LINK", "UNLINK"
	ResourceType string    `json:"resourceType"` // "service", "database", "worker", "link"
	ResourceName string    `json:"resourceName"`
	Details      string    `json:"details"`
	Timestamp    time.Time `json:"timestamp"`
}

// DockerRegistry contiene las credenciales de un registro de imágenes privado.
type DockerRegistry struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`     // ej: "GitHub Container Registry"
	Server   string `json:"server"`   // ej: "ghcr.io", "hub.docker.com"
	Username string `json:"username"`
	Password string `json:"password"`
}

// DeploymentRecord representa una versión histórica de despliegue de un servicio.
type DeploymentRecord struct {
	ID          int       `json:"id"`
	ServiceName string    `json:"serviceName"`
	ImageTag    string    `json:"imageTag"`
	EnvVars     string    `json:"envVars"`
	Port        int       `json:"port"`
	Domain      string    `json:"domain"`
	Expose      bool      `json:"expose"`
	DeployedAt  time.Time `json:"deployedAt"`
	Status      string    `json:"status"` // "success", "failed", "rolled_back"
}

// AlertSettings almacena la configuración de destinos de alertas salientes.
type AlertSettings struct {
	DiscordURL    string `json:"discordUrl"`
	TelegramToken string `json:"telegramToken"`
	TelegramChat  string `json:"telegramChat"`
	SlackURL      string `json:"slackUrl"`
	GenericURL    string `json:"genericUrl"`
	Enabled       bool   `json:"enabled"`
}

