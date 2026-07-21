package ports

type InstallTailscaleUseCase interface {
	Execute(authKey string) error
}
