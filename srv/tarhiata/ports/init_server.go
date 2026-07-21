package ports

type InitServerUseCase interface {
	Execute(acmeEmail string) error
}
