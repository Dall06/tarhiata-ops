package ports

import "github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"

type ConfigHandler interface {
	Execute(current *domain.ServerConfig) *domain.ServerConfig
}

type BootstrapHandler interface {
	Execute(config domain.ServerConfig)
}

type ServiceHandler interface {
	Execute(config domain.ServerConfig)
}

type DatabaseHandler interface {
	Execute(config domain.ServerConfig)
}

type ToolHandler interface {
	Execute(config domain.ServerConfig)
}

type ObservabilityHandler interface {
	Execute(config domain.ServerConfig)
}

type ShellHandler interface {
	Execute(config domain.ServerConfig)
}
