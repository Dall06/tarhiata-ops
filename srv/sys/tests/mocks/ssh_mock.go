package mocks

import (
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

type MockSSHExecutor struct {
	// Historial de comandos ejecutados
	CommandsExecuted []string

	// Configurar respuestas simuladas para comandos específicos
	// Key: subcadena del comando, Value: resultado simulado
	MockResponses map[string]*domain.CommandResult

	// Error global simulado si falla la conexión
	ConnectError error

	// RunCommandErr, si no es nil, se devuelve como error de Go real en RunCommand
	// (independiente del ExitCode simulado en MockResponses), para probar caídas de
	// transporte SSH en vez de solo fallas de exit code.
	RunCommandErr error

	// WriteRemoteFileErr, si no es nil, se devuelve como error real de WriteRemoteFile.
	WriteRemoteFileErr error

	// WriteRemoteFileCapture, si es true, guarda el último contenido escrito en LastWrittenContent.
	WriteRemoteFileCapture bool
	LastWrittenContent     string

	// CloseCalls cuenta cuántas veces se llamó Close(), para detectar fugas de conexión.
	CloseCalls int
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

	if m.RunCommandErr != nil {
		return &domain.CommandResult{Output: "", ExitCode: -1, Error: m.RunCommandErr}, m.RunCommandErr
	}

	for key, res := range m.MockResponses {
		if strings.Contains(cmd, key) {
			return res, nil
		}
	}

	// Respuesta exitosa por defecto
	return &domain.CommandResult{Output: "success", ExitCode: 0}, nil
}

// RunCommandStreaming replica la misma lógica de resolución de respuesta que RunCommand,
// pero además invoca onLine por cada línea del Output resuelto, para poder probar
// callers que dependen del streaming línea por línea sin necesitar una sesión SSH real.
func (m *MockSSHExecutor) RunCommandStreaming(cmd string, onLine func(line string)) (*domain.CommandResult, error) {
	res, err := m.RunCommand(cmd)
	if res != nil && onLine != nil {
		for _, line := range strings.Split(res.Output, "\n") {
			if strings.TrimSpace(line) != "" {
				onLine(line)
			}
		}
	}
	return res, err
}

func (m *MockSSHExecutor) Close() error {
	m.CloseCalls++
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

func (m *MockSSHExecutor) WriteRemoteFile(remotePath, content string) error {
	m.CommandsExecuted = append(m.CommandsExecuted, "WRITE "+remotePath)
	if m.WriteRemoteFileCapture {
		m.LastWrittenContent = content
	}
	return m.WriteRemoteFileErr
}
