package usecases

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/cli/ports"
	sysdomain "github.com/Dall06/tarhiata-ops/srv/sys/domain"
	sysports "github.com/Dall06/tarhiata-ops/srv/sys/ports"
	sysrepositories "github.com/Dall06/tarhiata-ops/srv/sys/repositories"
	sysusecases "github.com/Dall06/tarhiata-ops/srv/sys/usecases"
	"github.com/charmbracelet/huh"
)

type serviceHandler struct {
	repo ports.ConfigRepository
}

// NewServiceHandler crea el caso de uso para administrar servicios en CLI.
func NewServiceHandler(repo ports.ConfigRepository) ports.ServiceHandler {
	return &serviceHandler{repo: repo}
}

func (h *serviceHandler) Execute(config sysdomain.ServerConfig) {
	fmt.Printf("\n⏳ Conectando al clúster para sincronizar estado de servicios...")
	sshExec := sysrepositories.NewCryptoSSHExecutor()
	if err := sshExec.Connect(config); err != nil {
		fmt.Printf("❌ Error conectando por SSH: %v\n", err)
		return
	}
	defer func() {
		if errClose := sshExec.Close(); errClose != nil {
			slog.Warn("cli: error cerrando ssh en service handler", "error", errClose)
		}
	}()

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

	savedServices, err := h.repo.GetServices()
	if err != nil {
		fmt.Printf("❌ Error leyendo catálogo local: %v\n", err)
		return
	}

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
		h.runAddServiceWizard()
		return
	}
	if selectedAction == "map" {
		h.showNetworkMap(config)
		return
	}
	if selectedAction == "global_link" {
		h.runGlobalLinkWizard()
		return
	}

	stackName := strings.TrimPrefix(selectedAction, "manage_")
	h.runManageServiceMenu(stackName, sshExec)
}

func (h *serviceHandler) runGlobalLinkWizard() {
	fmt.Printf("\n🔗 --- ASISTENTE GLOBAL DE INTERCONEXIÓN ---")
	allSvc, errSvc := h.repo.GetServices()
	if errSvc != nil {
		slog.Warn("service_handler: error obteniendo servicios", "error", errSvc)
	}
	allDBs, errDB := h.repo.GetDatabases()
	if errDB != nil {
		slog.Warn("service_handler: error obteniendo bases de datos", "error", errDB)
	}

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

	svc, err := h.repo.GetService(originName)
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
		matched, errMatch := regexp.MatchString(`^[A-Za-z0-9_]+$`, envVarName)
		if errMatch != nil || !matched {
			fmt.Println("❌ Nombre de variable inválido. Solo se permiten letras, números y guiones bajos.")
			return
		}
		if svc.EnvFilePath == "" {
			svc.EnvFilePath = getEnvPath(svc.Name)
			if errSave := h.repo.SaveService(*svc); errSave != nil {
				slog.Warn("fallo guardando servicio con env path", "service", svc.Name, "error", errSave)
			}
		}

		f, errFile := os.OpenFile(svc.EnvFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if errFile != nil {
			fmt.Printf("❌ Error abriendo archivo .env: %v\n", errFile)
			return
		}
		defer f.Close()

		finalURL := fmt.Sprintf("%s%s", protocol, targetHost)
		if strings.Contains(targetHost, "://") {
			finalURL = targetHost
		}

		lineToAdd := fmt.Sprintf("\n%s=%s\n", envVarName, finalURL)
		if _, errWrite := f.WriteString(lineToAdd); errWrite != nil {
			fmt.Printf("❌ Error escribiendo archivo .env: %v\n", errWrite)
			return
		}
		fmt.Printf("✅ ¡Conexión establecida! %s ahora conoce a %s a través de la variable '%s'.\n", svc.Name, targetHost, envVarName)
		fmt.Println("👉 Recuerda entrar a Administrar el servicio origen y 'Desplegar / Actualizar' para aplicar los cambios al clúster.")
	}
}

func (h *serviceHandler) showNetworkMap(config sysdomain.ServerConfig) {
	services, errSvc := h.repo.GetServices()
	if errSvc != nil {
		slog.Warn("topology: fallo leyendo servicios", "error", errSvc)
	}
	databases, errDB := h.repo.GetDatabases()
	if errDB != nil {
		slog.Warn("topology: fallo leyendo bases de datos", "error", errDB)
	}

	fmt.Printf("\n\033[1;36m========================================================\033[0m\n")
	fmt.Println("\033[1;36m      🗺️   T A R H I A T A   T O P O L O G Y   M A P    \033[0m")
	fmt.Println("\033[1;36m========================================================\033[0m")
	fmt.Println()

	for _, svc := range services {
		fmt.Printf("\033[1;32m🚀 SERVICIO: %s\033[0m\n", svc.Name)
		fmt.Printf(" ├─ 🔌 \033[33mDNS Interno\033[0m : http://%s:%d \033[90m(Visible en Swarm)\033[0m\n", svc.Name, svc.Port)

		if svc.Expose {
			protocol := "http://"
			if svc.EnableSSL {
				protocol = "https://"
			}
			if svc.Domain != "" {
				fmt.Printf(" ├─ 🌐 \033[34mRed Pública\033[0m : %s%s\n", protocol, svc.Domain)
			}
			if svc.Domain == "" {
				fmt.Printf(" ├─ 🌐 \033[34mRed Pública\033[0m : http://%s/%s\n", config.Host, svc.Name)
			}
		}
		if !svc.Expose {
			fmt.Printf(" ├─ 🔒 \033[31mRed Pública\033[0m : [ACCESO DENEGADO - Privado]\n")
		}

		var mounts []sysdomain.ServiceMount
		if svc.MountsJSON != "" && svc.MountsJSON != "[]" {
			if err := json.Unmarshal([]byte(svc.MountsJSON), &mounts); err != nil {
				slog.Debug("error decodificando mounts json", "error", err)
			}
			fmt.Printf(" ├─ 📁 \033[35mMounts\033[0m      : %d archivos inyectados\n", len(mounts))
		}
		if svc.MountsJSON == "" || svc.MountsJSON == "[]" {
			fmt.Printf(" ├─ 📁 \033[35mMounts\033[0m      : [Ninguno]\n")
		}

		if svc.EnvFilePath != "" {
			fmt.Printf(" └─ 📝 \033[36mVariables\033[0m   : %s\n", svc.EnvFilePath)
			content, err := os.ReadFile(svc.EnvFilePath)
			if err == nil {
				lines := strings.Split(string(content), "\n")
				for _, line := range lines {
					if strings.Contains(line, "=") {
						fmt.Printf("    │  └─ \033[90m%s\033[0m\n", strings.TrimSpace(line))
					}
				}
			}
		}
		if svc.EnvFilePath == "" {
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
	fmt.Println("\033[1;36m========================================================\033[0m")
	fmt.Println()

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

	fmt.Printf("\n\033[1;36m========================================================\033[0m\n")
	fmt.Println("\033[90mPresiona Enter para continuar...\033[0m")
	if _, errScan := fmt.Scanln(); errScan != nil {
		slog.Debug("scanln skipped", "error", errScan)
	}
}

func (h *serviceHandler) runAddServiceWizard() {
	fmt.Printf("\n📦 Agregando nuevo servicio al catálogo (Aún no se desplegará)...")

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
				s = strings.TrimSpace(s)
				if s == "" {
					return fmt.Errorf("el nombre no puede estar vacío")
				}
				matched, errMatch := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, s)
				if errMatch != nil || !matched {
					return fmt.Errorf("el nombre debe contener únicamente letras, números, guiones y guiones bajos")
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
					return fmt.Errorf("la imagen no puede estar vacía")
				}
				return nil
			}),
			huh.NewInput().Title("Puerto interno de tu app (ej. 3000)").Value(&portStr),
		),
	).Run()
	if err != nil {
		return
	}

	if errConfirm := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("¿Hacer este servicio accesible desde Internet?").Value(&isPublic))).Run(); errConfirm != nil {
		return
	}

	if isPublic {
		if errDomain := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Dominio (Opcional)\n⚠️ Recomendado si tu app usa rutas absolutas y no soporta base-paths.").Value(&domainName))).Run(); errDomain != nil {
			return
		}

		if domainName != "" {
			if errSSL := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("¿Habilitar SSL Automático (HTTPS) para este dominio?").Value(&enableSSL))).Run(); errSSL != nil {
				return
			}
		}
	}

	if errEnv := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Ruta local de archivo .env (Opcional, vacío para crear después)").Value(&envFilePath))).Run(); errEnv != nil {
		return
	}

	var healthcheckCmd string
	if errHealth := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Comando Healthcheck (ej. curl -f http://localhost:3000 || exit 1) [Vacío para omitir]").Value(&healthcheckCmd))).Run(); errHealth != nil {
		return
	}

	if envFilePath == "" {
		var createEnv bool
		if errPrompt := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("No proveíste un archivo. ¿Deseas abrir el editor para crearlo ahora?").Value(&createEnv))).Run(); errPrompt != nil {
			return
		}

		if createEnv {
			tempFile := getEnvPath(serviceName)
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "nano"
			}
			editorParts := strings.Fields(editor)
			editorParts = append(editorParts, tempFile)
			cmd := exec.Command(editorParts[0], editorParts[1:]...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if errExec := cmd.Run(); errExec == nil {
				if _, statErr := os.Stat(tempFile); statErr == nil {
					envFilePath = tempFile
				}
			}
		}
	}

	port, errPort := strconv.Atoi(portStr)
	if errPort != nil || port <= 0 {
		port = 80
	}
	newService := sysdomain.SavedService{
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

	if errSave := h.repo.SaveService(newService); errSave != nil {
		fmt.Printf("❌ Error guardando servicio: %v\n", errSave)
		return
	}

	fmt.Printf("✅ ¡Servicio %s guardado en tu catálogo local! Ahora puedes seleccionarlo para desplegarlo.\n", serviceName)
}

func getEnvPath(serviceName string) string {
	home, errHome := os.UserHomeDir()
	if errHome != nil {
		home = os.TempDir()
	}
	envDir := filepath.Join(home, ".config", "tarhiata", "envs")
	if errMk := os.MkdirAll(envDir, 0700); errMk != nil {
		slog.Warn("fallo creando directorio de envs", "dir", envDir, "error", errMk)
	}
	return filepath.Join(envDir, serviceName+".env")
}

func (h *serviceHandler) runManageServiceMenu(serviceName string, sshExec sysports.SSHExecutor) {
	svc, err := h.repo.GetService(serviceName)
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
		cmd := fmt.Sprintf("docker service logs -f %s_%s", svc.Name, svc.Name)
		if errLogs := sshExec.InteractiveCommand(cmd); errLogs != nil {
			fmt.Println("Desconectado de los logs.")
		}

	case "change_image":
		var newImage string
		errForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(fmt.Sprintf("Imagen actual: %s\nIngresa la nueva imagen:tag (ej. node:18-alpine)", svc.ImageSource)).
					Value(&newImage),
			),
		).Run()
		if errForm == nil && newImage != "" {
			svc.ImageSource = newImage
			if errSave := h.repo.SaveService(*svc); errSave != nil {
				fmt.Printf("❌ Error guardando servicio: %v\n", errSave)
				return
			}
			fmt.Println("✅ Imagen actualizada localmente. Recuerda hacer un 'Desplegar / Actualizar' para aplicar los cambios.")
		}

	case "edit_network":
		portStr := fmt.Sprintf("%d", svc.Port)
		errForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Puerto interno de tu app (ej. 3000)").Value(&portStr).Validate(func(s string) error {
					if p, errVal := strconv.Atoi(s); errVal != nil || p <= 0 {
						return fmt.Errorf("puerto inválido. Debe ser un número mayor a 0")
					}
					return nil
				}),
			),
		).Run()
		if errForm != nil {
			return
		}

		var isPublic bool = svc.Expose
		var domainName string = svc.Domain

		if errPublic := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("¿Hacer este servicio accesible desde Internet?").Value(&isPublic))).Run(); errPublic != nil {
			return
		}

		if isPublic {
			if errDomain := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Dominio (Opcional)\n⚠️ Recomendado si tu app usa rutas absolutas y no soporta base-paths.").Value(&domainName))).Run(); errDomain != nil {
				return
			}
		}

		parsedPort, errPort := strconv.Atoi(portStr)
		if errPort != nil || parsedPort <= 0 {
			parsedPort = 80
		}
		svc.Port = parsedPort
		svc.Expose = isPublic
		svc.Domain = domainName
		if errSave := h.repo.SaveService(*svc); errSave != nil {
			fmt.Printf("❌ Error guardando servicio: %v\n", errSave)
			return
		}
		fmt.Println("✅ Configuración de red actualizada. Recuerda hacer un 'Desplegar / Actualizar' para aplicar los cambios.")

	case "edit_env":
		fmt.Printf("\n📝 Abriendo editor para variables de %s...\n", svc.Name)
		if svc.EnvFilePath == "" {
			svc.EnvFilePath = getEnvPath(svc.Name)
		}

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nano"
		}
		editorParts := strings.Fields(editor)
		editorParts = append(editorParts, svc.EnvFilePath)
		cmd := exec.Command(editorParts[0], editorParts[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if errRun := cmd.Run(); errRun == nil {
			if _, statErr := os.Stat(svc.EnvFilePath); statErr == nil {
				if errSave := h.repo.SaveService(*svc); errSave != nil {
					slog.Warn("falló al guardar servicio tras editar env", "error", errSave)
				}
				fmt.Println("✅ Archivo .env guardado localmente. Recuerda hacer un 'Desplegar / Actualizar' para aplicar los cambios.")
				return
			}
			fmt.Println("⚠️  No se guardó ningún archivo.")
			return
		}
		fmt.Println("❌ Error al abrir el editor.")

	case "link_service":
		allSvc, errSvc := h.repo.GetServices()
		if errSvc != nil {
			slog.Warn("link_service: fallo leyendo servicios", "error", errSvc)
		}
		allDBs, errDB := h.repo.GetDatabases()
		if errDB != nil {
			slog.Warn("link_service: fallo leyendo bases de datos", "error", errDB)
		}

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
		errForm := huh.NewForm(
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

		if errForm == nil && envVarName != "" {
			matched, errMatch := regexp.MatchString(`^[A-Za-z0-9_]+$`, envVarName)
			if errMatch != nil || !matched {
				fmt.Println("❌ Nombre de variable inválido. Solo se permiten letras, números y guiones bajos.")
				return
			}
			if svc.EnvFilePath == "" {
				svc.EnvFilePath = getEnvPath(svc.Name)
				if errSave := h.repo.SaveService(*svc); errSave != nil {
					slog.Warn("fallo guardando servicio con env path", "error", errSave)
				}
			}

			finalURL := fmt.Sprintf("%s%s", protocol, targetHost)
			if strings.Contains(targetHost, "://") {
				finalURL = targetHost
			}

			content, errRead := os.ReadFile(svc.EnvFilePath)
			if errRead != nil && !os.IsNotExist(errRead) {
				slog.Warn("aviso al leer archivo env", "path", svc.EnvFilePath, "error", errRead)
			}
			lines := strings.Split(string(content), "\n")
			var newLines []string
			found := false
			prefix := envVarName + "="
			for _, line := range lines {
				if strings.HasPrefix(line, prefix) {
					newLines = append(newLines, prefix+finalURL)
					found = true
				}
				if !strings.HasPrefix(line, prefix) && strings.TrimSpace(line) != "" {
					newLines = append(newLines, line)
				}
			}
			if !found {
				newLines = append(newLines, prefix+finalURL)
			}

			if errWrite := os.WriteFile(svc.EnvFilePath, []byte(strings.Join(newLines, "\n")+"\n"), 0644); errWrite != nil {
				fmt.Printf("❌ Error guardando archivo .env: %v\n", errWrite)
				return
			}
			fmt.Printf("✅ Variable '%s' vinculada exitosamente.\n", envVarName)
			var redeploy bool
			if errPrompt := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("🔄 ¿Deseas redesplegar el servicio ahora para aplicar los cambios?").Value(&redeploy))).Run(); errPrompt != nil {
				return
			}
			if redeploy {
				fmt.Printf("\n🚀 Redesplegando %s en el clúster...\n", svc.Name)
				deployConfig := sysdomain.DeployConfig{ImageSource: svc.ImageSource, IsURL: svc.IsURL, Port: svc.Port, Domain: svc.Domain, Expose: svc.Expose, EnableSSL: svc.EnableSSL, HealthcheckCmd: svc.HealthcheckCmd}
				customService := sysdomain.CustomService{Name: svc.Name, EnvVars: make(map[string]string)}
				if svc.EnvFilePath != "" {
					customService.Files = append(customService.Files, sysdomain.ServiceFile{FileName: ".env", LocalPath: svc.EnvFilePath})
				}
				if svc.MountsJSON != "" && svc.MountsJSON != "[]" {
					var mounts []sysdomain.ServiceMount
					if errDec := json.Unmarshal([]byte(svc.MountsJSON), &mounts); errDec != nil {
						slog.Debug("error decodificando mounts json al redesplegar", "error", errDec)
					}
					customService.Mounts = mounts
				}
				if errDeploy := sysusecases.NewDeployServiceUseCase(sshExec).Execute(customService, deployConfig); errDeploy != nil {
					fmt.Printf("❌ Falló el despliegue: %v\n", errDeploy)
					return
				}
				fmt.Printf("✅ ¡%s redesplegado exitosamente!\n", svc.Name)
			}
		}

	case "add_mount":
		var mounts []sysdomain.ServiceMount
		if svc.MountsJSON != "" && svc.MountsJSON != "[]" {
			if errMounts := json.Unmarshal([]byte(svc.MountsJSON), &mounts); errMounts != nil {
				slog.Warn("fallo deserializando montajes", "error", errMounts)
			}
		}

		fmt.Printf("\n📁 Archivos Inyectados Actualmente (%d):\n", len(mounts))
		for i, m := range mounts {
			fmt.Printf("  %d. %s -> %s\n", i+1, m.LocalPath, m.DestPath)
		}

		var mountAction string
		errForm := huh.NewForm(
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
		if errForm != nil || mountAction == "back" {
			return
		}

		if mountAction == "clear" {
			svc.MountsJSON = "[]"
			if errSave := h.repo.SaveService(*svc); errSave != nil {
				slog.Warn("fallo guardando servicio tras limpiar montajes", "error", errSave)
			}

			if _, errRm := sshExec.RunCommand(fmt.Sprintf("rm -rf /opt/tarhiata/services/%s/configs", svc.Name)); errRm != nil {
				slog.Warn("fallo al limpiar configs en host", "error", errRm)
			}

			fmt.Println("✅ Todos los archivos inyectados fueron eliminados física y lógicamente.")
			return
		}

		var localPath, destPath string
		errForm = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Ruta local del archivo (ej. /Users/diego/config.json)").Value(&localPath),
				huh.NewInput().Title("Ruta destino en el contenedor (ej. /app/config.json)").Value(&destPath),
			),
		).Run()

		if errForm == nil && localPath != "" && destPath != "" {
			mounts = append(mounts, sysdomain.ServiceMount{
				LocalPath: localPath,
				DestPath:  destPath,
			})
			newJSON, errMarshal := json.Marshal(mounts)
			if errMarshal != nil {
				fmt.Printf("❌ Error serializando montajes: %v\n", errMarshal)
				return
			}
			svc.MountsJSON = string(newJSON)
			if errSave := h.repo.SaveService(*svc); errSave != nil {
				slog.Warn("fallo guardando servicio con montajes", "error", errSave)
			}
			fmt.Printf("✅ Archivo %s agregado. Recuerda hacer un 'Desplegar / Actualizar' para montar el archivo.\n", localPath)
		}

	case "deploy":
		fmt.Printf("\n🚀 Desplegando %s en el clúster...\n", svc.Name)

		deployConfig := sysdomain.DeployConfig{
			ImageSource:    svc.ImageSource,
			IsURL:          svc.IsURL,
			Port:           svc.Port,
			Domain:         svc.Domain,
			Expose:         svc.Expose,
			EnableSSL:      svc.EnableSSL,
			HealthcheckCmd: svc.HealthcheckCmd,
		}

		customService := sysdomain.CustomService{
			Name:    svc.Name,
			EnvVars: make(map[string]string),
		}

		if svc.EnvFilePath != "" {
			customService.Files = append(customService.Files, sysdomain.ServiceFile{
				FileName:  ".env",
				LocalPath: svc.EnvFilePath,
			})
		}

		if svc.MountsJSON != "" && svc.MountsJSON != "[]" {
			var mounts []sysdomain.ServiceMount
			if errU := json.Unmarshal([]byte(svc.MountsJSON), &mounts); errU != nil {
				slog.Warn("fallo deserializando montajes", "error", errU)
			}
			customService.Mounts = mounts
		}

		deployer := sysusecases.NewDeployServiceUseCase(sshExec)
		if errDeploy := deployer.Execute(customService, deployConfig); errDeploy != nil {
			fmt.Printf("❌ Falló el despliegue: %v\n", errDeploy)
			return
		}
		fmt.Printf("✅ ¡%s desplegado exitosamente!\n", svc.Name)

	case "stop":
		fmt.Printf("\n🛑 Apagando %s...\n", svc.Name)
		res, errCmd := sshExec.RunCommand(fmt.Sprintf("docker stack rm %s", svc.Name))
		if errCmd != nil || res == nil || res.ExitCode != 0 {
			out := ""
			if res != nil {
				out = res.Output
			}
			if out == "" && errCmd != nil {
				out = errCmd.Error()
			}
			fmt.Printf("❌ Error apagando servicio: %s\n", out)
			return
		}
		fmt.Println("✅ Servicio apagado. Aún existe en tu catálogo local.")

	case "delete":
		if _, errRm := sshExec.RunCommand(fmt.Sprintf("docker stack rm %s", svc.Name)); errRm != nil {
			slog.Debug("aviso al apagar stack al borrar", "error", errRm)
		}
		if _, errRmDir := sshExec.RunCommand(fmt.Sprintf("rm -rf /opt/tarhiata/services/%s", svc.Name)); errRmDir != nil {
			slog.Debug("aviso al limpiar directorio de servicio", "error", errRmDir)
		}
		if errDel := h.repo.DeleteService(svc.Name); errDel != nil {
			fmt.Printf("❌ Error eliminando del catálogo: %v\n", errDel)
			return
		}
		fmt.Println("✅ Servicio eliminado del catálogo local.")
	}
}
