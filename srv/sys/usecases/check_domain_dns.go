package usecases

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

var validDomainRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$`)

type CheckDomainDNSUseCase struct {
	resolver ports.DNSResolver
}

func NewCheckDomainDNSUseCase(resolver ports.DNSResolver) *CheckDomainDNSUseCase {
	return &CheckDomainDNSUseCase{resolver: resolver}
}

// Execute normaliza un dominio o URL ingresado por el usuario y verifica si resuelve a la
// IP del servidor dado.
func (uc *CheckDomainDNSUseCase) Execute(rawDomain string, serverIP string) (domain.DNSCheckResult, error) {
	cleanDomain := strings.TrimPrefix(strings.TrimSpace(rawDomain), "https://")
	cleanDomain = strings.TrimPrefix(cleanDomain, "http://")
	if idx := strings.Index(cleanDomain, "/"); idx != -1 {
		cleanDomain = cleanDomain[:idx]
	}
	if idx := strings.Index(cleanDomain, ":"); idx != -1 {
		cleanDomain = cleanDomain[:idx]
	}
	cleanDomain = strings.ToLower(strings.TrimSpace(cleanDomain))

	if !validDomainRegex.MatchString(cleanDomain) {
		return domain.DNSCheckResult{}, fmt.Errorf("formato de dominio inválido")
	}

	ips, err := uc.resolver.LookupHost(cleanDomain)
	if err != nil || len(ips) == 0 {
		return domain.DNSCheckResult{
			Domain:      cleanDomain,
			ServerIP:    serverIP,
			ResolvedIPs: []string{},
			Matches:     false,
			Status:      "not_found",
		}, nil
	}

	isMatch := slices.Contains(ips, serverIP)
	status := "mismatch"
	if isMatch {
		status = "match"
	}

	return domain.DNSCheckResult{
		Domain:      cleanDomain,
		ServerIP:    serverIP,
		ResolvedIPs: ips,
		Matches:     isMatch,
		Status:      status,
	}, nil
}
