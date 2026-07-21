package ports

import "github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"

type DeployServiceUseCase interface {
	Execute(service domain.CustomService, config domain.DeployConfig) error
}
