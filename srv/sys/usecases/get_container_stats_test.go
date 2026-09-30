package usecases

import (
	"errors"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestGetContainerStatsUseCase_Execute(t *testing.T) {
	t.Run("conexión SSH fallida propaga error", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		mockSSH.ConnectError = errors.New("no se pudo conectar")
		uc := NewGetContainerStatsUseCase(mockSSH)

		if _, err := uc.Execute("my-app", domain.ServerConfig{Host: "1.2.3.4"}); err == nil {
			t.Fatal("se esperaba error de conexión SSH")
		}
	})

	t.Run("stats real se parsean del JSON de docker stats", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		mockSSH.MockResponses["docker stats"] = &domain.CommandResult{
			Output:   `{"container":"my-app","cpu":"3.2%","memUsage":"50MiB / 1GiB","memPerc":"5.0%","netIo":"1kB / 2kB","blockIo":"0B / 0B"}`,
			ExitCode: 0,
		}
		uc := NewGetContainerStatsUseCase(mockSSH)

		stats, err := uc.Execute("my-app", domain.ServerConfig{Host: "1.2.3.4"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stats.CPUPerc != "3.2%" {
			t.Errorf("expected CPUPerc 3.2%%, got %s", stats.CPUPerc)
		}
	})

	t.Run("salida no-JSON degrada a fallback en vez de error", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		mockSSH.MockResponses["docker stats"] = &domain.CommandResult{
			Output:   "esto no es json",
			ExitCode: 0,
		}
		uc := NewGetContainerStatsUseCase(mockSSH)

		stats, err := uc.Execute("my-app", domain.ServerConfig{Host: "1.2.3.4"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stats.Container != "my-app" {
			t.Errorf("se esperaba fallback con Container='my-app', got %+v", stats)
		}
	})

	t.Run("exit code distinto de cero degrada a fallback", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		mockSSH.MockResponses["docker stats"] = &domain.CommandResult{
			Output:   "",
			ExitCode: 1,
		}
		uc := NewGetContainerStatsUseCase(mockSSH)

		stats, err := uc.Execute("my-app", domain.ServerConfig{Host: "1.2.3.4"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stats.Container != "my-app" {
			t.Errorf("se esperaba fallback con Container='my-app', got %+v", stats)
		}
	})
}
