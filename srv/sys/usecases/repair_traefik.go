package usecases

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type RepairTraefikUseCase struct {
	sshExec ports.SSHExecutor
}

func NewRepairTraefikUseCase(sshExec ports.SSHExecutor) *RepairTraefikUseCase {
	return &RepairTraefikUseCase{sshExec: sshExec}
}

// acmeEmailRegex exige un email simple en ASCII: sin comillas, backslashes ni saltos de
// línea, que son los caracteres que romperían el YAML del stack de Traefik al interpolarse.
var acmeEmailRegex = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9._%+\-]*[a-zA-Z0-9])?@[a-zA-Z0-9](?:[a-zA-Z0-9.\-]*[a-zA-Z0-9])?\.[a-zA-Z]{2,}$`)

func (uc *RepairTraefikUseCase) Execute(acmeEmail string) (string, error) {
	acmeEmail = strings.TrimSpace(acmeEmail)
	acmeFlags := ""
	acmeVolMount := ""
	acmeVolSection := ""
	if acmeEmail != "" {
		if !acmeEmailRegex.MatchString(acmeEmail) {
			return "", fmt.Errorf("el email ACME '%s' no tiene un formato válido", acmeEmail)
		}
		acmeFlags = fmt.Sprintf("      - \"--certificatesresolvers.leresolver.acme.tlschallenge=true\"\n      - \"--certificatesresolvers.leresolver.acme.email=%s\"\n      - \"--certificatesresolvers.leresolver.acme.storage=/letsencrypt/acme.json\"\n", acmeEmail)
		acmeVolMount = "      - \"traefik_certs:/letsencrypt\"\n"
		acmeVolSection = "volumes:\n  traefik_certs:\n"
	}

	compose := fmt.Sprintf(`version: '3.8'
services:
  traefik:
    image: traefik:v3.1
    command:
      - "--api.dashboard=true"
      - "--providers.docker=true"
      - "--providers.docker.swarmmode=true"
      - "--providers.docker.network=tarhiata_public"
      - "--providers.docker.exposedbydefault=false"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
%s    ports:
      - "80:80"
      - "443:443"
    volumes:
      - "/var/run/docker.sock:/var/run/docker.sock:ro"
%s    networks:
      - tarhiata_public
    deploy:
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
      placement:
        constraints:
          - node.role == manager

networks:
  tarhiata_public:
    external: true

%s`, acmeFlags, acmeVolMount, acmeVolSection)

	if err := uc.sshExec.WriteRemoteFile("/tmp/traefik-repair.yml", compose); err != nil {
		return "", fmt.Errorf("error escribiendo compose de reparación: %w", err)
	}

	res, err := uc.sshExec.RunCommand("docker stack deploy -c /tmp/traefik-repair.yml tarhiata_proxy")
	if err != nil || res == nil || res.ExitCode != 0 {
		out := ""
		if res != nil {
			out = res.Output
		}
		if out == "" && err != nil {
			out = err.Error()
		}
		return "", fmt.Errorf("error redesplegando Traefik: %s", out)
	}

	output := strings.TrimSpace(res.Output)
	if output == "" {
		output = "Traefik reparado y redesplegado con configuración actualizada."
	}
	return output, nil
}
