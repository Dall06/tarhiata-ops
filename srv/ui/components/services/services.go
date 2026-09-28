package services

import (
	_ "embed"
)

//go:embed services.js
var JSContent []byte

// GetJSContent devuelve el contenido JS del componente de servicios.
func GetJSContent() []byte {
	return JSContent
}
