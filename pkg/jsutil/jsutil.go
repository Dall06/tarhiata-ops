package jsutil

import (
	_ "embed"
)

//go:embed utils.js
var JSContent []byte

// GetJSContent devuelve el contenido del archivo utils JS.
func GetJSContent() []byte {
	return JSContent
}
