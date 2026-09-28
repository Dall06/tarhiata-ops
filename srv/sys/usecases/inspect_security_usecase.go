package usecases

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

// InspectSecurityUseCase inspecciona el cortafuegos UFW y la protección Fail2Ban en el host remoto o local.
type InspectSecurityUseCase struct {
	executor ports.SSHExecutor
}

func NewInspectSecurityUseCase(executor ports.SSHExecutor) *InspectSecurityUseCase {
	return &InspectSecurityUseCase{executor: executor}
}

// Execute ejecuta comandos remotos para recolectar el estado de UFW y Fail2Ban.
func (uc *InspectSecurityUseCase) Execute() (domain.SecurityReport, error) {
	report := domain.SecurityReport{
		UFWRules: make([]domain.UFWRule, 0),
		Fail2Ban: domain.Fail2BanReport{
			Jails:     make([]string, 0),
			BannedIPs: make([]string, 0),
		},
	}

	// 1. Recolectar estado UFW
	ufwRes, err := uc.executor.RunCommand("which ufw >/dev/null 2>&1 && (ufw status numbered || ufw status) || echo 'NOT_INSTALLED'")
	if err == nil && ufwRes != nil && !strings.Contains(ufwRes.Output, "NOT_INSTALLED") {
		report.RawUFW = strings.TrimSpace(ufwRes.Output)
		report.UFWActive, report.UFWRules = parseUFWOutput(ufwRes.Output)
	}

	// 2. Recolectar estado Fail2Ban
	f2bRes, err := uc.executor.RunCommand("which fail2ban-client >/dev/null 2>&1 && (fail2ban-client status && (fail2ban-client status sshd 2>/dev/null || fail2ban-client status ssh 2>/dev/null || true)) || echo 'NOT_INSTALLED'")
	if err == nil && f2bRes != nil && !strings.Contains(f2bRes.Output, "NOT_INSTALLED") {
		report.Fail2Ban = parseFail2BanOutput(f2bRes.Output)
	}

	return report, nil
}

// parseUFWOutput procesa la salida de ufw status numbered.
func parseUFWOutput(output string) (bool, []domain.UFWRule) {
	rules := make([]domain.UFWRule, 0)
	lower := strings.ToLower(output)
	active := strings.Contains(lower, "status: active")

	lines := strings.Split(output, "\n")
	ruleRegexNumbered := regexp.MustCompile(`^\[\s*(\d+)\]\s+(.*?)\s+(ALLOW\s+IN|DENY\s+IN|REJECT\s+IN|ALLOW|DENY|REJECT)\s+(.*)$`)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "Status:") || strings.HasPrefix(trimmed, "To") || strings.HasPrefix(trimmed, "--") {
			continue
		}

		matches := ruleRegexNumbered.FindStringSubmatch(trimmed)
		if len(matches) >= 5 {
			toVal := strings.TrimSpace(matches[2])
			proto := "any"
			if strings.Contains(toVal, "/") {
				parts := strings.Split(toVal, "/")
				if len(parts) == 2 {
					proto = strings.TrimSpace(parts[1])
				}
			}

			rules = append(rules, domain.UFWRule{
				Number: matches[1],
				To:     toVal,
				Action: strings.TrimSpace(matches[3]),
				From:   strings.TrimSpace(matches[4]),
				Proto:  proto,
			})
			continue
		}

		// Formato no numerado fallback
		fields := strings.Fields(trimmed)
		if len(fields) >= 3 && (fields[1] == "ALLOW" || fields[1] == "DENY" || fields[1] == "REJECT") {
			rules = append(rules, domain.UFWRule{
				Number: strconv.Itoa(len(rules) + 1),
				To:     fields[0],
				Action: fields[1],
				From:   strings.Join(fields[2:], " "),
				Proto:  "any",
			})
		}
	}

	return active, rules
}

// parseFail2BanOutput procesa la salida combinada de fail2ban-client status y status jail.
func parseFail2BanOutput(output string) domain.Fail2BanReport {
	report := domain.Fail2BanReport{
		Active:    !strings.Contains(strings.ToLower(output), "not running") && len(strings.TrimSpace(output)) > 0,
		Jails:     make([]string, 0),
		BannedIPs: make([]string, 0),
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		// Extraer lista de jaulas (Jail list: sshd, nginx, etc.)
		if strings.Contains(lower, "jail list:") {
			idx := strings.Index(lower, "jail list:")
			jailsStr := trimmed[idx+len("jail list:"):]
			for _, j := range strings.Split(jailsStr, ",") {
				if cleaned := strings.TrimSpace(j); cleaned != "" {
					report.Jails = append(report.Jails, cleaned)
				}
			}
			continue
		}

		// Total banned
		if strings.Contains(lower, "total banned:") {
			fields := strings.Fields(trimmed)
			if len(fields) >= 1 {
				val, err := strconv.Atoi(fields[len(fields)-1])
				if err == nil {
					report.TotalBanned = val
				}
			}
			continue
		}

		// Currently failed
		if strings.Contains(lower, "currently failed:") {
			fields := strings.Fields(trimmed)
			if len(fields) >= 1 {
				val, err := strconv.Atoi(fields[len(fields)-1])
				if err == nil {
					report.FailedCount = val
				}
			}
			continue
		}

		// Banned IP list: 1.2.3.4 5.6.7.8
		if strings.Contains(lower, "banned ip list:") {
			idx := strings.Index(lower, "banned ip list:")
			ipsStr := trimmed[idx+len("banned ip list:"):]
			for _, ip := range strings.Fields(ipsStr) {
				if cleaned := strings.TrimSpace(ip); cleaned != "" {
					report.BannedIPs = append(report.BannedIPs, cleaned)
				}
			}
		}
	}

	return report
}
