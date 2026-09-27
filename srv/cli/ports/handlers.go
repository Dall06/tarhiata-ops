package ports

import (
	sysdomain "github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

type ConfigHandler interface {
	Execute(current *sysdomain.ServerConfig) *sysdomain.ServerConfig
}

type BootstrapHandler interface {
	Execute(config sysdomain.ServerConfig)
}

type ServiceHandler interface {
	Execute(config sysdomain.ServerConfig)
}

type DatabaseHandler interface {
	Execute(config sysdomain.ServerConfig)
}

type ToolHandler interface {
	Execute(config sysdomain.ServerConfig)
}

type ObservabilityHandler interface {
	Execute(config sysdomain.ServerConfig)
}

type ShellHandler interface {
	Execute(config sysdomain.ServerConfig)
}

type DashboardPresenter interface {
	RenderDashboard(config *sysdomain.ServerConfig)
}

type HostInspector interface {
	HandleMetrics(serverName string) error
	HandleInspect(serverName string) error
	HandleLogs(serviceName string, serverName string) error
	HandleUpdate(serverName string) error
	HandlePrune(serverName string) error
}
