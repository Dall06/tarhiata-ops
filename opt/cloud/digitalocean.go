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

const doTerraformTemplate = `
terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.34"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
  }
}

variable "do_token" {}

provider "digitalocean" {
  token = var.do_token
}

resource "tls_private_key" "node_key" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "digitalocean_ssh_key" "node_key" {
  name       = "tarhiata-key-%s"
  public_key = tls_private_key.node_key.public_key_openssh
}

resource "digitalocean_firewall" "swarm_internal" {
  name = "tarhiata-swarm-%s"

  droplet_ids = [digitalocean_droplet.node.id]

  inbound_rule {
    protocol         = "tcp"
    port_range       = "22"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "2377"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "7946"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "udp"
    port_range       = "7946"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "udp"
    port_range       = "4789"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "tcp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "udp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }
}

resource "digitalocean_droplet" "node" {
  image     = "ubuntu-22-04-x64"
  name      = "%s"
  region    = "%s"
  size      = "%s"
  ssh_keys  = [digitalocean_ssh_key.node_key.id]
  user_data = <<-EOF
              #!/bin/bash
              export DEBIAN_FRONTEND=noninteractive
              curl -fsSL https://get.docker.com -o get-docker.sh
              sh get-docker.sh
              EOF
}

output "public_ip" {
  value = digitalocean_droplet.node.ipv4_address
}

output "private_key" {
  value     = tls_private_key.node_key.private_key_pem
  sensitive = true
}
`

// ProvisionDigitalOcean aprovisiona directamente un Droplet en DigitalOcean usando Terraform.
func ProvisionDigitalOcean(workspace, token, nodeName, region, size string) (domain.NodeProvisionResult, error) {
	reg, _ := validator.NormalizeRegion("do", region)
	if strings.TrimSpace(size) == "" {
		size = "s-1vcpu-1gb"
	}

	tfContent := fmt.Sprintf(doTerraformTemplate, nodeName, nodeName, nodeName, reg, size)

	runner, err := terraform.NewRunner(workspace)
	if err != nil {
		return domain.NodeProvisionResult{}, err
	}

	vars := map[string]string{
		"do_token": token,
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

// DestroyDigitalOcean destruye una instancia aprovisionada en DigitalOcean.
func DestroyDigitalOcean(workspace, token, nodeName string) error {
	tf, err := tfexec.NewTerraform(workspace, "terraform")
	if err != nil {
		return fmt.Errorf("opt/cloud: error inicializando terraform: %w", err)
	}

	if err := tf.Init(context.Background(), tfexec.Upgrade(true)); err != nil {
		return fmt.Errorf("opt/cloud: error en terraform init: %w", err)
	}

	if err := tf.Destroy(context.Background(), tfexec.Var(fmt.Sprintf("do_token=%s", token))); err != nil {
		return fmt.Errorf("opt/cloud: error en terraform destroy: %w", err)
	}

	return nil
}
