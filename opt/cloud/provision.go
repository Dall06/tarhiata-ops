package cloud

import (
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

// Provision ejecuta el aprovisionamiento directo en el proveedor indicado ("vultr" o "digitalocean").
func Provision(provider, workspace, token, nodeName, region, plan string) (domain.NodeProvisionResult, error) {
	p := strings.ToLower(strings.TrimSpace(provider))
	if p == "digitalocean" || p == "do" {
		return ProvisionDigitalOcean(workspace, token, nodeName, region, plan)
	}
	return ProvisionVultr(workspace, token, nodeName, region, plan)
}

// Destroy ejecuta la destrucción de la infraestructura en el proveedor indicado.
func Destroy(provider, workspace, token, nodeName string) error {
	p := strings.ToLower(strings.TrimSpace(provider))
	if p == "digitalocean" || p == "do" {
		return DestroyDigitalOcean(workspace, token, nodeName)
	}
	return DestroyVultr(workspace, token, nodeName)
}

// Provisioner implementa ports.Provisioner directamente como adaptador simple en opt/cloud.
type Provisioner struct {
	Provider  string
	Workspace string
}

func NewProvisioner(provider, workspace string) *Provisioner {
	return &Provisioner{
		Provider:  provider,
		Workspace: workspace,
	}
}

func (p *Provisioner) ProvisionNode(token string, nodeName string, region string, plan string) (domain.NodeProvisionResult, error) {
	return Provision(p.Provider, p.Workspace, token, nodeName, region, plan)
}

func (p *Provisioner) DestroyNode(token string, nodeName string) error {
	return Destroy(p.Provider, p.Workspace, token, nodeName)
}
