package nodes

import (
	_ "embed"
)

//go:embed nodes.js
var JSContent []byte

// GetJSContent devuelve el contenido JS del componente de nodos.
func GetJSContent() []byte {
	return JSContent
}
