package fleet

import (
	_ "embed"
)

//go:embed fleet.js
var JSContent []byte

// GetJSContent devuelve el contenido del componente JS de fleet.
func GetJSContent() []byte {
	return JSContent
}
