package ports

import "github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"

type DeployerUseCase interface {
	DeployService(service domain.CustomService, config domain.DeployConfig) error
}
