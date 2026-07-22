package usecases

import (
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
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

	fmt.Printf("\n🚀 Desplegando Base de Datos: %s (%s)...\n", db.Name, db.Engine)

	constraint := `"node.role == manager"`
	if db.DeployType == "multi-node" {
		constraint = fmt.Sprintf(`"node.labels.type == db_%s"`, db.Name)
	}

	fmt.Printf("📁 Preparando almacenamiento persistente en el nodo (%s)...\n", db.DeployType)
	var uid string
	if db.Engine == "postgres" {
		uid = "70:70"
	} else {
		uid = "999:999"
	}

	// Usamos un servicio efímero de Swarm para crear la carpeta y asignar permisos en el nodo correcto
	initCmd := fmt.Sprintf(`docker service create --name init-perms-%s --restart-condition none --constraint %s --mount type=bind,source=/,destination=/host alpine sh -c "mkdir -p /host%s && chown -R %s /host%s"`, db.Name, constraint, db.VolumeHostPath, uid, db.VolumeHostPath)
	uc.ssh.RunCommand(initCmd)

	// Esperamos a que termine y limpiamos
	uc.ssh.RunCommand(fmt.Sprintf("docker service rm init-perms-%s", db.Name))

	// 2. Apagar la BD si ya existía para actualizarla
	uc.ssh.RunCommand(fmt.Sprintf("docker service rm %s", db.Name))

	// 3. Construir el comando de docker service create

	// Escapar comillas simples para evitar inyección de bash
	safePassword := strings.ReplaceAll(db.Password, "'", `'"'"'`)

	var createCmd string
	if db.Engine == "postgres" {
		createCmd = fmt.Sprintf(
			`docker service create \
			--name %s \
			--network tarhiata_internal \
			--mount type=bind,source=%s,destination=/var/lib/postgresql/data \
			-e POSTGRES_USER=admin \
			-e POSTGRES_PASSWORD='%s' \
			-e POSTGRES_DB=db \
			--constraint %s \
			postgres:15-alpine`,
			db.Name, db.VolumeHostPath, safePassword, constraint,
		)
	} else if db.Engine == "mongo" {
		createCmd = fmt.Sprintf(
			`docker service create \
			--name %s \
			--network tarhiata_internal \
			--mount type=bind,source=%s,destination=/data/db \
			-e MONGO_INITDB_ROOT_USERNAME=admin \
			-e MONGO_INITDB_ROOT_PASSWORD='%s' \
			--constraint %s \
			mongo:6`,
			db.Name, db.VolumeHostPath, safePassword, constraint,
		)
	} else {
		return fmt.Errorf("motor de base de datos no soportado: %s", db.Engine)
	}

	// 4. Ejecutar el despliegue
	res, err := uc.ssh.RunCommand(createCmd)
	if err != nil || res.ExitCode != 0 {
		return fmt.Errorf("error creando servicio de BD: %s", res.Output)
	}

	// Limpiamos el output por seguridad
	fmt.Printf("✅ ¡Base de datos '%s' (Single-Node) ha sido desplegada correctamente y está anclada a %s!\n", db.Name, db.VolumeHostPath)

	// Preparamos la URI segura local (ej postgres://admin:pass@service:5432)
	// Esto solo se imprime de forma local, no viaja por la red
	safeUri := fmt.Sprintf("%s://admin:********@%s:%d/db", db.Engine, db.Name, db.InternalPort)
	if db.Engine == "mongo" {
		safeUri = fmt.Sprintf("mongodb://admin:********@%s:27017/?authSource=admin", db.Name)
	}

	fmt.Printf("🔌 URI Interna (Oculta): %s\n", safeUri)
	return nil
}
