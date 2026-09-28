package ports

// WebServer define el contrato para el ciclo de vida del servidor web de Tarhiata-Ops Studio.
type WebServer interface {
	Start(port string) error
	Stop() error
}
