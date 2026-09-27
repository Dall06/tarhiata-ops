package domain

// PromptInput modela la captura de texto interactiva en terminal.
type PromptInput struct {
	Title       string
	Placeholder string
	Value       string
	Required    bool
}

// ConfirmDialog modela una confirmación booleana interactiva.
type ConfirmDialog struct {
	Title   string
	Default bool
	Value   bool
}

// HostMetricsDisplay modela los datos formateados para mostrar métricas en terminal.
type HostMetricsDisplay struct {
	ServerName string
	HostIP     string
	CPUUsage   string
	RAMUsage   string
	DiskUsage  string
	Uptime     string
	DockerVer  string
}

// ServiceTableItem modela un servicio desplegado formateado para vista en lista.
type ServiceTableItem struct {
	Name    string
	Image   string
	Status  string
	Port    string
	Domain  string
	Replicas string
}

// DeployFormModel modela los campos recolectados por el formulario interactivo de despliegue.
type DeployFormModel struct {
	Name        string
	Image       string
	Port        string
	Domain      string
	PreHook     string
	AutoMigrate bool
}
