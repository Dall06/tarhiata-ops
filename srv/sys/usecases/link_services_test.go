package usecases

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestLinkServices_Execute(t *testing.T) {
	mockSSH := mocks.NewMockSSHExecutor()

	usecase := NewLinkServicesUseCase(nil, mockSSH)

	link, err := usecase.Execute("api-backend", "db-postgres", "DATABASE_URL")
	if err != nil {
		t.Fatalf("se esperaba éxito en link, se obtuvo error: %v", err)
	}

	if link.EnvVarName != "DATABASE_URL" {
		t.Errorf("se esperaba ENV_VAR 'DATABASE_URL', se obtuvo '%s'", link.EnvVarName)
	}
}
