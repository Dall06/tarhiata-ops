package topology

import (
	_ "embed"
)

//go:embed topology.js
var JSContent []byte

// GetJSContent devuelve el contenido JS del componente de topología.
func GetJSContent() []byte {
	return JSContent
}
