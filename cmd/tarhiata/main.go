package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/repositories"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/infrastructure/ssh"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/infrastructure/terraform"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/usecases"
	"github.com/charmbracelet/huh"
)

func main() {
	fmt.Println("🚀 Tarhiata-ops CLI - Tu PaaS Personalizado en VPS")

	// 1. Inicializar Base de Datos Local (SQLite)
	homeDir, _ := os.UserHomeDir()
	dbPath := filepath.Join(homeDir, ".config", "tarhiata", "config.db")

	repo, err := repositories.NewSQLiteRepository(dbPath)
	if err != nil {
		fmt.Printf("❌ Error crítico iniciando base de datos: %v\n", err)
		os.Exit(1)
	}
	defer repo.Close()

	// 2. Intentar cargar configuración actual
	serverConfig, err := repo.GetServerConfig()
	if err != nil {
		fmt.Printf("❌ Error leyendo configuración local: %v\n", err)
	}

	// 3. Menú Principal Infinito
	for {
		var action string

		statusStr := "🔴 Sin Configurar"
		if serverConfig != nil {
			statusStr = fmt.Sprintf("🟢 Conectado a %s", serverConfig.Host)
		}

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(fmt.Sprintf("¿Qué te gustaría hacer hoy? [%s]", statusStr)).
					Options(
						huh.NewOption("⚙️  Configurar Credenciales del Servidor", "config"),
						huh.NewOption("🚀 Bootstrapper (Inicializar VPS virgen con Swarm/Traefik)", "bootstrap"),
						huh.NewOption("📦 Desplegar un Servicio (Tipo PaaS)", "deploy"),
						huh.NewOption("🗄️  Administrar Bases de Datos", "db"),
						huh.NewOption("🛠️  Herramientas Avanzadas (Observabilidad, VPN)", "tools"),
						huh.NewOption("💻 Abrir Shell Interactivo (Terminal Remota)", "shell"),
						huh.NewOption("❌ Salir", "exit"),
					).
					Value(&action),
			),
		)

		if err := form.Run(); err != nil {
			fmt.Println("Cancelado por el usuario.")
			os.Exit(0)
		}

		switch action {
		case "config":
			serverConfig = handleConfig(repo, serverConfig)
		case "bootstrap":
			if serverConfig == nil {
				fmt.Println("⚠️  Primero debes configurar las credenciales del servidor.")
				continue
			}
			handleBootstrap(*serverConfig)
		case "deploy":
			if serverConfig == nil {
				fmt.Println("⚠️  Primero debes configurar las credenciales del servidor.")
				continue
			}
			handleServices(*serverConfig, repo)
		case "db":
			if serverConfig == nil {
				fmt.Println("⚠️  Primero debes configurar las credenciales del servidor.")
				continue
			}
			handleDatabases(*serverConfig, repo)
		case "tools":
			if serverConfig == nil {
				fmt.Println("⚠️  Primero debes configurar las credenciales del servidor.")
				continue
			}
			handleTools(*serverConfig)
		case "shell":
			if serverConfig == nil {
				fmt.Println("⚠️  Primero debes configurar las credenciales del servidor.")
				continue
			}
			handleShell(*serverConfig)
		case "exit":
			fmt.Println("\n¡Hasta luego Ninja! 🥷")
			os.Exit(0)
		}
	}
}

func handleConfig(repo *repositories.SQLiteRepository, current *domain.ServerConfig) *domain.ServerConfig {
	var configType string
	
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("¿Dónde alojaremos el motor de Tarhiata-ops?").
				Options(
					huh.NewOption("🔌 Tengo un servidor existente (Requiere IP y SSH)", "existing"),
					huh.NewOption("🐳 Crear un servidor desde cero (DigitalOcean / Vultr)", "new"),
				).Value(&configType),
		),
	).Run()

	if err != nil {
		fmt.Println("Cancelado.")
		return current
	}

	var host, user, key, doToken string
	var portStr string = "22"

	if current != nil {
		host = current.Host
		portStr = fmt.Sprintf("%d", current.Port)
		user = current.User
		key = current.PrivateKey
		doToken = current.DOAPIToken
	}

	if configType == "existing" {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("IP del Servidor (Host)").Value(&host),
				huh.NewInput().Title("Puerto SSH").Value(&portStr),
				huh.NewInput().Title("Usuario").Value(&user),
				huh.NewInput().Title("Ruta de la Llave Privada (ej. ~/.ssh/id_rsa)").Value(&key),
				huh.NewInput().Title("DigitalOcean API Token (Opcional, para BDs)").Value(&doToken),
			),
		)

		if err := form.Run(); err != nil {
			fmt.Println("Cancelado.")
			return current
		}
	} else {
		// Modo Terraform (Desde cero)
		var providerName string
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Proveedor Cloud").
					Options(
						huh.NewOption("DigitalOcean", "digitalocean"),
						huh.NewOption("Vultr", "vultr"),
					).Value(&providerName),
			),
		).Run(); err != nil {
			fmt.Println("Cancelado.")
			return current
		}

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title(fmt.Sprintf("%s API Token (Obligatorio)", strings.Title(providerName))).Value(&doToken).Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("El Token es obligatorio")
					}
					return nil
				}),
			),
		)
		if err := form.Run(); err != nil {
			fmt.Println("Cancelado.")
			return current
		}

		fmt.Printf("\n⏳ [Terraform] Construyendo el servidor maestro en %s. Esto tardará un poco...\n", providerName)
		workspace := filepath.Join(os.TempDir(), "tarhiata_tf_master")
		
		var provisioner ports.Provisioner
		var region string
		if providerName == "digitalocean" {
			provisioner = terraform.NewDigitalOceanProvisioner(workspace)
			region = "nyc1" // DigitalOcean Region
		} else {
			provisioner = terraform.NewVultrProvisioner(workspace)
			region = "ewr" // Vultr Region (New Jersey)
		}
		
		newIP, privKey, err := provisioner.ProvisionNode(doToken, "tarhiata-master", region)
		if err != nil {
			fmt.Printf("❌ Error provisionando el servidor: %v\n", err)
			return current
		}

		host = newIP
		user = "root" // Ubuntu DO Droplet default root
		portStr = "22"
		
		// Guardar llave privada localmente
		homeDir, _ := os.UserHomeDir()
		keyDir := filepath.Join(homeDir, ".ssh")
		os.MkdirAll(keyDir, 0700)
		key = filepath.Join(keyDir, "tarhiata_master_rsa")
		
		if err := os.WriteFile(key, []byte(privKey), 0600); err != nil {
			fmt.Printf("❌ Error guardando la llave privada: %v\n", err)
			return current
		}
		
		fmt.Printf("✅ Servidor maestro creado exitosamente en %s\n", newIP)
	}

	port, _ := strconv.Atoi(portStr)
	newConfig := domain.ServerConfig{
		Host:       host,
		Port:       port,
		User:       user,
		PrivateKey: key,
		DOAPIToken: doToken,
	}

	if err := repo.SaveServerConfig(newConfig); err != nil {
		fmt.Printf("❌ Error guardando configuración: %v\n", err)
		return current
	}

	fmt.Println("✅ ¡Configuración guardada exitosamente!")
	return &newConfig
}

func handleBootstrap(config domain.ServerConfig) {
	var installObs bool
	var acmeEmail string
	var installTS bool
	var tsAuthKey string
	var exposeObs bool
	
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("¿Deseas instalar Tailscale (VPN) para gestionar el servidor de forma privada?").
				Value(&installTS),
		),
	).Run()
	if err != nil {
		return
	}

	if installTS {
		exposeObs = false
		huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Auth Key de Tailscale (Opcional, déjalo vacío para loguearte mediante URL en los logs)").
					Value(&tsAuthKey),
				huh.NewConfirm().
					Title("¿Deseas instalar el Stack de Observabilidad? (Estará protegido y oculto dentro de la VPN)").
					Value(&installObs),
			),
		).Run()
	} else {
		// Regla estricta: Si no hay VPN, NO instalamos observabilidad en el Bootstrapper.
		// El usuario debe ir explícitamente a la sección de herramientas si la quiere pública.
		installObs = false
		exposeObs = false
	}

	huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Correo para Let's Encrypt (SSL Automático). Déjalo vacío si no usarás dominios públicos.").
				Value(&acmeEmail),
		),
	).Run()

	fmt.Println("\n⏳ Conectando al servidor para inicializar Bootstrapper...")
	sshExec := ssh.NewCryptoSSHExecutor()
	if err := sshExec.Connect(config); err != nil {
		fmt.Printf("❌ Error conectando por SSH: %v\n", err)
		return
	}
	defer sshExec.Close()

	bootstrapper := usecases.NewBootstrapperUseCase(sshExec)
	fmt.Println("🚀 Ejecutando inicialización (Docker, Swarm, Firewall, Traefik)...")

	if err := bootstrapper.InitServer(installObs, acmeEmail, installTS, tsAuthKey, exposeObs); err != nil {
		fmt.Printf("❌ Falló el Bootstrapper: %v\n", err)
		// No hacemos return porque si Tailscale falló por AuthKey vacío, 
		// necesitamos mostrarle el comando al usuario para que se loguee.
	}

	fmt.Println("✅ ¡Servidor inicializado y protegido con éxito!")
	if installTS {
		fmt.Println("🌐 NOTA: Tailscale fue instalado. Si no proveíste AuthKey o la conexión falló, abre la 'Consola SSH' en el menú y corre 'tailscale up' para autenticarte.")
	}
	if installObs {
		fmt.Printf("📊 Portainer (Dashboard) disponible en: http://[IP_DE_TAILSCALE]:9000\n")
		fmt.Printf("📝 Dozzle (Logs en vivo) disponible en: http://[IP_DE_TAILSCALE]:8888\n")
	}
}

func handleTools(config domain.ServerConfig) {
	var action string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("🛠️ Herramientas Adicionales").
				Options(
					huh.NewOption("📈 Instalar Observabilidad PÚBLICA (Portainer/Dozzle sin VPN - ⚠️ Inseguro)", "obs_public"),
					huh.NewOption("🔙 Volver", "back"),
				).
				Value(&action),
		),
	).Run()
	if err != nil || action == "back" {
		return
	}

	if action == "obs_public" {
		var confirm bool
		huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("⚠️ ¿Estás completamente seguro? Cualquiera con tu IP podrá ver el Login de Portainer y Dozzle").Value(&confirm))).Run()
		if !confirm {
			return
		}
		
		fmt.Println("\n⏳ Conectando al servidor...")
		sshExec := ssh.NewCryptoSSHExecutor()
		if err := sshExec.Connect(config); err != nil {
			fmt.Println("❌ Error SSH:", err)
			return
		}
		defer sshExec.Close()

		bootstrapper := usecases.NewBootstrapperUseCase(sshExec)
		fmt.Println("🚀 Desplegando Observabilidad Pública...")
		if err := bootstrapper.DeployObservability(true); err != nil {
			fmt.Println("❌ Error:", err)
		} else {
			fmt.Printf("✅ Observabilidad Pública Instalada exitosamente.\n")
			fmt.Printf("📊 Portainer: http://%s:9000\n", config.Host)
			fmt.Printf("📝 Dozzle: http://%s:8888\n", config.Host)
		}
	}
}

func handleShell(config domain.ServerConfig) {
	fmt.Println("\n💻 Abriendo túnel seguro interactivo (Escribe 'exit' para salir)...")
	sshExec := ssh.NewCryptoSSHExecutor()
	if err := sshExec.Connect(config); err != nil {
		fmt.Printf("❌ Error conectando por SSH: %v\n", err)
		return
	}
	defer sshExec.Close()

	if err := sshExec.InteractiveShell(); err != nil {
		fmt.Printf("\nSesión terminada: %v\n", err)
	}
}

func handleServices(config domain.ServerConfig, repo *repositories.SQLiteRepository) {
	fmt.Println("\n⏳ Conectando al clúster para sincronizar estado de servicios...")
	sshExec := ssh.NewCryptoSSHExecutor()
	if err := sshExec.Connect(config); err != nil {
		fmt.Printf("❌ Error conectando por SSH: %v\n", err)
		return
	}
	defer sshExec.Close()

	// 1. Obtener los que están corriendo en Swarm
	res, err := sshExec.RunCommand("docker stack ls --format '{{.Name}}'")
	runningStacks := make(map[string]bool)
	if err == nil && res.ExitCode == 0 {
		lines := strings.Split(strings.TrimSpace(res.Output), "\n")
		for _, line := range lines {
			if line != "" {
				runningStacks[line] = true
			}
		}
	}

	// 2. Obtener el catálogo guardado en SQLite
	savedServices, err := repo.GetServices()
	if err != nil {
		fmt.Printf("❌ Error leyendo catálogo local: %v\n", err)
		return
	}

	// 3. Construir Menú
	var selectedAction string
	options := []huh.Option[string]{
		huh.NewOption("➕ Agregar Servicio al Catálogo", "add_new"),
		huh.NewOption("🗺️  Ver Mapa de Interconexión (URLs)", "map"),
		huh.NewOption("🔗 Vincular Servicios Rápidamente", "global_link"),
	}

	for _, svc := range savedServices {
		statusIcon := "🔴"
		if runningStacks[svc.Name] {
			statusIcon = "🟢"
		}
		options = append(options, huh.NewOption(fmt.Sprintf("📦 %s %s", statusIcon, svc.Name), "manage_"+svc.Name))
	}
	options = append(options, huh.NewOption("🔙 Volver al Menú Principal", "back"))

	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Catálogo de Servicios").
				Options(options...).
				Value(&selectedAction),
		),
	).Run()

	if err != nil || selectedAction == "back" {
		return
	}

	if selectedAction == "add_new" {
		runAddServiceWizard(repo)
	} else if selectedAction == "map" {
		showNetworkMap(repo, config)
	} else if selectedAction == "global_link" {
		runGlobalLinkWizard(repo)
	} else {
		stackName := strings.TrimPrefix(selectedAction, "manage_")
		runManageServiceMenu(stackName, repo, sshExec)
	}
}

func runGlobalLinkWizard(repo *repositories.SQLiteRepository) {
	fmt.Println("\n🔗 --- ASISTENTE GLOBAL DE INTERCONEXIÓN ---")
	allSvc, _ := repo.GetServices()
	allDBs, _ := repo.GetDatabases()

	if len(allSvc) == 0 {
		fmt.Println("⚠️  No tienes servicios creados. Crea al menos un servicio origen primero.")
		return
	}

	var originOptions []huh.Option[string]
	for _, s := range allSvc {
		originOptions = append(originOptions, huh.NewOption(fmt.Sprintf("📦 %s", s.Name), s.Name))
	}

	var originName string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title("1. Selecciona el Servicio Origen (Quien recibirá la variable)").Options(originOptions...).Value(&originName),
		),
	).Run()
	if err != nil || originName == "" {
		return
	}

	// Obtener el servicio original para poder guardar su .env
	svc, err := repo.GetService(originName)
	if err != nil || svc == nil {
		fmt.Println("❌ Error leyendo el servicio origen.")
		return
	}

	var linkOptions []huh.Option[string]
	for _, s := range allSvc {
		val := fmt.Sprintf("%s:%d", s.Name, s.Port)
		label := fmt.Sprintf("🌐 Servicio: %s", s.Name)
		if s.Name == svc.Name {
			label += " (Auto-conexión)"
		}
		linkOptions = append(linkOptions, huh.NewOption(label, val))
	}
	for _, dbInfo := range allDBs {
		val := fmt.Sprintf("%s://admin:password@%s:%d/db", dbInfo.Engine, dbInfo.Name, dbInfo.InternalPort)
		linkOptions = append(linkOptions, huh.NewOption(fmt.Sprintf("🗄️ BD: %s (%s)", dbInfo.Name, dbInfo.Engine), val))
	}

	if len(linkOptions) == 0 {
		fmt.Println("⚠️  No hay otros objetivos disponibles.")
		return
	}

	var targetHost, protocol, envVarName string
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title("2. Selecciona el Destino (A quién se conectará)").Options(linkOptions...).Value(&targetHost),
			huh.NewSelect[string]().Title("3. Protocolo de conexión").Options(
				huh.NewOption("http:// (API REST)", "http://"),
				huh.NewOption("ws:// (WebSockets)", "ws://"),
				huh.NewOption("grpc:// (gRPC)", "grpc://"),
				huh.NewOption("tcp:// (Raw TCP)", "tcp://"),
				huh.NewOption("[Ninguno] - Solo inyectar host:puerto", ""),
				huh.NewOption("[Autodetectado] - Ignorar si es Base de Datos", ""),
			).Value(&protocol),
			huh.NewInput().Title("4. Nombre de la Variable (ej. API_URL, DATABASE_URL)").Value(&envVarName),
		),
	).Run()

	if err == nil && envVarName != "" {
		if svc.EnvFilePath == "" {
			svc.EnvFilePath = filepath.Join(os.TempDir(), "tarhiata_"+svc.Name+".env")
			repo.SaveService(*svc)
		}

		f, err := os.OpenFile(svc.EnvFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("❌ Error abriendo archivo .env: %v\n", err)
			return
		}
		defer f.Close()

		finalURL := fmt.Sprintf("%s%s", protocol, targetHost)
		if strings.Contains(targetHost, "://") {
			finalURL = targetHost
		}

		lineToAdd := fmt.Sprintf("\n%s=%s\n", envVarName, finalURL)
		if _, err := f.WriteString(lineToAdd); err != nil {
			fmt.Printf("❌ Error escribiendo archivo .env: %v\n", err)
		} else {
			fmt.Printf("✅ ¡Conexión establecida! %s ahora conoce a %s a través de la variable '%s'.\n", svc.Name, targetHost, envVarName)
			fmt.Println("👉 Recuerda entrar a Administrar el servicio origen y 'Desplegar / Actualizar' para aplicar los cambios al clúster.")
		}
	}
}

func showNetworkMap(repo *repositories.SQLiteRepository, config domain.ServerConfig) {
	services, _ := repo.GetServices()
	databases, _ := repo.GetDatabases()

	fmt.Println("\n\033[1;36m========================================================\033[0m")
	fmt.Println("\033[1;36m      🗺️   T A R H I A T A   T O P O L O G Y   M A P    \033[0m")
	fmt.Println("\033[1;36m========================================================\033[0m\n")

	for _, svc := range services {
		fmt.Printf("\033[1;32m🚀 SERVICIO: %s\033[0m\n", svc.Name)
		fmt.Printf(" ├─ 🔌 \033[33mDNS Interno\033[0m : http://%s:%d \033[90m(Visible en Swarm)\033[0m\n", svc.Name, svc.Port)
		
		if svc.Expose {
			if svc.Domain != "" {
				protocol := "http://"
				if svc.EnableSSL {
					protocol = "https://"
				}
				fmt.Printf(" ├─ 🌐 \033[34mRed Pública\033[0m : %s%s\n", protocol, svc.Domain)
			} else {
				fmt.Printf(" ├─ 🌐 \033[34mRed Pública\033[0m : http://%s/%s\n", config.Host, svc.Name)
			}
		} else {
			fmt.Printf(" ├─ 🔒 \033[31mRed Pública\033[0m : [ACCESO DENEGADO - Privado]\n")
		}

		var mounts []domain.ServiceMount
		if svc.MountsJSON != "" && svc.MountsJSON != "[]" {
			json.Unmarshal([]byte(svc.MountsJSON), &mounts)
			fmt.Printf(" ├─ 📁 \033[35mMounts\033[0m      : %d archivos inyectados\n", len(mounts))
		} else {
			fmt.Printf(" ├─ 📁 \033[35mMounts\033[0m      : [Ninguno]\n")
		}

		if svc.EnvFilePath != "" {
			fmt.Printf(" └─ 📝 \033[36mVariables\033[0m   : %s\n", svc.EnvFilePath)
			// Leer el archivo .env e imprimir las variables vinculadas
			content, err := os.ReadFile(svc.EnvFilePath)
			if err == nil {
				lines := strings.Split(string(content), "\n")
				for _, line := range lines {
					if strings.Contains(line, "=") {
						fmt.Printf("    │  └─ \033[90m%s\033[0m\n", strings.TrimSpace(line))
					}
				}
			}
		} else {
			fmt.Printf(" └─ 📝 \033[36mVariables\033[0m   : [Ninguno]\n")
		}
		fmt.Println()
	}

	for _, dbInfo := range databases {
		fmt.Printf("\033[1;34m🗄️  BASE DE DATOS: %s (%s)\033[0m\n", dbInfo.Name, dbInfo.Engine)
		fmt.Printf(" ├─ 🔌 \033[33mDNS Interno\033[0m : %s://admin:password@%s:%d/db\n", dbInfo.Engine, dbInfo.Name, dbInfo.InternalPort)
		fmt.Printf(" └─ 🔒 \033[31mRed Pública\033[0m : [ACCESO DENEGADO - Seguro por defecto]\n\n")
	}

	fmt.Println("\033[1;36m========================================================\033[0m")
	fmt.Println("\033[1;36m      🕸️   GRAFO DE DEPENDENCIAS (Interconexiones)    \033[0m")
	fmt.Println("\033[1;36m========================================================\033[0m\n")

	hasConnections := false
	for _, svc := range services {
		if svc.EnvFilePath != "" {
			content, err := os.ReadFile(svc.EnvFilePath)
			if err == nil {
				lines := strings.Split(string(content), "\n")
				for _, line := range lines {
					if strings.Contains(line, "=") {
						parts := strings.SplitN(line, "=", 2)
						val := parts[1]
						
						for _, otherSvc := range services {
							if otherSvc.Name != svc.Name && strings.Contains(val, otherSvc.Name) {
								fmt.Printf(" \033[1;32m[%s]\033[0m ────(\033[36m%s\033[0m)────▶ \033[1;32m[%s]\033[0m\n", svc.Name, parts[0], otherSvc.Name)
								hasConnections = true
							}
						}
						for _, dbInfo := range databases {
							if strings.Contains(val, dbInfo.Name) {
								fmt.Printf(" \033[1;32m[%s]\033[0m ────(\033[36m%s\033[0m)────▶ \033[1;34m[%s (BD)]\033[0m\n", svc.Name, parts[0], dbInfo.Name)
								hasConnections = true
							}
						}
					}
				}
			}
		}
	}
	
	if !hasConnections {
		fmt.Println(" \033[90mNingún servicio está interconectado mediante variables aún.\033[0m")
	}

	fmt.Println("\n\033[1;36m========================================================\033[0m")
	fmt.Println("\033[90mPresiona Enter para continuar...\033[0m")
	fmt.Scanln()
}

func runAddServiceWizard(repo *repositories.SQLiteRepository) {
	fmt.Println("\n📦 Agregando nuevo servicio al catálogo (Aún no se desplegará)...")

	var (
		serviceName string
		imageType   string
		imageSource string
		portStr     string = "80"
		isPublic    bool
		domainName  string
		enableSSL   bool
		envFilePath string
	)

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Nombre del Servicio (ej. api)").Value(&serviceName).Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return fmt.Errorf("El nombre no puede estar vacío")
				}
				return nil
			}),
			huh.NewSelect[string]().Title("Origen de la Imagen").
				Options(
					huh.NewOption("🐳 Docker Hub", "hub"),
					huh.NewOption("🔗 URL Directa (ZIP/TAR)", "url"),
				).Value(&imageType),
			huh.NewInput().Title("Nombre de Imagen o URL").Value(&imageSource).Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return fmt.Errorf("La imagen no puede estar vacía")
				}
				return nil
			}),
			huh.NewInput().Title("Puerto interno de tu app (ej. 3000)").Value(&portStr),
		),
	).Run()
	if err != nil {
		return
	}

	if err := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("¿Hacer este servicio accesible desde Internet?").Value(&isPublic))).Run(); err != nil {
		return
	}

	if isPublic {
		if err := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Dominio (Opcional, deja vacío para usar IP)").Value(&domainName))).Run(); err != nil {
			return
		}
		
		if domainName != "" {
			if err := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("¿Habilitar SSL Automático (HTTPS) para este dominio?").Value(&enableSSL))).Run(); err != nil {
				return
			}
		}
	}

	if err := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Ruta local de archivo .env (Opcional, vacío para crear después)").Value(&envFilePath))).Run(); err != nil {
		return
	}

	var healthcheckCmd string
	if err := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Comando Healthcheck (ej. curl -f http://localhost:3000 || exit 1) [Vacío para omitir]").Value(&healthcheckCmd))).Run(); err != nil {
		return
	}

	if envFilePath == "" {
		var createEnv bool
		if err := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("No proveíste un archivo. ¿Deseas abrir el editor para crearlo ahora?").Value(&createEnv))).Run(); err != nil {
			return
		}
		
		if createEnv {
			tempFile := filepath.Join(os.TempDir(), "tarhiata_"+serviceName+".env")
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "nano"
			}
			cmd := exec.Command(editor, tempFile)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err == nil {
				if _, statErr := os.Stat(tempFile); statErr == nil {
					envFilePath = tempFile
				}
			}
		}
	}

	port, _ := strconv.Atoi(portStr)
	newService := domain.SavedService{
		Name:           serviceName,
		ImageSource:    imageSource,
		IsURL:          imageType == "url",
		Port:           port,
		Domain:         domainName,
		Expose:         isPublic,
		EnvFilePath:    envFilePath,
		EnableSSL:      enableSSL,
		HealthcheckCmd: healthcheckCmd,
	}

	if err := repo.SaveService(newService); err != nil {
		fmt.Printf("❌ Error guardando servicio: %v\n", err)
		return
	}

	fmt.Printf("✅ ¡Servicio %s guardado en tu catálogo local! Ahora puedes seleccionarlo para desplegarlo.\n", serviceName)
}

func runManageServiceMenu(serviceName string, repo *repositories.SQLiteRepository, sshExec ports.SSHExecutor) {
	svc, err := repo.GetService(serviceName)
	if err != nil || svc == nil {
		fmt.Println("❌ No se encontró el servicio en la base de datos.")
		return
	}

	var action string
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(fmt.Sprintf("Administrando: %s", svc.Name)).
				Options(
					huh.NewOption("🚀 Desplegar / Actualizar ahora", "deploy"),
					huh.NewOption("🔄 Cambiar Imagen / Versión", "change_image"),
					huh.NewOption("🔧 Editar Configuración de Red (Puerto/Dominio)", "edit_network"),
					huh.NewOption("📁 Inyectar Archivo de Configuración (Mount)", "add_mount"),
					huh.NewOption("📊 Ver Logs en Vivo", "logs"),
					huh.NewOption("📝 Editar Variables de Entorno", "edit_env"),
					huh.NewOption("🔗 Vincular con otro Servicio / BD", "link_service"),
					huh.NewOption("🛑 Apagar (Eliminar de Swarm)", "stop"),
					huh.NewOption("🗑️ Eliminar del Catálogo Local", "delete"),
					huh.NewOption("🔙 Volver", "back"),
				).
				Value(&action),
		),
	).Run()

	if err != nil || action == "back" {
		return
	}

	switch action {
	case "logs":
		fmt.Printf("\n📊 Conectando a los logs de %s (Presiona Ctrl+C para salir)...\n", svc.Name)
		// Swarm nombra el servicio combinando <nombre_stack>_<nombre_servicio>
		cmd := fmt.Sprintf("docker service logs -f %s_%s", svc.Name, svc.Name)
		if err := sshExec.InteractiveCommand(cmd); err != nil {
			// Es normal que salga con error al hacer Ctrl+C
			fmt.Println("Desconectado de los logs.")
		}

	case "change_image":
		var newImage string
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(fmt.Sprintf("Imagen actual: %s\nIngresa la nueva imagen:tag (ej. nginx:latest)", svc.ImageSource)).
					Value(&newImage),
			),
		).Run()
		if err == nil && newImage != "" {
			svc.ImageSource = newImage
			repo.SaveService(*svc)
			fmt.Println("✅ Imagen actualizada localmente. Recuerda hacer un 'Desplegar / Actualizar' para aplicar los cambios.")
		}

	case "edit_network":
		portStr := fmt.Sprintf("%d", svc.Port)
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Puerto interno de tu app (ej. 3000)").Value(&portStr),
			),
		).Run()
		if err != nil {
			return
		}
		
		var isPublic bool = svc.Expose
		var domainName string = svc.Domain
		
		if err := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("¿Hacer este servicio accesible desde Internet?").Value(&isPublic))).Run(); err != nil {
			return
		}

		if isPublic {
			if err := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Dominio (Opcional, deja vacío para usar IP)").Value(&domainName))).Run(); err != nil {
				return
			}
		}

		svc.Port, _ = strconv.Atoi(portStr)
		svc.Expose = isPublic
		svc.Domain = domainName
		repo.SaveService(*svc)
		fmt.Println("✅ Configuración de red actualizada. Recuerda hacer un 'Desplegar / Actualizar' para aplicar los cambios.")

	case "edit_env":
		fmt.Printf("\n📝 Abriendo editor para variables de %s...\n", svc.Name)
		if svc.EnvFilePath == "" {
			svc.EnvFilePath = filepath.Join(os.TempDir(), "tarhiata_"+svc.Name+".env")
		}
		
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nano"
		}
		
		cmd := exec.Command(editor, svc.EnvFilePath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			if _, statErr := os.Stat(svc.EnvFilePath); statErr == nil {
				// Guardamos la nueva ruta por si antes no tenía
				repo.SaveService(*svc)
				fmt.Println("✅ Archivo .env guardado localmente. Recuerda hacer un 'Desplegar / Actualizar' para aplicar los cambios.")
			} else {
				fmt.Println("⚠️  No se guardó ningún archivo.")
			}
		} else {
			fmt.Println("❌ Error al abrir el editor.")
		}

	case "link_service":
		allSvc, _ := repo.GetServices()
		allDBs, _ := repo.GetDatabases()
		
		var linkOptions []huh.Option[string]
		for _, s := range allSvc {
			val := fmt.Sprintf("%s:%d", s.Name, s.Port)
			label := fmt.Sprintf("🌐 Servicio: %s", s.Name)
			if s.Name == svc.Name {
				label += " (Este mismo servicio)"
			}
			linkOptions = append(linkOptions, huh.NewOption(label, val))
		}
		for _, dbInfo := range allDBs {
			val := fmt.Sprintf("%s://admin:password@%s:%d/db", dbInfo.Engine, dbInfo.Name, dbInfo.InternalPort)
			linkOptions = append(linkOptions, huh.NewOption(fmt.Sprintf("🗄️ BD: %s (%s)", dbInfo.Name, dbInfo.Engine), val))
		}
		
		if len(linkOptions) == 0 {
			fmt.Println("⚠️  No hay otros servicios o bases de datos para vincular.")
			return
		}
		
		var targetHost, protocol, envVarName string
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().Title("Selecciona el objetivo a vincular").Options(linkOptions...).Value(&targetHost),
				huh.NewSelect[string]().Title("Protocolo de conexión (Para servicios)").Options(
					huh.NewOption("http:// (API REST)", "http://"),
					huh.NewOption("ws:// (WebSockets)", "ws://"),
					huh.NewOption("grpc:// (gRPC)", "grpc://"),
					huh.NewOption("tcp:// (Raw TCP)", "tcp://"),
					huh.NewOption("[Ninguno] - Solo inyectar host:puerto", ""),
					huh.NewOption("[Autodetectado] - Ignorar si elegiste una Base de Datos", ""),
				).Value(&protocol),
				huh.NewInput().Title("Nombre de la Variable (ej. API_URL, DATABASE_URL)").Value(&envVarName),
			),
		).Run()
		
		if err == nil && envVarName != "" {
			if svc.EnvFilePath == "" {
				svc.EnvFilePath = filepath.Join(os.TempDir(), "tarhiata_"+svc.Name+".env")
				repo.SaveService(*svc)
			}
			
			f, err := os.OpenFile(svc.EnvFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Printf("❌ Error abriendo archivo .env: %v\n", err)
				return
			}
			defer f.Close()
			
			// Si el targetHost ya contiene un protocolo (ej. BDs postgres://), no agregamos el seleccionado
			finalURL := fmt.Sprintf("%s%s", protocol, targetHost)
			if strings.Contains(targetHost, "://") {
				finalURL = targetHost
			}

			lineToAdd := fmt.Sprintf("\n%s=%s\n", envVarName, finalURL)
			if _, err := f.WriteString(lineToAdd); err != nil {
				fmt.Printf("❌ Error escribiendo archivo .env: %v\n", err)
			} else {
				fmt.Printf("✅ Variable '%s' autogenerada exitosamente. Recuerda 'Desplegar / Actualizar'.\n", envVarName)
			}
		}

	case "add_mount":
		var mounts []domain.ServiceMount
		if svc.MountsJSON != "" && svc.MountsJSON != "[]" {
			json.Unmarshal([]byte(svc.MountsJSON), &mounts)
		}

		fmt.Printf("\n📁 Archivos Inyectados Actualmente (%d):\n", len(mounts))
		for i, m := range mounts {
			fmt.Printf("  %d. %s -> %s\n", i+1, m.LocalPath, m.DestPath)
		}

		var mountAction string
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("¿Qué deseas hacer?").
					Options(
						huh.NewOption("➕ Añadir nuevo archivo", "add"),
						huh.NewOption("🗑️ Eliminar TODOS los archivos", "clear"),
						huh.NewOption("🔙 Volver", "back"),
					).Value(&mountAction),
			),
		).Run()
		if err != nil || mountAction == "back" {
			return
		}

		if mountAction == "clear" {
			svc.MountsJSON = "[]"
			repo.SaveService(*svc)
			fmt.Println("✅ Todos los archivos inyectados fueron eliminados. Recuerda Desplegar para aplicar.")
			return
		}

		var localPath, destPath string
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Ruta local del archivo (ej. /Users/diego/config.json)").Value(&localPath),
				huh.NewInput().Title("Ruta destino en el contenedor (ej. /app/config.json)").Value(&destPath),
			),
		).Run()
		
		if err == nil && localPath != "" && destPath != "" {
			mounts = append(mounts, domain.ServiceMount{
				LocalPath: localPath,
				DestPath:  destPath,
			})
			newJSON, _ := json.Marshal(mounts)
			svc.MountsJSON = string(newJSON)
			repo.SaveService(*svc)
			fmt.Printf("✅ Archivo %s agregado. Recuerda hacer un 'Desplegar / Actualizar' para montar el archivo.\n", localPath)
		}

	case "deploy":
		fmt.Printf("\n🚀 Desplegando %s en el clúster...\n", svc.Name)
		
		deployConfig := usecases.DeployConfig{
			ImageSource:    svc.ImageSource,
			IsURL:          svc.IsURL,
			Port:           svc.Port,
			Domain:         svc.Domain,
			Expose:         svc.Expose,
			EnableSSL:      svc.EnableSSL,
			HealthcheckCmd: svc.HealthcheckCmd,
		}

		customService := domain.CustomService{
			Name:    svc.Name,
			EnvVars: make(map[string]string),
		}

		if svc.EnvFilePath != "" {
			customService.Files = append(customService.Files, domain.ServiceFile{
				FileName:  ".env",
				LocalPath: svc.EnvFilePath,
			})
		}

		if svc.MountsJSON != "" && svc.MountsJSON != "[]" {
			var mounts []domain.ServiceMount
			json.Unmarshal([]byte(svc.MountsJSON), &mounts)
			customService.Mounts = mounts
		}

		deployer := usecases.NewDeployerUseCase(sshExec)
		if err := deployer.DeployService(customService, deployConfig); err != nil {
			fmt.Printf("❌ Falló el despliegue: %v\n", err)
			return
		}
		fmt.Printf("✅ ¡%s desplegado exitosamente!\n", svc.Name)

	case "stop":
		fmt.Printf("\n🛑 Apagando %s...\n", svc.Name)
		res, err := sshExec.RunCommand(fmt.Sprintf("docker stack rm %s", svc.Name))
		if err != nil || res.ExitCode != 0 {
			fmt.Printf("❌ Error apagando servicio: %v\n", res.Output)
		} else {
			fmt.Println("✅ Servicio apagado. Aún existe en tu catálogo local.")
		}

	case "delete":
		// Apagamos primero por si acaso
		sshExec.RunCommand(fmt.Sprintf("docker stack rm %s", svc.Name))
		if err := repo.DeleteService(svc.Name); err != nil {
			fmt.Printf("❌ Error eliminando del catálogo: %v\n", err)
		} else {
			fmt.Println("✅ Servicio eliminado del catálogo local.")
		}
	}
}

// --- Gestión de Bases de Datos ---

func handleDatabases(config domain.ServerConfig, repo *repositories.SQLiteRepository) {
	dbs, err := repo.GetDatabases()
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
		runAddDatabaseWizard(repo)
	} else {
		dbName := strings.TrimPrefix(selectedAction, "manage_")
		// runManageDatabaseMenu(dbName, repo, config) -> Para futura implementación del despliegue en sí
		fmt.Printf("\n🚧 Gestión del despliegue para %s en desarrollo (Siguiente iteración).\n", dbName)
	}
}

func runAddDatabaseWizard(repo *repositories.SQLiteRepository) {
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

	if err := repo.SaveDatabase(newDB); err != nil {
		fmt.Printf("❌ Error guardando BD: %v\n", err)
	} else {
		fmt.Printf("✅ Base de datos %s guardada exitosamente.\n", dbName)
	}
}
