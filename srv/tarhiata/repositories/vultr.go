package repositories

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/pkg/terraform"
	"github.com/hashicorp/terraform-exec/tfexec"
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

// ProvisionNode delega la ejecución al Runner genérico de Terraform
func (p *VultrProvisioner) ProvisionNode(token string, nodeName string, region string) (string, string, error) {
	tfTemplate := `
terraform {
  required_providers {
    vultr = {
      source  = "vultr/vultr"
      version = "~> 2.19"
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
  rate_limit  = 100
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

resource "vultr_firewall_group" "swarm_internal" {
  description = "tarhiata-swarm-%s"
}

resource "vultr_firewall_rule" "ssh" {
  firewall_group_id = vultr_firewall_group.swarm_internal.id
  protocol          = "tcp"
  ip_type           = "v4"
  subnet            = "0.0.0.0"
  subnet_size       = 0
  port              = "22"
}

resource "vultr_firewall_rule" "swarm_2377" {
  firewall_group_id = vultr_firewall_group.swarm_internal.id
  protocol          = "tcp"
  ip_type           = "v4"
  subnet            = "0.0.0.0"
  subnet_size       = 0
  port              = "2377"
}

resource "vultr_firewall_rule" "swarm_7946_tcp" {
  firewall_group_id = vultr_firewall_group.swarm_internal.id
  protocol          = "tcp"
  ip_type           = "v4"
  subnet            = "0.0.0.0"
  subnet_size       = 0
  port              = "7946"
}

resource "vultr_firewall_rule" "swarm_7946_udp" {
  firewall_group_id = vultr_firewall_group.swarm_internal.id
  protocol          = "udp"
  ip_type           = "v4"
  subnet            = "0.0.0.0"
  subnet_size       = 0
  port              = "7946"
}

resource "vultr_firewall_rule" "swarm_4789_udp" {
  firewall_group_id = vultr_firewall_group.swarm_internal.id
  protocol          = "udp"
  ip_type           = "v4"
  subnet            = "0.0.0.0"
  subnet_size       = 0
  port              = "4789"
}

resource "vultr_instance" "node" {
  plan              = "vc2-1c-1gb"
  region            = "%s"
  os_id             = 1743
  label             = "%s"
  hostname          = "%s"
  ssh_key_ids       = [vultr_ssh_key.node_key.id]
  firewall_group_id = vultr_firewall_group.swarm_internal.id
  
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
	tfContent := fmt.Sprintf(tfTemplate, nodeName, nodeName, region, nodeName, nodeName)

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

	pkStr = strings.TrimSpace(pkStr)

	return ipStr, pkStr, nil
}

func (p *VultrProvisioner) DestroyNode(token string, nodeName string) error {
	tf, err := tfexec.NewTerraform(p.workspace, "terraform")
	if err != nil {
		return fmt.Errorf("error inicializando terraform: %w", err)
	}

	err = tf.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		return fmt.Errorf("error en terraform init: %w", err)
	}

	err = tf.Destroy(context.Background(),
		tfexec.Var(fmt.Sprintf("vultr_api_key=%s", token)),
	)
	if err != nil {
		return fmt.Errorf("error en terraform destroy: %w", err)
	}

	return nil
}
