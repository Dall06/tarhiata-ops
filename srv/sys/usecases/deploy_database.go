package usecases

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/Dall06/tarhiata-ops/pkg/dockerutil"
	"github.com/Dall06/tarhiata-ops/pkg/validator"
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
	if !validator.IsSafePath(db.VolumeHostPath) {
		return fmt.Errorf("ruta de volumen inválida: %q", db.VolumeHostPath)
	}
	if db.Password == "" {
		db.Password = fmt.Sprintf("admin_%s_pass", db.Name)
	}

	fmt.Printf("\n🚀 Desplegando Base de Datos: %s (%s)...\n", db.Name, db.Engine)

	if err := uc.ensurePreflight(); err != nil {
		return fmt.Errorf("pre-flight check fallido: %w", err)
	}

	constraint := uc.resolveConstraint(db, config)
	uc.prepareStorage(db.VolumeHostPath, db.CleanExistingData, db.Engine, db.DeployType)

	serviceName := fmt.Sprintf("tarhiata-db-%s", db.Name)

	// Apagar la BD si ya existía para actualizarla
	if _, errRmSvc := uc.ssh.RunCommand(fmt.Sprintf("docker service rm %s", serviceName)); errRmSvc != nil {
		slog.Debug("aviso al remover servicio previo (si existía)", "service", serviceName, "error", errRmSvc)
	}

	createCmd, errBuild := uc.buildCreateCommand(db, serviceName, constraint)
	if errBuild != nil {
		return errBuild
	}

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
	fmt.Printf("🔌 URI Interna (Oculta): %s\n", dockerutil.BuildSafeURI(db.Engine, serviceName, db.InternalPort))

	syncUC := NewSyncClusterStateUseCase(nil, uc.ssh)
	if errSync := syncUC.ExportStateToRemote(config.Name); errSync != nil {
		slog.Warn("Fallo al exportar estado de sincronización al VPS", "error", errSync)
	}

	return nil
}

func (uc *DeployDatabaseUseCase) ensurePreflight() error {
	resDocker, errDocker := uc.ssh.RunCommand("command -v docker")
	if errDocker != nil || resDocker == nil || resDocker.ExitCode != 0 || strings.TrimSpace(resDocker.Output) == "" {
		errMsg := "Docker no está instalado o no se encuentra en el PATH del VPS"
		if resDocker != nil && strings.TrimSpace(resDocker.Output) != "" {
			errMsg = resDocker.Output
		}
		if errDocker != nil {
			errMsg = fmt.Sprintf("%s (%v)", errMsg, errDocker)
		}
		return fmt.Errorf("%s", errMsg)
	}

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

	resNet, errNet := uc.ssh.RunCommand("docker network create --driver overlay --attachable tarhiata_internal 2>&1 || true")
	if errNet != nil {
		slog.Warn("aviso al asegurar red overlay tarhiata_internal", "error", errNet)
	}
	if resNet != nil && strings.Contains(resNet.Output, "error") && !strings.Contains(resNet.Output, "already exists") {
		slog.Warn("aviso al crear red tarhiata_internal", "output", resNet.Output)
	}
	return nil
}

func (uc *DeployDatabaseUseCase) resolveConstraint(db domain.SavedDatabase, config domain.ServerConfig) string {
	if db.DeployType == "multi-node" && db.TargetNode == "" {
		return fmt.Sprintf(`"node.labels.type == db_%s"`, db.Name)
	}
	if db.TargetNode == "" || db.TargetNode == "manager" {
		return `"node.role == manager"`
	}

	if db.TargetNode == "worker" {
		return uc.resolveWorkerConstraint(db.Name, "worker", config, `"node.role == worker"`)
	}
	if db.TargetNode == "db" {
		return uc.resolveWorkerConstraint(db.Name, "db", config, `"node.labels.type == db"`)
	}

	// Validar si coincide con un nodo o hostname real
	resCheckNode, errCheckNode := uc.ssh.RunCommand("docker node ls --format '{{.Hostname}}|{{.ID}}'")
	if errCheckNode == nil && resCheckNode != nil && resCheckNode.ExitCode == 0 {
		for _, line := range strings.Split(resCheckNode.Output, "\n") {
			parts := strings.Split(strings.TrimSpace(line), "|")
			for _, part := range parts {
				if strings.EqualFold(strings.TrimSpace(part), db.TargetNode) {
					return fmt.Sprintf(`"node.hostname == %s"`, db.TargetNode)
				}
			}
		}
	}

	slog.Info("TargetNode no corresponde a un hostname de Swarm. Usando node.role == manager", "targetNode", db.TargetNode)
	return `"node.role == manager"`
}

func (uc *DeployDatabaseUseCase) resolveWorkerConstraint(dbName, nodeType string, config domain.ServerConfig, targetConstraint string) string {
	filter := "role=worker"
	if nodeType == "db" {
		filter = "label=type=db"
	}
	resNode, errNode := uc.ssh.RunCommand(fmt.Sprintf("docker node ls --filter %s --format '{{.ID}}'", filter))
	if errNode == nil && resNode != nil && strings.TrimSpace(resNode.Output) != "" {
		return targetConstraint
	}

	token := config.VultrAPIToken
	if token == "" {
		token = config.DOAPIToken
	}
	if token == "" {
		fmt.Printf("⚠️ No hay nodos %s activos y no hay token de nube configurado. Usando Manager...\n", nodeType)
		return `"node.role == manager"`
	}

	nodeName := fmt.Sprintf("worker-%s", dbName)
	if nodeType == "db" {
		nodeName = fmt.Sprintf("worker-db-%s", dbName)
	}
	fmt.Printf("🏗️ No hay nodos %s activos. Aprovisionando VM '%s' en la nube...\n", nodeType, nodeName)
	workerUC := NewProvisionWorkerUseCase(uc.ssh)
	_, errProv := workerUC.ExecuteWithRegion(config, nodeName, nodeType, "")
	if errProv != nil {
		fmt.Printf("⚠️ No se pudo aprovisionar VM %s automática (%v). Usando Manager...\n", nodeType, errProv)
		return `"node.role == manager"`
	}
	return targetConstraint
}

func (uc *DeployDatabaseUseCase) prepareStorage(volumeHostPath string, cleanExisting bool, engine string, deployType string) {
	fmt.Printf("📁 Preparando almacenamiento persistente en el nodo (%s)...\n", deployType)
	uid := "999:999"
	if strings.EqualFold(engine, "postgres") {
		uid = "70:70"
	}

	checkCmd := fmt.Sprintf("test -d %s && ls -A %s", volumeHostPath, volumeHostPath)
	resCheck, errCheck := uc.ssh.RunCommand(checkCmd)
	hasExistingData := errCheck == nil && resCheck != nil && strings.TrimSpace(resCheck.Output) != ""

	if hasExistingData && cleanExisting {
		fmt.Printf("🧹 [Recovery Mode] Limpiando datos antiguos en %s antes de desplegar...\n", volumeHostPath)
		if _, errRm := uc.ssh.RunCommand(fmt.Sprintf("rm -rf %s/*", volumeHostPath)); errRm != nil {
			slog.Warn("aviso al limpiar datos antiguos", "path", volumeHostPath, "error", errRm)
		}
	}
	if hasExistingData && !cleanExisting {
		fmt.Printf("📦 [Recovery Mode] ¡Se detectaron datos previos en %s! Reutilizando volumen host...\n", volumeHostPath)
	}

	if _, errMk := uc.ssh.RunCommand(fmt.Sprintf("mkdir -p %s && chown -R %s %s", volumeHostPath, uid, volumeHostPath)); errMk != nil {
		slog.Warn("aviso al preparar permisos de directorio", "path", volumeHostPath, "error", errMk)
	}
}

func (uc *DeployDatabaseUseCase) buildCreateCommand(db domain.SavedDatabase, serviceName, constraint string) (string, error) {
	safePassword := strings.ReplaceAll(db.Password, "'", `'"'"'`)
	engineLower := strings.ToLower(db.Engine)

	switch engineLower {
	case "postgres":
		return fmt.Sprintf(
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
		), nil
	case "mongo", "mongodb":
		return fmt.Sprintf(
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
		), nil
	case "mysql", "mariadb":
		return fmt.Sprintf(
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
		), nil
	case "redis":
		return fmt.Sprintf(
			`docker service create \
			--name %s \
			--detach=true \
			--network tarhiata_internal \
			--mount type=bind,source=%s,destination=/data \
			--constraint %s \
			redis:7-alpine redis-server --requirepass '%s'`,
			serviceName, db.VolumeHostPath, constraint, safePassword,
		), nil
	case "minio", "s3":
		return fmt.Sprintf(
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
		), nil
	default:
		return "", fmt.Errorf("motor de base de datos no soportado: %s", db.Engine)
	}
}

