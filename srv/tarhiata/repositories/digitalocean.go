package repositories

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/pkg/terraform"
	"github.com/hashicorp/terraform-exec/tfexec"
)

// DigitalOceanProvisioner implementa ports.Provisioner usando Terraform
type DigitalOceanProvisioner struct {
	workspace string
}

func NewDigitalOceanProvisioner(workspace string) *DigitalOceanProvisioner {
	return &DigitalOceanProvisioner{
		workspace: workspace,
	}
}

// ProvisionNode delega la ejecución al Runner genérico de Terraform
func (p *DigitalOceanProvisioner) ProvisionNode(token string, nodeName string, region string) (string, string, error) {
	tfTemplate := `
terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
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

resource "digitalocean_droplet" "node" {
  image  = "ubuntu-22-04-x64"
  name   = "%s"
  region = "%s"
  size   = "s-1vcpu-1gb"
  ssh_keys = [digitalocean_ssh_key.node_key.fingerprint]
  
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
	tfContent := fmt.Sprintf(tfTemplate, nodeName, nodeName, region)

	runner, err := terraform.NewRunner(p.workspace)
	if err != nil {
		return "", "", err
	}

	vars := map[string]string{
		"do_token": token,
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

func (p *DigitalOceanProvisioner) DestroyNode(token string, nodeName string) error {
	tf, err := tfexec.NewTerraform(p.workspace, "terraform")
	if err != nil {
		return fmt.Errorf("error inicializando terraform: %w", err)
	}

	err = tf.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		return fmt.Errorf("error en terraform init: %w", err)
	}

	err = tf.Destroy(context.Background(),
		tfexec.Var(fmt.Sprintf("do_token=%s", token)),
		tfexec.Var(fmt.Sprintf("node_name=%s", nodeName)),
		tfexec.Var("region=nyc1"), // Asumimos default
	)
	if err != nil {
		return fmt.Errorf("error en terraform destroy: %w", err)
	}

	return nil
}
