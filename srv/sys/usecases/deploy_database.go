package usecases

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type DeployDatabaseUseCase struct {
	ssh ports.SSHExecutor
}

func NewDeployDatabaseUseCase(ssh ports.SSHExecutor) ports.DeployDatabaseUseCase {
	return &DeployDatabaseUseCase{ssh: ssh}
}

func (uc *DeployDatabaseUseCase) Execute(db domain.SavedDatabase, config domain.ServerConfig) error {
	if db.DeployType == "external" {
		return fmt.Errorf("las bases de datos externas no se despliegan, solo se guardan en el catálogo")
	}

	if db.VolumeHostPath == "" {
		db.VolumeHostPath = fmt.Sprintf("/opt/data/db-%s", db.Name)
	}
	if db.Password == "" {
		db.Password = fmt.Sprintf("admin_%s_pass", db.Name)
	}

	fmt.Printf("\n🚀 Desplegando Base de Datos: %s (%s)...\n", db.Name, db.Engine)

	// 0. Pre-flight: Verificar Docker disponible en el VPS
	resDocker, errDocker := uc.ssh.RunCommand("command -v docker")
	if errDocker != nil || resDocker == nil || resDocker.ExitCode != 0 || strings.TrimSpace(resDocker.Output) == "" {
		errMsg := "Docker no está instalado o no se encuentra en el PATH del VPS"
		if resDocker != nil && strings.TrimSpace(resDocker.Output) != "" {
			errMsg = resDocker.Output
		}
		if errDocker != nil {
			errMsg = fmt.Sprintf("%s (%v)", errMsg, errDocker)
		}
		return fmt.Errorf("pre-flight check fallido: %s", errMsg)
	}

	// 1. Pre-flight: Verificar e inicializar Docker Swarm si está inactivo
	resSwarm, errSwarm := uc.ssh.RunCommand("docker info --format '{{.Swarm.LocalNodeState}}'")
	swarmState := ""
	if resSwarm != nil {
		swarmState = strings.TrimSpace(resSwarm.Output)
	}
	if errSwarm != nil || swarmState != "active" {
		slog.Info("Docker Swarm inactivo en el VPS. Inicializando Swarm...", "state", swarmState)
		resInit, errInit := uc.ssh.RunCommand("docker swarm init")
		if errInit != nil || resInit == nil || resInit.ExitCode != 0 {
			initOut := ""
			if resInit != nil {
				initOut = resInit.Output
			}
			return fmt.Errorf("falló auto-inicialización de Docker Swarm: %s (err: %v)", initOut, errInit)
		}
	}

	// 2. Pre-flight: Asegurar que las redes overlay existan de forma idempotente
	resNet, errNet := uc.ssh.RunCommand("docker network create --driver overlay --attachable tarhiata_internal 2>&1 || true")
	if errNet != nil {
		slog.Warn("aviso al asegurar red overlay tarhiata_internal", "error", errNet)
	}
	if resNet != nil && strings.Contains(resNet.Output, "error") && !strings.Contains(resNet.Output, "already exists") {
		slog.Warn("aviso al crear red tarhiata_internal", "output", resNet.Output)
	}

	constraint := `"node.role == manager"`
	if db.TargetNode != "" {
		if db.TargetNode == "worker" {
			resNode, errNode := uc.ssh.RunCommand("docker node ls --filter role=worker --format '{{.ID}}'")
			if errNode == nil && resNode != nil && strings.TrimSpace(resNode.Output) != "" {
				constraint = `"node.role == worker"`
			} else {
				token := config.VultrAPIToken
				if token == "" { token = config.DOAPIToken }
				if token != "" {
					nodeName := fmt.Sprintf("worker-%s", db.Name)
					fmt.Printf("🏗️  No hay nodos Worker activos. Aprovisionando VM '%s' en la nube vía Terraform...\n", nodeName)
					workerUC := NewProvisionWorkerUseCase(uc.ssh)
					_, errProv := workerUC.ExecuteWithRegion(config, nodeName, "worker", "")
					if errProv == nil {
						constraint = `"node.role == worker"`
					} else {
						fmt.Printf("⚠️ No se pudo aprovisionar VM Worker automática (%v). Usando nodo Manager como respaldo...\n", errProv)
						constraint = `"node.role == manager"`
					}
				} else {
					fmt.Println("⚠️  No hay nodos Worker activos y no se ha configurado un Token de API de Nube. Usando nodo Manager...")
					constraint = `"node.role == manager"`
				}
			}
		} else if db.TargetNode == "db" {
			resNode, errNode := uc.ssh.RunCommand("docker node ls --filter label=type=db --format '{{.ID}}'")
			if errNode == nil && resNode != nil && strings.TrimSpace(resNode.Output) != "" {
				constraint = `"node.labels.type == db"`
			} else {
				token := config.VultrAPIToken
				if token == "" { token = config.DOAPIToken }
				if token != "" {
					nodeName := fmt.Sprintf("worker-db-%s", db.Name)
					fmt.Printf("🏗️  No hay nodos Worker DB activos. Aprovisionando VM '%s' en la nube vía Terraform...\n", nodeName)
					workerUC := NewProvisionWorkerUseCase(uc.ssh)
					_, errProv := workerUC.ExecuteWithRegion(config, nodeName, "db", "")
					if errProv == nil {
						constraint = `"node.labels.type == db"`
					} else {
						fmt.Printf("⚠️ No se pudo aprovisionar VM Worker DB automática (%v). Usando nodo Manager como respaldo...\n", errProv)
						constraint = `"node.role == manager"`
					}
				} else {
					fmt.Println("⚠️  No hay nodos Worker DB activos y no se ha configurado un Token de API de Nube. Usando nodo Manager...")
					constraint = `"node.role == manager"`
				}
			}
		} else if db.TargetNode == "manager" {
			constraint = `"node.role == manager"`
		} else {
			// Validar si el targetNode coincide con un hostname o ID real en Swarm
			isRealNode := false
			resCheckNode, errCheckNode := uc.ssh.RunCommand("docker node ls --format '{{.Hostname}}|{{.ID}}'")
			if errCheckNode == nil && resCheckNode != nil && resCheckNode.ExitCode == 0 {
				for _, line := range strings.Split(resCheckNode.Output, "\n") {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					parts := strings.Split(line, "|")
					for _, part := range parts {
						if strings.EqualFold(strings.TrimSpace(part), db.TargetNode) {
							isRealNode = true
							break
						}
					}
					if isRealNode {
						break
					}
				}
			}
			if isRealNode {
				constraint = fmt.Sprintf(`"node.hostname == %s"`, db.TargetNode)
			} else {
				slog.Info("TargetNode no corresponde a un hostname de Swarm. Usando node.role == manager", "targetNode", db.TargetNode)
				constraint = `"node.role == manager"`
			}
		}
	} else if db.DeployType == "multi-node" {
		constraint = fmt.Sprintf(`"node.labels.type == db_%s"`, db.Name)
	}

	fmt.Printf("📁 Preparando almacenamiento persistente en el nodo (%s)...\n", db.DeployType)
	var uid string
	engineLower := strings.ToLower(db.Engine)
	if engineLower == "postgres" {
		uid = "70:70"
	} else {
		uid = "999:999"
	}

	// Preparar directorio host directamente por SSH sin contenedores efímeros
	checkCmd := fmt.Sprintf("test -d %s && ls -A %s", db.VolumeHostPath, db.VolumeHostPath)
	resCheck, errCheck := uc.ssh.RunCommand(checkCmd)
	hasExistingData := errCheck == nil && resCheck != nil && strings.TrimSpace(resCheck.Output) != ""

	if hasExistingData {
		if db.CleanExistingData {
			fmt.Printf("🧹 [Recovery Mode] Limpiando datos antiguos en %s antes de desplegar...\n", db.VolumeHostPath)
			if _, errRm := uc.ssh.RunCommand(fmt.Sprintf("rm -rf %s/*", db.VolumeHostPath)); errRm != nil {
				slog.Warn("aviso al limpiar datos antiguos", "path", db.VolumeHostPath, "error", errRm)
			}
			if _, errMk := uc.ssh.RunCommand(fmt.Sprintf("mkdir -p %s && chown -R %s %s", db.VolumeHostPath, uid, db.VolumeHostPath)); errMk != nil {
				slog.Warn("aviso al preparar permisos de directorio", "path", db.VolumeHostPath, "error", errMk)
			}
		} else {
			fmt.Printf("📦 [Recovery Mode] ¡Se detectaron datos previos en %s! Reutilizando volumen host para recuperación de base de datos...\n", db.VolumeHostPath)
			if _, errMk := uc.ssh.RunCommand(fmt.Sprintf("mkdir -p %s && chown -R %s %s", db.VolumeHostPath, uid, db.VolumeHostPath)); errMk != nil {
				slog.Warn("aviso al preparar permisos de directorio", "path", db.VolumeHostPath, "error", errMk)
			}
		}
	} else {
		if _, errMk := uc.ssh.RunCommand(fmt.Sprintf("mkdir -p %s && chown -R %s %s", db.VolumeHostPath, uid, db.VolumeHostPath)); errMk != nil {
			slog.Warn("aviso al crear directorio de datos", "path", db.VolumeHostPath, "error", errMk)
		}
	}

	serviceName := fmt.Sprintf("tarhiata-db-%s", db.Name)

	// 2. Apagar la BD si ya existía para actualizarla
	if _, errRmSvc := uc.ssh.RunCommand(fmt.Sprintf("docker service rm %s", serviceName)); errRmSvc != nil {
		slog.Debug("aviso al remover servicio previo (si existía)", "service", serviceName, "error", errRmSvc)
	}

	// 3. Construir el comando de docker service create
	safePassword := strings.ReplaceAll(db.Password, "'", `'"'"'`)

	var createCmd string
	switch engineLower {
	case "postgres":
		createCmd = fmt.Sprintf(
			`docker service create \
			--name %s \
			--detach=true \
			--network tarhiata_internal \
			--mount type=bind,source=%s,destination=/var/lib/postgresql/data \
			-e POSTGRES_USER=admin \
			-e POSTGRES_PASSWORD='%s' \
			-e POSTGRES_DB=db \
			--constraint %s \
			postgres:15-alpine`,
			serviceName, db.VolumeHostPath, safePassword, constraint,
		)
	case "mongo", "mongodb":
		createCmd = fmt.Sprintf(
			`docker service create \
			--name %s \
			--detach=true \
			--network tarhiata_internal \
			--mount type=bind,source=%s,destination=/data/db \
			-e MONGO_INITDB_ROOT_USERNAME=admin \
			-e MONGO_INITDB_ROOT_PASSWORD='%s' \
			--constraint %s \
			mongo:6`,
			serviceName, db.VolumeHostPath, safePassword, constraint,
		)
	case "mysql", "mariadb":
		createCmd = fmt.Sprintf(
			`docker service create \
			--name %s \
			--detach=true \
			--network tarhiata_internal \
			--mount type=bind,source=%s,destination=/var/lib/mysql \
			-e MYSQL_ROOT_PASSWORD='%s' \
			-e MYSQL_DATABASE=db \
			--constraint %s \
			mysql:8`,
			serviceName, db.VolumeHostPath, safePassword, constraint,
		)
	case "redis":
		createCmd = fmt.Sprintf(
			`docker service create \
			--name %s \
			--detach=true \
			--network tarhiata_internal \
			--mount type=bind,source=%s,destination=/data \
			--constraint %s \
			redis:7-alpine redis-server --requirepass '%s'`,
			serviceName, db.VolumeHostPath, constraint, safePassword,
		)
	case "minio", "s3":
		createCmd = fmt.Sprintf(
			`docker service create \
			--name %s \
			--detach=true \
			--network tarhiata_internal \
			--mount type=bind,source=%s,destination=/data \
			-e MINIO_ROOT_USER=admin \
			-e MINIO_ROOT_PASSWORD='%s' \
			--constraint %s \
			minio/minio:latest server /data --console-address ":9001"`,
			serviceName, db.VolumeHostPath, safePassword, constraint,
		)
	default:
		return fmt.Errorf("motor de base de datos no soportado: %s", db.Engine)
	}

	// 4. Ejecutar el despliegue
	res, err := uc.ssh.RunCommand(createCmd)
	if err != nil || res == nil || res.ExitCode != 0 {
		errMsg := ""
		if res != nil {
			errMsg = res.Output
		}
		if errMsg == "" && err != nil {
			errMsg = err.Error()
		}
		return fmt.Errorf("error creando servicio de BD/Storage: %s", errMsg)
	}

	fmt.Printf("✅ ¡Servidor de Almacenamiento/BD '%s' (%s) desplegado correctamente en %s!\n", db.Name, db.Engine, db.VolumeHostPath)

	safeUri := fmt.Sprintf("%s://admin:********@%s:%d/db", db.Engine, serviceName, db.InternalPort)
	if engineLower == "mongo" || engineLower == "mongodb" {
		safeUri = fmt.Sprintf("mongodb://admin:********@%s:27017/?authSource=admin", serviceName)
	} else if engineLower == "redis" {
		safeUri = fmt.Sprintf("redis://:********@%s:6379", serviceName)
	} else if engineLower == "minio" || engineLower == "s3" {
		safeUri = fmt.Sprintf("s3://admin:********@%s:9000 (Console :9001)", serviceName)
	}

	fmt.Printf("🔌 URI Interna (Oculta): %s\n", safeUri)

	syncUC := NewSyncClusterStateUseCase(nil, uc.ssh)
	if errSync := syncUC.ExportStateToRemote(); errSync != nil {
		slog.Warn("Fallo al exportar estado de sincronización al VPS", "error", errSync)
	}

	return nil
}
