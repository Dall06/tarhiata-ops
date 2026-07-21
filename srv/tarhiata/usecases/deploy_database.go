package usecases

import (
	"fmt"

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

	// 1. Crear el volumen físico en el disco del host
	fmt.Printf("📁 Creando directorio local: %s...\n", db.VolumeHostPath)
	mkdirCmd := fmt.Sprintf("mkdir -p %s", db.VolumeHostPath)
	if _, err := uc.ssh.RunCommand(mkdirCmd); err != nil {
		return fmt.Errorf("error creando directorio físico: %w", err)
	}

	// 2. Apagar la BD si ya existía para actualizarla
	uc.ssh.RunCommand(fmt.Sprintf("docker service rm %s", db.Name))

	// 3. Construir el comando de docker service create
	var createCmd string
	if db.Engine == "postgres" {
		createCmd = fmt.Sprintf(
			`docker service create \
			--name %s \
			--network tarhiata_internal \
			--mount type=bind,source=%s,destination=/var/lib/postgresql/data \
			-e POSTGRES_USER=admin \
			-e POSTGRES_PASSWORD=%s \
			-e POSTGRES_DB=db \
			--constraint "node.role == manager" \
			postgres:15-alpine`,
			db.Name, db.VolumeHostPath, db.Password,
		)
	} else if db.Engine == "mongo" {
		createCmd = fmt.Sprintf(
			`docker service create \
			--name %s \
			--network tarhiata_internal \
			--mount type=bind,source=%s,destination=/data/db \
			-e MONGO_INITDB_ROOT_USERNAME=admin \
			-e MONGO_INITDB_ROOT_PASSWORD=%s \
			--constraint "node.role == manager" \
			mongo:6`,
			db.Name, db.VolumeHostPath, db.Password,
		)
	} else {
		return fmt.Errorf("motor de base de datos no soportado: %s", db.Engine)
	}

	// 4. Ejecutar el despliegue
	res, err := uc.ssh.RunCommand(createCmd)
	if err != nil || res.ExitCode != 0 {
		return fmt.Errorf("error creando servicio de BD: %s", res.Output)
	}

	// 5. Configurar permisos especiales (A veces Postgres requiere ajustes de chown dependiendo del usuario)
	// Como estamos ejecutando todo por root y type=bind crea las carpetas como root, postgres (uid 70)
	// en Alpine necesita ser dueño.
	if db.Engine == "postgres" {
		// Damos unos segundos a que levante el contenedor para luego cambiar el permiso local
		chownCmd := fmt.Sprintf("chown -R 70:70 %s", db.VolumeHostPath)
		uc.ssh.RunCommand(chownCmd)
		// Y reiniciamos el servicio para que tome los permisos
		uc.ssh.RunCommand(fmt.Sprintf("docker service update --force %s", db.Name))
	} else if db.Engine == "mongo" {
		// Mongo suele usar el uid 999
		chownCmd := fmt.Sprintf("chown -R 999:999 %s", db.VolumeHostPath)
		uc.ssh.RunCommand(chownCmd)
		uc.ssh.RunCommand(fmt.Sprintf("docker service update --force %s", db.Name))
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
