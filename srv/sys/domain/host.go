package domain

import "time"

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
