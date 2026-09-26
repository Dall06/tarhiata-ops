package mocks

import "github.com/Dall06/tarhiata-ops/srv/sys/domain"

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

func (m *MockProvisioner) ProvisionNode(token string, nodeName string, region string, plan string) (domain.NodeProvisionResult, error) {
	m.NodesCreated = append(m.NodesCreated, nodeName)
	return domain.NodeProvisionResult{
		PublicIP:   m.MockIP,
		PrivateKey: m.MockPrivKey,
	}, m.MockError
}

func (m *MockProvisioner) DestroyNode(token string, nodeName string) error {
	return nil
}

