package repositories

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/terraform-exec/tfexec"
)

// DigitalOceanProvisioner implementa ports.Provisioner usando Terraform
type DigitalOceanProvisioner struct {
	workspace string // Directorio donde se guardan los .tf y el estado
}

func NewDigitalOceanProvisioner(workspace string) *DigitalOceanProvisioner {
	return &DigitalOceanProvisioner{
		workspace: workspace,
	}
}

// ProvisionNode descarga terraform (si no existe), genera el .tf y ejecuta tf apply
func (p *DigitalOceanProvisioner) ProvisionNode(token string, nodeName string, region string) (string, string, error) {
	fmt.Println("⏳ [Terraform] Preparando binario de Terraform...")
	
	// 1. Descarga e instala Terraform automáticamente de forma silenciosa
	installer := &releases.ExactVersion{
		Product: product.Terraform,
		Version: version.Must(version.NewVersion("1.5.7")),
	}

	execPath, err := installer.Install(context.Background())
	if err != nil {
		return "", "", fmt.Errorf("error instalando terraform: %w", err)
	}

	// 2. Crea el directorio de trabajo si no existe
	if err := os.MkdirAll(p.workspace, 0755); err != nil {
		return "", "", fmt.Errorf("error creando workspace de terraform: %w", err)
	}

	// 3. Escribe el main.tf dinámicamente (Plantilla HCL para DigitalOcean Droplet + Firewall + SSH)
	tfTemplate := fmt.Sprintf(`
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
  name       = "tarhiata-key-%%s"
  public_key = tls_private_key.node_key.public_key_openssh
}

resource "digitalocean_droplet" "node" {
  image  = "ubuntu-22-04-x64"
  name   = "%%s"
  region = "%%s"
  size   = "s-1vcpu-1gb"
  ssh_keys = [digitalocean_ssh_key.node_key.fingerprint]
  
  # user_data (Cloud-init) para instalar Docker de fábrica
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
`)
	// Usamos Sprintf dos veces porque el template original usa %s y queremos que evalúe nodeName y region.
	// Haremos fmt.Sprintf del template ya con nodeName para SSH y Droplet
	tfTemplate = strings.ReplaceAll(tfTemplate, "%%s", "%s")
	tfContent := fmt.Sprintf(tfTemplate, nodeName, nodeName, region)

	tfFilePath := filepath.Join(p.workspace, "main.tf")
	if err := os.WriteFile(tfFilePath, []byte(tfContent), 0644); err != nil {
		return "", "", fmt.Errorf("error escribiendo main.tf: %w", err)
	}

	fmt.Println("🚀 [Terraform] Inicializando módulos...")
	tf, err := tfexec.NewTerraform(p.workspace, execPath)
	if err != nil {
		return "", "", fmt.Errorf("error creando instancia tfexec: %w", err)
	}

	err = tf.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		return "", "", fmt.Errorf("error en tf init: %w", err)
	}

	fmt.Println("🏗️  [Terraform] Aprovisionando infraestructura (esto puede tomar 1-2 minutos)...")
	err = tf.Apply(context.Background(), tfexec.Var("do_token="+token))
	if err != nil {
		return "", "", fmt.Errorf("error en tf apply: %w", err)
	}

	// 4. Extraer el output de la IP Pública y Llave Privada
	outputs, err := tf.Output(context.Background())
	if err != nil {
		return "", "", fmt.Errorf("error leyendo tf outputs: %w", err)
	}

	publicIP := outputs["public_ip"].Value
	ipStr := string(publicIP)
	if len(ipStr) > 2 {
		ipStr = ipStr[1 : len(ipStr)-1]
	}

	privateKey := outputs["private_key"].Value
	// Remover espacios invisibles o JSON string wrappers de Terraform
	pkStr := strings.Trim(string(privateKey), "\"")
	pkStr = strings.ReplaceAll(pkStr, "\\n", "\n")

	return ipStr, pkStr, nil
}

func (p *DigitalOceanProvisioner) DestroyNode(token string, nodeName string) error {
	// A implementar con tf.Destroy()
	return nil
}
