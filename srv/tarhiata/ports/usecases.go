package ports

import "github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"

type InitServerUseCase interface {
	Execute(acmeEmail string) error
}

type InstallTailscaleUseCase interface {
	Execute(authKey string) error
}

type DeployObservabilityUseCase interface {
	Execute(exposePublic bool) error
}

type DeployDatabaseUseCase interface {
	Execute(db domain.SavedDatabase, config domain.ServerConfig) error
}

type DeployServiceUseCase interface {
	Execute(service domain.CustomService, config domain.DeployConfig) error
}

type UpdateServerUseCase interface {
	Execute() error
}
