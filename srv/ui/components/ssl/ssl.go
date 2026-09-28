package ssl

import (
	_ "embed"
	"time"
)

//go:embed ssl.js
var JSContent []byte

// GetJSContent devuelve el contenido JS del componente de ssl.
func GetJSContent() []byte {
	return JSContent
}

// CertificateInfo describe el estado TLS de un dominio público.
type CertificateInfo struct {
	Domain        string `json:"domain"`
	ServiceName   string `json:"serviceName"`
	Status        string `json:"status"` // active, expiring_soon, expired, http_only
	Issuer        string `json:"issuer"`
	ExpiryDate    string `json:"expiryDate"`
	DaysRemaining int    `json:"daysRemaining"`
}

// MaintenancePayload representa la solicitud para alternar el modo 503 de un servicio.
type MaintenancePayload struct {
	ServiceName string `json:"serviceName"`
	Enable      bool   `json:"enable"`
}

// CalculateRemainingDays calcula los días restantes hasta la fecha de expiración.
func CalculateRemainingDays(expiry time.Time, now time.Time) int {
	diff := expiry.Sub(now)
	days := int(diff.Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// DetermineSSLStatus categoriza el estado del certificado TLS.
func DetermineSSLStatus(hasCert bool, daysRemaining int) string {
	if !hasCert {
		return "http_only"
	}
	if daysRemaining <= 0 {
		return "expired"
	}
	if daysRemaining <= 15 {
		return "expiring_soon"
	}
	return "active"
}
