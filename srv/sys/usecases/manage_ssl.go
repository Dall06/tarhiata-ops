package usecases

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type SSLStatusItem = domain.SSLStatusItem

type ManageSSLMaintenanceUseCase struct {
	repo ports.ConfigRepository
	ssh  ports.SSHExecutor
}

func NewManageSSLMaintenanceUseCase(repo ports.ConfigRepository, ssh ports.SSHExecutor) *ManageSSLMaintenanceUseCase {
	return &ManageSSLMaintenanceUseCase{repo: repo, ssh: ssh}
}

func determineSSLStatus(daysLeft int) string {
	if daysLeft <= 0 {
		return "expired"
	}
	if daysLeft < 15 {
		return "expiring_soon"
	}
	return "active"
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
		if closeErr := conn.Close(); closeErr != nil {
			slog.Warn("manage_ssl: error cerrando conexión TLS", "domain", s.Domain, "error", closeErr)
		}

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
			item.Status = determineSSLStatus(daysLeft)
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
	defer func() {
		if closeErr := uc.ssh.Close(); closeErr != nil {
			slog.Warn("manage_ssl: error cerrando conexión SSH", "error", closeErr)
		}
	}()

	containerName := fmt.Sprintf("tarhiata-app-%s", svc.Name)

	if enable {
		// Activar etiqueta Traefik de respuesta 503 Maintenance
		cmd := fmt.Sprintf(`docker service update \
			--label-add "traefik.http.middlewares.maint-%s.replacepathregex.regex=.*" \
			--label-add "traefik.http.middlewares.maint-%s.replacepathregex.replacement=/" \
			--label-add "traefik.http.routers.%s.middlewares=maint-%s" \
			%s 2>&1`, svc.Name, svc.Name, svc.Name, svc.Name, containerName)
		out, err := uc.ssh.RunCommand(cmd)
		if err != nil {
			outMsg := ""
			if out != nil {
				outMsg = out.Output
			}
			slog.Warn("manage_ssl: docker service update fallo al activar mantenimiento, probando fallback", "error", err, "out", outMsg)
			fallbackCmd := fmt.Sprintf(`docker update --restart=no %s 2>&1`, containerName)
			fOut, fErr := uc.ssh.RunCommand(fallbackCmd)
			if fErr != nil {
				fMsg := ""
				if fOut != nil {
					fMsg = fOut.Output
				}
				return fmt.Errorf("fallo al activar modo mantenimiento en '%s': %w (%s)", svc.Name, err, outMsg+fMsg)
			}
		}
		return nil
	}

	// Remover modo mantenimiento
	cmd := fmt.Sprintf(`docker service update \
		--label-rm "traefik.http.middlewares.maint-%s.replacepathregex.regex" \
		--label-rm "traefik.http.middlewares.maint-%s.replacepathregex.replacement" \
		--label-rm "traefik.http.routers.%s.middlewares" \
		%s 2>&1`, svc.Name, svc.Name, svc.Name, containerName)
	out, err := uc.ssh.RunCommand(cmd)
	if err != nil {
		outMsg := ""
		if out != nil {
			outMsg = out.Output
		}
		slog.Warn("manage_ssl: docker service update fallo al desactivar mantenimiento", "error", err, "out", outMsg)
	}
	return nil
}

// ReloadTraefik fuerza el reinicio o recarga de la configuración de Traefik para renovar certificados
func (uc *ManageSSLMaintenanceUseCase) ReloadTraefik(config domain.ServerConfig) error {
	if uc.ssh == nil {
		return fmt.Errorf("ejecutor SSH no configurado")
	}
	if err := uc.ssh.Connect(config); err != nil {
		return fmt.Errorf("error de conexión SSH: %w", err)
	}
	defer func() {
		if closeErr := uc.ssh.Close(); closeErr != nil {
			slog.Warn("manage_ssl: error cerrando conexión SSH", "error", closeErr)
		}
	}()

	cmd := `docker service update --force tarhiata-traefik 2>/dev/null || docker restart tarhiata-traefik 2>&1`
	out, err := uc.ssh.RunCommand(cmd)
	if err != nil {
		outMsg := ""
		if out != nil {
			outMsg = out.Output
		}
		slog.Warn("manage_ssl: fallo reiniciando traefik", "error", err, "out", outMsg)
		return fmt.Errorf("error recargando Traefik: %w", err)
	}
	return nil
}
