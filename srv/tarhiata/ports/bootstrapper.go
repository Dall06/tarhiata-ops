package ports

type BootstrapperUseCase interface {
	InitServer(installObservability bool, acmeEmail string, installTS bool, tsAuthKey string, exposeObs bool) error
	InstallTailscale(authKey string) error
	DeployObservability(exposePublic bool) error
}
