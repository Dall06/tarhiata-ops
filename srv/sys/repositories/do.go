package repositories

import (
	"github.com/Dall06/tarhiata-ops/opt/cloud"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

// DigitalOceanProvisioner implementa ports.Provisioner para DigitalOcean delegando en opt/cloud.
type DigitalOceanProvisioner struct {
	workspace string
}

func NewDigitalOceanProvisioner(workspace string) *DigitalOceanProvisioner {
	return &DigitalOceanProvisioner{
		workspace: workspace,
	}
}

func (p *DigitalOceanProvisioner) ProvisionNode(token string, nodeName string, region string, plan string) (domain.NodeProvisionResult, error) {
	return cloud.ProvisionDigitalOcean(p.workspace, token, nodeName, region, plan)
}

func (p *DigitalOceanProvisioner) DestroyNode(token string, nodeName string) error {
	return cloud.DestroyDigitalOcean(p.workspace, token, nodeName)
}
