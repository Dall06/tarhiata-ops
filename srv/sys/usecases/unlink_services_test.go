package usecases

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestUnlinkServices_Execute(t *testing.T) {
	mockSSH := mocks.NewMockSSHExecutor()

	usecase := NewUnlinkServicesUseCase(nil, mockSSH)

	err := usecase.Execute("api-backend", "db-postgres")
	if err != nil {
		t.Fatalf("se esperaba éxito al eliminar link, se obtuvo error: %v", err)
	}
}
