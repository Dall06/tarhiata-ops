package handlers

import (
	"fmt"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/repositories"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/usecases"
	"github.com/charmbracelet/huh"
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
	case "multi-node":
		fmt.Println("🚧 [Próximamente] Se activará el módulo de Terraform para provisionar la nueva VM y anclar el Stack PLG.")
		return
	}

	newObs := domain.SavedObservability{
		ID:          1,
		DeployType:  deployType,
		ExternalURL: externalURL,
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

		obsUC := usecases.NewDeployObservabilityUseCase(sshExec)
		if err := obsUC.ExecutePersistent(exposePublic); err != nil {
			fmt.Println("❌ Error en despliegue:", err)
		} else {
			fmt.Println("✅ ¡Stack de Observabilidad desplegado exitosamente!")
			if obs.DeployType == "single-node" {
				fmt.Printf("👉 Grafana: http://%s:3001 (User: admin / Pass: admin)\n", config.Host)
				fmt.Printf("👉 Portainer: http://%s:9000\n", config.Host)
			}
		}
	} else if action == "delete" {
		var confirm bool
		huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("⚠️ ¿Seguro que quieres apagar y eliminar el Stack? (Los datos persistirán en el volumen físico)").Value(&confirm))).Run()
		if confirm {
			if obs.DeployType != "external" {
				sshExec := repositories.NewCryptoSSHExecutor()
				if err := sshExec.Connect(config); err == nil {
					sshExec.RunCommand("docker stack rm tarhiata_obs")
					sshExec.Close()
				}
			}
			h.repo.DeleteObservability()
			fmt.Println("✅ Observabilidad eliminada del catálogo y stack apagado.")
		}
	}
}
