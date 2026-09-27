package dockerutil

import (
	"fmt"
	"strings"
)

// IsDockerError detecta si una salida de comando Docker corresponde a un mensaje de error conocido.
func IsDockerError(out string) bool {
	l := strings.ToLower(out)
	return strings.Contains(l, "no such service") ||
		strings.Contains(l, "no such container") ||
		strings.Contains(l, "error response from daemon") ||
		strings.Contains(l, "invalid service name") ||
		strings.Contains(l, "not found")
}

// ExtractDomain extrae el nombre de dominio configurado en una regla de router de Traefik (Host(`...`) o PathPrefix(`...`)).
func ExtractDomain(rule string) string {
	if strings.Contains(rule, "Host(`") {
		start := strings.Index(rule, "Host(`") + 6
		end := strings.Index(rule[start:], "`)")
		if end != -1 {
			return rule[start : start+end]
		}
	}
	if strings.Contains(rule, "PathPrefix(`") {
		start := strings.Index(rule, "PathPrefix(`") + 12
		end := strings.Index(rule[start:], "`)")
		if end != -1 {
			return rule[start : start+end]
		}
	}
	return ""
}

// BuildSafeURI construye la cadena de conexión interna enmascarada para una base de datos o servicio.
func BuildSafeURI(engine, serviceName string, port int) string {
	engineLower := strings.ToLower(engine)
	switch engineLower {
	case "mongo", "mongodb":
		return fmt.Sprintf("mongodb://admin:********@%s:27017/?authSource=admin", serviceName)
	case "redis":
		return fmt.Sprintf("redis://:********@%s:6379", serviceName)
	case "minio", "s3":
		return fmt.Sprintf("s3://admin:********@%s:9000 (Console :9001)", serviceName)
	default:
		return fmt.Sprintf("%s://admin:********@%s:%d/db", engine, serviceName, port)
	}
}
