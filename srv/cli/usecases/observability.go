package usecases

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Dall06/tarhiata-ops/opt/cloud"
	"github.com/Dall06/tarhiata-ops/srv/cli/ports"
	sysdomain "github.com/Dall06/tarhiata-ops/srv/sys/domain"
	sysrepositories "github.com/Dall06/tarhiata-ops/srv/sys/repositories"
	sysusecases "github.com/Dall06/tarhiata-ops/srv/sys/usecases"
	"github.com/charmbracelet/huh"
)

type observabilityHandler struct {
	repo ports.ConfigRepository
}

// NewObservabilityHandler crea el caso de uso de observabilidad interactiva para CLI.
func NewObservabilityHandler(repo ports.ConfigRepository) ports.ObservabilityHandler {
	return &observabilityHandler{repo: repo}
}

func (h *observabilityHandler) Execute(config sysdomain.ServerConfig) {
	obs, err := h.repo.GetObservability()
	if err != nil {
		fmt.Printf("❌ Error leyendo configuración de observabilidad: %v\n", err)
		return
	}

	var selectedAction string
	var options []huh.Option[string]

	if obs == nil {
		options = append(options, huh.NewOption("➕ Configurar Stack de Observabilidad", "configure"))
	}
	if obs != nil {
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
		return
	}
	h.runManageMenu(obs, config)
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
		if errForm := huh.NewForm(huh.NewGroup(huh.NewInput().Title("URL del Panel de Observabilidad (ej. https://mi-grafana.com)").Value(&externalURL))).Run(); errForm != nil {
			return
		}
	}

	bytes := make([]byte, 8)
	if _, errRand := rand.Read(bytes); errRand != nil {
		slog.Warn("falló lectura de entropía para password de grafana", "error", errRand)
	}
	grafanaPassword := hex.EncodeToString(bytes)

	newObs := sysdomain.SavedObservability{
		ID:              1,
		DeployType:      deployType,
		ExternalURL:     externalURL,
		GrafanaPassword: grafanaPassword,
	}

	if err := h.repo.SaveObservability(newObs); err != nil {
		fmt.Printf("❌ Error guardando configuración: %v\n", err)
		return
	}
	fmt.Println("✅ Configuración de Observabilidad guardada exitosamente.")
}

func (h *observabilityHandler) runManageMenu(obs *sysdomain.SavedObservability, config sysdomain.ServerConfig) {
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
		if errPrompt := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("⚠️ ¿Exponer Grafana/Portainer al internet público? (Se recomienda No, para usar VPN)").Value(&exposePublic))).Run(); errPrompt != nil {
			slog.Warn("cli: cancelado prompt expose public", "error", errPrompt)
			return
		}

		fmt.Println("\n⏳ Conectando al servidor principal...")
		sshExec := sysrepositories.NewCryptoSSHExecutor()
		if err := sshExec.Connect(config); err != nil {
			fmt.Println("❌ Error SSH:", err)
			return
		}
		defer func() {
			if errClose := sshExec.Close(); errClose != nil {
				slog.Warn("cli: error cerrando ssh en observability deploy", "error", errClose)
			}
		}()

		if obs.DeployType == "multi-node" {
			if obs.NodeIP == "" {
				workerUC := sysusecases.NewProvisionWorkerUseCase(sshExec)
				nodeName := "tarhiata-obs-worker"
				newIP, errProv := workerUC.Execute(config, nodeName, "obs")
				if newIP != "" {
					obs.NodeIP = newIP
					if errSave := h.repo.SaveObservability(*obs); errSave != nil {
						slog.Warn("cli: error guardando IP de nodo en observabilidad", "error", errSave)
					}
				}
				if errProv != nil {
					fmt.Println("❌ Error provisionando nodo de logs:", errProv)
					return
				}
			}
		}

		fmt.Println("🚀 Desplegando Stack de Logs y Métricas...")
		fmt.Printf("🔒 Credenciales de Grafana generadas automáticamente: admin / %s\n", obs.GrafanaPassword)

		obsUC := sysusecases.NewDeployObservabilityUseCase(sshExec)
		if err := obsUC.ExecutePersistent(exposePublic, obs.DeployType, obs.GrafanaPassword); err != nil {
			fmt.Println("❌ Error en despliegue:", err)
			return
		}
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
		return
	}

	if action == "delete" {
		var confirm bool

		msg := "⚠️ ¿Seguro que quieres apagar y eliminar el Stack? (Los datos en el servidor principal persistirán)"
		if obs.DeployType == "multi-node" {
			msg = "⚠️ PELIGRO: Esto DESTRUIRÁ el servidor dedicado y borrará TODOS los logs guardados de forma irreversible. ¿Continuar?"
		}

		if errForm := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title(msg).Value(&confirm))).Run(); errForm != nil {
			slog.Warn("cli: cancelado prompt confirm delete observability", "error", errForm)
			return
		}

		if confirm {
			if obs.DeployType != "external" {
				sshExec := sysrepositories.NewCryptoSSHExecutor()
				if errConn := sshExec.Connect(config); errConn == nil {
					defer func() {
						if errClose := sshExec.Close(); errClose != nil {
							slog.Warn("cli: error cerrando ssh en observability delete", "error", errClose)
						}
					}()
					if res, errCmd := sshExec.RunCommand("docker stack rm tarhiata_obs"); errCmd != nil || (res != nil && res.ExitCode != 0) {
						slog.Warn("falló comando docker stack rm", "error", errCmd)
					}
					if obs.DeployType == "multi-node" {
						nodeName := "tarhiata-obs-worker"
						if res, errCmd := sshExec.RunCommand(fmt.Sprintf("docker node rm -f %s", nodeName)); errCmd != nil || (res != nil && res.ExitCode != 0) {
							slog.Warn("falló comando docker node rm", "node", nodeName, "error", errCmd)
						}
					}
				}

				if obs.DeployType == "multi-node" {
					fmt.Println("⏳ Destruyendo servidor dedicado de logs en la nube (Vultr)...")
					homeDir, errHome := os.UserHomeDir()
					if errHome != nil {
						homeDir = os.TempDir()
					}
					nodeName := "tarhiata-obs-worker"
					workspace := filepath.Join(homeDir, ".config", "tarhiata", "terraform", "worker_"+nodeName)
					prov := cloud.NewProvisioner("vultr", workspace)

					if errDestroy := prov.DestroyNode(config.VultrAPIToken, nodeName); errDestroy != nil {
						fmt.Printf("⚠️ Hubo un problema al intentar destruir la instancia: %v (Por favor verifique en su panel de Vultr)\n", errDestroy)
						fmt.Println("❌ Operación abortada para evitar pérdida de estado. Repare el nodo manualmente o reintente.")
						return
					}
					fmt.Println("🔥 Servidor dedicado destruido y eliminado de la facturación.")
					if errRm := os.RemoveAll(workspace); errRm != nil {
						slog.Warn("fallo al limpiar workspace de terraform", "workspace", workspace, "error", errRm)
					}
				}
			}
			if errDel := h.repo.DeleteObservability(); errDel != nil {
				fmt.Printf("⚠️ Error al eliminar observabilidad de SQLite: %v\n", errDel)
			}
			fmt.Println("✅ Observabilidad eliminada del catálogo y stack apagado.")
		}
	}
}
