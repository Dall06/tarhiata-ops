package store

import (
	_ "embed"
)

//go:embed state.js
var JSContent []byte

// GetJSContent devuelve el contenido del store JS.
func GetJSContent() []byte {
	return JSContent
}
