package mocks

import (
	sysdomain "github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

// MockConfigRepository simula ports.ConfigRepository para pruebas en srv/ui.
type MockConfigRepository struct {
	Configs       map[string]sysdomain.ServerConfig
	SaveErr       error
	LoadErr       error
	DeleteErr     error
	ListErr       error
	SaveCalled    bool
	LoadCalled    bool
	DeleteCalled  bool
	ListCalled    bool
}

// NewMockConfigRepository inicializa una instancia de MockConfigRepository.
func NewMockConfigRepository() *MockConfigRepository {
	return &MockConfigRepository{
		Configs: make(map[string]sysdomain.ServerConfig),
	}
}

func (m *MockConfigRepository) Save(config sysdomain.ServerConfig) error {
	m.SaveCalled = true
	if m.SaveErr != nil {
		return m.SaveErr
	}
	m.Configs[config.Name] = config
	return nil
}

func (m *MockConfigRepository) Load(name string) (sysdomain.ServerConfig, error) {
	m.LoadCalled = true
	if m.LoadErr != nil {
		return sysdomain.ServerConfig{}, m.LoadErr
	}
	cfg, ok := m.Configs[name]
	if !ok {
		return sysdomain.ServerConfig{Name: name}, nil
	}
	return cfg, nil
}

func (m *MockConfigRepository) Delete(name string) error {
	m.DeleteCalled = true
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	delete(m.Configs, name)
	return nil
}

func (m *MockConfigRepository) List() ([]sysdomain.ServerConfig, error) {
	m.ListCalled = true
	if m.ListErr != nil {
		return nil, m.ListErr
	}
	var list []sysdomain.ServerConfig
	for _, c := range m.Configs {
		list = append(list, c)
	}
	return list, nil
}
