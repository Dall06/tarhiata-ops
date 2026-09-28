package databases

import (
	_ "embed"
)

//go:embed databases.js
var JSContent []byte

// GetJSContent devuelve el contenido JS del componente de bases de datos.
func GetJSContent() []byte {
	return JSContent
}
