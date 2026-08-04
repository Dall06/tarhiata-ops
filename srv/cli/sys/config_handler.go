package sys

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
	"github.com/Dall06/tarhiata-ops/srv/sys/repositories"
	"github.com/charmbracelet/huh"
)

type configHandler struct {
	repo ports.ConfigRepository
}

func NewConfigHandler(repo ports.ConfigRepository) ports.ConfigHandler {
	return &configHandler{repo: repo}
}

func (h *configHandler) Execute(current *domain.ServerConfig) *domain.ServerConfig {
	var configType string

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("¿Dónde alojaremos el motor de Tarhiata-ops?").
				Options(
					huh.NewOption("🔌 Tengo un servidor existente (Requiere IP y SSH)", "existing"),
					huh.NewOption("🐳 Crear un servidor desde cero (Vultr)", "new"),
				).Value(&configType),
		),
	).Run()

	if err != nil {
		fmt.Println("Cancelado.")
		return current
	}

	var host, user, key, doToken, vultrToken, cloudProvider string
	var portStr string = "22"

	if current != nil {
		host = current.Host
		portStr = fmt.Sprintf("%d", current.Port)
		user = current.User
		key = current.PrivateKey
		vultrToken = current.VultrAPIToken
		doToken = current.DOAPIToken
		cloudProvider = current.CloudProvider
	}

	if configType == "existing" {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("IP del Servidor (Host)").Value(&host),
				huh.NewInput().Title("Puerto SSH").Value(&portStr),
				huh.NewInput().Title("Usuario").Value(&user),
				huh.NewInput().Title("Ruta de la Llave Privada (ej. ~/.ssh/id_rsa)").Value(&key),
				huh.NewInput().Title("Vultr API Token (Opcional)").Value(&vultrToken),
				huh.NewInput().Title("DigitalOcean API Token (Opcional)").Value(&doToken),
			),
		)

		if err := form.Run(); err != nil {
			fmt.Println("Cancelado.")
			return current
		}
	} else {
		// Modo Terraform (Desde cero)
		var providerName string
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Proveedor Cloud").
					Options(
						huh.NewOption("Vultr", "vultr"),
						huh.NewOption("Vultr / Cloud Provider API Key", "vultr"),
					).Value(&providerName),
			),
		).Run(); err != nil {
			fmt.Println("Cancelado.")
			return current
		}

		var tokenPtr *string
		if providerName == "vultr" {
			tokenPtr = &vultrToken
		} else {
			tokenPtr = &doToken
		}

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title(fmt.Sprintf("%s API Token (Obligatorio)", strings.Title(providerName))).Value(tokenPtr).Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("El Token es obligatorio")
					}
					return nil
				}),
			),
		)
		if err := form.Run(); err != nil {
			fmt.Println("Cancelado.")
			return current
		}

		var selectedPlan string
		planOptions := []huh.Option[string]{}
		if providerName == "vultr" {
			planOptions = []huh.Option[string]{
				huh.NewOption("🪙 Micro ($5/mes) - 1 vCPU / 1GB RAM", "vc2-1c-1gb"),
				huh.NewOption("🌿 Starter ($10/mes) - 1 vCPU / 2GB RAM", "vc2-1c-2gb"),
				huh.NewOption("🚀 Scale ($20/mes) - 2 vCPU / 4GB RAM", "vc2-2c-4gb"),
			}
		} else {
			planOptions = []huh.Option[string]{
				huh.NewOption("🪙 Micro ($6/mes) - 1 vCPU / 1GB RAM", "s-1vcpu-1gb"),
				huh.NewOption("🌿 Starter ($12/mes) - 1 vCPU / 2GB RAM", "s-1vcpu-2gb"),
				huh.NewOption("🚀 Scale ($24/mes) - 2 vCPU / 4GB RAM", "s-2vcpu-4gb"),
			}
		}

		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Selecciona el Plan de la VM").
					Options(planOptions...).
					Value(&selectedPlan),
			),
		).Run(); err != nil {
			fmt.Println("Cancelado.")
			return current
		}

		fmt.Printf("\n⏳ [Terraform] Construyendo el servidor maestro en %s (Plan: %s). Esto tardará un poco...\n", providerName, selectedPlan)
		homeDir, _ := os.UserHomeDir()
		workspace := filepath.Join(homeDir, ".config", "tarhiata", "terraform", "tarhiata_master")

		var provisioner ports.Provisioner
		var region string
		var activeToken string
		if providerName == "vultr" {
			provisioner = repositories.NewVultrProvisioner(workspace)
			region = "ewr"
			activeToken = vultrToken
		} else {
			provisioner = repositories.NewDigitalOceanProvisioner(workspace)
			region = "nyc1"
			activeToken = doToken
		}

		newIP, privKeyContent, err := provisioner.ProvisionNode(activeToken, "tarhiata-manager", region, selectedPlan)
		if err != nil {
			fmt.Printf("❌ Error provisionando el servidor: %v\n", err)
			return current
		}

		host = newIP
		user = "root" // Ubuntu DO Droplet default root
		portStr = "22"

		// Guardar llave privada localmente

		keyDir := filepath.Join(homeDir, ".ssh")
		os.MkdirAll(keyDir, 0700)
		key = filepath.Join(keyDir, "tarhiata_master_rsa")

		if err := os.WriteFile(key, []byte(privKeyContent), 0600); err != nil {
			fmt.Printf("❌ Error guardando la llave privada: %v\n", err)
			return current
		}

		fmt.Printf("✅ Servidor maestro creado exitosamente en %s\n", newIP)
		cloudProvider = providerName
	}

	if cloudProvider == "" {
		cloudProvider = "vultr" // Default fallback
	}

	port, _ := strconv.Atoi(portStr)
	newConfig := domain.ServerConfig{
		Host:       host,
		Port:       port,
		User:       user,
		PrivateKey: key,
		DOAPIToken: doToken,
		VultrAPIToken: vultrToken,
		CloudProvider: cloudProvider,
	}

	if err := h.repo.SaveServerConfig(newConfig); err != nil {
		fmt.Printf("❌ Error guardando configuración: %v\n", err)
		return current
	}

	fmt.Println("✅ ¡Configuración guardada exitosamente!")
	return &newConfig
}
