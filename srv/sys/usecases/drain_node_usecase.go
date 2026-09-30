package usecases

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

// DrainNodeResult reporta el estado de disponibilidad y las tareas activas tras una operación en un nodo.
type DrainNodeResult struct {
	NodeID       string `json:"nodeId"`
	Hostname     string `json:"hostname"`
	Availability string `json:"availability"` // "active", "drain", "pause"
	RunningTasks int    `json:"runningTasks"`
	Message      string `json:"message"`
}

// DrainNodeUseCase gestiona el drenado seguro de tareas y mantenimiento de nodos en Docker Swarm.
type DrainNodeUseCase struct {
	executor ports.SSHExecutor
}

func NewDrainNodeUseCase(executor ports.SSHExecutor) *DrainNodeUseCase {
	return &DrainNodeUseCase{executor: executor}
}

// Execute cambia la disponibilidad de un nodo Swarm y audita las tareas en migración.
func (uc *DrainNodeUseCase) Execute(nodeID, availability string, config domain.ServerConfig) (*DrainNodeResult, error) {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return nil, errors.New("identificador de nodo requerido")
	}

	avail := strings.ToLower(strings.TrimSpace(availability))
	if avail != "active" && avail != "drain" && avail != "pause" {
		return nil, fmt.Errorf("disponibilidad inválida: %s (debe ser active, drain o pause)", availability)
	}

	if err := uc.executor.Connect(config); err != nil {
		return nil, fmt.Errorf("error conectando SSH: %w", err)
	}
	defer func() {
		if clErr := uc.executor.Close(); clErr != nil {
			slog.Warn("drain_node: error cerrando conexión SSH", "error", clErr)
		}
	}()

	// 1. Actualizar disponibilidad del nodo
	updateCmd := fmt.Sprintf("docker node update --availability %s %s", avail, nodeID)
	cmdRes, err := uc.executor.RunCommand(updateCmd)
	if err != nil {
		return nil, fmt.Errorf("error ejecutando cambio de disponibilidad: %w", err)
	}
	if cmdRes != nil && cmdRes.ExitCode != 0 {
		return nil, fmt.Errorf("docker node update falló: %s", cmdRes.Output)
	}

	// 2. Consultar tareas en ejecución restantes en el nodo
	psCmd := fmt.Sprintf("docker node ps %s --filter desired-state=running --format '{{.ID}}' 2>/dev/null | grep -v '^$' | wc -l", nodeID)
	psRes, err := uc.executor.RunCommand(psCmd)
	runningTasks := 0
	if err == nil && psRes != nil {
		countStr := strings.TrimSpace(psRes.Output)
		if count, errConv := strconv.Atoi(countStr); errConv == nil {
			runningTasks = count
		}
	}

	msg := fmt.Sprintf("Disponibilidad del nodo %s cambiada a %s", nodeID, avail)
	if avail == "drain" {
		if runningTasks == 0 {
			msg = fmt.Sprintf("Nodo %s drenado completamente. Listo para mantenimiento.", nodeID)
		}
		if runningTasks > 0 {
			msg = fmt.Sprintf("Nodo %s en proceso de drenado (%d tareas migrando).", nodeID, runningTasks)
		}
	}

	return &DrainNodeResult{
		NodeID:       nodeID,
		Availability: avail,
		RunningTasks: runningTasks,
		Message:      msg,
	}, nil
}
