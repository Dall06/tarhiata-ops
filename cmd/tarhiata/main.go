package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/handlers"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/repositories"
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
			serverConfig = handlers.NewConfigHandler(repo).Execute(serverConfig)
		case "bootstrap":
			if serverConfig == nil {
				fmt.Println("⚠️  Primero debes configurar las credenciales del servidor.")
				continue
			}
			handlers.NewBootstrapHandler(repo).Execute(*serverConfig)
		case "deploy":
			if serverConfig == nil {
				fmt.Println("⚠️  Primero debes configurar las credenciales del servidor.")
				continue
			}
			handlers.NewServiceHandler(repo).Execute(*serverConfig)
		case "db":
			if serverConfig == nil {
				fmt.Println("⚠️  Primero debes configurar las credenciales del servidor.")
				continue
			}
			handlers.NewDatabaseHandler(repo).Execute(*serverConfig)
		case "tools":
			if serverConfig == nil {
				fmt.Println("⚠️  Primero debes configurar las credenciales del servidor.")
				continue
			}
			handlers.NewToolHandler(repo).Execute(*serverConfig)
		case "shell":
			if serverConfig == nil {
				fmt.Println("⚠️  Primero debes configurar las credenciales del servidor.")
				continue
			}
			handlers.NewShellHandler(repo).Execute(*serverConfig)
		case "exit":
			fmt.Println("\n¡Hasta luego Ninja! 🥷")
			os.Exit(0)
		}
	}
}
