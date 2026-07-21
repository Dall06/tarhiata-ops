package domain

// ServerConfig contiene los datos necesarios para establecer la conexión.
type ServerConfig struct {
	Host       string
	Port       int
	User       string
	PrivateKey string // Ruta a la llave SSH (ej: ~/.ssh/id_rsa)
	DOAPIToken string // DigitalOcean API Token (Para Terraform)
}

// CommandResult encapsula la respuesta del servidor tras ejecutar un comando.
type CommandResult struct {
	Output   string
	ExitCode int
	Error    error
}

// ServiceFile representa un archivo de configuración local que se enviará al servidor.
type ServiceFile struct {
	FileName  string // Ej: "secrets.json", ".env"
	LocalPath string // Ruta local en la máquina del usuario (ej. /tmp/secrets.json)
}

// CustomService representa el stack estándar a desplegar con sus archivos adjuntos.
type CustomService struct {
	Name        string
	ComposeFile string // Nombre o ruta del stack.yml estándar
	Files       []ServiceFile // Archivos extra que se copiarán al servidor
	Mounts      []ServiceMount // Archivos que se montarán en el contenedor
	EnvVars     map[string]string
}

// SavedService representa la configuración persistida de un servicio en el catálogo local.
type SavedService struct {
	ID          int
	Name        string
	ImageSource string
	IsURL       bool
	Port        int
	Domain      string
	Expose      bool
	EnvFilePath    string // Ruta local al archivo .env asociado (si aplica)
	EnableSSL      bool
	HealthcheckCmd string // Comando para matar zombies (ej: curl -f http://localhost/ || exit 1)
	MountsJSON     string // Archivos extra a inyectar (JSON array de ServiceMount)
}

// ServiceMount define un mapeo de archivo local hacia el contenedor.
type ServiceMount struct {
	LocalPath string `json:"local_path"`
	DestPath  string `json:"dest_path"`
}

// SavedDatabase representa una base de datos en el catálogo.
type SavedDatabase struct {
	ID             int
	Name           string
	Engine         string // "postgres", "mongo"
	DeployType     string // "external", "single-node", "multi-node"
	ExternalURL    string // Si es externa
	InternalPort   int    // Puerto interno en el cluster (ej 5432)
	VolumeHostPath string // Ruta en el servidor para persistencia local
	NodeIP         string // Si es multi-node
}

// DeployConfig contiene las opciones estilo "Vercel" que define el usuario.
type DeployConfig struct {
	ImageSource    string // URL o nombre en Docker Hub
	IsURL          bool   // True si es un ZIP/TAR, False si es DockerHub
	Port           int    // Puerto interno del contenedor (ej. 3000)
	Domain         string // Dominio (ej. api.gymbro.com). Vacío = enruta por Path/IP
	Expose         bool   // Si es true, Traefik lo rutea hacia afuera. Si es false, queda interno.
	EnableSSL      bool   // Si es true, añade el resolver de Let's Encrypt
	HealthcheckCmd string // Comando para healthcheck
}
