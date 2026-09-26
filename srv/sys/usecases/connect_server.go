package usecases

import (
	"fmt"
	"strings"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

// ConnectServerUseCase implementa ports.ConnectServerUseCase.
type ConnectServerUseCase struct {
	executor ports.SSHExecutor
}

// NewConnectServerUseCase crea una nueva instancia del caso de uso.
func NewConnectServerUseCase(executor ports.SSHExecutor) ports.ConnectServerUseCase {
	return &ConnectServerUseCase{
		executor: executor,
	}
}

// Execute establece la conexión y recopila la telemetría inicial del host.
func (uc *ConnectServerUseCase) Execute(config domain.ServerConfig) (*domain.ConnectionResult, error) {
	start := time.Now()

	targetHost := strings.TrimSpace(config.Host)
	if config.IsLocal() && targetHost == "" {
		targetHost = "localhost"
	}

	name := strings.TrimSpace(config.Name)
	if name == "" {
		name = targetHost
	}

	result := &domain.ConnectionResult{
		Name:       name,
		IsLocal:    config.IsLocal(),
		TargetHost: targetHost,
	}

	if err := uc.executor.Connect(config); err != nil {
		result.Connected = false
		result.LatencyMs = time.Since(start).Milliseconds()
		result.Message = fmt.Sprintf("Error conectando a %s: %v", targetHost, err)
		result.Errors = []string{err.Error()}
		return result, nil
	}

	result.Connected = true

	// 1. Obtener Sistema Operativo y Arquitectura
	result.OS = "Unknown"
	osRes, err := uc.executor.RunCommand("uname -s -m")
	if err == nil && osRes != nil && osRes.ExitCode == 0 {
		result.OS = strings.TrimSpace(osRes.Output)
	}

	// 2. Verificar disponibilidad de Docker
	dockerVerRes, err := uc.executor.RunCommand("docker --version")
	if err == nil && dockerVerRes != nil && dockerVerRes.ExitCode == 0 {
		result.DockerActive = true
		result.DockerVersion = strings.TrimSpace(dockerVerRes.Output)
	}

	// 3. Verificar estado de Docker Swarm
	if result.DockerActive {
		swarmRes, errSwarm := uc.executor.RunCommand("docker info --format '{{.Swarm.LocalNodeState}}'")
		if errSwarm == nil && swarmRes != nil && strings.TrimSpace(swarmRes.Output) == "active" {
			result.SwarmActive = true
		}
	}

	result.LatencyMs = time.Since(start).Milliseconds()

	// 4. Construir mensaje de estado
	if !result.DockerActive {
		result.Message = fmt.Sprintf("Conectado exitosamente a %s (%s), pero Docker no se encuentra instalado o en ejecución.", targetHost, result.OS)
		return result, nil
	}

	if !result.SwarmActive {
		result.Message = fmt.Sprintf("Conectado a %s (%s). Docker operacional (%s). Swarm listo para inicializar.", targetHost, result.OS, result.DockerVersion)
		return result, nil
	}

	result.Message = fmt.Sprintf("Conectado a %s (%s). Clúster Docker Swarm activo y operacional.", targetHost, result.OS)
	return result, nil
}
