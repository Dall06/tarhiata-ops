package usecases

import (
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
	"strings"
	"testing"
)

// MockSSHExecutor simula la conexión SSH para pruebas
type MockSSHExecutor struct {
	CommandsExecuted []string
	MockExitCode     int
	MockOutput       string
	MockError        error
}

func (m *MockSSHExecutor) Connect(config domain.ServerConfig) error { return nil }
func (m *MockSSHExecutor) Close() error                             { return nil }
func (m *MockSSHExecutor) CheckConnection() bool                    { return true }
func (m *MockSSHExecutor) RunCommand(cmd string) (*domain.CommandResult, error) {
	m.CommandsExecuted = append(m.CommandsExecuted, cmd)
	return &domain.CommandResult{Output: m.MockOutput, ExitCode: m.MockExitCode}, m.MockError
}
func (m *MockSSHExecutor) InteractiveShell() error             { return nil }
func (m *MockSSHExecutor) InteractiveCommand(cmd string) error { return nil }

func TestBootstrapper_InitServer(t *testing.T) {
	tests := []struct {
		name                 string
		installObservability bool
		installTS            bool
		expectCommands       []string
	}{
		{
			name:                 "Instalación Base (Sin VPN ni Observabilidad)",
			installObservability: false,
			installTS:            false,
			expectCommands: []string{
				"killall -9 apt apt-get dpkg",
				"command -v docker",
				"daemon.json",
				"docker swarm init",
				"ufw default deny incoming",
				"docker stack deploy -c /tmp/traefik-stack.yml tarhiata_proxy",
			},
		},
		{
			name:                 "Instalación Completa (VPN + Observabilidad)",
			installObservability: true,
			installTS:            true,
			expectCommands: []string{
				"command -v tailscale",
				"tailscale up --authkey=testkey",
				"portainer",
				"dozzle",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSSH := &MockSSHExecutor{MockExitCode: 0} // Simulamos que todos los comandos SSH tienen éxito
			uc := NewBootstrapperUseCase(mockSSH)

			tsKey := ""
			if tt.installTS {
				tsKey = "testkey"
			}
			err := uc.InitServer(tt.installObservability, "test@test.com", tt.installTS, tsKey, false)
			if err != nil {
				t.Fatalf("InitServer falló inesperadamente: %v", err)
			}

			// Concatenar todos los comandos ejecutados para búsqueda fácil
			allCmds := strings.Join(mockSSH.CommandsExecuted, " ||| ")

			for _, expect := range tt.expectCommands {
				if !strings.Contains(allCmds, expect) {
					t.Errorf("Se esperaba que se ejecutara un comando conteniendo '%s', pero no se encontró.\nComandos: %s", expect, allCmds)
				}
			}
		})
	}
}
