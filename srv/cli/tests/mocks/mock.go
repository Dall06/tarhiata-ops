package mocks

import (
	sysports "github.com/Dall06/tarhiata-ops/srv/sys/ports"
	sysmocks "github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

// MockConfigRepository provee un mock para tests de CLI reusando el mock de sys.
type MockConfigRepository = sysmocks.MockConfigRepository

// NewMockConfigRepository crea una nueva instancia de mock para CLI tests.
func NewMockConfigRepository() *MockConfigRepository {
	return sysmocks.NewMockConfigRepository()
}

// Ensure interface compliance
var _ sysports.ConfigRepository = (*MockConfigRepository)(nil)
