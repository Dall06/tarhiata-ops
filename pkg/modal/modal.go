package modal

import (
	_ "embed"
)

//go:embed modal.js
var JSContent []byte

// GetJSContent devuelve el contenido del modal JS.
func GetJSContent() []byte {
	return JSContent
}
