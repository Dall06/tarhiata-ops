package ports

type DeployObservabilityUseCase interface {
	Execute(exposePublic bool) error
}
