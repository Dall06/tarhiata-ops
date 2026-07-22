package usecases

import (
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/tests/mocks"
)

func TestInitServer_Execute(t *testing.T) {
	tests := []struct {
		name                string
		email               string
		domain              string
		expectedCmdContains []string
	}{
		{
			name:  "Bootstrap successful with traefik and letsencrypt",
			email: "admin@test.com",
			expectedCmdContains: []string{
				"PermitRootLogin prohibit-password", // ssh hardening
				"fail2ban",
				"ufw allow 80/tcp",
				"ufw allow 443/tcp",
				"docker swarm init",
				"docker network create --driver overlay tarhiata_public",
				"tarhiata_proxy", // El stack de traefik
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSSH := mocks.NewMockSSHExecutor()
			// Evitar que falle el check de swarm init
			mockSSH.MockResponses["Swarm: active"] = &domain.CommandResult{Output: "inactive", ExitCode: 1} // Simular que no es parte de un swarm aún
			// Forzar que cree la red pública
			mockSSH.MockResponses["docker network ls"] = &domain.CommandResult{Output: "", ExitCode: 1}

			bootstrapper := NewInitServerUseCase(mockSSH)
			err := bootstrapper.Execute(tt.email)

			if err != nil {
				t.Fatalf("InitServer.Execute() error = %v", err)
			}

			// Validar que los comandos críticos fueron ejecutados
			for _, expectedCmd := range tt.expectedCmdContains {
				found := false
				for _, executedCmd := range mockSSH.CommandsExecuted {
					if strings.Contains(executedCmd, expectedCmd) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Se esperaba que se ejecutara el comando que contenga '%s', pero no se encontró.\nComandos ejecutados: %v", expectedCmd, mockSSH.CommandsExecuted)
				}
			}
		})
	}
}
