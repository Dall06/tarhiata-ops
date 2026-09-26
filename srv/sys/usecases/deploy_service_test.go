package usecases

import (
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestDeployServiceUseCase_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		dockerVersion *domain.CommandResult
		swarmState    *domain.CommandResult
		service       domain.CustomService
		config        domain.DeployConfig
		expectError   bool
		expectedError string
	}{
		{
			name:          "Deploy fails when Docker is not installed on VPS",
			dockerVersion: &domain.CommandResult{Output: "docker: command not found", ExitCode: 127},
			service:       domain.CustomService{Name: "my-app"},
			config:        domain.DeployConfig{ImageSource: "nginx:alpine", Port: 80},
			expectError:   true,
			expectedError: "Docker no está disponible",
		},
		{
			name:          "Deploy succeeds with domain and SSL on active swarm",
			dockerVersion: &domain.CommandResult{Output: "Docker version 24.0.5", ExitCode: 0},
			swarmState:    &domain.CommandResult{Output: "active", ExitCode: 0},
			service:       domain.CustomService{Name: "web-prod"},
			config: domain.DeployConfig{
				ImageSource: "nginx:alpine",
				Port:        80,
				Domain:      "app.example.com",
				Expose:      true,
				EnableSSL:   true,
				TargetNode:  "manager",
			},
			expectError: false,
		},
		{
			name:          "Deploy succeeds and auto-initializes swarm when inactive",
			dockerVersion: &domain.CommandResult{Output: "Docker version 24.0.5", ExitCode: 0},
			swarmState:    &domain.CommandResult{Output: "inactive", ExitCode: 0},
			service:       domain.CustomService{Name: "api-backend"},
			config: domain.DeployConfig{
				ImageSource: "node:18-alpine",
				Port:        3000,
				Domain:      "",
				Expose:      true,
				TargetNode:  "vps-primary",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSSH := mocks.NewMockSSHExecutor()
			if tt.dockerVersion != nil {
				mockSSH.MockResponses["docker --version"] = tt.dockerVersion
			}
			if tt.swarmState != nil {
				mockSSH.MockResponses["docker info --format"] = tt.swarmState
			}

			uc := NewDeployServiceUseCase(mockSSH)
			err := uc.Execute(tt.service, tt.config)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.expectedError)
				}
				if !strings.Contains(err.Error(), tt.expectedError) {
					t.Fatalf("expected error containing %q, got %v", tt.expectedError, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestFormatNodeConstraint_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty node defaults to manager role",
			input:    "",
			expected: "node.role == manager",
		},
		{
			name:     "Manager node defaults to manager role",
			input:    "manager",
			expected: "node.role == manager",
		},
		{
			name:     "Master node defaults to manager role",
			input:    "master",
			expected: "node.role == manager",
		},
		{
			name:     "Worker node sets worker role",
			input:    "worker",
			expected: "node.role == worker",
		},
		{
			name:     "VPS label fallback to manager role",
			input:    "vps-nyc1-prod",
			expected: "node.role == manager",
		},
		{
			name:     "Explicit equality constraint preserved",
			input:    "node.labels.zone == us-east",
			expected: "node.labels.zone == us-east",
		},
		{
			name:     "Specific hostname constraint",
			input:    "worker-node-01",
			expected: "node.hostname == worker-node-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := formatNodeConstraint(tt.input)
			if actual != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, actual)
			}
		})
	}
}
