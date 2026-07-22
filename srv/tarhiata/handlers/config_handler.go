package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/repositories"
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
					huh.NewOption("🐳 Crear un servidor desde cero (DigitalOcean / Vultr)", "new"),
				).Value(&configType),
		),
	).Run()

	if err != nil {
		fmt.Println("Cancelado.")
		return current
	}

	var host, user, key, doToken string
	var portStr string = "22"

	if current != nil {
		host = current.Host
		portStr = fmt.Sprintf("%d", current.Port)
		user = current.User
		key = current.PrivateKey
		doToken = current.DOAPIToken
	}

	if configType == "existing" {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("IP del Servidor (Host)").Value(&host),
				huh.NewInput().Title("Puerto SSH").Value(&portStr),
				huh.NewInput().Title("Usuario").Value(&user),
				huh.NewInput().Title("Ruta de la Llave Privada (ej. ~/.ssh/id_rsa)").Value(&key),
				huh.NewInput().Title("DigitalOcean API Token (Opcional, para BDs)").Value(&doToken),
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
						huh.NewOption("DigitalOcean", "digitalocean"),
						huh.NewOption("Vultr", "vultr"),
					).Value(&providerName),
			),
		).Run(); err != nil {
			fmt.Println("Cancelado.")
			return current
		}

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title(fmt.Sprintf("%s API Token (Obligatorio)", strings.Title(providerName))).Value(&doToken).Validate(func(s string) error {
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

		fmt.Printf("\n⏳ [Terraform] Construyendo el servidor maestro en %s. Esto tardará un poco...\n", providerName)
		homeDir, _ := os.UserHomeDir()
		workspace := filepath.Join(homeDir, ".config", "tarhiata", "terraform", "tarhiata_master")

		var provisioner ports.Provisioner
		var region string
		if providerName == "digitalocean" {
			provisioner = repositories.NewDigitalOceanProvisioner(workspace)
			region = "nyc1" // DigitalOcean Region
		} else {
			provisioner = repositories.NewVultrProvisioner(workspace)
			region = "ewr" // Vultr Region (New Jersey)
		}

		newIP, privKey, err := provisioner.ProvisionNode(doToken, "tarhiata-master", region)
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

		if err := os.WriteFile(key, []byte(privKey), 0600); err != nil {
			fmt.Printf("❌ Error guardando la llave privada: %v\n", err)
			return current
		}

		fmt.Printf("✅ Servidor maestro creado exitosamente en %s\n", newIP)
	}

	port, _ := strconv.Atoi(portStr)
	newConfig := domain.ServerConfig{
		Host:       host,
		Port:       port,
		User:       user,
		PrivateKey: key,
		DOAPIToken: doToken,
	}

	if err := h.repo.SaveServerConfig(newConfig); err != nil {
		fmt.Printf("❌ Error guardando configuración: %v\n", err)
		return current
	}

	fmt.Println("✅ ¡Configuración guardada exitosamente!")
	return &newConfig
}
