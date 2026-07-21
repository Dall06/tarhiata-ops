package repositories

import (
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/pkg/terraform"
)

// VultrProvisioner implementa ports.Provisioner usando Terraform para Vultr
type VultrProvisioner struct {
	workspace string
}

func NewVultrProvisioner(workspace string) *VultrProvisioner {
	return &VultrProvisioner{
		workspace: workspace,
	}
}

// ProvisionNode genera la plantilla HCL y delega la ejecución al Runner genérico de Terraform
func (p *VultrProvisioner) ProvisionNode(token string, nodeName string, region string) (string, string, error) {
	tfTemplate := `
terraform {
  required_providers {
    vultr = {
      source  = "vultr/vultr"
      version = "~> 2.15"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
  }
}

variable "vultr_api_key" {}

provider "vultr" {
  api_key     = var.vultr_api_key
  rate_limit  = 700
  retry_limit = 3
}

resource "tls_private_key" "node_key" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "vultr_ssh_key" "node_key" {
  name     = "tarhiata-key-%s"
  ssh_key  = tls_private_key.node_key.public_key_openssh
}

resource "vultr_instance" "node" {
  plan        = "vc2-1c-1gb"
  region      = "%s"
  os_id       = 1743 # Ubuntu 22.04 LTS x64
  label       = "%s"
  hostname    = "%s"
  ssh_key_ids = [vultr_ssh_key.node_key.id]
  
  user_data = <<-EOF
              #!/bin/bash
              export DEBIAN_FRONTEND=noninteractive
              curl -fsSL https://get.docker.com -o get-docker.sh
              sh get-docker.sh
              EOF
}

output "public_ip" {
  value = vultr_instance.node.main_ip
}

output "private_key" {
  value     = tls_private_key.node_key.private_key_pem
  sensitive = true
}
`
	tfContent := fmt.Sprintf(tfTemplate, nodeName, region, nodeName, nodeName)

	runner, err := terraform.NewRunner(p.workspace)
	if err != nil {
		return "", "", err
	}

	vars := map[string]string{
		"vultr_api_key": token,
	}

	outputs, err := runner.Apply(tfContent, vars)
	if err != nil {
		return "", "", err
	}

	ipStr := outputs["public_ip"]
	pkStr := outputs["private_key"]

	// Limpiar un poco más por seguridad, aunque el Runner ya lo hace
	pkStr = strings.TrimSpace(pkStr)

	return ipStr, pkStr, nil
}

func (p *VultrProvisioner) DestroyNode(token string, nodeName string) error {
	return nil
}
