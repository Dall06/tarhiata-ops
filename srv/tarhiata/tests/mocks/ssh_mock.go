package mocks

import (
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
)

type MockSSHExecutor struct {
	// Historial de comandos ejecutados
	CommandsExecuted []string
	
	// Configurar respuestas simuladas para comandos específicos
	// Key: subcadena del comando, Value: resultado simulado
	MockResponses map[string]*domain.CommandResult
	
	// Error global simulado si falla la conexión
	ConnectError error
}

func NewMockSSHExecutor() *MockSSHExecutor {
	return &MockSSHExecutor{
		CommandsExecuted: []string{},
		MockResponses:    make(map[string]*domain.CommandResult),
	}
}

func (m *MockSSHExecutor) Connect(config domain.ServerConfig) error {
	return m.ConnectError
}

func (m *MockSSHExecutor) RunCommand(cmd string) (*domain.CommandResult, error) {
	m.CommandsExecuted = append(m.CommandsExecuted, cmd)
	
	for key, res := range m.MockResponses {
		if strings.Contains(cmd, key) {
			return res, nil
		}
	}
	
	// Respuesta exitosa por defecto
	return &domain.CommandResult{Output: "success", ExitCode: 0}, nil
}

func (m *MockSSHExecutor) Close() error {
	return nil
}

func (m *MockSSHExecutor) CheckConnection() bool {
	return true
}

func (m *MockSSHExecutor) InteractiveShell() error {
	return nil
}

func (m *MockSSHExecutor) InteractiveCommand(cmd string) error {
	return nil
}
