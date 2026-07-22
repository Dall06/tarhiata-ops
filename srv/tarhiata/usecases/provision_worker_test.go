package usecases

import (
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/tests/mocks"
)

func TestProvisionWorker_Execute(t *testing.T) {
	tests := []struct {
		name                string
		config              domain.ServerConfig
		nodeName            string
		labelType           string
		expectError         bool
		expectWorkerCmds    []string
		expectManagerCmds   []string
		expectedIP          string
	}{
		{
			name: "Provision worker success",
			config: domain.ServerConfig{
				DOAPIToken: "mock-token",
				Host:       "1.1.1.1",
			},
			nodeName:  "worker-1",
			labelType: "db",
			expectError: false,
			expectedIP:  "2.2.2.2",
			expectWorkerCmds: []string{
				"cloud-init status --wait",
				"iptables -I DOCKER-USER",
				"docker swarm join",
			},
			expectManagerCmds: []string{
				"docker swarm join-token worker -q",
				"docker node update --label-add type=db worker-1",
			},
		},
		{
			name: "Fails without DO Token",
			config: domain.ServerConfig{
				DOAPIToken: "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManagerSSH := mocks.NewMockSSHExecutor()
			mockManagerSSH.MockResponses["join-token"] = &domain.CommandResult{Output: "SWMTKN-MOCK-TOKEN\n", ExitCode: 0}

			mockProvisioner := mocks.NewMockProvisioner()
			mockProvisioner.MockIP = "2.2.2.2"
			mockProvisioner.MockPrivKey = "mock-key-content"

			mockWorkerSSH := mocks.NewMockSSHExecutor()

			uc := &ProvisionWorkerUseCase{
				managerSSH: mockManagerSSH,
				Provisioner: mockProvisioner,
				WorkerSSHFactory: func() ports.SSHExecutor {
					return mockWorkerSSH
				},
			}

			ip, err := uc.Execute(tt.config, tt.nodeName, tt.labelType)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ip != tt.expectedIP {
				t.Errorf("expected IP %s, got %s", tt.expectedIP, ip)
			}

			// Validate Worker Commands
			allWorkerCmds := strings.Join(mockWorkerSSH.CommandsExecuted, " ||| ")
			for _, expected := range tt.expectWorkerCmds {
				if !strings.Contains(allWorkerCmds, expected) {
					t.Errorf("worker SSH missed command: %s (executed: %s)", expected, allWorkerCmds)
				}
			}

			// Validate Manager Commands
			allManagerCmds := strings.Join(mockManagerSSH.CommandsExecuted, " ||| ")
			for _, expected := range tt.expectManagerCmds {
				if !strings.Contains(allManagerCmds, expected) {
					t.Errorf("manager SSH missed command: %s (executed: %s)", expected, allManagerCmds)
				}
			}
		})
	}
}
