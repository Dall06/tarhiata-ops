package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

	if err != nil {
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
				if err != nil {
					fmt.Println("❌ Error provisionando nodo:", err)
					return
				}
				db.NodeIP = newIP
				h.repo.SaveDatabase(*db) // Actualizamos la BD con la nueva IP
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
		huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("⚠️ ¿Seguro que quieres apagar y eliminar la BD? (Los datos persistirán en el volumen físico)").Value(&confirm))).Run()
		if confirm {
			if db.DeployType != "external" {
				sshExec := repositories.NewCryptoSSHExecutor()
				if err := sshExec.Connect(config); err == nil {
					sshExec.RunCommand(fmt.Sprintf("docker service rm %s", db.Name))
					sshExec.Close()
				}
			}
			h.repo.DeleteDatabase(db.Name)
			fmt.Println("✅ Base de datos eliminada del catálogo y apagada.")
		}
	}
}
