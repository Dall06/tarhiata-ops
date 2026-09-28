package domain

// StatusResponse representa la respuesta devuelta por /api/status.
type StatusResponse struct {
	Connected      bool        `json:"connected"`
	Configured     bool        `json:"configured"`
	SwarmActive    bool        `json:"swarmActive"`
	IsLeader       bool        `json:"isLeader"`
	SwarmConfigured bool       `json:"swarmConfigured"`
	Message        string      `json:"message,omitempty"`
	Error          string      `json:"error,omitempty"`
	Telemetry      interface{} `json:"telemetry,omitempty"`
}

// RollbackRequest representa la solicitud de reversión de versión de un servicio.
type RollbackRequest struct {
	ServiceName string `json:"serviceName"`
}

// RestartRequest representa la solicitud para reiniciar un contenedor o servicio.
type RestartRequest struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"` // service o database
}
