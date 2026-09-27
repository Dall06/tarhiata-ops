package repositories

import (
	"github.com/Dall06/tarhiata-ops/opt/cloud"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

// VultrProvisioner implementa ports.Provisioner para Vultr delegando en opt/cloud.
type VultrProvisioner struct {
	workspace string
}

func NewVultrProvisioner(workspace string) *VultrProvisioner {
	return &VultrProvisioner{
		workspace: workspace,
	}
}

func (p *VultrProvisioner) ProvisionNode(token string, nodeName string, region string, plan string) (domain.NodeProvisionResult, error) {
	return cloud.ProvisionVultr(p.workspace, token, nodeName, region, plan)
}

func (p *VultrProvisioner) DestroyNode(token string, nodeName string) error {
	return cloud.DestroyVultr(p.workspace, token, nodeName)
}
