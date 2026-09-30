package usecases

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

// ACMEStorage representa la estructura estándar del archivo acme.json generado por Traefik v3.
type ACMEStorage struct {
	Letsencrypt struct {
		Certificates []struct {
			Domain struct {
				Main string   `json:"main"`
				SANs []string `json:"sans"`
			} `json:"domain"`
			Certificate string `json:"certificate"`
		} `json:"Certificates"`
	} `json:"letsencrypt"`
}

// ACMECertificateSummary contiene el estado preventivo de un certificado TLS emitido.
type ACMECertificateSummary struct {
	Domain        string    `json:"domain"`
	SANs          []string  `json:"sans,omitempty"`
	Issuer        string    `json:"issuer"`
	NotBefore     time.Time `json:"notBefore"`
	NotAfter      time.Time `json:"notAfter"`
	DaysRemaining int       `json:"daysRemaining"`
	Status        string    `json:"status"` // "active", "expiring_soon", "expired"
}

// InspectSSLAcmeUseCase lee el almacén de certificados ACME de Traefik e inspecciona su vigencia.
type InspectSSLAcmeUseCase struct {
	executor ports.SSHExecutor
}

func NewInspectSSLAcmeUseCase(executor ports.SSHExecutor) *InspectSSLAcmeUseCase {
	return &InspectSSLAcmeUseCase{executor: executor}
}

// Execute extrae los certificados de /opt/tarhiata/traefik/acme.json y calcula su vigencia preventiva.
func (uc *InspectSSLAcmeUseCase) Execute(config domain.ServerConfig) ([]ACMECertificateSummary, error) {
	if err := uc.executor.Connect(config); err != nil {
		return nil, fmt.Errorf("error conectando SSH: %w", err)
	}
	defer func() {
		if clErr := uc.executor.Close(); clErr != nil {
			slog.Warn("inspect_ssl_acme: error cerrando conexión SSH", "error", clErr)
		}
	}()

	cmd := "cat /opt/tarhiata/traefik/acme.json 2>/dev/null || cat /opt/tarhiata/acme.json 2>/dev/null || echo '{}'"
	cmdRes, err := uc.executor.RunCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("error leyendo archivo acme.json: %w", err)
	}

	rawJSON := "{}"
	if cmdRes != nil && strings.TrimSpace(cmdRes.Output) != "" {
		rawJSON = strings.TrimSpace(cmdRes.Output)
	}

	return uc.ParseACMEJSON(rawJSON, time.Now())
}

// ParseACMEJSON procesa el contenido JSON de acme.json y extrae los metadatos X.509.
func (uc *InspectSSLAcmeUseCase) ParseACMEJSON(rawJSON string, now time.Time) ([]ACMECertificateSummary, error) {
	var storage ACMEStorage
	if err := json.Unmarshal([]byte(rawJSON), &storage); err != nil {
		return []ACMECertificateSummary{}, nil
	}

	var results []ACMECertificateSummary

	for _, certItem := range storage.Letsencrypt.Certificates {
		certData, err := base64.StdEncoding.DecodeString(certItem.Certificate)
		if err != nil {
			continue
		}

		cert, err := x509.ParseCertificate(certData)
		if err != nil {
			continue
		}

		days := int(cert.NotAfter.Sub(now).Hours() / 24)
		if days < 0 {
			days = 0
		}

		status := "active"
		if now.After(cert.NotAfter) {
			status = "expired"
		}
		if now.Before(cert.NotAfter) && days <= 15 {
			status = "expiring_soon"
		}

		domainName := certItem.Domain.Main
		if domainName == "" && len(cert.DNSNames) > 0 {
			domainName = cert.DNSNames[0]
		}

		results = append(results, ACMECertificateSummary{
			Domain:        domainName,
			SANs:          certItem.Domain.SANs,
			Issuer:        cert.Issuer.CommonName,
			NotBefore:     cert.NotBefore,
			NotAfter:      cert.NotAfter,
			DaysRemaining: days,
			Status:        status,
		})
	}

	return results, nil
}
