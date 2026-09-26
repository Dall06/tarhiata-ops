package sys

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
	"github.com/Dall06/tarhiata-ops/srv/sys/repositories"
	"github.com/Dall06/tarhiata-ops/srv/sys/usecases"
)

// HostHandler gestiona los comandos CLI relacionados con la inspección del host/VPS directo.
type HostHandler struct {
	repo ports.ConfigRepository
}

// NewHostHandler crea una nueva instancia de HostHandler.
func NewHostHandler(repo ports.ConfigRepository) *HostHandler {
	return &HostHandler{repo: repo}
}

// resolveTargetConfig obtiene la configuración del servidor por nombre o la activa.
func (h *HostHandler) resolveTargetConfig(serverName string) (*domain.ServerConfig, error) {
	name := strings.TrimSpace(serverName)
	if name != "" {
		cfg, err := h.repo.GetServerConfigByName(name)
		if err != nil {
			return nil, fmt.Errorf("error buscando servidor '%s': %w", name, err)
		}
		if cfg != nil {
			return cfg, nil
		}
		return nil, fmt.Errorf("servidor '%s' no encontrado en el catálogo", name)
	}

	cfg, err := h.repo.GetServerConfig()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo servidor activo: %w", err)
	}
	if cfg == nil {
		return nil, fmt.Errorf("no hay ningún servidor activo configurado")
	}
	return cfg, nil
}

// HandleMetrics ejecuta y muestra las métricas de hardware de la máquina.
func (h *HostHandler) HandleMetrics(serverName string) error {
	cfg, err := h.resolveTargetConfig(serverName)
	if err != nil {
		return err
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	defer func() {
		if clErr := sshExec.Close(); clErr != nil {
			slog.Warn("cli: error cerrando sshExec en host metrics", "error", clErr)
		}
	}()

	uc := usecases.NewInspectHostUseCase(sshExec)
	m, err := uc.ExecuteMetricsOnly(*cfg)
	if err != nil {
		return fmt.Errorf("fallo inspeccionando host: %w", err)
	}

	fmt.Printf("\n┌─ TELEMETRÍA DE HOST: %s (%s) ──────────────────────────┐\n", cfg.Name, cfg.Host)
	fmt.Printf("│ Sistema Operativo : %-50s │\n", m.OS)
	fmt.Printf("│ Hostname          : %-50s │\n", m.Hostname)
	fmt.Printf("│ CPU Núcleos / Uso : %d Cores · %-41s │\n", m.CPUCores, fmt.Sprintf("%.1f%% %s", m.CPUPercent, renderBar(m.CPUPercent)))
	fmt.Printf("│ Memoria RAM       : %-50s │\n", fmt.Sprintf("%d MB / %d MB (%.1f%%) %s", m.MemoryUsedMB, m.MemoryTotalMB, m.MemoryPercent, renderBar(m.MemoryPercent)))
	fmt.Printf("│ Espacio en Disco  : %-50s │\n", fmt.Sprintf("%.2f GB / %.2f GB (%.1f%%) %s", m.DiskUsedGB, m.DiskTotalGB, m.DiskPercent, renderBar(m.DiskPercent)))
	fmt.Printf("│ Carga Promedio    : %-50s │\n", m.LoadAvg)
	fmt.Printf("│ Uptime            : %-50s │\n", m.Uptime)
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")
	return nil
}

// HandleServices lista los servicios del sistema operativo en ejecución directa.
func (h *HostHandler) HandleServices(serverName string) error {
	cfg, err := h.resolveTargetConfig(serverName)
	if err != nil {
		return err
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	defer func() {
		if clErr := sshExec.Close(); clErr != nil {
			slog.Warn("cli: error cerrando sshExec en host services", "error", clErr)
		}
	}()

	uc := usecases.NewInspectHostUseCase(sshExec)
	services, err := uc.ExecuteServicesOnly(*cfg)
	if err != nil {
		return fmt.Errorf("fallo obteniendo servicios: %w", err)
	}

	fmt.Printf("\n┌─ SERVICIOS ACTIVOS DEL SISTEMA: %s (%d encontrados) ───────────────┐\n", cfg.Name, len(services))
	fmt.Printf("│ %-30s │ %-8s │ %-8s │ %-30s │\n", "SERVICIO", "ESTADO", "SUB", "DESCRIPCIÓN")
	fmt.Println("├────────────────────────────────+──────────+──────────+────────────────────────────────┤")

	for _, s := range services {
		name := s.Name
		if len(name) > 30 {
			name = name[:27] + "..."
		}
		desc := s.Description
		if len(desc) > 30 {
			desc = desc[:27] + "..."
		}
		fmt.Printf("│ %-30s │ %-8s │ %-8s │ %-30s │\n", name, s.ActiveState, s.SubState, desc)
	}
	fmt.Println("└────────────────────────────────┴──────────┴──────────┴────────────────────────────────┘")
	return nil
}

// HandleInspect ejecuta inspección completa (métricas + servicios).
func (h *HostHandler) HandleInspect(serverName string) error {
	if err := h.HandleMetrics(serverName); err != nil {
		return err
	}
	return h.HandleServices(serverName)
}

// HandleDevices lista los dispositivos de hardware conectados al VPS (almacenamiento, GPU, USB, pantallas, PCI).
func (h *HostHandler) HandleDevices(serverName string) error {
	cfg, err := h.resolveTargetConfig(serverName)
	if err != nil {
		return err
	}

	sshExec := repositories.NewCryptoSSHExecutor()
	defer func() {
		if clErr := sshExec.Close(); clErr != nil {
			slog.Warn("cli: error cerrando sshExec en host devices", "error", clErr)
		}
	}()

	uc := usecases.NewListDevicesUseCase(sshExec)
	devs, err := uc.Execute(*cfg)
	if err != nil {
		return fmt.Errorf("fallo inspeccionando dispositivos de hardware: %w", err)
	}

	fmt.Printf("\n┌─ HARDWARE Y DISPOSITIVOS: %s (%s) ──────────────────────────┐\n", cfg.Name, cfg.Host)
	fmt.Printf("│ Servidor : %-59s │\n", cfg.Name)
	fmt.Printf("│ Dirección: %-59s │\n", cfg.Host)
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────┘")

	// 1. Almacenamiento
	fmt.Printf("\n💾 ALMACENAMIENTO (%d unidades detectadas):\n", len(devs.Storage))
	if len(devs.Storage) == 0 {
		fmt.Println("   (No se detectaron unidades de almacenamiento)")
	}
	if len(devs.Storage) > 0 {
		fmt.Printf("   %-12s %-10s %-8s %-6s %-10s %-20s %s\n", "DISPOSITIVO", "TAMAÑO", "TIPO", "TECN.", "FORMATO", "MONTAJE", "MODELO")
		fmt.Println("   " + strings.Repeat("─", 80))
		for _, s := range devs.Storage {
			tech := "SSD"
			if s.Rotational {
				tech = "HDD"
			}
			mp := s.MountPoint
			if mp == "" {
				mp = "—"
			}
			fs := s.FSType
			if fs == "" {
				fs = "—"
			}
			model := s.Model
			if model == "" {
				model = "—"
			}
			fmt.Printf("   %-12s %-10s %-8s %-6s %-10s %-20s %s\n", s.Name, s.Size, s.Type, tech, fs, mp, model)
		}
	}

	// 2. Unidades GPU
	fmt.Printf("\n🎮 ACELERADORES GRÁFICOS / GPU (%d detectados):\n", len(devs.GPUs))
	if len(devs.GPUs) == 0 {
		fmt.Println("   (No se detectaron GPUs dedicadas o aceleradores)")
	}
	if len(devs.GPUs) > 0 {
		for i, g := range devs.GPUs {
			vram := g.MemoryTotal
			if vram == "" {
				vram = "Memoria compartida / No reportada"
			}
			pci := g.PCIAddress
			if pci == "" {
				pci = "Bus interno"
			}
			fmt.Printf("   [%d] %s (Fabricante: %s)\n", i+1, g.Model, g.Vendor)
			fmt.Printf("       Bus: %s | VRAM: %s\n", pci, vram)
		}
	}

	// 3. Dispositivos USB
	fmt.Printf("\n🔌 PERIFÉRICOS USB (%d detectados):\n", len(devs.USB))
	if len(devs.USB) == 0 {
		fmt.Println("   (No se detectaron periféricos USB)")
	}
	if len(devs.USB) > 0 {
		for _, u := range devs.USB {
			fmt.Printf("   • [Bus %s Dev %s] ID %s: %s\n", u.Bus, u.Device, u.ID, u.Description)
		}
	}

	// 4. Salidas de Pantalla / Video
	fmt.Printf("\n🖥️  SALIDAS DE VIDEO Y PANTALLAS (%d detectadas):\n", len(devs.Displays))
	if len(devs.Displays) == 0 {
		fmt.Println("   (No se detectaron salidas de video o pantallas conectadas)")
	}
	if len(devs.Displays) > 0 {
		for _, d := range devs.Displays {
			statusIcon := "🟢"
			if d.Status != "connected" {
				statusIcon = "⚪"
			}
			resInfo := ""
			if d.Resolution != "" {
				resInfo = fmt.Sprintf(" (%s)", d.Resolution)
			}
			fmt.Printf("   %s Conector: %-15s Estado: %-14s%s\n", statusIcon, d.Connector, d.Status, resInfo)
		}
	}

	// 5. Buses PCI Principales
	if len(devs.PCI) > 0 {
		fmt.Printf("\n🖧  BUSES Y CONTROLADORES PCI (%d componentes):\n", len(devs.PCI))
		for _, p := range devs.PCI {
			fmt.Printf("   • %-10s %-25s %s %s\n", p.Address, p.Class, p.Vendor, p.Device)
		}
	}

	fmt.Println()
	return nil
}

func renderBar(percent float64) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	totalSlots := 10
	filled := int(percent / 10.0)
	if filled > totalSlots {
		filled = totalSlots
	}
	bar := "["
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := filled; i < totalSlots; i++ {
		bar += "░"
	}
	bar += "]"
	return bar
}
