package handlers

import (
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
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
		// h.runManageDatabaseMenu(dbName, config) -> Para futura implementación del despliegue en sí
		fmt.Printf("\n🚧 Gestión del despliegue para %s en desarrollo (Siguiente iteración).\n", dbName)
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
		fmt.Println("🚧 [Próximamente] Se activará el módulo de Terraform para provisionar la nueva VM y unirla al Swarm.")
		return
	}

	newDB := domain.SavedDatabase{
		Name:           dbName,
		Engine:         engine,
		DeployType:     deployType,
		ExternalURL:    externalURL,
		InternalPort:   internalPort,
		VolumeHostPath: hostPath,
	}

	if err := h.repo.SaveDatabase(newDB); err != nil {
		fmt.Printf("❌ Error guardando BD: %v\n", err)
	} else {
		fmt.Printf("✅ Base de datos %s guardada exitosamente.\n", dbName)
	}
}
