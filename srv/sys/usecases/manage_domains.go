package usecases

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type DomainRule struct {
	Domain         string `json:"domain"`
	RedirectTarget string `json:"redirectTarget"` // Vaciador si es alias normal, o p.ej. "example.com" para redirección 301
}

type ManageDomainsUseCase struct {
	repo ports.ConfigRepository
	ssh  ports.SSHExecutor
}

func NewManageDomainsUseCase(repo ports.ConfigRepository, ssh ports.SSHExecutor) *ManageDomainsUseCase {
	return &ManageDomainsUseCase{repo: repo, ssh: ssh}
}

// GetServiceDomains obtiene el dominio principal y los dominios personalizados/redirecciones asociados
func (uc *ManageDomainsUseCase) GetServiceDomains(serviceName string) (string, []DomainRule, error) {
	svc, err := uc.repo.GetService(serviceName)
	if err != nil || svc == nil {
		return "", nil, fmt.Errorf("servicio '%s' no encontrado", serviceName)
	}

	var rules []DomainRule
	if svc.CustomDomains != "" {
		_ = json.Unmarshal([]byte(svc.CustomDomains), &rules)
	}
	return svc.Domain, rules, nil
}

// AddCustomDomain agrega un alias o regla de redirección CNAME al servicio y actualiza Traefik en Docker Swarm
func (uc *ManageDomainsUseCase) AddCustomDomain(serviceName, customDomain, redirectTarget string, config domain.ServerConfig) error {
	svc, err := uc.repo.GetService(serviceName)
	if err != nil || svc == nil {
		return fmt.Errorf("servicio '%s' no encontrado", serviceName)
	}

	customDomain = strings.TrimSpace(strings.ToLower(customDomain))
	redirectTarget = strings.TrimSpace(strings.ToLower(redirectTarget))
	if customDomain == "" {
		return fmt.Errorf("el dominio personalizado no puede estar vacío")
	}

	var rules []DomainRule
	if svc.CustomDomains != "" {
		_ = json.Unmarshal([]byte(svc.CustomDomains), &rules)
	}

	// Verificar si ya existe
	for _, r := range rules {
		if r.Domain == customDomain {
			return fmt.Errorf("el dominio '%s' ya está registrado para este servicio", customDomain)
		}
	}

	rules = append(rules, DomainRule{
		Domain:         customDomain,
		RedirectTarget: redirectTarget,
	})

	bytes, _ := json.Marshal(rules)
	svc.CustomDomains = string(bytes)
	if err := uc.repo.SaveService(*svc); err != nil {
		return fmt.Errorf("error al guardar dominio en BD: %w", err)
	}

	// Sincronizar reglas Traefik en Docker Swarm vía SSH
	return uc.syncTraefikDomains(*svc, rules, config)
}

// RemoveCustomDomain elimina un dominio personalizado o alias del servicio
func (uc *ManageDomainsUseCase) RemoveCustomDomain(serviceName, customDomain string, config domain.ServerConfig) error {
	svc, err := uc.repo.GetService(serviceName)
	if err != nil || svc == nil {
		return fmt.Errorf("servicio '%s' no encontrado", serviceName)
	}

	customDomain = strings.TrimSpace(strings.ToLower(customDomain))
	var rules []DomainRule
	if svc.CustomDomains != "" {
		_ = json.Unmarshal([]byte(svc.CustomDomains), &rules)
	}

	var filtered []DomainRule
	found := false
	for _, r := range rules {
		if r.Domain == customDomain {
			found = true
			continue
		}
		filtered = append(filtered, r)
	}

	if !found {
		return fmt.Errorf("el dominio '%s' no pertenece al servicio '%s'", customDomain, serviceName)
	}

	bytes, _ := json.Marshal(filtered)
	svc.CustomDomains = string(bytes)
	if err := uc.repo.SaveService(*svc); err != nil {
		return fmt.Errorf("error al actualizar BD: %w", err)
	}

	return uc.syncTraefikDomains(*svc, filtered, config)
}

func (uc *ManageDomainsUseCase) syncTraefikDomains(svc domain.SavedService, rules []DomainRule, config domain.ServerConfig) error {
	if uc.ssh == nil {
		return nil
	}
	if err := uc.ssh.Connect(config); err != nil {
		return fmt.Errorf("error de conexión SSH: %w", err)
	}
	defer uc.ssh.Close()

	containerName := fmt.Sprintf("tarhiata-app-%s", svc.Name)

	// Construir Host rule para Traefik
	hosts := []string{}
	if svc.Domain != "" {
		hosts = append(hosts, fmt.Sprintf("Host(`%s`)", svc.Domain))
	}
	for _, r := range rules {
		hosts = append(hosts, fmt.Sprintf("Host(`%s`)", r.Domain))
	}

	if len(hosts) == 0 {
		return nil
	}

	ruleStr := strings.Join(hosts, " || ")
	cmd := fmt.Sprintf(`docker service update \
		--label-add "traefik.http.routers.%s.rule=%s" \
		--label-add "traefik.http.routers.%s-tls.rule=%s" \
		%s`, svc.Name, ruleStr, svc.Name, ruleStr, containerName)

	uc.ssh.RunCommand(cmd)
	return nil
}
