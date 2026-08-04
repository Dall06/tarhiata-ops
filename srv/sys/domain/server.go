package domain

import "time"

// ServerConfig contiene los datos necesarios para establecer la conexión.
type ServerConfig struct {
	Host          string `json:"host"`
	Port          int    `json:"port"`
	User          string `json:"user"`
	PrivateKey    string `json:"privateKey"`    // Ruta a la llave SSH (ej: ~/.ssh/id_rsa)
	DOAPIToken    string `json:"doApiToken"`   // Token de API Vultr/Cloud (Para Terraform)
	VultrAPIToken string `json:"vultrApiToken"` // Vultr API Key (Para Terraform)
	CloudProvider string `json:"cloudProvider"` // "vultr" o "custom"
}

// CommandResult encapsula la respuesta del servidor tras ejecutar un comando.
type CommandResult struct {
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
	Error    error  `json:"error"`
}

// ServiceFile representa un archivo de configuración local que se enviará al servidor.
type ServiceFile struct {
	FileName  string `json:"file_name"`  // Ej: "secrets.json", ".env"
	LocalPath string `json:"local_path"` // Ruta local en la máquina del usuario (ej. /tmp/secrets.json)
}

// CustomService representa el stack estándar a desplegar con sus archivos adjuntos.
type CustomService struct {
	Name          string            `json:"name"`
	ComposeFile   string            `json:"compose_file"` // Nombre o ruta del stack.yml estándar
	Files         []ServiceFile     `json:"files"`        // Archivos extra que se copiarán al servidor
	Mounts        []ServiceMount    `json:"mounts"`       // Archivos que se montarán en el contenedor
	EnvVars       map[string]string `json:"env_vars"`
	PreDeployHook string            `json:"pre_deploy_hook"` // Pre-deploy migration hook (ej. "npx prisma db push")
}

// SavedService representa la configuración persistida de un servicio en el catálogo local.
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

// ServiceMount define un mapeo de archivo local hacia el contenedor.
type ServiceMount struct {
	LocalPath string `json:"local_path"`
	DestPath  string `json:"dest_path"`
}

// SavedDatabase representa una base de datos en el catálogo.
type SavedDatabase struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Engine         string `json:"engine"`           // "postgres", "mongo"
	DeployType     string `json:"deployType"`      // "external", "single-node", "multi-node"
	ExternalURL    string `json:"externalUrl"`     // Si es externa
	InternalPort   int    `json:"internalPort"`    // Puerto interno en el cluster (ej 5432)
	VolumeHostPath string `json:"volumeHostPath"` // Ruta en el servidor para persistencia local
	NodeIP         string `json:"nodeIp"`          // Si es multi-node
	Password          string `json:"password"`            // Contraseña auto-generada
	TargetNode        string `json:"targetNode"`          // Restricción de afinidad (ej: manager, worker-1)
	ReuseExistingData  bool   `json:"reuseExistingData"`  // Reutiliza /opt/data/db-<name> en modo Recovery
	CleanExistingData  bool   `json:"cleanExistingData"`  // Limpia el directorio host antes de desplegar
}

// DeployConfig contiene las opciones estilo "Vercel" que define el usuario.
type DeployConfig struct {
	ImageSource    string `json:"imageSource"`    // URL o nombre en Docker Hub
	IsURL          bool   `json:"isUrl"`          // True si es un ZIP/TAR, False si es DockerHub
	Port           int    `json:"port"`            // Puerto interno del contenedor (ej. 3000)
	Domain         string `json:"domain"`          // Dominio (ej. api.gymbro.com). Vacío = enruta por Path/IP
	Expose         bool   `json:"expose"`          // Si es true, Traefik lo rutea hacia afuera. Si es false, queda interno.
	EnableSSL      bool   `json:"enableSSL"`      // Si es true, añade el resolver de Let's Encrypt
	HealthcheckCmd string `json:"healthcheckCmd"` // Comando para healthcheck
	TargetNode     string `json:"targetNode"`     // Restricción de afinidad de nodo (manager, worker, hostname)
}

// SavedObservability representa la configuración del stack de logs y métricas
type SavedObservability struct {
	ID              int    `json:"id"`
	DeployType      string `json:"deploy_type"`  // "external", "single-node", "multi-node"
	ExternalURL     string `json:"external_url"` // ej: URL de Datadog o Grafana Cloud
	NodeIP          string `json:"node_ip"`      // Si es multi-node
	GrafanaPassword string `json:"grafana_password"`
}

// ServiceLink representa la interconexión entre dos servicios o bases de datos mediante una variable de entorno.
type ServiceLink struct {
	ID         int    `json:"id"`
	SourceSvc  string `json:"sourceSvc"`  // Nombre del servicio origen (ej: "frontend-app")
	TargetSvc  string `json:"targetSvc"`  // Nombre del servicio o BD destino (ej: "db-postgres")
	EnvVarName string `json:"envVarName"` // Variable inyectada (ej: "DATABASE_URL")
	TargetURL  string `json:"targetUrl"`  // URL interna resuelta (ej: "postgres://admin:***@tarhiata-db-postgres:5432/db")
}

// SavedPreviewEnv representa un entorno de pruebas / preview temporal efímero.
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

// SavedRegistryCredential representa credenciales guardadas para un Docker Registry privado.
type SavedRegistryCredential struct {
	ID        int    `json:"id"`
	Server    string `json:"server"`    // ej: "docker.io", "ghcr.io", "666666.dkr.ecr.us-east-1.amazonaws.com"
	Username  string `json:"username"`  // ej: "myorg"
	Password  string `json:"password"`  // Cifrada/Almacenada
	CreatedAt string `json:"createdAt"`
}

// MigrationFile representa un archivo de migración SQL o script registrado para una BD.
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

// DatabaseMigrationRequest contiene los parámetros para la ejecución interactiva de migraciones y regresiones.
type DatabaseMigrationRequest struct {
	TargetDB   string   `json:"targetDB"`   // Nombre de la BD en el catálogo (ej: "postgres-prod")
	TargetNode string   `json:"targetNode"` // Afinidad de nodo ("manager", "worker", IP)
	Filenames  []string `json:"filenames"`  // Lista de archivos .sql a ejecutar en orden
	Action     string   `json:"action"`     // "up" (aplicar) o "down" (regresión)
	SqlContent string   `json:"sqlContent"` // Opcional: Sentencia SQL rápida directa
}

// SavedBackup representa un snapshot / backup realizado de una BD o Volumen.
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

// BackupRequest especifica la solicitud de creación de backup o restauración.
type BackupRequest struct {
	TargetName  string `json:"targetName"`  // BD o App
	TargetType  string `json:"targetType"`  // "database" o "volume"
	TargetNode  string `json:"targetNode"`  // Afinidad de nodo ("manager", "worker-1", etc.)
	BackupID    int    `json:"backupId"`    // Para restauración
	S3Target    string `json:"s3Target,omitempty"`    // Nombre de la instancia MinIO/S3 o "custom"
	BucketName  string `json:"bucketName,omitempty"`  // Nombre del Bucket (ej: "backups")
	CustomS3URL string `json:"customS3Url,omitempty"` // URL externa (ej: "https://s3.amazonaws.com" o Cloudflare R2)
	AccessKey   string `json:"accessKey,omitempty"`   // Clave de Acceso S3
	SecretKey   string `json:"secretKey,omitempty"`   // Clave Secreta S3
}

// AuditLog representa una entrada inmutable en el registro de auditoría.
type AuditLog struct {
	ID           int       `json:"id"`
	Action       string    `json:"action"`       // "DEPLOY", "EDIT", "DELETE", "PROVISION", "LINK", "UNLINK"
	ResourceType string    `json:"resourceType"` // "service", "database", "worker", "link"
	ResourceName string    `json:"resourceName"`
	Details      string    `json:"details"`
	Timestamp    time.Time `json:"timestamp"`
}

// DockerRegistry contiene las credenciales de un registro de imágenes privado (Docker Hub, GHCR, AWS ECR).
type DockerRegistry struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`     // ej: "GitHub Container Registry"
	Server   string `json:"server"`   // ej: "ghcr.io", "hub.docker.com"
	Username string `json:"username"`
	Password string `json:"password"`
}

// ContainerStats representa métricas en tiempo real de un contenedor (cgroups / docker stats).
type ContainerStats struct {
	Container string `json:"container"`
	CPUPerc   string `json:"cpu"`
	MemUsage  string `json:"memUsage"`
	MemPerc   string `json:"memPerc"`
	NetIO     string `json:"netIo"`
	BlockIO   string `json:"blockIo"`
}

// DBHealthStats contiene métricas detalladas de salud de un motor de BD.
type DBHealthStats struct {
	Engine            string  `json:"engine"`
	ActiveConnections int     `json:"activeConnections"`
	MaxConnections    int     `json:"maxConnections"`
	UptimeSeconds     int64   `json:"uptimeSeconds"`
	QPS               float64 `json:"qps"`
	Status            string  `json:"status"` // "Healthy", "Warning", "Critical"
	Details           string  `json:"details"`
}

// VultrPlan contiene la información de precios y specs de una instancia Vultr.
type VultrPlan struct {
	ID          string   `json:"id"`
	VCPU        int      `json:"vcpu_count"`
	RAM         int      `json:"ram"`
	Disk        int      `json:"disk"`
	Bandwidth   int      `json:"bandwidth"`
	MonthlyCost float64  `json:"monthly_cost"`
	Type        string   `json:"type"`
	Locations   []string `json:"locations,omitempty"`
}

// VultrRegion contiene la información geográfica de un Data Center de Vultr.
type VultrRegion struct {
	ID        string   `json:"id"`
	City      string   `json:"city"`
	Country   string   `json:"country"`
	Continent string   `json:"continent"`
	Options   []string `json:"options,omitempty"`
}

// SSHKeyInfo contiene los metadatos de una llave SSH autorizada en el VPS.
type SSHKeyInfo struct {
	Fingerprint string `json:"fingerprint"`
	Comment     string `json:"comment"`
	Type        string `json:"type"`
	KeyContent  string `json:"key_content"`
	IsVultrKey  bool   `json:"is_vultr_key"`
	Protected   bool   `json:"protected"`
}


