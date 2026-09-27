package cloud

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/pkg/terraform"
	"github.com/Dall06/tarhiata-ops/pkg/validator"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/hashicorp/terraform-exec/tfexec"
)

const vultrTerraformTemplate = `
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
  plan              = "%s"
  region            = "%s"
  os_id             = 1743
  label             = "%s"
  hostname          = "%s"
  ssh_key_ids       = [vultr_ssh_key.node_key.id]
  firewall_group_id = vultr_firewall_group.swarm_internal.id
  activation_email  = false
  backups           = "disabled"
  enable_ipv6       = true
  
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

// ProvisionVultr aprovisiona directamente una máquina virtual en Vultr usando Terraform.
func ProvisionVultr(workspace, token, nodeName, region, plan string) (domain.NodeProvisionResult, error) {
	reg, _ := validator.NormalizeRegion("vultr", region)
	if strings.TrimSpace(plan) == "" {
		plan = "vc2-1c-1gb"
	}

	tfContent := fmt.Sprintf(vultrTerraformTemplate, nodeName, nodeName, plan, reg, nodeName, nodeName)

	runner, err := terraform.NewRunner(workspace)
	if err != nil {
		return domain.NodeProvisionResult{}, err
	}

	vars := map[string]string{
		"vultr_api_key": token,
	}

	outputs, err := runner.Apply(tfContent, vars)
	if err != nil {
		return domain.NodeProvisionResult{}, err
	}

	return domain.NodeProvisionResult{
		PublicIP:   outputs["public_ip"],
		PrivateKey: strings.TrimSpace(outputs["private_key"]),
	}, nil
}

// DestroyVultr destruye una instancia aprovisionada en Vultr.
func DestroyVultr(workspace, token, nodeName string) error {
	tf, err := tfexec.NewTerraform(workspace, "terraform")
	if err != nil {
		return fmt.Errorf("opt/cloud: error inicializando terraform: %w", err)
	}

	if err := tf.Init(context.Background(), tfexec.Upgrade(true)); err != nil {
		return fmt.Errorf("opt/cloud: error en terraform init: %w", err)
	}

	if err := tf.Destroy(context.Background(), tfexec.Var(fmt.Sprintf("vultr_api_key=%s", token))); err != nil {
		return fmt.Errorf("opt/cloud: error en terraform destroy: %w", err)
	}

	return nil
}
