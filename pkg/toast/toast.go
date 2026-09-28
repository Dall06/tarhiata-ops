package toast

import (
	_ "embed"
)

//go:embed toast.js
var JSContent []byte

// GetJSContent devuelve el contenido del toast JS.
func GetJSContent() []byte {
	return JSContent
}
