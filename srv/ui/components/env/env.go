package env

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed env.js
var JSContent []byte

// GetJSContent devuelve el contenido JS del componente de env.
func GetJSContent() []byte {
	return JSContent
}

// Payload representa el cuerpo de la petición para leer o actualizar variables .env.
type Payload struct {
	ServiceName string `json:"serviceName"`
	RawContent  string `json:"rawContent"`
}

// EnvVar representa un par clave-valor de variable de entorno.
type EnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ParseEnvFile parsea un texto raw estilo .env a un mapa de claves y valores.
func ParseEnvFile(raw string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		idx := strings.Index(trimmed, "=")
		if idx == -1 {
			continue
		}
		k := strings.TrimSpace(trimmed[:idx])
		v := strings.TrimSpace(trimmed[idx+1:])
		if (strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"")) || (strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'")) {
			if len(v) >= 2 {
				v = v[1 : len(v)-1]
			}
		}
		if k != "" {
			result[k] = v
		}
	}
	return result
}

// FormatEnvFile genera un string con formato .env a partir de un mapa, ordenado alfabéticamente.
func FormatEnvFile(vars map[string]string) string {
	if len(vars) == 0 {
		return ""
	}
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("%s=%s\n", k, vars[k]))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}
