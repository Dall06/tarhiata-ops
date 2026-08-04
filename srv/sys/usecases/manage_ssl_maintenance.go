package usecases

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type SSLStatusItem struct {
	Domain       string `json:"domain"`
	ServiceName  string `json:"serviceName"`
	IsSSL        bool   `json:"isSSL"`
	Status       string `json:"status"` // "active", "expiring_soon", "expired", "http_only"
	DaysRemaining int   `json:"daysRemaining"`
	Issuer       string `json:"issuer"`
	ExpiryDate   string `json:"expiryDate"`
}

type ManageSSLMaintenanceUseCase struct {
	repo ports.ConfigRepository
	ssh  ports.SSHExecutor
}

func NewManageSSLMaintenanceUseCase(repo ports.ConfigRepository, ssh ports.SSHExecutor) *ManageSSLMaintenanceUseCase {
	return &ManageSSLMaintenanceUseCase{repo: repo, ssh: ssh}
}

// InspectSSL inspecta la validez de los certificados SSL de los dominios registrados
func (uc *ManageSSLMaintenanceUseCase) InspectSSL() ([]SSLStatusItem, error) {
	services, err := uc.repo.GetServices()
	if err != nil {
		return nil, err
	}

	var results []SSLStatusItem
	for _, s := range services {
		if !s.Expose || s.Domain == "" {
			continue
		}

		item := SSLStatusItem{
			Domain:      s.Domain,
			ServiceName: s.Name,
			IsSSL:       s.EnableSSL,
			Status:      "http_only",
		}

		if !s.EnableSSL {
			results = append(results, item)
			continue
		}

		// Inspeccionar certificado TLS en puerto 443
		dialer := &net.Dialer{Timeout: 3 * time.Second}
		conn, err := tls.DialWithDialer(dialer, "tcp", fmt.Sprintf("%s:443", s.Domain), &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			item.Status = "expired"
			results = append(results, item)
			continue
		}

		state := conn.ConnectionState()
		conn.Close()

		if len(state.PeerCertificates) > 0 {
			cert := state.PeerCertificates[0]
			now := time.Now()
			daysLeft := int(cert.NotAfter.Sub(now).Hours() / 24)

			item.Issuer = cert.Issuer.CommonName
			if item.Issuer == "" && len(cert.Issuer.Organization) > 0 {
				item.Issuer = cert.Issuer.Organization[0]
			}
			item.ExpiryDate = cert.NotAfter.Format("2006-01-02")
			item.DaysRemaining = daysLeft

			if daysLeft <= 0 {
				item.Status = "expired"
			} else if daysLeft < 15 {
				item.Status = "expiring_soon"
			} else {
				item.Status = "active"
			}
		}

		results = append(results, item)
	}

	return results, nil
}

// ToggleMaintenanceMode activa o desactiva el modo mantenimiento (503 Drain) en Traefik para un servicio
func (uc *ManageSSLMaintenanceUseCase) ToggleMaintenanceMode(serviceName string, enable bool, config domain.ServerConfig) error {
	svc, err := uc.repo.GetService(serviceName)
	if err != nil || svc == nil {
		return fmt.Errorf("servicio '%s' no encontrado", serviceName)
	}

	if uc.ssh == nil {
		return fmt.Errorf("ejecutor SSH no configurado")
	}
	if err := uc.ssh.Connect(config); err != nil {
		return fmt.Errorf("error de conexión SSH: %w", err)
	}
	defer uc.ssh.Close()

	containerName := fmt.Sprintf("tarhiata-app-%s", svc.Name)

	if enable {
		// Activar etiqueta Traefik de respuesta 503 Maintenance
		cmd := fmt.Sprintf(`docker service update \
			--label-add "traefik.http.middlewares.maint-%s.replacepathregex.regex=.*" \
			--label-add "traefik.http.middlewares.maint-%s.replacepathregex.replacement=/" \
			--label-add "traefik.http.routers.%s.middlewares=maint-%s" \
			%s`, svc.Name, svc.Name, svc.Name, svc.Name, containerName)
		uc.ssh.RunCommand(cmd)
	} else {
		// Remover modo mantenimiento
		cmd := fmt.Sprintf(`docker service update \
			--label-rm "traefik.http.middlewares.maint-%s.replacepathregex.regex" \
			--label-rm "traefik.http.middlewares.maint-%s.replacepathregex.replacement" \
			--label-rm "traefik.http.routers.%s.middlewares" \
			%s`, svc.Name, svc.Name, svc.Name, containerName)
		uc.ssh.RunCommand(cmd)
	}

	return nil
}
