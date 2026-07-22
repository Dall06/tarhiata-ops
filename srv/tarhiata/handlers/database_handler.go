package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/repositories"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/usecases"
	"github.com/charmbracelet/huh"
)

type databaseHandler struct {
	repo ports.ConfigRepository
}

func NewDatabaseHandler(repo ports.ConfigRepository) ports.DatabaseHandler {
	return &databaseHandler{repo: repo}
}

func (h *databaseHandler) Execute(config domain.ServerConfig) {
	dbs, err := h.repo.GetDatabases()
	if err != nil {
		fmt.Printf("❌ Error leyendo bases de datos: %v\n", err)
		return
	}

	var selectedAction string
	options := []huh.Option[string]{
		huh.NewOption("➕ Agregar Base de Datos", "add_new"),
	}

	for _, dbInfo := range dbs {
		options = append(options, huh.NewOption(fmt.Sprintf("🗄️  %s (%s - %s)", dbInfo.Name, dbInfo.Engine, dbInfo.DeployType), "manage_"+dbInfo.Name))
	}
	options = append(options, huh.NewOption("🔙 Volver al Menú Principal", "back"))

	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Gestión de Bases de Datos").
				Options(options...).
				Value(&selectedAction),
		),
	).Run()

	if err != nil || selectedAction == "back" {
		return
	}

	if selectedAction == "add_new" {
		h.runAddDatabaseWizard()
	} else {
		dbName := strings.TrimPrefix(selectedAction, "manage_")
		h.runManageDatabaseMenu(dbName, config)
	}
}

func (h *databaseHandler) runAddDatabaseWizard() {
	fmt.Println("\n🗄️  Agregando Base de Datos al catálogo...")

	var dbName, engine, deployType, externalURL, hostPath string
	var internalPort int

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Nombre (ej. mi-postgres)").Value(&dbName),
			huh.NewSelect[string]().Title("Motor de Base de Datos").
				Options(
					huh.NewOption("PostgreSQL", "postgres"),
					huh.NewOption("MongoDB", "mongo"),
				).Value(&engine),
			huh.NewSelect[string]().Title("Topología de Despliegue").
				Options(
					huh.NewOption("1. Externa (URL pública, ej. Supabase)", "external"),
					huh.NewOption("2. Clúster Dedicado (Nuevo VPS - Requiere Terraform)", "multi-node"),
					huh.NewOption("3. Todo-en-Uno (Contenedor local con volumen)", "single-node"),
				).Value(&deployType),
		),
	).Run()

	if err != nil || dbName == "" {
		return
	}

	if matched, _ := regexp.MatchString(`^[A-Za-z0-9_-]+$`, dbName); !matched {
		fmt.Println("❌ Nombre de base de datos inválido. Solo se permiten letras, números y guiones.")
		return
	}

	switch deployType {
	case "external":
		if err := huh.NewForm(huh.NewGroup(huh.NewInput().Title("URL de Conexión (ej. postgres://user:pass@...)").Value(&externalURL))).Run(); err != nil {
			return
		}
	case "single-node":
		if engine == "postgres" {
			internalPort = 5432
		} else {
			internalPort = 27017
		}
		defaultPath := fmt.Sprintf("/opt/tarhiata/data/%s", dbName)
		if err := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Ruta del Volumen en Host").Value(&defaultPath))).Run(); err != nil {
			return
		}
		hostPath = defaultPath
	case "multi-node":
		if engine == "postgres" {
			internalPort = 5432
		} else {
			internalPort = 27017
		}
		defaultPath := fmt.Sprintf("/opt/tarhiata/data/%s", dbName)
		if err := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Ruta del Volumen en Host del Nuevo Servidor").Value(&defaultPath))).Run(); err != nil {
			return
		}
		hostPath = defaultPath
	}

	newDB := domain.SavedDatabase{
		Name:           dbName,
		Engine:         engine,
		DeployType:     deployType,
		ExternalURL:    externalURL,
		InternalPort:   internalPort,
		VolumeHostPath: hostPath,
	}

	if deployType != "external" {
		b := make([]byte, 16)
		rand.Read(b)
		newDB.Password = hex.EncodeToString(b)
	}

	if err := h.repo.SaveDatabase(newDB); err != nil {
		fmt.Printf("❌ Error guardando BD: %v\n", err)
	} else {
		fmt.Printf("✅ Base de datos %s guardada exitosamente.\n", dbName)
	}
}

func (h *databaseHandler) runManageDatabaseMenu(dbName string, config domain.ServerConfig) {
	db, err := h.repo.GetDatabase(dbName)
	if err != nil || db == nil {
		fmt.Println("❌ No se encontró la base de datos.")
		return
	}

	var action string
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(fmt.Sprintf("Administrando BD: %s (%s)", db.Name, db.Engine)).
				Options(
					huh.NewOption("🚀 Desplegar / Actualizar ahora", "deploy"),
					huh.NewOption("🛑 Eliminar / Apagar BD", "delete"),
					huh.NewOption("🔙 Volver", "back"),
				).
				Value(&action),
		),
	).Run()

	if err != nil || action == "back" {
		return
	}

	if action == "deploy" {
		if db.DeployType == "external" {
			fmt.Println("⚠️ Las bases de datos externas no se pueden desplegar, ya existen en otro lugar.")
			return
		}

		fmt.Println("\n⏳ Conectando al servidor principal...")
		sshExec := repositories.NewCryptoSSHExecutor()
		if err := sshExec.Connect(config); err != nil {
			fmt.Println("❌ Error SSH:", err)
			return
		}
		defer sshExec.Close()

		if db.DeployType == "multi-node" {
			if db.NodeIP == "" {
				// Necesitamos provisionar el nodo
				workerUC := usecases.NewProvisionWorkerUseCase(sshExec)
				nodeName := fmt.Sprintf("tarhiata-db-%s", db.Name)
				newIP, err := workerUC.Execute(config, nodeName, "db_"+db.Name)
				if newIP != "" {
					db.NodeIP = newIP
					h.repo.SaveDatabase(*db) // Actualizamos la BD con la nueva IP (Evita Nodos Zombie)
				}
				if err != nil {
					fmt.Println("❌ Error provisionando nodo:", err)
					return
				}
			}
		}

		dbUC := usecases.NewDeployDatabaseUseCase(sshExec)
		if err := dbUC.Execute(*db, config); err != nil {
			fmt.Println("❌ Error en despliegue:", err)
		} else {
			if db.DeployType == "multi-node" {
				fmt.Printf("✅ Base de Datos anclada al nodo Worker: %s\n", db.NodeIP)
			}
		}
	} else if action == "delete" {
		var confirm bool

		if db.DeployType == "multi-node" {
			var typedName string
			huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("⚠️ PELIGRO: Esto DESTRUIRÁ el servidor dedicado y borrará TODOS los datos irreversiblemente. Si necesitas un respaldo (dump), cancélalo ahora.\nEscribe el nombre de la BD para confirmar:").
						Value(&typedName),
				),
			).Run()
			if typedName != db.Name {
				fmt.Println("❌ Nombre incorrecto. Operación abortada.")
				return
			}
			confirm = true
		} else {
			msg := "⚠️ ¿Seguro que quieres apagar y eliminar la BD? (Los datos en el servidor principal persistirán temporalmente)"
			huh.NewForm(huh.NewGroup(huh.NewConfirm().Title(msg).Value(&confirm))).Run()
		}

		if confirm {
			var deleteVolume bool
			if db.DeployType == "single-node" {
				huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("🗑️ ¿Deseas eliminar permanentemente la carpeta de datos físicos (volumen)?").Value(&deleteVolume))).Run()
			}

			if db.DeployType != "external" {
				sshExec := repositories.NewCryptoSSHExecutor()
				if err := sshExec.Connect(config); err == nil {
					serviceName := fmt.Sprintf("tarhiata-db-%s", db.Name)
					sshExec.RunCommand(fmt.Sprintf("docker service rm %s", serviceName))

					if db.DeployType == "single-node" && deleteVolume {
						// SECURITY CHECK: Solo permitir borrar dentro de un path seguro para evitar Inyección de Rutas (rm -rf /)
						if !strings.HasPrefix(db.VolumeHostPath, "/opt/") && !strings.HasPrefix(db.VolumeHostPath, "/var/lib/docker/") {
							fmt.Println("❌ Operación abortada: Ruta de volumen inválida o insegura para borrado automático.")
							return
						}
						fmt.Println("🧹 Limpiando volumen de datos huérfano...")
						sshExec.RunCommand(fmt.Sprintf("rm -rf %s", db.VolumeHostPath))
					}
					if db.DeployType == "multi-node" {
						nodeName := fmt.Sprintf("tarhiata-db-%s", db.Name)
						sshExec.RunCommand(fmt.Sprintf("docker node rm -f %s", nodeName))
					}
					sshExec.Close()
				}

				if db.DeployType == "multi-node" {
					fmt.Println("⏳ Destruyendo servidor dedicado en la nube (DigitalOcean)...")
					homeDir, _ := os.UserHomeDir()
					nodeName := fmt.Sprintf("tarhiata-db-%s", db.Name)
					workspace := filepath.Join(homeDir, ".config", "tarhiata", "terraform", "worker_"+nodeName)
					prov := repositories.NewDigitalOceanProvisioner(workspace)

					if err := prov.DestroyNode(config.DOAPIToken, nodeName); err != nil {
						fmt.Printf("⚠️ Hubo un problema al intentar destruir el Droplet: %v (Por favor verifique en su panel de DigitalOcean)\n", err)
						fmt.Println("❌ Operación abortada para evitar pérdida de estado. Repare el nodo manualmente o reintente.")
						return
					} else {
						fmt.Println("🔥 Servidor dedicado destruido y eliminado de la facturación.")
						os.RemoveAll(workspace) // Limpiar basura de Terraform (GAP 3)
					}
				}
			}
			h.repo.DeleteDatabase(db.Name)
			fmt.Println("✅ Base de datos eliminada del catálogo y apagada.")
		}
	}
}
