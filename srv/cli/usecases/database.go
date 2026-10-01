package usecases

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Dall06/tarhiata-ops/opt/cloud"
	"github.com/Dall06/tarhiata-ops/srv/cli/ports"
	sysdomain "github.com/Dall06/tarhiata-ops/srv/sys/domain"
	sysrepositories "github.com/Dall06/tarhiata-ops/srv/sys/repositories"
	sysusecases "github.com/Dall06/tarhiata-ops/srv/sys/usecases"
	"github.com/charmbracelet/huh"
)

type databaseHandler struct {
	repo ports.ConfigRepository
}

// NewDatabaseHandler inicializa el caso de uso de gestión de bases de datos para CLI.
func NewDatabaseHandler(repo ports.ConfigRepository) ports.DatabaseHandler {
	return &databaseHandler{repo: repo}
}

func (h *databaseHandler) Execute(config sysdomain.ServerConfig) {
	dbs, err := h.repo.GetDatabases(config.Name)
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
		h.runAddDatabaseWizard(config)
		return
	}

	dbName := strings.TrimPrefix(selectedAction, "manage_")
	h.runManageDatabaseMenu(dbName, config)
}

func (h *databaseHandler) runAddDatabaseWizard(config sysdomain.ServerConfig) {
	fmt.Println("\n🗄️  Agregando Base de Datos al catálogo...")

	var dbName, engine, deployType, externalURL, hostPath string
	var internalPort int

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Nombre (ej. mi-postgres)").Value(&dbName),
			huh.NewSelect[string]().Title("Motor de Base de Datos / Almacenamiento").
				Options(
					huh.NewOption("PostgreSQL", "postgres"),
					huh.NewOption("MongoDB", "mongo"),
					huh.NewOption("MySQL", "mysql"),
					huh.NewOption("Redis", "redis"),
					huh.NewOption("MinIO (Almacenamiento de Objetos S3)", "minio"),
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

	matched, errMatch := regexp.MatchString(`^[A-Za-z0-9_-]+$`, dbName)
	if errMatch != nil || !matched {
		fmt.Println("❌ Nombre de base de datos inválido. Solo se permiten letras, números y guiones.")
		return
	}

	switch deployType {
	case "external":
		if errForm := huh.NewForm(huh.NewGroup(huh.NewInput().Title("URL de Conexión (ej. postgres://user:pass@...)").Value(&externalURL))).Run(); errForm != nil {
			return
		}
	case "single-node":
		internalPort = 27017
		if engine == "postgres" {
			internalPort = 5432
		}
		defaultPath := fmt.Sprintf("/opt/tarhiata/data/%s", dbName)
		if errForm := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Ruta del Volumen en Host").Value(&defaultPath))).Run(); errForm != nil {
			return
		}
		hostPath = defaultPath
	case "multi-node":
		internalPort = 27017
		if engine == "postgres" {
			internalPort = 5432
		}
		defaultPath := fmt.Sprintf("/opt/tarhiata/data/%s", dbName)
		if errForm := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Ruta del Volumen en Host del Nuevo Servidor").Value(&defaultPath))).Run(); errForm != nil {
			return
		}
		hostPath = defaultPath
	}

	newDB := sysdomain.SavedDatabase{
		Name:           dbName,
		Engine:         engine,
		DeployType:     deployType,
		ExternalURL:    externalURL,
		InternalPort:   internalPort,
		VolumeHostPath: hostPath,
		ServerName:     config.Name,
	}

	if deployType != "external" {
		b := make([]byte, 16)
		if _, errEnt := rand.Read(b); errEnt != nil {
			slog.Warn("falló lectura de entropía para password de base de datos", "error", errEnt)
		}
		newDB.Password = hex.EncodeToString(b)
	}

	if err := h.repo.SaveDatabase(newDB); err != nil {
		fmt.Printf("❌ Error guardando BD: %v\n", err)
		return
	}
	fmt.Printf("✅ Base de datos %s guardada exitosamente.\n", dbName)
}

func (h *databaseHandler) runManageDatabaseMenu(dbName string, config sysdomain.ServerConfig) {
	db, err := h.repo.GetDatabase(dbName, config.Name)
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
		sshExec := sysrepositories.NewCryptoSSHExecutor()
		if err := sshExec.Connect(config); err != nil {
			fmt.Println("❌ Error SSH:", err)
			return
		}
		defer func() {
			if errClose := sshExec.Close(); errClose != nil {
				slog.Warn("cli: error cerrando ssh en deploy database", "error", errClose)
			}
		}()

		if db.DeployType == "multi-node" {
			if db.NodeIP == "" {
				workerUC := sysusecases.NewProvisionWorkerUseCase(sshExec)
				nodeName := fmt.Sprintf("tarhiata-db-%s", db.Name)
				newIP, errProv := workerUC.Execute(config, nodeName, "db_"+db.Name)
				if newIP != "" {
					db.NodeIP = newIP
					if errSave := h.repo.SaveDatabase(*db); errSave != nil {
						slog.Warn("cli: error actualizando IP de nodo en base de datos", "error", errSave)
					}
				}
				if errProv != nil {
					fmt.Println("❌ Error provisionando nodo:", errProv)
					return
				}
			}
		}

		dbUC := sysusecases.NewDeployDatabaseUseCase(sshExec)
		if err := dbUC.Execute(*db, config); err != nil {
			fmt.Println("❌ Error en despliegue:", err)
			return
		}
		if db.DeployType == "multi-node" {
			fmt.Printf("✅ Base de Datos anclada al nodo Worker: %s\n", db.NodeIP)
		}
		return
	}

	if action == "delete" {
		var confirm bool

		if db.DeployType == "multi-node" {
			var typedName string
			if errForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("⚠️ PELIGRO: Esto DESTRUIRÁ el servidor dedicado y borrará TODOS los datos irreversiblemente. Si necesitas un respaldo (dump), cancélalo ahora.\nEscribe el nombre de la BD para confirmar:").
						Value(&typedName),
				),
			).Run(); errForm != nil {
				slog.Error("error en formulario de confirmación multi-node", "error", errForm)
				return
			}
			if typedName != db.Name {
				fmt.Println("❌ Nombre incorrecto. Operación abortada.")
				return
			}
			confirm = true
		}
		if db.DeployType != "multi-node" {
			msg := "⚠️ ¿Seguro que quieres apagar y eliminar la BD? (Los datos en el servidor principal persistirán temporalmente)"
			if errForm := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title(msg).Value(&confirm))).Run(); errForm != nil {
				slog.Error("error en formulario de confirmación", "error", errForm)
			}
		}

		if confirm {
			var deleteVolume bool
			if db.DeployType == "single-node" {
				if errVol := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("🗑️ ¿Deseas eliminar permanentemente la carpeta de datos físicos (volumen)?").Value(&deleteVolume))).Run(); errVol != nil {
					slog.Error("error en formulario de eliminación de volumen", "error", errVol)
				}
			}

			if db.DeployType != "external" {
				sshExec := sysrepositories.NewCryptoSSHExecutor()
				if errConn := sshExec.Connect(config); errConn == nil {
					defer func() {
						if errClose := sshExec.Close(); errClose != nil {
							slog.Warn("cli: error cerrando ssh en delete database", "error", errClose)
						}
					}()
					serviceName := fmt.Sprintf("tarhiata-db-%s", db.Name)
					if res, errCmd := sshExec.RunCommand(fmt.Sprintf("docker service rm %s", serviceName)); errCmd != nil || (res != nil && res.ExitCode != 0) {
						slog.Warn("falló comando docker service rm para bd", "service", serviceName, "error", errCmd)
					}

					if db.DeployType == "single-node" && deleteVolume {
						if !strings.HasPrefix(db.VolumeHostPath, "/opt/") && !strings.HasPrefix(db.VolumeHostPath, "/var/lib/docker/") {
							fmt.Println("❌ Operación abortada: Ruta de volumen inválida o insegura para borrado automático.")
							return
						}
						fmt.Println("🧹 Limpiando volumen de datos huérfano...")
						if res, errCmd := sshExec.RunCommand(fmt.Sprintf("rm -rf %s", db.VolumeHostPath)); errCmd != nil || (res != nil && res.ExitCode != 0) {
							slog.Warn("falló comando rm volumen", "path", db.VolumeHostPath, "error", errCmd)
						}
					}
					if db.DeployType == "multi-node" {
						nodeName := fmt.Sprintf("tarhiata-db-%s", db.Name)
						if res, errCmd := sshExec.RunCommand(fmt.Sprintf("docker node rm -f %s", nodeName)); errCmd != nil || (res != nil && res.ExitCode != 0) {
							slog.Warn("falló comando docker node rm para nodo bd", "node", nodeName, "error", errCmd)
						}
					}
				}

				if db.DeployType == "multi-node" {
					fmt.Println("⏳ Destruyendo servidor dedicado en la nube (Vultr)...")
					homeDir, errHome := os.UserHomeDir()
					if errHome != nil {
						homeDir = os.TempDir()
					}
					nodeName := fmt.Sprintf("tarhiata-db-%s", db.Name)
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
			if errDel := h.repo.DeleteDatabase(db.Name, config.Name); errDel != nil {
				slog.Warn("fallo al eliminar base de datos del catálogo", "db", db.Name, "error", errDel)
			}
			fmt.Println("✅ Base de datos eliminada del catálogo y apagada.")
		}
	}
}
