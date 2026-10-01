package usecases

import (
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestJoinExistingWorker_Execute(t *testing.T) {
	tests := []struct {
		name              string
		managerJoinToken  *domain.CommandResult
		workerDockerCheck *domain.CommandResult
		managerHost       string
		nodeName          string
		labelType         string
		expectError       bool
		expectWorkerCmds  []string
		expectManagerCmds []string
	}{
		{
			name:             "join existing worker success, docker ya instalado",
			managerJoinToken: &domain.CommandResult{Output: "SWMTKN-MOCK-TOKEN\n", ExitCode: 0},
			workerDockerCheck: &domain.CommandResult{Output: "/usr/bin/docker\n", ExitCode: 0},
			managerHost:       "1.1.1.1",
			nodeName:          "worker-existente",
			labelType:         "database",
			expectError:       false,
			expectWorkerCmds: []string{
				"insecure-registries",
				"docker swarm join --token SWMTKN-MOCK-TOKEN 1.1.1.1:2377",
			},
			expectManagerCmds: []string{
				"docker swarm join-token worker -q",
				"docker node update --label-add type=database worker-existente",
			},
		},
		{
			name:             "join existing worker sin docker: lo instala primero",
			managerJoinToken: &domain.CommandResult{Output: "SWMTKN-MOCK-TOKEN\n", ExitCode: 0},
			workerDockerCheck: &domain.CommandResult{Output: "", ExitCode: 1},
			managerHost:       "9.9.9.9",
			nodeName:          "worker-2",
			labelType:         "worker",
			expectError:       false,
			expectWorkerCmds: []string{
				"get-docker.sh",
				"docker swarm join --token SWMTKN-MOCK-TOKEN 9.9.9.9:2377",
			},
		},
		{
			name:             "falla si el manager no tiene swarm activo",
			managerJoinToken: &domain.CommandResult{Output: "this node is not a swarm manager", ExitCode: 1},
			managerHost:      "1.1.1.1",
			nodeName:         "worker-1",
			labelType:        "worker",
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManagerSSH := mocks.NewMockSSHExecutor()
			mockManagerSSH.MockResponses["join-token"] = tt.managerJoinToken

			mockWorkerSSH := mocks.NewMockSSHExecutor()
			if tt.workerDockerCheck != nil {
				mockWorkerSSH.MockResponses["command -v docker"] = tt.workerDockerCheck
			}

			uc := NewJoinExistingWorkerUseCase(mockManagerSSH, mockWorkerSSH)
			err := uc.Execute(tt.managerHost, tt.nodeName, tt.labelType)

			if tt.expectError && err == nil {
				t.Fatalf("se esperaba un error y no ocurrió ninguno")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("no se esperaba error, se obtuvo: %v", err)
			}

			for _, expected := range tt.expectWorkerCmds {
				found := false
				for _, cmd := range mockWorkerSSH.CommandsExecuted {
					if strings.Contains(cmd, expected) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("se esperaba que el worker ejecutara un comando conteniendo %q, comandos: %v", expected, mockWorkerSSH.CommandsExecuted)
				}
			}

			for _, expected := range tt.expectManagerCmds {
				found := false
				for _, cmd := range mockManagerSSH.CommandsExecuted {
					if strings.Contains(cmd, expected) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("se esperaba que el manager ejecutara un comando conteniendo %q, comandos: %v", expected, mockManagerSSH.CommandsExecuted)
				}
			}
		})
	}
}
