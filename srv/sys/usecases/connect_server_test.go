package usecases

import (
	"errors"
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestConnectServerUseCase(t *testing.T) {
	tests := []struct {
		name                 string
		config               domain.ServerConfig
		connectErr           error
		mockResponses        map[string]*domain.CommandResult
		expectedConnected    bool
		expectedDockerActive bool
		expectedSwarmActive  bool
		expectedMsgContains  string
	}{
		{
			name: "connection failure",
			config: domain.ServerConfig{
				Host: "192.168.1.100",
				Port: 22,
			},
			connectErr:           errors.New("dial tcp 192.168.1.100:22: i/o timeout"),
			expectedConnected:    false,
			expectedDockerActive: false,
			expectedSwarmActive:  false,
			expectedMsgContains:  "Error conectando",
		},
		{
			name: "connection success without docker",
			config: domain.ServerConfig{
				Host: "10.0.0.5",
				Port: 22,
			},
			connectErr: nil,
			mockResponses: map[string]*domain.CommandResult{
				"uname -s -m":      {Output: "Linux x86_64", ExitCode: 0},
				"docker --version": {Output: "command not found: docker", ExitCode: 127},
			},
			expectedConnected:    true,
			expectedDockerActive: false,
			expectedSwarmActive:  false,
			expectedMsgContains:  "Docker no se encuentra instalado",
		},
		{
			name: "connection success with docker but swarm inactive",
			config: domain.ServerConfig{
				Host: "10.0.0.5",
				Port: 22,
			},
			connectErr: nil,
			mockResponses: map[string]*domain.CommandResult{
				"uname -s -m":      {Output: "Linux x86_64", ExitCode: 0},
				"docker --version": {Output: "Docker version 26.1.3", ExitCode: 0},
				"Swarm.LocalNodeState": {Output: "inactive", ExitCode: 0},
			},
			expectedConnected:    true,
			expectedDockerActive: true,
			expectedSwarmActive:  false,
			expectedMsgContains:  "Swarm listo para inicializar",
		},
		{
			name: "connection success with swarm active",
			config: domain.ServerConfig{
				Host: "vps.example.com",
				Port: 22,
			},
			connectErr: nil,
			mockResponses: map[string]*domain.CommandResult{
				"uname -s -m":      {Output: "Linux x86_64", ExitCode: 0},
				"docker --version": {Output: "Docker version 26.1.3", ExitCode: 0},
				"Swarm.LocalNodeState": {Output: "active", ExitCode: 0},
			},
			expectedConnected:    true,
			expectedDockerActive: true,
			expectedSwarmActive:  true,
			expectedMsgContains:  "Clúster Docker Swarm activo",
		},
		{
			name: "localhost connection success",
			config: domain.ServerConfig{
				Host: "localhost",
			},
			connectErr: nil,
			mockResponses: map[string]*domain.CommandResult{
				"uname -s -m":      {Output: "Darwin arm64", ExitCode: 0},
				"docker --version": {Output: "Docker version 27.0.3", ExitCode: 0},
				"Swarm.LocalNodeState": {Output: "active", ExitCode: 0},
			},
			expectedConnected:    true,
			expectedDockerActive: true,
			expectedSwarmActive:  true,
			expectedMsgContains:  "Clúster Docker Swarm activo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSSH := mocks.NewMockSSHExecutor()
			mockSSH.ConnectError = tt.connectErr
			mockSSH.MockResponses = tt.mockResponses

			uc := NewConnectServerUseCase(mockSSH)
			result, err := uc.Execute(tt.config)

			if err != nil {
				t.Fatalf("unexpected error executing usecase: %v", err)
			}
			if result.Connected != tt.expectedConnected {
				t.Errorf("expected Connected = %v, got %v", tt.expectedConnected, result.Connected)
			}
			if result.DockerActive != tt.expectedDockerActive {
				t.Errorf("expected DockerActive = %v, got %v", tt.expectedDockerActive, result.DockerActive)
			}
			if result.SwarmActive != tt.expectedSwarmActive {
				t.Errorf("expected SwarmActive = %v, got %v", tt.expectedSwarmActive, result.SwarmActive)
			}
			if !strings.Contains(result.Message, tt.expectedMsgContains) {
				t.Errorf("expected message to contain %q, got %q", tt.expectedMsgContains, result.Message)
			}
		})
	}
}
