package domain

// CliCommand representa una entidad de comando ejecutable en la interfaz CLI.
type CliCommand struct {
	Name        string
	Description string
	Category    string
	Flags       []string
}

// MenuOption representa una opción navegable en los menús interactivos de la CLI.
type MenuOption struct {
	Key         string
	Title       string
	Description string
	Danger      bool
}

// TerminalTheme define los colores y estilos visuales utilizados por la CLI.
type TerminalTheme struct {
	BgColor      string
	CardBgColor  string
	BorderColor  string
	TextColor    string
	SubtextColor string
	AccentColor  string
	SuccessColor string
	ErrorColor   string
}

// DashboardCard representa una tarjeta métrica en el dashboard visual.
type DashboardCard struct {
	Title string
	Value string
	Unit  string
	Color string
}
