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
	ExecutePersistent(exposePublic bool, deployType string, grafanaPassword string) error
}

type DeployDatabaseUseCase interface {
	Execute(db domain.SavedDatabase, config domain.ServerConfig) error
}

type ProvisionWorkerUseCase interface {
	Execute(config domain.ServerConfig, nodeName string, labelType string) (string, error)
}

type DeployServiceUseCase interface {
	Execute(service domain.CustomService, config domain.DeployConfig) error
}

type UpdateServerUseCase interface {
	Execute() error
}
