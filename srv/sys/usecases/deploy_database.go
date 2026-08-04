package usecases

import (
	"fmt"
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
			constraint = fmt.Sprintf(`"node.hostname == %s"`, db.TargetNode)
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
	resCheck, _ := uc.ssh.RunCommand(checkCmd)
	hasExistingData := resCheck != nil && strings.TrimSpace(resCheck.Output) != ""

	if hasExistingData {
		if db.CleanExistingData {
			fmt.Printf("🧹 [Recovery Mode] Limpiando datos antiguos en %s antes de desplegar...\n", db.VolumeHostPath)
			uc.ssh.RunCommand(fmt.Sprintf("rm -rf %s/*", db.VolumeHostPath))
			uc.ssh.RunCommand(fmt.Sprintf("mkdir -p %s && chown -R %s %s", db.VolumeHostPath, uid, db.VolumeHostPath))
		} else {
			fmt.Printf("📦 [Recovery Mode] ¡Se detectaron datos previos en %s! Reutilizando volumen host para recuperación de base de datos...\n", db.VolumeHostPath)
			uc.ssh.RunCommand(fmt.Sprintf("mkdir -p %s && chown -R %s %s", db.VolumeHostPath, uid, db.VolumeHostPath))
		}
	} else {
		uc.ssh.RunCommand(fmt.Sprintf("mkdir -p %s && chown -R %s %s", db.VolumeHostPath, uid, db.VolumeHostPath))
	}

	serviceName := fmt.Sprintf("tarhiata-db-%s", db.Name)

	// 2. Apagar la BD si ya existía para actualizarla
	uc.ssh.RunCommand(fmt.Sprintf("docker service rm %s", serviceName))

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
	if err != nil || res.ExitCode != 0 {
		return fmt.Errorf("error creando servicio de BD/Storage: %s", res.Output)
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
	_ = syncUC.ExportStateToRemote()

	return nil
}
