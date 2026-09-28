package telemetry

import (
	_ "embed"
)

//go:embed telemetry.js
var JSContent []byte

// GetJSContent devuelve el contenido JS del componente de telemetría.
func GetJSContent() []byte {
	return JSContent
}
