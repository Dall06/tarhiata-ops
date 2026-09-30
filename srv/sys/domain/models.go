package domain

import "time"

// NodeProvisionResult contiene la IP pública y la clave privada SSH tras aprovisionar infraestructura en la nube.
type NodeProvisionResult struct {
	PublicIP   string `json:"publicIp"`
	PrivateKey string `json:"privateKey"`
}

// ConnectionResult contiene el diagnóstico completo de conectividad y salud del host.
type ConnectionResult struct {
	Name          string   `json:"name"`
	Connected     bool     `json:"connected"`
	IsLocal       bool     `json:"isLocal"`
	TargetHost    string   `json:"targetHost"`
	OS            string   `json:"os"`
	DockerActive  bool     `json:"dockerActive"`
	DockerVersion string   `json:"dockerVersion"`
	SwarmActive   bool     `json:"swarmActive"`
	LatencyMs     int64    `json:"latencyMs"`
	Message       string   `json:"message"`
	Errors        []string `json:"errors,omitempty"`
}

// CommandResult encapsula la respuesta del servidor tras ejecutar un comando remoto o local.
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

// ServiceMount define un mapeo de archivo local hacia el contenedor.
type ServiceMount struct {
	LocalPath string `json:"local_path"`
	DestPath  string `json:"dest_path"`
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

// DeployConfig contiene las opciones estilo "Vercel" que define el usuario para despliegue.
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

// DatabaseMigrationRequest contiene los parámetros para la ejecución interactiva de migraciones y regresiones.
type DatabaseMigrationRequest struct {
	TargetDB   string   `json:"targetDB"`   // Nombre de la BD en el catálogo (ej: "postgres-prod")
	TargetNode string   `json:"targetNode"` // Afinidad de nodo ("manager", "worker", IP)
	Filenames  []string `json:"filenames"`  // Lista de archivos .sql a ejecutar en orden
	Action     string   `json:"action"`     // "up" (aplicar) o "down" (regresión)
	SqlContent string   `json:"sqlContent"` // Opcional: Sentencia SQL rápida directa
}

// BackupRequest especifica la solicitud de creación de backup o restauración.
type BackupRequest struct {
	TargetName  string `json:"targetName"`  // BD o App
	TargetType  string `json:"targetType"`  // "database" o "volume"
	Engine      string `json:"engine,omitempty"` // "postgres", "mysql", "mongo", "redis"
	TargetNode  string `json:"targetNode"`  // Afinidad de nodo ("manager", "worker-1", etc.)
	BackupID    int    `json:"backupId"`    // Para restauración
	Server      string `json:"server,omitempty"` // VPS de destino
	S3Target    string `json:"s3Target,omitempty"`    // Nombre de la instancia MinIO/S3 o "custom"
	BucketName  string `json:"bucketName,omitempty"`  // Nombre del Bucket (ej: "backups")
	CustomS3URL string `json:"customS3Url,omitempty"` // URL externa (ej: "https://s3.amazonaws.com" o Cloudflare R2)
	AccessKey   string `json:"accessKey,omitempty"`   // Clave de Acceso S3
	SecretKey   string `json:"secretKey,omitempty"`   // Clave Secreta S3
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

// VultrSSHKey representa una llave SSH registrada en la cuenta de Vultr del usuario.
type VultrSSHKey struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SSHKey      string `json:"ssh_key"`
	DateCreated string `json:"date_created"`
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

// StorageDevice representa una unidad o partición de almacenamiento conectada al host.
type StorageDevice struct {
	Name       string `json:"name"`       // Ej: "sda", "nvme0n1", "vda"
	Size       string `json:"size"`       // Ej: "50G", "1TB"
	Type       string `json:"type"`       // Ej: "disk", "part", "rom"
	MountPoint string `json:"mountPoint"` // Ej: "/", "/data", "[SWAP]"
	Model      string `json:"model"`      // Ej: "Samsung SSD 980" o "QEMU HARDDISK"
	Rotational bool   `json:"rotational"` // false = SSD / NVMe, true = HDD
	FSType     string `json:"fsType"`     // Ej: "ext4", "xfs", "btrfs"
}

// GPUDevice representa una tarjeta o acelerador gráfico conectado al bus del host.
type GPUDevice struct {
	Model       string `json:"model"`                 // Ej: "NVIDIA RTX 4090", "Virtio GPU", "UHD Graphics 630"
	Vendor      string `json:"vendor"`                // Ej: "NVIDIA Corporation", "Intel Corporation", "Advanced Micro Devices"
	MemoryTotal string `json:"memoryTotal,omitempty"` // Ej: "24576 MiB"
	Driver      string `json:"driver,omitempty"`      // Ej: "nvidia", "i915", "virtio-pci"
	PCIAddress  string `json:"pciAddress,omitempty"`  // Ej: "00:02.0"
}

// USBDevice representa un periférico o controlador conectado a los puertos USB del host.
type USBDevice struct {
	Bus          string `json:"bus"`                    // Ej: "001"
	Device       string `json:"device"`                 // Ej: "002"
	ID           string `json:"id"`                     // Ej: "1d6b:0002" (VendorID:ProductID)
	Description  string `json:"description"`            // Ej: "Linux Foundation 2.0 root hub"
	Manufacturer string `json:"manufacturer,omitempty"` // Ej: "Kingston", "Logitech"
}

// DisplayDevice representa una salida o pantalla física/virtual conectada al host (HDMI, DP, etc.).
type DisplayDevice struct {
	Connector  string `json:"connector"`            // Ej: "HDMI-1", "DP-1", "Virtual-1"
	Status     string `json:"status"`               // "connected" | "disconnected"
	Resolution string `json:"resolution,omitempty"` // Ej: "1920x1080", "2560x1440"
}

// PCIDevice representa un bus, puente o tarjeta en la topología PCI del host.
type PCIDevice struct {
	Address string `json:"address"` // Ej: "00:00.0"
	Class   string `json:"class"`   // Ej: "Host bridge", "Ethernet controller"
	Vendor  string `json:"vendor"`  // Ej: "Intel Corporation"
	Device  string `json:"device"`  // Ej: "82540EM Gigabit Ethernet Controller"
}

// HostDevices agrupa todos los componentes y periféricos de hardware detectados en el servidor.
type HostDevices struct {
	ServerName string          `json:"serverName"`
	Host       string          `json:"host"`
	IsLocal    bool            `json:"isLocal"`
	Timestamp  time.Time       `json:"timestamp"`
	Storage    []StorageDevice `json:"storage"`
	GPUs       []GPUDevice     `json:"gpus"`
	USB        []USBDevice     `json:"usb"`
	Displays   []DisplayDevice `json:"displays"`
	PCI        []PCIDevice     `json:"pci"`
}

// HostMetrics contiene las métricas de hardware y sistema operativo de un host (VPS o local).
type HostMetrics struct {
	CPUPercent    float64 `json:"cpuPercent"`    // % de uso de CPU (0.0 - 100.0)
	CPUCores      int     `json:"cpuCores"`      // Número de núcleos lógicos
	MemoryUsedMB  int64   `json:"memoryUsedMb"`  // RAM usada en MB
	MemoryTotalMB int64   `json:"memoryTotalMb"` // RAM total en MB
	MemoryPercent float64 `json:"memoryPercent"` // % de uso de RAM
	DiskUsedGB    float64 `json:"diskUsedGb"`    // Espacio en disco usado en GB
	DiskTotalGB   float64 `json:"diskTotalGb"`   // Espacio en disco total en GB
	DiskPercent   float64 `json:"diskPercent"`   // % de uso de disco
	Uptime        string  `json:"uptime"`        // Tiempo activo legible
	LoadAvg       string  `json:"loadAvg"`       // Promedios de carga (1m 5m 15m)
	Hostname      string  `json:"hostname"`      // Nombre de host
	OS            string  `json:"os"`            // Sistema operativo y arquitectura
}

// HostSystemService representa un servicio directo del sistema operativo del host (ej: systemd).
type HostSystemService struct {
	Name        string `json:"name"`        // Ej: "docker.service", "ssh.service"
	LoadState   string `json:"loadState"`   // Ej: "loaded"
	ActiveState string `json:"activeState"` // Ej: "active"
	SubState    string `json:"subState"`    // Ej: "running"
	Description string `json:"description"` // Ej: "Docker Application Container Engine"
}

// HostInspection consolida las métricas y la lista de servicios activos de una máquina.
type HostInspection struct {
	ServerName string              `json:"serverName"`
	Host       string              `json:"host"`
	IsLocal    bool                `json:"isLocal"`
	Timestamp  time.Time           `json:"timestamp"`
	Metrics    HostMetrics         `json:"metrics"`
	Services   []HostSystemService `json:"services"`
}

// SwarmServiceInfo encapsula los metadatos de un servicio orquestado en Docker Swarm.
type SwarmServiceInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Mode     string `json:"mode"`
	Replicas string `json:"replicas"`
	Image    string `json:"image"`
	Ports    string `json:"ports"`
	Domain   string `json:"domain"`
	Expose   bool   `json:"expose"`
}

// SwarmDBInfo encapsula el estado y conexión de una base de datos registrada en el cluster.
type SwarmDBInfo struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Engine       string `json:"engine"`
	DeployType   string `json:"deployType"` // "single-node", "multi-node", "external"
	InternalPort int    `json:"internalPort"`
	ExternalURL  string `json:"externalUrl"`
	TargetNode   string `json:"targetNode"`
	Status       string `json:"status"` // "running", "external", "offline"
	InternalDNS  string `json:"internalDns"`
}

// SwarmNodeInfo representa un nodo miembro del clúster Swarm (Manager o Worker).
type SwarmNodeInfo struct {
	ID            string `json:"id"`
	Hostname      string `json:"hostname"`
	Status        string `json:"status"`
	Availability  string `json:"availability"`
	ManagerStatus string `json:"managerStatus"`
	EngineVersion string `json:"engineVersion"`
}

// SwarmStatus consolida la salud, servicios, bases de datos, nodos y accesos web del clúster Swarm.
type SwarmStatus struct {
	Active        bool               `json:"active"`
	NodeState     string             `json:"nodeState"`
	DockerVersion string             `json:"dockerVersion"`
	Services      []SwarmServiceInfo `json:"services"`
	Databases     []SwarmDBInfo      `json:"databases"`
	Nodes         []SwarmNodeInfo    `json:"nodes"`
	Dashboards    map[string]string  `json:"dashboards"`
}

// MetricPoint representa un punto temporal de uso de recursos de un servicio.
type MetricPoint struct {
	Timestamp string  `json:"timestamp"`
	CPU       float64 `json:"cpu"`     // %
	Memory    float64 `json:"memory"`  // MB
	Network   float64 `json:"network"` // KB/s
	Disk      float64 `json:"disk"`    // MB/s
}

// ServiceMetrics consolida el historial de métricas para un rango temporal.
type ServiceMetrics struct {
	ServiceName string        `json:"serviceName"`
	Range       string        `json:"range"`
	Points      []MetricPoint `json:"points"`
}

// SSLStatusItem representa el estado del certificado SSL de un servicio expuesto.
type SSLStatusItem struct {
	Domain        string `json:"domain"`
	ServiceName   string `json:"serviceName"`
	IsSSL         bool   `json:"isSSL"`
	Status        string `json:"status"` // "active", "expiring_soon", "expired", "http_only"
	DaysRemaining int    `json:"daysRemaining"`
	Issuer        string `json:"issuer"`
	ExpiryDate    string `json:"expiryDate"`
}

// NodeInfo representa los detalles de un nodo Swarm incluyendo etiquetas y rol.
type NodeInfo struct {
	ID            string            `json:"id"`
	Hostname      string            `json:"hostname"`
	Role          string            `json:"role"`
	Status        string            `json:"status"`
	Availability  string            `json:"availability"`
	IsLeader      bool              `json:"isLeader"`
	EngineVersion string            `json:"engineVersion"`
	Labels        map[string]string `json:"labels"`
}

// ProvisionCloudRequest encapsula los parámetros para aprovisionar un servidor en la nube vía IaC.
type ProvisionCloudRequest struct {
	Name        string `json:"name"`
	Provider    string `json:"provider"` // "vultr" | "digitalocean"
	APIToken    string `json:"apiToken"`
	Region      string `json:"region"`
	Plan        string `json:"plan"`
	SetAsActive bool   `json:"isActive"`
}

// BootstrapMasterInput define los parámetros de arranque inicial rápido de una app con base de datos.
type BootstrapMasterInput struct {
	AppName      string `json:"app_name"`
	Image        string `json:"image"`
	Port         int    `json:"port"`
	Domain       string `json:"domain"`
	ExposePublic bool   `json:"expose_public"`
	DBEngine     string `json:"db_engine"`
	EnvVarName   string `json:"env_var_name"`
	TargetNode   string `json:"target_node,omitempty"`
}

// BootstrapMasterResult consolida el resultado del arranque del stack master.
type BootstrapMasterResult struct {
	App         SavedService   `json:"app"`
	Database    *SavedDatabase `json:"database,omitempty"`
	Link        *ServiceLink   `json:"link,omitempty"`
	UnlinkedOld []string       `json:"unlinked_old,omitempty"`
}

// CreatePreviewEnvInput encapsula los datos para instanciar un entorno efímero.
type CreatePreviewEnvInput struct {
	Name        string `json:"name"`
	Image       string `json:"image"`
	ImageSource string `json:"imageSource,omitempty"`
	Port        int    `json:"port"`
	Domain      string `json:"domain"`
	LinkDBName  string `json:"link_db_name,omitempty"`
	LinkDbName  string `json:"linkDbName,omitempty"`
	TargetNode  string `json:"target_node,omitempty"`
	NodeTarget  string `json:"targetNode,omitempty"`
}

// SnapshotDownloadResult contiene los bytes decodificados y el nombre del archivo de backup.
type SnapshotDownloadResult struct {
	Data     []byte `json:"-"`
	Filename string `json:"filename"`
}

// UFWRule representa una regla de puerto y tráfico en el firewall UFW.
type UFWRule struct {
	Number string `json:"number"`
	To     string `json:"to"`
	Action string `json:"action"`
	From   string `json:"from"`
	Proto  string `json:"proto"`
}

// Fail2BanReport contiene el estado de las jaulas activas e IPs bloqueadas por Fail2Ban.
type Fail2BanReport struct {
	Active      bool     `json:"active"`
	Jails       []string `json:"jails"`
	TotalBanned int      `json:"totalBanned"`
	BannedIPs   []string `json:"bannedIps"`
	FailedCount int      `json:"failedCount"`
}

// SecurityReport consolida el estado del firewall UFW y Fail2Ban.
type SecurityReport struct {
	UFWActive bool           `json:"ufwActive"`
	UFWRules  []UFWRule      `json:"ufwRules"`
	Fail2Ban  Fail2BanReport `json:"fail2ban"`
	RawUFW    string         `json:"rawUfw"`
}

// SystemDiagnosticReport consolida una instantánea completa del servidor para auditoría y soporte.
type SystemDiagnosticReport struct {
	Timestamp   time.Time          `json:"timestamp"`
	ServerName  string             `json:"serverName"`
	Host        string             `json:"host"`
	Telemetry   HostInspection     `json:"telemetry"`
	Security    SecurityReport     `json:"security"`
	SwarmStatus SwarmStatus        `json:"swarmStatus"`
	Databases   []SavedDatabase    `json:"databases"`
	GeneratedBy string             `json:"generatedBy"`
}

// DNSCheckResult representa el resultado de verificar si un dominio resuelve a la IP del servidor.
type DNSCheckResult struct {
	Domain      string   `json:"domain"`
	ServerIP    string   `json:"server_ip"`
	ResolvedIPs []string `json:"resolved_ips"`
	Matches     bool     `json:"matches"`
	Status      string   `json:"status"` // "match", "mismatch" o "not_found"
}
