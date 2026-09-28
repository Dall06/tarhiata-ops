package apiclient

import (
	_ "embed"
)

//go:embed api.js
var JSContent []byte

// GetJSContent devuelve el contenido del cliente JS.
func GetJSContent() []byte {
	return JSContent
}
