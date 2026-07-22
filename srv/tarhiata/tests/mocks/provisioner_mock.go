package mocks

type MockProvisioner struct {
	MockIP       string
	MockPrivKey  string
	MockError    error
	NodesCreated []string
}

func NewMockProvisioner() *MockProvisioner {
	return &MockProvisioner{
		NodesCreated: []string{},
	}
}

func (m *MockProvisioner) ProvisionNode(token string, nodeName string, region string) (string, string, error) {
	m.NodesCreated = append(m.NodesCreated, nodeName)
	return m.MockIP, m.MockPrivKey, m.MockError
}

func (m *MockProvisioner) DestroyNode(token string, nodeName string) error {
	return nil
}
