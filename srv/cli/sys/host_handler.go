package sys

import (
	"fmt"
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
	defer sshExec.Close()

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
	defer sshExec.Close()

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
