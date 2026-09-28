package logs

import (
	_ "embed"
	"strconv"
	"strings"
)

//go:embed logs.js
var JSContent []byte

// GetJSContent devuelve el contenido JS del componente de logs.
func GetJSContent() []byte {
	return JSContent
}

// Response representa la respuesta con los registros del contenedor.
type Response struct {
	Logs string `json:"logs"`
}

// FilterLogs filtra las líneas de log que contienen la subcadena de búsqueda (case-insensitive).
func FilterLogs(raw string, query string) []string {
	if raw == "" {
		return []string{}
	}
	lines := strings.Split(raw, "\n")
	if strings.TrimSpace(query) == "" {
		return lines
	}
	q := strings.ToLower(strings.TrimSpace(query))
	var filtered []string
	for _, l := range lines {
		if strings.Contains(strings.ToLower(l), q) {
			filtered = append(filtered, l)
		}
	}
	return filtered
}

// SanitizeTailLines valida y limita la cantidad de líneas de cola para evitar desbordes de memoria.
func SanitizeTailLines(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return 100
	}
	if n > 2000 {
		return 2000
	}
	return n
}
