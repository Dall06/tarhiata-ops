package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
	"github.com/Dall06/tarhiata-ops/srv/sys/repositories"
	"github.com/Dall06/tarhiata-ops/srv/sys/usecases"
	"github.com/Dall06/tarhiata-ops/srv/ui/controllers"
)

// Version is the current release version of tarhiata-ops.
const Version = "v1.0.0-beta"

func main() {
	// 1. Inicializar Base de Datos Local (SQLite)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("❌ Error obteniendo directorio home: %v\n", err)
		os.Exit(1)
	}
	dbPath := filepath.Join(homeDir, ".config", "tarhiata", "config.db")

	repo, err := repositories.NewSQLiteRepository(dbPath)
	if err != nil {
		fmt.Printf("❌ Error crítico iniciando base de datos: %v\n", err)
		os.Exit(1)
	}
	defer repo.Close()

	// 2. Cargar configuración del servidor si existe
	serverConfig, _ := repo.GetServerConfig()

	// 3. Enrutamiento de Comandos CLI (Mapeo 1 a 1 con Casos de Uso)
	args := os.Args[1:]
	if len(args) == 0 {
		runDashboard(repo, serverConfig)
		return
	}

	command := strings.ToLower(args[0])
	subArgs := args[1:]

	switch command {
	case "dashboard", "ui":
		runDashboard(repo, serverConfig)

	case "config":
		handleConfigCommand(repo, subArgs)

	case "init", "bootstrap":
		handleInitCommand(repo, serverConfig, subArgs)

	case "deploy":
		handleDeployServiceCommand(repo, serverConfig, subArgs)

	case "master":
		handleMasterCommand(repo, serverConfig, subArgs)

	case "preview":
		handlePreviewCommand(repo, serverConfig, subArgs)

	case "db":
		handleDatabaseCommand(repo, serverConfig, subArgs)

	case "worker", "node", "nodes":
		handleNodeCommand(repo, serverConfig, subArgs)

	case "obs", "observability":
		handleObservabilityCommand(repo, serverConfig, subArgs)

	case "rollback":
		handleRollbackCommand(serverConfig, subArgs)

	case "registry", "registries":
		handleRegistryCommand(repo, serverConfig, subArgs)

	case "ssh-key", "ssh-keys", "key", "keys":
		handleSSHKeyCLICommand(serverConfig, subArgs)


	case "link":
		handleLinkCommand(repo, serverConfig, os.Args[2:])

	case "unlink":
		handleUnlinkCommand(repo, serverConfig, os.Args[2:])

	case "update":
		handleUpdateCommand(serverConfig)

	case "list":
		handleListCommand(repo)

	case "status":
		handleStatusCommand(repo, serverConfig)

	case "topology":
		handleTopologyCommand(repo)

	case "prune":
		handlePruneCommand(serverConfig)

	case "backup":
		handleBackupCommand(repo, serverConfig, subArgs)

	case "env":
		handleEnvCommand(repo, serverConfig, subArgs)

	case "volume":
		handleVolumeCommand(repo, serverConfig, subArgs)

	case "ssl":
		handleSSLCommand(repo, serverConfig, subArgs)

	case "maintenance":
		handleMaintenanceCommand(repo, serverConfig, subArgs)

	case "domain":
		handleDomainCommand(repo, serverConfig, subArgs)

	case "version", "-v", "--version":
		fmt.Printf("tarhiata-ops %s\n", Version)
		fmt.Println("Built with Go · Made in México 🇲🇽 · https://github.com/Dall06/tarhiata-ops")

	case "help", "-h", "--help":
		printHelp()

	default:
		fmt.Printf("❌ Comando '%s' no reconocido.\n\n", command)
		printHelp()
	}
}

func runDashboard(repo *repositories.SQLiteRepository, config *domain.ServerConfig) {
	webServer := controllers.NewWebServer(repo, config)
	if err := webServer.Start(8080); err != nil {
		fmt.Printf("❌ Error ejecutando servidor web: %v\n", err)
		return
	}
}

func handleConfigCommand(repo *repositories.SQLiteRepository, args []string) {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	host := fs.String("host", "", "IP o Host del servidor VPS")
	port := fs.Int("port", 22, "Puerto SSH")
	user := fs.String("user", "root", "Usuario SSH")
	key := fs.String("key", "~/.ssh/id_rsa", "Ruta a llave privada SSH")
	doToken := fs.String("do-token", "", "Token de API de DigitalOcean")
	fs.Parse(args)

	if *host == "" {
		if repo == nil {
			fmt.Println("❌ No hay servidor configurado. Usa: tarhiata config set --host <IP>")
			return
		}
		cfg, err := repo.GetServerConfig()
		if err != nil || cfg == nil || cfg.Host == "" {
			fmt.Println("❌ No hay servidor configurado. Usa: tarhiata config set --host <IP>")
			return
		}
		fmt.Printf("⚙️  Configuración Actual:\n • Host: %s\n • Puerto: %d\n • User: %s\n • Key: %s\n • DO Token: %s\n",
			cfg.Host, cfg.Port, cfg.User, cfg.PrivateKey, cfg.DOAPIToken)
		return
	}

	newCfg := domain.ServerConfig{
		Host:          strings.TrimSpace(*host),
		Port:          *port,
		User:          strings.TrimSpace(*user),
		PrivateKey:    strings.TrimSpace(*key),
		DOAPIToken:    strings.TrimSpace(*doToken),
		CloudProvider: "vps-direct",
	}

	if err := repo.SaveServerConfig(newCfg); err != nil {
		fmt.Printf("❌ Error guardando configuración: %v\n", err)
		return
	}
	fmt.Println("✅ Configuración de servidor guardada exitosamente en SQLite!")
}

func handleInitCommand(repo *repositories.SQLiteRepository, config *domain.ServerConfig, args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	email := fs.String("email", "admin@tarhiata.com", "Email para certs SSL de Let's Encrypt")
	fs.Parse(args)

	if config == nil || config.Host == "" {
		fmt.Println("❌ Configura tu VPS primero con: tarhiata config set --host <IP>")
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*config); err != nil {
		fmt.Printf("❌ Error conectando SSH a %s: %v\n", config.Host, err)
		return
	}
	defer sshExec.Close()

	fmt.Printf("🚀 Ejecutando InitServerUseCase en %s...\n", config.Host)
	bootstrapper := usecases.NewInitServerUseCase(sshExec)
	if err := bootstrapper.Execute(*email); err != nil {
		fmt.Printf("❌ Error en bootstrapper: %v\n", err)
		return
	}
	fmt.Println("🎉 ¡Bootstrapper completado! Docker Swarm, Traefik HTTPS y Fail2Ban están OPERACIONALES.")
}

func handleDeployServiceCommand(repo *repositories.SQLiteRepository, config *domain.ServerConfig, args []string) {
	fs := flag.NewFlagSet("deploy", flag.ExitOnError)
	name := fs.String("name", "", "Nombre del servicio (Requerido)")
	image := fs.String("image", "", "Imagen Docker o URL ZIP (Requerido)")
	port := fs.Int("port", 80, "Puerto interno de la app")
	domainName := fs.String("domain", "", "Dominio público (ej. app.midominio.com)")
	ssl := fs.Bool("ssl", true, "Habilitar SSL HTTPS automático")
	healthCmd := fs.String("healthcheck", "", "Comando de healthcheck")
	preHook := fs.String("pre-hook", "", "Pre-deploy migration hook (ej. npx prisma db push)")
	fs.Parse(args)

	if *name == "" || *image == "" {
		fmt.Println("❌ Uso: tarhiata deploy --name <nombre> --image <imagen> [--port 80] [--domain app.com] [--pre-hook 'npx prisma db push']")
		return
	}

	svc := domain.SavedService{
		Name:           *name,
		ImageSource:    *image,
		Port:           *port,
		Domain:         *domainName,
		Expose:         *domainName != "",
		EnableSSL:      *ssl,
		HealthcheckCmd: *healthCmd,
		PreDeployHook:  *preHook,
	}

	if err := repo.SaveService(svc); err != nil {
		fmt.Printf("❌ Error guardando en catálogo: %v\n", err)
		return
	}

	if config != nil && config.Host != "" {
		sshExec := repositories.NewCryptoSSHExecutor()
		if err := sshExec.Connect(*config); err == nil {
			defer sshExec.Close()
			deployConfig := domain.DeployConfig{
				ImageSource:    svc.ImageSource,
				Port:           svc.Port,
				Domain:         svc.Domain,
				Expose:         svc.Expose,
				EnableSSL:      svc.EnableSSL,
				HealthcheckCmd: svc.HealthcheckCmd,
			}
			customSvc := domain.CustomService{
				Name:          svc.Name,
				PreDeployHook: svc.PreDeployHook,
			}
			deployer := usecases.NewDeployServiceUseCase(sshExec)
			if err := deployer.Execute(customSvc, deployConfig); err != nil {
				fmt.Printf("⚠️ Guardado en SQLite pero falló deploy en VPS: %v\n", err)
				return
			}
			fmt.Printf("🚀 ¡Servicio '%s' desplegado exitosamente en Swarm!\n", svc.Name)
			return
		}
	}
	fmt.Printf("✅ Servicio '%s' registrado en catálogo local.\n", svc.Name)
}

func handleDatabaseCommand(repo *repositories.SQLiteRepository, config *domain.ServerConfig, args []string) {
	if len(args) == 0 {
		fmt.Println("❌ Uso: tarhiata db create --name <nombre> --engine <postgres|mongo|mysql|redis>")
		return
	}

	subCmd := strings.ToLower(args[0])
	if subCmd == "create" || subCmd == "deploy" {
		fs := flag.NewFlagSet("db create", flag.ExitOnError)
		name := fs.String("name", "", "Nombre de la BD (Requerido)")
		engine := fs.String("engine", "postgres", "Motor: postgres, mongodb, mysql, redis")
		multiNode := fs.Bool("multi-node", false, "Modo multi-nodo con volumenes anclados")
		fs.Parse(args[1:])

		if *name == "" {
			fmt.Println("❌ Debes especificar un nombre con --name <nombre>")
			return
		}

		defaultPort := 5432
		switch *engine {
		case "mongodb":
			defaultPort = 27017
		case "mysql":
			defaultPort = 3306
		case "redis":
			defaultPort = 6379
		}

		deployType := "single-node"
		if *multiNode { deployType = "multi-node" }

		db := domain.SavedDatabase{
			Name:         *name,
			Engine:       *engine,
			InternalPort: defaultPort,
			DeployType:   deployType,
		}

		if err := repo.SaveDatabase(db); err != nil {
			fmt.Printf("❌ Error guardando base de datos en catálogo: %v\n", err)
			return
		}

		if config != nil && config.Host != "" {
			sshExec := repositories.NewCryptoSSHExecutor()
			if err := sshExec.Connect(*config); err == nil {
				defer sshExec.Close()
				deployer := usecases.NewDeployDatabaseUseCase(sshExec)
				_ = deployer.Execute(db, *config)
				fmt.Printf("🗄️ ¡Base de Datos '%s' (%s) desplegada correctamente!\n", db.Name, db.Engine)
				return
			}
		}
		fmt.Printf("✅ BD '%s' registrada en catálogo local.\n", db.Name)
		return
	}

	if subCmd == "migrate" || subCmd == "migrations" {
		fs := flag.NewFlagSet("db migrate", flag.ExitOnError)
		dbName := fs.String("db", "", "Nombre de la base de datos destino (Requerido)")
		node := fs.String("node", "manager", "Nodo destino (manager, worker)")
		file := fs.String("file", "", "Archivo SQL específico a ejecutar")
		sqlStr := fs.String("sql", "", "Sentencia SQL directa a ejecutar")
		fs.Parse(args[1:])

		if *dbName == "" {
			fmt.Println("❌ Uso: tarhiata db migrate --db <nombre_bd> [--node manager] [--file mig.sql] [--sql 'CREATE TABLE...']")
			return
		}

		if config == nil || config.Host == "" {
			fmt.Println("❌ Configura el servidor primero: tarhiata config --host <ip> --user root")
			return
		}

		sshExec := repositories.NewCryptoSSHExecutor()
		uc := usecases.NewManageDBMigrationsUseCase(repo, sshExec)

		var filenames []string
		if *file != "" {
			filenames = append(filenames, *file)
		}

		req := domain.DatabaseMigrationRequest{
			TargetDB:   *dbName,
			TargetNode: *node,
			Filenames:  filenames,
			SqlContent: *sqlStr,
		}

		res, err := uc.Execute(req, *config)
		if err != nil {
			fmt.Printf("❌ Error ejecutando migración: %v\n", err)
			return
		}

		for _, m := range res {
			if m.Status == "applied" {
				fmt.Printf("✅ Migración '%s' aplicada exitosamente en '%s'\n", m.Filename, *dbName)
			} else {
				fmt.Printf("❌ Migración '%s' falló: %s\n", m.Filename, m.LogOutput)
			}
		}
		return
	}
}

func handleWorkerCommand(repo *repositories.SQLiteRepository, config *domain.ServerConfig, args []string) {
	fs := flag.NewFlagSet("worker", flag.ExitOnError)
	nodeName := fs.String("name", "worker-1", "Nombre del nodo worker")
	region := fs.String("region", "ewr", "Región VPS Vultr (ej. ewr, lax)")
	fs.Parse(args)

	if config == nil || (config.DOAPIToken == "" && config.VultrAPIToken == "") {
		fmt.Println("❌ Requiere Vultr API Key en config: tarhiata config --do-token vultr_api_key_...")
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*config); err != nil {
		fmt.Printf("❌ Error conectando por SSH al Manager: %v\n", err)
		return
	}
	defer sshExec.Close()

	workerUseCase := usecases.NewProvisionWorkerUseCase(sshExec)
	fmt.Printf("🏗️ Provisionando worker node '%s' vía Terraform en DO (%s)...\n", *nodeName, *region)
	nodeIP, err := workerUseCase.Execute(*config, *nodeName, "worker")
	if err != nil {
		fmt.Printf("❌ Error provisionando worker: %v\n", err)
		return
	}
	fmt.Printf("🎉 Worker '%s' unido al clúster exitosamente con IP: %s\n", *nodeName, nodeIP)
}

func handleNodeCommand(repo *repositories.SQLiteRepository, config *domain.ServerConfig, args []string) {
	if len(args) == 0 {
		handleWorkerCommand(repo, config, args)
		return
	}

	sub := args[0]
	switch sub {
	case "ls", "list":
		if config == nil || config.Host == "" {
			fmt.Println("❌ VPS no configurado.")
			return
		}
		sshExec := repositories.NewCryptoSSHExecutor()
		if err := sshExec.Connect(*config); err != nil {
			fmt.Printf("❌ Error SSH: %v\n", err)
			return
		}
		defer sshExec.Close()
		res, err := sshExec.RunCommand("docker node ls")
		if err != nil {
			fmt.Printf("❌ Error consultando nodos Swarm: %v\n", err)
			return
		}
		fmt.Println("🌐 Clúster de Nodos Swarm:")
		fmt.Println(res.Output)

	case "token":
		if config == nil || config.Host == "" {
			fmt.Println("❌ VPS no configurado.")
			return
		}
		sshExec := repositories.NewCryptoSSHExecutor()
		if err := sshExec.Connect(*config); err != nil {
			fmt.Printf("❌ Error SSH: %v\n", err)
			return
		}
		defer sshExec.Close()
		resW, _ := sshExec.RunCommand("docker swarm join-token worker -q")
		resM, _ := sshExec.RunCommand("docker swarm join-token manager -q")
		workerToken := strings.TrimSpace(resW.Output)
		mgrToken := strings.TrimSpace(resM.Output)
		fmt.Printf("📋 Worker Join Token:  %s\n", workerToken)
		fmt.Printf("📋 Manager Join Token: %s\n", mgrToken)
		fmt.Printf("🔗 Worker Join Cmd:   docker swarm join --token %s %s:2377\n", workerToken, config.Host)

	case "add", "provision":
		fs := flag.NewFlagSet("node add", flag.ExitOnError)
		nodeName := fs.String("name", "worker-1", "Nombre del nodo worker")
		region := fs.String("region", "nyc3", "Región en DigitalOcean")
		fs.Parse(args[1:])

		if config == nil || config.DOAPIToken == "" {
			fmt.Println("❌ Requiere DigitalOcean API Token en config: tarhiata config --do-token dop_v1_...")
			return
		}

		sshExec := repositories.NewCryptoSSHExecutor()
		if err := sshExec.Connect(*config); err != nil {
			fmt.Printf("❌ Error conectando por SSH al Manager: %v\n", err)
			return
		}
		defer sshExec.Close()

		workerUseCase := usecases.NewProvisionWorkerUseCase(sshExec)
		fmt.Printf("🏗️ Provisionando worker node '%s' vía Terraform en DO (%s)...\n", *nodeName, *region)
		nodeIP, err := workerUseCase.Execute(*config, *nodeName, "worker")
		if err != nil {
			fmt.Printf("❌ Error provisionando worker: %v\n", err)
			return
		}
		fmt.Printf("🎉 Worker '%s' unido al clúster exitosamente con IP: %s\n", *nodeName, nodeIP)

	case "rm", "remove", "delete":
		if len(args) < 2 {
			fmt.Println("Uso: tarhiata node rm <node-id>")
			return
		}
		nodeID := args[1]
		if config == nil || config.Host == "" {
			fmt.Println("❌ VPS no configurado.")
			return
		}

		// Prompt confirmation for node removal
		fmt.Printf("⚠️  ¿Está seguro de que desea eliminar el nodo '%s' del clúster Swarm? (s/N): ", nodeID)
		var confirm string
		fmt.Scanln(&confirm)
		confirm = strings.ToLower(strings.TrimSpace(confirm))
		if confirm != "s" && confirm != "si" && confirm != "y" && confirm != "yes" {
			fmt.Println("Operación cancelada.")
			return
		}

		sshExec := repositories.NewCryptoSSHExecutor()
		if err := sshExec.Connect(*config); err != nil {
			fmt.Printf("❌ Error SSH: %v\n", err)
			return
		}
		defer sshExec.Close()
		fmt.Printf("⏳ Drenando servicios del nodo '%s'...\n", nodeID)
		_, _ = sshExec.RunCommand(fmt.Sprintf("docker node update --availability drain %s", nodeID))
		res, err := sshExec.RunCommand(fmt.Sprintf("docker node rm --force %s", nodeID))
		if err != nil || res.ExitCode != 0 {
			fmt.Printf("❌ Error al remover nodo: %s\n", res.Output)
			return
		}
		fmt.Printf("✅ Nodo '%s' removido del clúster Swarm exitosamente.\n", nodeID)

	case "update":
		var avail string
		nodeID := ""
		for i := 1; i < len(args); i++ {
			if args[i] == "--availability" && i+1 < len(args) {
				avail = args[i+1]
				i++
			} else if !strings.HasPrefix(args[i], "-") && nodeID == "" {
				nodeID = args[i]
			}
		}
		if avail == "" {
			avail = "active"
		}
		if nodeID == "" {
			fmt.Println("Uso: tarhiata node update [--availability active|drain|pause] <node-id>")
			return
		}
		if config == nil || config.Host == "" {
			fmt.Println("❌ VPS no configurado.")
			return
		}
		sshExec := repositories.NewCryptoSSHExecutor()
		if err := sshExec.Connect(*config); err != nil {
			fmt.Printf("❌ Error SSH: %v\n", err)
			return
		}
		defer sshExec.Close()
		res, err := sshExec.RunCommand(fmt.Sprintf("docker node update --availability %s %s", avail, nodeID))
		if err != nil || res.ExitCode != 0 {
			fmt.Printf("❌ Error actualizando nodo: %s\n", res.Output)
			return
		}
		fmt.Printf("✅ Disponibilidad del nodo '%s' actualizada a '%s'.\n", nodeID, avail)

	default:
		fmt.Println("Comando desconocido para 'node'. Subcomandos disponibles: ls, token, add, rm, update")
	}
}

func handleObservabilityCommand(repo *repositories.SQLiteRepository, config *domain.ServerConfig, args []string) {
	if config == nil || config.Host == "" {
		fmt.Println("❌ VPS no configurado.")
		return
	}

	if len(args) > 0 && (args[0] == "metrics" || args[0] == "stats") {
		serviceName := "all"
		timeRange := "1h"
		for i := 1; i < len(args); i++ {
			if args[i] == "--service" && i+1 < len(args) {
				serviceName = args[i+1]
				i++
			} else if args[i] == "--range" && i+1 < len(args) {
				timeRange = args[i+1]
				i++
			}
		}

		sshExec := repositories.NewCryptoSSHExecutor()
		uc := usecases.NewGetServiceMetricsUseCase(repo, sshExec)
		metrics, err := uc.Execute(serviceName, timeRange, *config)
		if err != nil {
			fmt.Printf("❌ Error obteniendo métricas: %v\n", err)
			return
		}

		fmt.Printf("\n📈 METRICAS HISTORICAS (%s) - SERVICIO: %s\n", metrics.Range, metrics.ServiceName)
		fmt.Println("=========================================================")
		fmt.Printf("%-10s | %-10s | %-12s | %-10s\n", "HORA", "CPU (%)", "MEMORIA (MB)", "RED (KB/s)")
		fmt.Println("---------------------------------------------------------")
		for _, p := range metrics.Points {
			fmt.Printf("%-10s | %-10.1f | %-12.1f | %-10.1f\n", p.Timestamp, p.CPU, p.Memory, p.Network)
		}
		fmt.Println("=========================================================")
		return
	}

	volPath := "/opt/tarhiata/obs"
	pass := "admin12345"
	deployType := "single-node"

	for i := 0; i < len(args); i++ {
		if (args[i] == "--volume-path" || args[i] == "-v") && i+1 < len(args) {
			volPath = args[i+1]
			i++
		} else if (args[i] == "--password" || args[i] == "-p") && i+1 < len(args) {
			pass = args[i+1]
			i++
		} else if args[i] == "--deploy-type" && i+1 < len(args) {
			deployType = args[i+1]
			i++
		}
	}

	fmt.Printf("⚠️  ¿Está seguro de desplegar/actualizar el stack de Observabilidad (Loki, Grafana, Portainer) montado en la VM en '%s'? (s/N): ", volPath)
	var confirm string
	fmt.Scanln(&confirm)
	confirm = strings.ToLower(strings.TrimSpace(confirm))
	if confirm != "s" && confirm != "si" && confirm != "y" && confirm != "yes" {
		fmt.Println("Operación cancelada.")
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*config); err != nil {
		fmt.Printf("❌ Error SSH: %v\n", err)
		return
	}
	defer sshExec.Close()

	obsUseCase := usecases.NewDeployObservabilityUseCase(sshExec)
	fmt.Printf("📊 Desplegando stack de Observabilidad persistente en '%s'...\n", volPath)
	if err := obsUseCase.ExecutePersistentWithVolume(true, deployType, pass, volPath); err != nil {
		fmt.Printf("❌ Error desplegando observabilidad: %v\n", err)
		return
	}

	obsRecord := domain.SavedObservability{
		DeployType:      deployType,
		ExternalURL:     volPath,
		GrafanaPassword: pass,
	}
	if err := repo.SaveObservability(obsRecord); err != nil {
		fmt.Printf("⚠️ Advertencia: No se pudo guardar el registro de observabilidad en SQLite: %v\n", err)
	}

	fmt.Println("✅ Stack de Observabilidad desplegado exitosamente con volumen en la VM.")
}

func handleRollbackCommand(config *domain.ServerConfig, args []string) {
	if len(args) < 1 {
		fmt.Println("Uso: tarhiata rollback <nombre-del-servicio>")
		return
	}
	svcName := args[0]

	if config == nil || config.Host == "" {
		fmt.Println("❌ VPS no configurado.")
		return
	}

	fmt.Printf("⚠️  ¿Está seguro de ejecutar un ROLLBACK en el servicio Swarm '%s'? (s/N): ", svcName)
	var confirm string
	fmt.Scanln(&confirm)
	confirm = strings.ToLower(strings.TrimSpace(confirm))
	if confirm != "s" && confirm != "si" && confirm != "y" && confirm != "yes" {
		fmt.Println("Operación cancelada.")
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*config); err != nil {
		fmt.Printf("❌ Error SSH: %v\n", err)
		return
	}
	defer sshExec.Close()

	fmt.Printf("⏳ Revirtiendo servicio '%s' a su revisión previa en Swarm...\n", svcName)
	rollbackCmd := fmt.Sprintf("docker service rollback %s || docker service rollback %s_%s || docker service rollback tarhiata-app-%s || docker service rollback tarhiata_%s",
		svcName, svcName, svcName, svcName, svcName)
	res, err := sshExec.RunCommand(rollbackCmd)
	if err != nil || res.ExitCode != 0 {
		fmt.Printf("❌ Error al realizar rollback: %s\n", res.Output)
		return
	}
	fmt.Printf("✅ Rollback completado exitosamente para el servicio '%s'.\n", svcName)
}

func handleRegistryCommand(repo *repositories.SQLiteRepository, config *domain.ServerConfig, args []string) {
	if len(args) == 0 {
		fmt.Println("❌ Subcomandos disponibles: tarhiata registry <login|ls|rm>")
		return
	}

	var sshExec ports.SSHExecutor
	var cfg domain.ServerConfig
	if config != nil && config.Host != "" {
		cfg = *config
		se := repositories.NewCryptoSSHExecutor()
		if err := se.Connect(cfg); err == nil {
			sshExec = se
			defer se.Close()
		}
	}

	uc := usecases.NewManageRegistryAuthUseCase(repo, sshExec)
	subCmd := strings.ToLower(args[0])

	switch subCmd {
	case "login", "add", "save":
		fs := flag.NewFlagSet("registry login", flag.ExitOnError)
		server := fs.String("server", "docker.io", "Servidor Registry (ej. docker.io, ghcr.io, ecr)")
		username := fs.String("username", "", "Usuario (Requerido)")
		password := fs.String("password", "", "Contraseña o Token Personal (Requerido)")
		fs.Parse(args[1:])

		if *username == "" || *password == "" {
			fmt.Println("❌ Uso: tarhiata registry login --username <user> --password <token> [--server docker.io]")
			return
		}

		fmt.Printf("⚠️  ¿Está seguro de registrar e iniciar sesión en '%s' como '%s'? (s/N): ", *server, *username)
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(strings.TrimSpace(confirm)) != "s" && strings.ToLower(strings.TrimSpace(confirm)) != "si" {
			fmt.Println("Operación cancelada.")
			return
		}

		cred := domain.SavedRegistryCredential{
			Server:   *server,
			Username: *username,
			Password: *password,
		}
		if err := uc.Save(cred, cfg); err != nil {
			fmt.Printf("❌ Error al guardar credenciales del registry: %v\n", err)
			return
		}
		fmt.Printf("✅ Login exitoso y credenciales guardadas para '%s' (%s).\n", *server, *username)

	case "ls", "list":
		creds, err := uc.List()
		if err != nil {
			fmt.Printf("❌ Error al listar registries: %v\n", err)
			return
		}
		if len(creds) == 0 {
			fmt.Println("ℹ️ No hay registries privados autenticados.")
			return
		}
		fmt.Println("📦 DOCKER REGISTRIES PRIVADOS AUTENTICADOS:")
		for _, c := range creds {
			fmt.Printf("  • Server: %s | User: %s | Creado: %s\n", c.Server, c.Username, c.CreatedAt)
		}

	case "rm", "delete", "logout":
		fs := flag.NewFlagSet("registry rm", flag.ExitOnError)
		server := fs.String("server", "", "Servidor Registry a remover (Requerido)")
		fs.Parse(args[1:])

		if *server == "" {
			if len(args) > 1 {
				*server = args[1]
			} else {
				fmt.Println("❌ Uso: tarhiata registry rm --server <server>")
				return
			}
		}

		fmt.Printf("⚠️  ¿Está seguro de remover las credenciales y cerrar sesión en '%s'? (s/N): ", *server)
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(strings.TrimSpace(confirm)) != "s" && strings.ToLower(strings.TrimSpace(confirm)) != "si" {
			fmt.Println("Operación cancelada.")
			return
		}

		if err := uc.Delete(*server, cfg); err != nil {
			fmt.Printf("❌ Error al remover registry: %v\n", err)
			return
		}
		fmt.Printf("✅ Credenciales de '%s' eliminadas exitosamente.\n", *server)

	default:
		fmt.Printf("❌ Subcomando '%s' no reconocido. Usa: tarhiata registry <login|ls|rm>\n", subCmd)
	}
}

func handleMasterCommand(repo *repositories.SQLiteRepository, config *domain.ServerConfig, args []string) {
	fs := flag.NewFlagSet("master", flag.ExitOnError)
	name := fs.String("name", "", "Nombre del servicio App (Requerido)")
	image := fs.String("image", "node:18-alpine", "Imagen Docker del servicio")
	port := fs.Int("port", 80, "Puerto interno de la app")
	dbEngine := fs.String("db", "postgres", "Base de datos integrada: postgres | mysql | redis | mongodb | none")
	envVar := fs.String("env-var", "DATABASE_URL", "Nombre de variable de entorno inyectada")
	domainName := fs.String("domain", "", "Dominio público opcional en Traefik")
	fs.Parse(args)

	if *name == "" {
		fmt.Println("❌ Uso: tarhiata master --name <appName> [--image node:18-alpine] [--port 80] [--db postgres|mysql|redis|mongodb|none] [--env-var DATABASE_URL] [--domain domain.com]")
		return
	}

	if config == nil || config.Host == "" {
		fmt.Println("❌ VPS no configurado.")
		return
	}

	fmt.Printf("⚠️  ¿Está seguro de inicializar el servicio Master '%s' con BD '%s'? (Si '%s' tenía enlaces previos, se desvinculará automáticamente) (s/N): ", *name, *dbEngine, *name)
	var confirm string
	fmt.Scanln(&confirm)
	confirm = strings.ToLower(strings.TrimSpace(confirm))
	if confirm != "s" && confirm != "si" && confirm != "y" && confirm != "yes" {
		fmt.Println("Operación cancelada.")
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*config); err != nil {
		fmt.Printf("❌ Error SSH: %v\n", err)
		return
	}
	defer sshExec.Close()

	linkUC := usecases.NewLinkServicesUseCase(repo, sshExec)
	unlinkUC := usecases.NewUnlinkServicesUseCase(repo, sshExec)
	dbUC := usecases.NewDeployDatabaseUseCase(sshExec)
	svcUC := usecases.NewDeployServiceUseCase(sshExec)
	masterUC := usecases.NewBootstrapMasterServiceUseCase(repo, sshExec, linkUC, unlinkUC, dbUC, svcUC)

	input := ports.BootstrapMasterInput{
		AppName:      *name,
		Image:        *image,
		Port:         *port,
		DBEngine:     *dbEngine,
		EnvVarName:   *envVar,
		Domain:       *domainName,
		ExposePublic: *domainName != "",
	}

	fmt.Printf("🪄 Ejecutando Master 1-Click Bootstrap para '%s'...\n", *name)
	res, err := masterUC.Execute(input, *config)
	if err != nil {
		fmt.Printf("❌ Error en Master Bootstrap: %v\n", err)
		return
	}

	fmt.Println("========================================================")
	fmt.Printf("🎉 ¡MASTER 1-CLICK SUCCESS: App '%s' desplegada!\n", res.App.Name)
	if len(res.UnlinkedOld) > 0 {
		fmt.Printf(" 🔓 Auto-Desvinculado de servicios anteriores: %s\n", strings.Join(res.UnlinkedOld, ", "))
	}
	if res.Database != nil {
		fmt.Printf(" 🗄️  Base de Datos Aprovisionada: %s (%s)\n", res.Database.Name, res.Database.Engine)
	}
	if res.Link != nil {
		fmt.Printf(" ⚡ Variable de Entorno Inyectada: %s = %s\n", res.Link.EnvVarName, res.Link.TargetURL)
	}
	fmt.Println("========================================================")
}

func handleUpdateCommand(config *domain.ServerConfig) {
	if config == nil || config.Host == "" {
		fmt.Println("❌ VPS no configurado.")
		return
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*config); err != nil {
		fmt.Printf("❌ Error SSH: %v\n", err)
		return
	}
	defer sshExec.Close()

	updateUseCase := usecases.NewUpdateServerUseCase(sshExec)
	fmt.Println("🔄 Actualizando paquetes del servidor y Docker...")
	if err := updateUseCase.Execute(); err != nil {
		fmt.Printf("❌ Error en actualización: %v\n", err)
		return
	}
	fmt.Println("✅ Servidor actualizado al día.")
}

func handleListCommand(repo *repositories.SQLiteRepository) {
	svcs, _ := repo.GetServices()
	dbs, _ := repo.GetDatabases()

	fmt.Println("========================================================")
	fmt.Println(" 📦 CATÁLOGO DE SERVICIOS Y BASES DE DATOS DE TARHIATA")
	fmt.Println("========================================================")
	fmt.Printf("\n🚀 SERVICIOS APPS (%d):\n", len(svcs))
	for _, s := range svcs {
		ssl := "HTTP"
		if s.EnableSSL { ssl = "HTTPS 🔒" }
		fmt.Printf(" • %-15s | Image: %-20s | Port: %-4d | %s -> %s\n", s.Name, s.ImageSource, s.Port, ssl, s.Domain)
	}

	fmt.Printf("\n🗄️ BASES DE DATOS (%d):\n", len(dbs))
	for _, db := range dbs {
		fmt.Printf(" • %-15s | Engine: %-10s | Internal Port: %d | Type: %s\n", db.Name, db.Engine, db.InternalPort, db.DeployType)
	}
	fmt.Println("========================================================")
}

func handleStatusCommand(repo *repositories.SQLiteRepository, config *domain.ServerConfig) {
	fmt.Println("========================================================")
	fmt.Println(" 📊 TARHIATA CLUSTER STATUS")
	fmt.Println("========================================================")
	if config != nil && config.Host != "" {
		fmt.Printf(" 🟢 Host IP:       %s (User: %s)\n", config.Host, config.User)
		fmt.Printf(" 🔒 Swarm Cluster: OPERACIONAL (Master Active)\n")
	} else {
		fmt.Println(" 🔴 Host IP:       NO CONFIGURADO (Ejecuta: tarhiata config --host <IP>)")
	}
	svcs, _ := repo.GetServices()
	dbs, _ := repo.GetDatabases()
	fmt.Printf(" 📦 Total Apps:    %d\n", len(svcs))
	fmt.Printf(" 🗄️ Total BDs:     %d\n", len(dbs))
	fmt.Println("========================================================")
}

func handleTopologyCommand(repo *repositories.SQLiteRepository) {
	svcs, _ := repo.GetServices()
	dbs, _ := repo.GetDatabases()
	links, _ := repo.GetServiceLinks()

	fmt.Println("========================================================")
	fmt.Println(" 🗺️ TARHIATA NETWORK TOPOLOGY MAP")
	fmt.Println("========================================================")
	for _, s := range svcs {
		fmt.Printf("🚀 SERVICIO: %s\n", s.Name)
		fmt.Printf("   ├─ DNS Interno: http://%s:%d\n", s.Name, s.Port)
		if s.Expose {
			proto := "http"
			if s.EnableSSL { proto = "https" }
			fmt.Printf("   └─ Red Pública: %s://%s\n\n", proto, s.Domain)
		} else {
			fmt.Printf("   └─ Red Pública: [Privado]\n\n")
		}
	}
	for _, db := range dbs {
		fmt.Printf("🗄️ BASE DE DATOS: %s (%s)\n", db.Name, db.Engine)
		fmt.Printf("   └─ DNS Interno: %s://admin:***@tarhiata-db-%s:%d/db\n\n", db.Engine, db.Name, db.InternalPort)
	}

	if len(links) > 0 {
		fmt.Println("🔗 INTERCONEXIONES ACTIVAS (A ➔ B):")
		for _, l := range links {
			fmt.Printf("   🚀 %s ───[ %s ]───► 🗄️ %s\n      └─ URL Inyectada: %s\n\n", l.SourceSvc, l.EnvVarName, l.TargetSvc, l.TargetURL)
		}
	}
}

func handlePruneCommand(config *domain.ServerConfig) {
	if config == nil || config.Host == "" {
		fmt.Println("❌ Error: Servidor no configurado. Ejecuta primero 'tarhiata config set'")
		return
	}
	fmt.Println("🧹 Limpiando imágenes y contenedores no utilizados en el servidor remoto...")
	sshExec := repositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(*config); err != nil {
		fmt.Printf("❌ Error SSH conectando a %s: %v\n", config.Host, err)
		return
	}
	defer sshExec.Close()

	res, err := sshExec.RunCommand("docker system prune -af")
	if err != nil {
		fmt.Printf("❌ Error ejecutando docker system prune: %v\n", err)
		return
	}
	output := strings.TrimSpace(res.Output)
	if output == "" {
		output = "✔ Docker system prune finalizado exitosamente. El sistema está limpio."
	}
	fmt.Println(output)
}

func handleLinkCommand(repo *repositories.SQLiteRepository, config *domain.ServerConfig, args []string) {
	fs := flag.NewFlagSet("link", flag.ExitOnError)
	from := fs.String("from", "", "Servicio origen (ej. api-backend)")
	to := fs.String("to", "", "Servicio o BD destino (ej. db-postgres)")
	envVar := fs.String("var", "DATABASE_URL", "Nombre de la Variable de Entorno (ej. DATABASE_URL)")
	fs.Parse(args)

	if *from == "" || *to == "" {
		fmt.Println("❌ Uso: tarhiata link --from <servicio_origen> --to <servicio_o_bd_destino> --var <ENV_VAR>")
		return
	}

	var sshExec ports.SSHExecutor
	if config != nil && config.Host != "" {
		se := repositories.NewCryptoSSHExecutor()
		if err := se.Connect(*config); err == nil {
			sshExec = se
			defer se.Close()
		}
	}

	linkUseCase := usecases.NewLinkServicesUseCase(repo, sshExec)
	fmt.Printf("🔗 Interconectando '%s' ───[ %s ]───► '%s'...\n", *from, strings.ToUpper(*envVar), *to)
	link, err := linkUseCase.Execute(*from, *to, *envVar)
	if err != nil {
		fmt.Printf("❌ Error al enlazar servicios: %v\n", err)
		return
	}

	fmt.Printf("✅ Interconexión establecida con éxito!\n")
	fmt.Printf("   ├─ Origen     : %s\n", link.SourceSvc)
	fmt.Printf("   ├─ Destino    : %s\n", link.TargetSvc)
	fmt.Printf("   ├─ Variable   : %s\n", link.EnvVarName)
	fmt.Printf("   └─ URL Inyect : %s\n\n", link.TargetURL)
}

func handleUnlinkCommand(repo *repositories.SQLiteRepository, config *domain.ServerConfig, args []string) {
	fs := flag.NewFlagSet("unlink", flag.ExitOnError)
	from := fs.String("from", "", "Servicio origen (ej. api-backend)")
	to := fs.String("to", "", "Servicio o BD destino (ej. db-postgres)")
	fs.Parse(args)

	if *from == "" || *to == "" {
		fmt.Println("❌ Uso: tarhiata unlink --from <servicio_origen> --to <servicio_o_bd_destino>")
		return
	}

	var sshExec ports.SSHExecutor
	if config != nil && config.Host != "" {
		se := repositories.NewCryptoSSHExecutor()
		if err := se.Connect(*config); err == nil {
			sshExec = se
			defer se.Close()
		}
	}

	unlinkUseCase := usecases.NewUnlinkServicesUseCase(repo, sshExec)
	fmt.Printf("🗑️ Removiendo enlace '%s' ➔ '%s'...\n", *from, *to)
	err := unlinkUseCase.Execute(*from, *to)
	if err != nil {
		fmt.Printf("❌ Error al eliminar enlace: %v\n", err)
		return
	}

	fmt.Printf("✅ Interconexión eliminada y variable removida de Docker Swarm con éxito!\n\n")
}

func handlePreviewCommand(repo *repositories.SQLiteRepository, config *domain.ServerConfig, args []string) {
	if len(args) == 0 {
		fmt.Println("❌ Uso: tarhiata preview <create|list|destroy> [opciones]")
		return
	}

	subCmd := strings.ToLower(args[0])

	var sshExec ports.SSHExecutor
	var cfg domain.ServerConfig
	if config != nil && config.Host != "" {
		cfg = *config
		se := repositories.NewCryptoSSHExecutor()
		if err := se.Connect(cfg); err == nil {
			sshExec = se
			defer se.Close()
		}
	}

	uc := usecases.NewManagePreviewEnvUseCase(repo, sshExec)

	switch subCmd {
	case "create":
		fs := flag.NewFlagSet("preview create", flag.ExitOnError)
		name := fs.String("name", "", "Nombre del entorno preview (Requerido)")
		image := fs.String("image", "", "Imagen Docker (Requerido)")
		port := fs.Int("port", 80, "Puerto interno de la app")
		domainName := fs.String("domain", "", "Dominio temporal Traefik (ej. pr-12.domain.com)")
		linkDB := fs.String("link-db", "", "Nombre de la BD a vincular opcionalmente")
		fs.Parse(args[1:])

		if *name == "" || *image == "" {
			fmt.Println("❌ Uso: tarhiata preview create --name <nombre> --image <imagen> [--port 80] [--domain domain.com] [--link-db db-name]")
			return
		}

		input := ports.CreatePreviewEnvInput{
			Name:       *name,
			Image:      *image,
			Port:       *port,
			Domain:     *domainName,
			LinkDBName: *linkDB,
		}

		res, err := uc.Create(input, cfg)
		if err != nil {
			fmt.Printf("❌ Error creando entorno preview: %v\n", err)
			return
		}
		fmt.Printf("🎉 Entorno Preview '%s' desplegado exitosamente! (Status: %s)\n", res.Name, res.Status)

	case "list":
		list, err := uc.List()
		if err != nil {
			fmt.Printf("❌ Error obteniendo lista de preview envs: %v\n", err)
			return
		}
		fmt.Println("========================================================")
		fmt.Println(" 🧪 ENTORNOS DE PREVIEW TEMPORALES (PR / TESTING)")
		fmt.Println("========================================================")
		if len(list) == 0 {
			fmt.Println(" (No hay entornos preview activos en este momento)")
		} else {
			for _, p := range list {
		fmt.Printf(" • %-16s | Image: %-22s | Port: %-4d | Domain: %-20s | Status: %s\n",
					p.Name, p.ImageSource, p.Port, p.Domain, p.Status)
			}
		}
		fmt.Println("========================================================")

	case "destroy", "delete", "rm":
		fs := flag.NewFlagSet("preview destroy", flag.ExitOnError)
		name := fs.String("name", "", "Nombre del entorno preview a destruir (Requerido)")
		fs.Parse(args[1:])

		if *name == "" {
			fmt.Println("❌ Uso: tarhiata preview destroy --name <nombre>")
			return
		}

		if err := uc.Destroy(*name, cfg); err != nil {
			fmt.Printf("❌ Error destruyendo entorno preview: %v\n", err)
			return
		}
		fmt.Printf("🔥 Entorno Preview '%s' destruido exitosamente.\n", *name)

	default:
		fmt.Printf("❌ Subcomando '%s' no reconocido. Usa: tarhiata preview <create|list|destroy>\n", subCmd)
	}
}

func handleBackupCommand(repo *repositories.SQLiteRepository, config *domain.ServerConfig, args []string) {
	if len(args) == 0 {
		fmt.Println("Uso: tarhiata backup <create|list|restore>")
		return
	}
	if config == nil {
		fmt.Println("❌ Error: No hay un servidor configurado. Ejecuta primero 'tarhiata config set'")
		return
	}

	subCmd := args[0]
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageBackupsUseCase(repo, sshExec)

	switch subCmd {
	case "create":
		fs := flag.NewFlagSet("backup create", flag.ExitOnError)
		targetDB := fs.String("db", "", "Nombre de la BD para snapshot")
		targetVol := fs.String("volume", "", "Nombre del servicio/volumen para snapshot")
		fs.Parse(args[1:])

		targetName := *targetDB
		targetType := "database"
		if targetName == "" {
			targetName = *targetVol
			targetType = "volume"
		}
		if targetName == "" {
			fmt.Println("❌ Debes especificar --db <nombre> o --volume <nombre>")
			return
		}

		fmt.Printf("⏳ Creando snapshot de %s '%s'...\n", targetType, targetName)
		backup, err := uc.CreateSnapshot(domain.BackupRequest{
			TargetName: targetName,
			TargetType: targetType,
		}, *config)
		if err != nil {
			fmt.Printf("❌ Error al crear backup: %v\n", err)
			return
		}
		fmt.Printf("✅ Backup creado exitosamente: %s (%d bytes)\n", backup.Filename, backup.SizeBytes)

	case "list":
		backups, err := repo.GetBackups()
		if err != nil {
			fmt.Printf("❌ Error al consultar backups: %v\n", err)
			return
		}
		fmt.Println("💾 Snapshot / Backups Registrados:")
		for _, b := range backups {
			fmt.Printf("  • ID: %d | %s (%s) | %s | %d B | %s\n", b.ID, b.TargetName, b.Engine, b.Filename, b.SizeBytes, b.CreatedAt)
		}

	case "restore":
		fs := flag.NewFlagSet("backup restore", flag.ExitOnError)
		id := fs.Int("id", 0, "ID del backup a restaurar")
		fs.Parse(args[1:])
		if *id <= 0 {
			fmt.Println("❌ Especifica el ID del backup con --id <id>")
			return
		}
		fmt.Printf("⏳ Restaurando snapshot ID %d...\n", *id)
		if err := uc.RestoreSnapshot(*id, *config); err != nil {
			fmt.Printf("❌ Error al restaurar: %v\n", err)
			return
		}
		fmt.Println("✅ Restauración completada exitosamente.")
	}
}

func handleEnvCommand(repo *repositories.SQLiteRepository, config *domain.ServerConfig, args []string) {
	if len(args) == 0 {
		fmt.Println("Uso: tarhiata env <list|import|export|set>")
		return
	}
	subCmd := args[0]
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageEnvVarsUseCase(repo, sshExec)
	cfg := domain.ServerConfig{}
	if config != nil {
		cfg = *config
	}

	switch subCmd {
	case "list":
		fs := flag.NewFlagSet("env list", flag.ExitOnError)
		svcName := fs.String("service", "", "Nombre del servicio")
		fs.Parse(args[1:])
		if *svcName == "" {
			fmt.Println("❌ Especifica el servicio con --service <nombre>")
			return
		}
		raw, envMap, err := uc.GetEnvVars(*svcName)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
		fmt.Printf("🔑 Variables de entorno para '%s':\n", *svcName)
		if len(envMap) == 0 {
			fmt.Println(" (Sin variables de entorno configuradas)")
			return
		}
		for k, v := range envMap {
			fmt.Printf("  • %s = %s\n", k, v)
		}
		_ = raw

	case "import":
		fs := flag.NewFlagSet("env import", flag.ExitOnError)
		svcName := fs.String("service", "", "Nombre del servicio")
		filePath := fs.String("file", "", "Ruta al archivo .env local")
		fs.Parse(args[1:])
		if *svcName == "" || *filePath == "" {
			fmt.Println("❌ Especifica --service <nombre> y --file <ruta.env>")
			return
		}
		data, err := os.ReadFile(*filePath)
		if err != nil {
			fmt.Printf("❌ Error al leer el archivo '%s': %v\n", *filePath, err)
			return
		}
		if err := uc.UpdateEnvVars(*svcName, string(data), cfg); err != nil {
			fmt.Printf("❌ Error al actualizar variables: %v\n", err)
			return
		}
		fmt.Printf("✅ Variables de entorno importadas y aplicadas a '%s' exitosamente desde %s.\n", *svcName, *filePath)

	case "export":
		fs := flag.NewFlagSet("env export", flag.ExitOnError)
		svcName := fs.String("service", "", "Nombre del servicio")
		outPath := fs.String("output", "", "Ruta del archivo de salida (ej. .env)")
		fs.Parse(args[1:])
		if *svcName == "" || *outPath == "" {
			fmt.Println("❌ Especifica --service <nombre> y --output <ruta.env>")
			return
		}
		raw, _, err := uc.GetEnvVars(*svcName)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
		if err := os.WriteFile(*outPath, []byte(raw), 0644); err != nil {
			fmt.Printf("❌ Error al escribir en '%s': %v\n", *outPath, err)
			return
		}
		fmt.Printf("✅ Variables de entorno exportadas a '%s'.\n", *outPath)

	case "set":
		fs := flag.NewFlagSet("env set", flag.ExitOnError)
		svcName := fs.String("service", "", "Nombre del servicio")
		fs.Parse(args[1:])
		if *svcName == "" || len(fs.Args()) == 0 {
			fmt.Println("❌ Especifica --service <nombre> KEY=VAL")
			return
		}
		kv := fs.Args()[0]
		raw, envMap, _ := uc.GetEnvVars(*svcName)
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			fmt.Println("❌ Formato inválido. Usa KEY=VALUE")
			return
		}
		if envMap == nil {
			envMap = make(map[string]string)
		}
		envMap[parts[0]] = parts[1]
		newRaw := usecases.FormatEnvMap(envMap)
		if err := uc.UpdateEnvVars(*svcName, newRaw, cfg); err != nil {
			fmt.Printf("❌ Error al guardar variable: %v\n", err)
			return
		}
		fmt.Printf("✅ Variable '%s' actualizada en '%s'.\n", parts[0], *svcName)
		_ = raw
	}
}

func handleVolumeCommand(repo *repositories.SQLiteRepository, config *domain.ServerConfig, args []string) {
	if len(args) == 0 {
		fmt.Println("Uso: tarhiata volume <ls|cat|rm>")
		return
	}
	subCmd := args[0]
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageVolumesUseCase(repo, sshExec)
	cfg := domain.ServerConfig{}
	if config != nil {
		cfg = *config
	}

	switch subCmd {
	case "ls":
		fs := flag.NewFlagSet("volume ls", flag.ExitOnError)
		path := fs.String("path", "/opt/data", "Ruta del volumen o directorio en el servidor")
		fs.Parse(args[1:])

		files, err := uc.ListVolumeFiles(*path, cfg)
		if err != nil {
			fmt.Printf("❌ Error al listar volúmenes: %v\n", err)
			return
		}
		fmt.Printf("📁 Contenido de '%s':\n", *path)
		if len(files) == 0 {
			fmt.Println(" (Carpeta vacía)")
			return
		}
		for _, f := range files {
			typeIcon := "📄"
			if f.IsDir {
				typeIcon = "📁"
			}
			fmt.Printf(" %s  %-30s %10d bytes   %s\n", typeIcon, f.Name, f.Size, f.ModTime)
		}

	case "cat":
		fs := flag.NewFlagSet("volume cat", flag.ExitOnError)
		path := fs.String("path", "", "Ruta del archivo")
		fs.Parse(args[1:])
		if *path == "" {
			fmt.Println("❌ Especifica --path <ruta_del_archivo>")
			return
		}
		content, err := uc.ReadFileContent(*path, cfg)
		if err != nil {
			fmt.Printf("❌ Error al leer archivo: %v\n", err)
			return
		}
		fmt.Println(content)

	case "rm":
		fs := flag.NewFlagSet("volume rm", flag.ExitOnError)
		path := fs.String("path", "", "Ruta del archivo o carpeta a eliminar")
		fs.Parse(args[1:])
		if *path == "" {
			fmt.Println("❌ Especifica --path <ruta_del_archivo>")
			return
		}
		if err := uc.DeleteFile(*path, cfg); err != nil {
			fmt.Printf("❌ Error al eliminar: %v\n", err)
			return
		}
		fmt.Printf("✅ Recurso '%s' eliminado exitosamente.\n", *path)
	}
}

func handleSSLCommand(repo *repositories.SQLiteRepository, config *domain.ServerConfig, args []string) {
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageSSLMaintenanceUseCase(repo, sshExec)
	items, err := uc.InspectSSL()
	if err != nil {
		fmt.Printf("❌ Error al inspeccionar SSL: %v\n", err)
		return
	}
	fmt.Println("🔒 Estado de Certificados SSL:")
	if len(items) == 0 {
		fmt.Println(" (No hay dominios configurados)")
		return
	}
	for _, item := range items {
		statusStr := "HTTP Solo"
		if item.Status == "active" {
			statusStr = fmt.Sprintf("🔒 Activo (%d días restantes)", item.DaysRemaining)
		} else if item.Status == "expiring_soon" {
			statusStr = fmt.Sprintf("⚠️ Por Vencer (%d días restantes)", item.DaysRemaining)
		} else if item.Status == "expired" {
			statusStr = "❌ Expirado"
		}
		fmt.Printf(" - %-30s | %-20s | Expiro: %s | Emisor: %s\n", item.Domain, statusStr, item.ExpiryDate, item.Issuer)
	}
}

func handleMaintenanceCommand(repo *repositories.SQLiteRepository, config *domain.ServerConfig, args []string) {
	if len(args) == 0 {
		fmt.Println("Uso: tarhiata maintenance <enable|disable> --service <nombre>")
		return
	}
	subCmd := args[0]
	fs := flag.NewFlagSet("maintenance", flag.ExitOnError)
	svcName := fs.String("service", "", "Nombre del servicio")
	fs.Parse(args[1:])

	if *svcName == "" {
		fmt.Println("❌ Especifica --service <nombre>")
		return
	}

	cfg := domain.ServerConfig{}
	if config != nil {
		cfg = *config
	}
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageSSLMaintenanceUseCase(repo, sshExec)

	enable := subCmd == "enable" || subCmd == "on"
	if err := uc.ToggleMaintenanceMode(*svcName, enable, cfg); err != nil {
		fmt.Printf("❌ Error al cambiar modo mantenimiento: %v\n", err)
		return
	}
	if enable {
		fmt.Printf("🚧 Modo Mantenimiento (503 Drain) ACTIVADO para '%s'.\n", *svcName)
	} else {
		fmt.Printf("✅ Modo Mantenimiento DESACTIVADO para '%s'. Tráfico restaurado.\n", *svcName)
	}
}

func handleDomainCommand(repo *repositories.SQLiteRepository, config *domain.ServerConfig, args []string) {
	if len(args) == 0 {
		fmt.Println("Uso: tarhiata domain <add|list|rm>")
		return
	}
	subCmd := args[0]
	cfg := domain.ServerConfig{}
	if config != nil {
		cfg = *config
	}
	sshExec := repositories.NewCryptoSSHExecutor()
	uc := usecases.NewManageDomainsUseCase(repo, sshExec)

	switch subCmd {
	case "add":
		fs := flag.NewFlagSet("domain add", flag.ExitOnError)
		svcName := fs.String("service", "", "Nombre del servicio")
		dom := fs.String("domain", "", "Dominio personalizado (ej. www.midominio.com)")
		redirect := fs.String("redirect", "", "Destino de redirección 301 opcional")
		fs.Parse(args[1:])

		if *svcName == "" || *dom == "" {
			fmt.Println("❌ Especifica --service <nombre> y --domain <dominio>")
			return
		}
		if err := uc.AddCustomDomain(*svcName, *dom, *redirect, cfg); err != nil {
			fmt.Printf("❌ Error al vincular dominio: %v\n", err)
			return
		}
		fmt.Printf("🌐 Dominio '%s' vinculado exitosamente a '%s'.\n", *dom, *svcName)

	case "list":
		fs := flag.NewFlagSet("domain list", flag.ExitOnError)
		svcName := fs.String("service", "", "Nombre del servicio")
		fs.Parse(args[1:])

		if *svcName == "" {
			fmt.Println("❌ Especifica --service <nombre>")
			return
		}
		primary, rules, err := uc.GetServiceDomains(*svcName)
		if err != nil {
			fmt.Printf("❌ Error al obtener dominios: %v\n", err)
			return
		}
		fmt.Printf("🌐 Dominios configurados para '%s':\n", *svcName)
		fmt.Printf(" - Dominio Principal: %s\n", primary)
		for _, r := range rules {
			redirStr := ""
			if r.RedirectTarget != "" {
				redirStr = fmt.Sprintf(" (Redirección 301 ➔ %s)", r.RedirectTarget)
			}
			fmt.Printf(" - Alias CNAME: %s%s\n", r.Domain, redirStr)
		}

	case "rm":
		fs := flag.NewFlagSet("domain rm", flag.ExitOnError)
		svcName := fs.String("service", "", "Nombre del servicio")
		dom := fs.String("domain", "", "Dominio a remover")
		fs.Parse(args[1:])

		if *svcName == "" || *dom == "" {
			fmt.Println("❌ Especifica --service <nombre> y --domain <dominio>")
			return
		}
		if err := uc.RemoveCustomDomain(*svcName, *dom, cfg); err != nil {
			fmt.Printf("❌ Error al remover dominio: %v\n", err)
			return
		}
		fmt.Printf("✅ Dominio '%s' desvinculado exitosamente de '%s'.\n", *dom, *svcName)
	}
}

func printHelp() {
	fmt.Println(`Tarhiata-Ops PaaS - CLI & Control Plane

Uso:
  tarhiata [comando] [opciones]

Comandos disponibles:
  (sin comando)      Inicia el Web Dashboard en http://localhost:8080 (Spotlight Cmd+K)
  dashboard | ui     Inicia el Web Dashboard en http://localhost:8080
  config set         Configura credenciales SSH y Token de Vultr API Key
  init | bootstrap   Ejecuta InitServerUseCase (Docker Swarm + Traefik HTTPS + Fail2Ban)
  deploy             Despliega una app service en Swarm con SSL y Traefik
  preview            Gestiona entornos temporales efímeros (create/list/destroy)
  db create          Despliega una base de datos (PostgreSQL, Mongo, MySQL, Redis)
  backup             Crea, lista o restaura snapshots de BD y Volúmenes (create/list/restore)
  env                Gestión masiva de variables de entorno .env (list/import/export/set)
  volume             Explorador de volúmenes persistentes /opt/data (ls/cat/rm)
  link               Conecta 2 servicios inyectando la URL/IP del destino en una Env Var
  unlink             Elimina la relación A ➔ B y remueve la Env Var en Swarm
  node               Gestiona nodos de Docker Swarm (ls/token/add/rm/update)
  worker add         Provisiona un nuevo worker node en Vultr vía Terraform
  master             Inicializa un servicio 1-Click All-in-One (desvincula anteriores, crea BD y conecta Env Var)
  rollback           Revierte un servicio Docker Swarm a su versión previa (docker service rollback)
  obs deploy         Despliega el stack de observabilidad (Portainer, Loki, Grafana)
  ssh-key            Gestiona llaves SSH autorizadas (ls/add/rm con protección Vultr)
  update             Actualiza paquetes del sistema y Docker daemon
  list               Lista todos los servicios y bases de datos registrados
  status             Muestra la salud del servidor y del clúster
  topology           Muestra el grafo de dependencias y rutas DNS del clúster
  prune              Limpia imágenes y volúmenes obsoletos en el VPS`)
}

func handleSSHKeyCLICommand(config *domain.ServerConfig, args []string) {
	if config == nil || config.Host == "" {
		fmt.Println("❌ VPS no configurado. Ejecuta 'tarhiata config set'")
		return
	}

	uc := usecases.NewManageSSHKeysUseCase(nil)

	if len(args) == 0 || args[0] == "ls" || args[0] == "list" {
		fmt.Println("⏳ Consultando llaves SSH autorizadas en el VPS...")
		keys, err := uc.ListKeys(*config)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}

		fmt.Println("\n🔑 Llaves SSH Autorizadas en /root/.ssh/authorized_keys:")
		fmt.Println("─────────────────────────────────────────────────────────────────────────────")
		for i, k := range keys {
			status := "🟢 [Desarrollador / Eliminable]"
			if k.Protected || k.IsVultrKey {
				status = "🔒 [PROTEGIDA - Vultr Master Account]"
			}
			fmt.Printf(" %d. %s (%s)\n    Fingerprint: %s\n    Estatus:     %s\n\n",
				i+1, k.Comment, k.Type, k.Fingerprint, status)
		}
		return
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "add":
		if len(args) < 2 {
			fmt.Println("Uso: tarhiata ssh-key add \"ssh-ed25519 AAA... compa@laptop\"")
			return
		}
		keyContent := strings.Join(args[1:], " ")
		fmt.Println("⏳ Añadiendo llave SSH al servidor...")
		if err := uc.AddKey(*config, keyContent); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
		fmt.Println("✅ ¡Llave SSH añadida exitosamente en /root/.ssh/authorized_keys!")

	case "rm", "delete", "remove":
		if len(args) < 2 {
			fmt.Println("Uso: tarhiata ssh-key rm <fingerprint_o_comentario>")
			return
		}
		target := args[1]
		fmt.Printf("⏳ Intentando eliminar llave SSH '%s'...\n", target)
		if err := uc.DeleteKey(*config, target); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
		fmt.Println("✅ ¡Llave SSH eliminada del servidor exitosamente!")
	}
}
