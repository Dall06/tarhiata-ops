package validator

import (
	"fmt"
	"regexp"
	"strings"
)

var identifierRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)
var whitespaceRegex = regexp.MustCompile(`\s+`)
var safePathRegex = regexp.MustCompile(`^/[a-zA-Z0-9_\-./]*$`)

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

// IsSafeCommand evalúa si un comando ingresado no contiene secuencias destructivas de alto
// riesgo en terminales expuestas. Es una red de seguridad contra que un administrador ya
// autenticado se autodestruya el servidor por accidente (fat-finger), NO un límite de
// seguridad contra un atacante: quien llega a este endpoint ya está autenticado con el
// mismo nivel de acceso que el resto del panel y, por diseño, puede ejecutar cualquier
// comando a propósito.
// ponytail: blocklist de patrones conocidos, no un parser de shell — variantes
// suficientemente creativas (variables, sustitución, encoding) pueden evadirla. Endurecerla
// de verdad requeriría un allowlist o deshabilitar la terminal libre, que es una decisión de
// producto, no un fix de este alcance.
func IsSafeCommand(cmd string) bool {
	c := strings.ToLower(strings.TrimSpace(cmd))
	c = whitespaceRegex.ReplaceAllString(c, " ")
	blocked := []string{
		"rm -rf /",
		"rm -rf /*",
		"rm -fr /",
		"rm --recursive --force /",
		"mkfs",
		"dd if=",
		"dd of=/dev/",
		"> /dev/sd",
		"> /dev/nvme",
		"reboot",
		"shutdown",
		"poweroff",
		"halt",
		"init 0",
		"init 6",
		":(){ :|:& };:",
		"chmod -r 000 /",
		"chown -r nobody /",
	}
	for _, b := range blocked {
		if strings.Contains(c, b) {
			return false
		}
	}
	return true
}

// IsSafePath valida que una ruta absoluta de filesystem no contenga metacaracteres de shell
// (espacios, comillas, $, `, ;, |, &) ni secuencias de traversal, antes de interpolarla
// cruda en comandos de shell remotos vía SSH (mkdir, rm, docker --mount, etc).
func IsSafePath(path string) bool {
	if path == "" || len(path) > 512 {
		return false
	}
	if !safePathRegex.MatchString(path) {
		return false
	}
	return !strings.Contains(path, "..")
}

// ShellQuote envuelve s en comillas simples POSIX-seguras para interpolarlo en un
// comando de shell remoto (SSH), neutralizando cualquier metacaracter ($, `, ;, |, &,
// espacios) sin importar el contenido.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
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
