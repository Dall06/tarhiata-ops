package validator

import (
	"fmt"
	"regexp"
	"strings"
)

var identifierRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)

// IsIdentifier valida si una cadena es un identificador alfanumérico seguro (para nombres de servicio, BD o contenedores).
func IsIdentifier(s string) bool {
	if s == "" || len(s) > 128 {
		return false
	}
	return identifierRegex.MatchString(s)
}

// IsNodeID valida si una cadena cumple con el formato de ID de nodo Swarm (hasta 64 caracteres alfanuméricos seguros).
func IsNodeID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, r := range id {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.') {
			return false
		}
	}
	return true
}

// IsSafeCommand evalúa si un comando ingresado no contiene secuencias destructivas de alto riesgo en terminales expuestas.
func IsSafeCommand(cmd string) bool {
	c := strings.TrimSpace(strings.ToLower(cmd))
	blocked := []string{
		"rm -rf /",
		"rm -rf /*",
		"mkfs",
		"dd if=",
		"reboot",
		"shutdown",
		"init 0",
		":(){ :|:& };:",
	}
	for _, b := range blocked {
		if strings.Contains(c, b) {
			return false
		}
	}
	return true
}

// NormalizeRegion normaliza alias de regiones comunes para proveedores de nube a su código canónico.
func NormalizeRegion(provider, region string) (string, error) {
	p := strings.ToLower(strings.TrimSpace(provider))
	r := strings.ToLower(strings.TrimSpace(region))

	switch p {
	case "vultr":
		switch r {
		case "mex", "mexico", "mexico-city", "cdmx", "mx", "":
			return "mex", nil
		case "nyc1", "nyc2", "nyc3":
			return "ewr", nil
		case "sfo1", "sfo2", "sfo3":
			return "sjc", nil
		case "ams2", "ams3":
			return "ams", nil
		case "lon1":
			return "lhr", nil
		case "fra1":
			return "fra", nil
		case "sgp1":
			return "sgp", nil
		default:
			return r, nil
		}
	case "digitalocean", "do":
		switch r {
		case "nyc", "newyork", "":
			return "nyc1", nil
		case "sfo", "sanfrancisco":
			return "sfo3", nil
		case "ams", "amsterdam":
			return "ams3", nil
		case "fra", "frankfurt":
			return "fra1", nil
		case "lon", "london":
			return "lon1", nil
		case "sgp", "singapore":
			return "sgp1", nil
		default:
			return r, nil
		}
	default:
		if r == "" {
			return "default", nil
		}
		return r, nil
	}
}

// SanitizeContainerName asegura que un nombre de contenedor no contenga caracteres inválidos.
func SanitizeContainerName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if !IsIdentifier(trimmed) {
		return "", fmt.Errorf("nombre de contenedor inválido: %q", name)
	}
	return trimmed, nil
}
