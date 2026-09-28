package volumes

import (
	_ "embed"
	"path/filepath"
	"strings"
)

//go:embed volumes.js
var JSContent []byte

// GetJSContent devuelve el contenido JS del componente de volumes.
func GetJSContent() []byte {
	return JSContent
}

// VolumeItem describe un archivo o directorio remoto.
type VolumeItem struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"isDir"`
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
}

// WritePayload contiene los datos para escribir o actualizar un archivo.
type WritePayload struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// MkdirPayload contiene la ruta para crear un nuevo directorio.
type MkdirPayload struct {
	Path string `json:"path"`
}

// SanitizeVolumePath asegura que la ruta no intente escapar de /opt/data o contenga path traversals peligrosos.
func SanitizeVolumePath(rawPath string, basePath string) string {
	baseClean := strings.TrimSpace(basePath)
	if baseClean == "" || baseClean == "." {
		baseClean = "/opt/data"
	}
	baseClean = filepath.Clean(baseClean)

	cleaned := filepath.Clean(strings.TrimSpace(rawPath))
	if cleaned == "." || cleaned == "/" || !strings.HasPrefix(cleaned, baseClean) {
		return baseClean
	}
	return cleaned
}
