package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/repositories"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/usecases"
	"github.com/charmbracelet/huh"
	"os"
	"path/filepath"
)

type observabilityHandler struct {
	repo ports.ConfigRepository
}

func NewObservabilityHandler(repo ports.ConfigRepository) ports.ObservabilityHandler {
	return &observabilityHandler{repo: repo}
}

func (h *observabilityHandler) Execute(config domain.ServerConfig) {
	obs, err := h.repo.GetObservability()
	if err != nil {
		fmt.Printf("❌ Error leyendo configuración de observabilidad: %v\n", err)
		return
	}

	var selectedAction string
	var options []huh.Option[string]

	if obs == nil {
		options = append(options, huh.NewOption("➕ Configurar Stack de Observabilidad", "configure"))
	} else {
		options = append(options, huh.NewOption(fmt.Sprintf("📊 Administrar Stack (Tipo: %s)", obs.DeployType), "manage"))
	}
	options = append(options, huh.NewOption("🔙 Volver al Menú Principal", "back"))

	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Gestión de Logs y Métricas").
				Options(options...).
				Value(&selectedAction),
		),
	).Run()

	if err != nil || selectedAction == "back" {
		return
	}

	if selectedAction == "configure" {
		h.runConfigureWizard()
	} else {
		h.runManageMenu(obs, config)
	}
}

func (h *observabilityHandler) runConfigureWizard() {
	fmt.Println("\n📊 Configurando Stack de Observabilidad...")

	var deployType, externalURL string

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title("Topología de Despliegue").
				Options(
					huh.NewOption("1. Externa (URL pública, ej. Datadog / Grafana Cloud)", "external"),
					huh.NewOption("2. Clúster Dedicado (Nuevo VPS - Requiere Terraform)", "multi-node"),
					huh.NewOption("3. Todo-en-Uno (Stack PLG local con volumen físico)", "single-node"),
				).Value(&deployType),
		),
	).Run()

	if err != nil {
		return
	}

	switch deployType {
	case "external":
		if err := huh.NewForm(huh.NewGroup(huh.NewInput().Title("URL del Panel de Observabilidad (ej. https://mi-grafana.com)").Value(&externalURL))).Run(); err != nil {
			return
		}
	}

	bytes := make([]byte, 8)
	rand.Read(bytes)
	grafanaPassword := hex.EncodeToString(bytes)

	newObs := domain.SavedObservability{
		ID:              1,
		DeployType:      deployType,
		ExternalURL:     externalURL,
		GrafanaPassword: grafanaPassword,
	}

	if err := h.repo.SaveObservability(newObs); err != nil {
		fmt.Printf("❌ Error guardando configuración: %v\n", err)
	} else {
		fmt.Println("✅ Configuración de Observabilidad guardada exitosamente.")
	}
}

func (h *observabilityHandler) runManageMenu(obs *domain.SavedObservability, config domain.ServerConfig) {
	var action string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Administrando Observabilidad").
				Options(
					huh.NewOption("🚀 Desplegar / Actualizar ahora", "deploy"),
					huh.NewOption("🛑 Eliminar / Apagar Stack", "delete"),
					huh.NewOption("🔙 Volver", "back"),
				).
				Value(&action),
		),
	).Run()

	if err != nil || action == "back" {
		return
	}

	if action == "deploy" {
		if obs.DeployType == "external" {
			fmt.Printf("✅ Tienes observabilidad externa configurada en: %s\n", obs.ExternalURL)
			return
		}

		var exposePublic bool
		huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("⚠️ ¿Exponer Grafana/Portainer al internet público? (Se recomienda No, para usar VPN)").Value(&exposePublic))).Run()

		fmt.Println("\n⏳ Conectando al servidor principal...")
		sshExec := repositories.NewCryptoSSHExecutor()
		if err := sshExec.Connect(config); err != nil {
			fmt.Println("❌ Error SSH:", err)
			return
		}
		defer sshExec.Close()

		if obs.DeployType == "multi-node" {
			if obs.NodeIP == "" {
				workerUC := usecases.NewProvisionWorkerUseCase(sshExec)
				nodeName := "tarhiata-obs-worker"
				newIP, err := workerUC.Execute(config, nodeName, "obs")
				if newIP != "" {
					obs.NodeIP = newIP
					h.repo.SaveObservability(*obs) // (Evita Nodos Zombie)
				}
				if err != nil {
					fmt.Println("❌ Error provisionando nodo de logs:", err)
					return
				}
			}
		}

		fmt.Println("🚀 Desplegando Stack de Logs y Métricas...")
		fmt.Printf("🔒 Credenciales de Grafana generadas automáticamente: admin / %s\n", obs.GrafanaPassword)

		// Llamar al UseCase
		obsUC := usecases.NewDeployObservabilityUseCase(sshExec)
		if err := obsUC.ExecutePersistent(exposePublic, obs.DeployType, obs.GrafanaPassword); err != nil {
			fmt.Println("❌ Error en despliegue:", err)
		} else {
			fmt.Println("✅ ¡Stack de Observabilidad desplegado exitosamente!")
			fmt.Println("\n========================================================")
			fmt.Println("📌 PARA ACCEDER A TUS PANELES (Vía VPN o Local):")
			fmt.Println("   1. Abre tu archivo local (en tu PC): /etc/hosts")
			fmt.Println("   2. Agrega la siguiente línea al final:")
			fmt.Printf("      %s grafana.tarhiata.local portainer.tarhiata.local dozzle.tarhiata.local\n", config.Host)
			fmt.Println("   3. Abre en tu navegador:")
			fmt.Println("      - Grafana: http://grafana.tarhiata.local")
			fmt.Println("      - Portainer: http://portainer.tarhiata.local")
			fmt.Println("      - Dozzle (logs): http://dozzle.tarhiata.local")
			fmt.Println("========================================================")
			if obs.DeployType == "multi-node" {
				fmt.Printf("✅ Logs anclados al nodo Worker: %s\n", obs.NodeIP)
			}
		}
	} else if action == "delete" {
		var confirm bool

		msg := "⚠️ ¿Seguro que quieres apagar y eliminar el Stack? (Los datos en el servidor principal persistirán)"
		if obs.DeployType == "multi-node" {
			msg = "⚠️ PELIGRO: Esto DESTRUIRÁ el servidor dedicado y borrará TODOS los logs guardados de forma irreversible. ¿Continuar?"
		}

		huh.NewForm(huh.NewGroup(huh.NewConfirm().Title(msg).Value(&confirm))).Run()
		if confirm {
			if obs.DeployType != "external" {
				sshExec := repositories.NewCryptoSSHExecutor()
				if err := sshExec.Connect(config); err == nil {
					sshExec.RunCommand("docker stack rm tarhiata_obs")
					if obs.DeployType == "multi-node" {
						nodeName := "tarhiata-obs-worker"
						sshExec.RunCommand(fmt.Sprintf("docker node rm -f %s", nodeName))
					}
					sshExec.Close()
				}

				if obs.DeployType == "multi-node" {
					fmt.Println("⏳ Destruyendo servidor dedicado de logs en la nube (DigitalOcean)...")
					homeDir, _ := os.UserHomeDir()
					nodeName := "tarhiata-obs-worker"
					workspace := filepath.Join(homeDir, ".config", "tarhiata", "terraform", "worker_"+nodeName)
					prov := repositories.NewDigitalOceanProvisioner(workspace)

					if err := prov.DestroyNode(config.DOAPIToken, nodeName); err != nil {
						fmt.Printf("⚠️ Hubo un problema al intentar destruir el Droplet: %v (Por favor verifique en su panel de DigitalOcean)\n", err)
						fmt.Println("❌ Operación abortada para evitar pérdida de estado. Repare el nodo manualmente o reintente.")
						return
					} else {
						fmt.Println("🔥 Servidor dedicado destruido y eliminado de la facturación.")
						os.RemoveAll(workspace)
					}
				}
			}
			h.repo.DeleteObservability()
			fmt.Println("✅ Observabilidad eliminada del catálogo y stack apagado.")
		}
	}
}
