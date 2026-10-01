package usecases

import (
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestDeployRegistryUseCase_Execute(t *testing.T) {
	t.Run("despliega el stack del registry correctamente", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewDeployRegistryUseCase(mockSSH)

		if err := uc.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var deployed bool
		for _, c := range mockSSH.CommandsExecuted {
			if strings.Contains(c, "docker stack deploy -c /tmp/registry-stack.yml "+RegistryStackName) {
				deployed = true
			}
		}
		if !deployed {
			t.Errorf("no se ejecutó el deploy del stack del registry: %v", mockSSH.CommandsExecuted)
		}
	})

	t.Run("exit code distinto de cero propaga error", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		mockSSH.MockResponses["docker stack deploy"] = &domain.CommandResult{Output: "manifest inválido", ExitCode: 1}
		uc := NewDeployRegistryUseCase(mockSSH)

		if err := uc.Execute(); err == nil || !strings.Contains(err.Error(), "manifest inválido") {
			t.Fatalf("se esperaba error con la salida del comando, got: %v", err)
		}
	})
}
