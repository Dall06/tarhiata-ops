package terminal

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed terminal.js
var JSContent []byte

// GetJSContent devuelve el contenido JS del componente de terminal.
func GetJSContent() []byte {
	return JSContent
}

// ExecRequest representa la petición para ejecutar un comando en la terminal web o de host.
type ExecRequest struct {
	Name      string `json:"name"`
	Command   string `json:"command"`
	Container string `json:"container,omitempty"`
}

// ExecResponse representa la salida devuelta al navegador tras ejecutar el comando.
type ExecResponse struct {
	Output    string `json:"output"`
	ExitCode  int    `json:"exitCode"`
	Connected bool   `json:"connected"`
}

// FormatPrompt genera la cadena de prompt adecuada según el host y contenedor.
func FormatPrompt(host string, container string) string {
	h := strings.TrimSpace(host)
	if h == "" {
		h = "vps"
	}
	c := strings.TrimSpace(container)
	if c != "" {
		return fmt.Sprintf("root@%s:#", c)
	}
	return fmt.Sprintf("root@%s:~$", h)
}

// SanitizeTerminalCommand limpia espacios y valida comandos vacíos.
func SanitizeTerminalCommand(cmd string) string {
	return strings.TrimSpace(cmd)
}
