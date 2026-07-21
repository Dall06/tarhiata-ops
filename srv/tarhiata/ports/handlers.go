package ports

// CLIHandler define el contrato para el controlador principal de la interfaz de línea de comandos.
type CLIHandler interface {
	Run()
}
