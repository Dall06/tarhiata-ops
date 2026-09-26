package usecases

import (
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

// GetSwarmStatusUseCase consulta el estado en vivo de Docker Swarm, sus servicios y nodos.
type GetSwarmStatusUseCase struct {
	executor ports.SSHExecutor
}

// NewGetSwarmStatusUseCase crea una nueva instancia del caso de uso.
func NewGetSwarmStatusUseCase(executor ports.SSHExecutor) *GetSwarmStatusUseCase {
	return &GetSwarmStatusUseCase{
		executor: executor,
	}
}

// Execute recopila el estado del cluster Swarm, la lista de servicios y nodos conectados.
func (uc *GetSwarmStatusUseCase) Execute(config domain.ServerConfig) (*domain.SwarmStatus, error) {
	if err := uc.executor.Connect(config); err != nil {
		return nil, fmt.Errorf("error conectando a %s: %w", config.Host, err)
	}

	status := &domain.SwarmStatus{
		Services:   []domain.SwarmServiceInfo{},
		Nodes:      []domain.SwarmNodeInfo{},
		Dashboards: make(map[string]string),
	}

	// 1. Verificar versión de Docker
	resDocker, err := uc.executor.RunCommand("docker --version")
	if err != nil || resDocker == nil || resDocker.ExitCode != 0 {
		status.Active = false
		status.NodeState = "inactive"
		return status, nil
	}
	status.DockerVersion = strings.TrimSpace(resDocker.Output)

	// 2. Verificar estado de Swarm
	resSwarm, err := uc.executor.RunCommand("docker info --format '{{.Swarm.LocalNodeState}}'")
	if err != nil || resSwarm == nil || strings.TrimSpace(resSwarm.Output) != "active" {
		status.Active = false
		status.NodeState = "inactive"
		return status, nil
	}
	status.Active = true
	status.NodeState = "active"

	// 3. Listar servicios de Swarm
	resServices, err := uc.executor.RunCommand("docker service ls --format '{{.ID}}\t{{.Name}}\t{{.Mode}}\t{{.Replicas}}\t{{.Image}}\t{{.Ports}}'")
	if err == nil && resServices != nil && resServices.ExitCode == 0 {
		status.Services = ParseSwarmServices(resServices.Output)
		// Consultar etiquetas de Traefik para saber si son públicos/privados y su dominio
		resInspect, errInsp := uc.executor.RunCommand("docker service inspect $(docker service ls -q) --format '{{.Spec.Name}}\t{{index .Spec.Labels \"traefik.enable\"}}\t{{index .Spec.Labels (printf \"traefik.http.routers.%s.rule\" .Spec.Name)}}' 2>/dev/null")
		if errInsp == nil && resInspect != nil && resInspect.ExitCode == 0 {
			enrichSwarmServicesWithInspect(status.Services, resInspect.Output)
		}
	}

	// 4. Listar nodos de Swarm
	resNodes, err := uc.executor.RunCommand("docker node ls --format '{{.ID}}\t{{.Hostname}}\t{{.Status}}\t{{.Availability}}\t{{.ManagerStatus}}\t{{.EngineVersion}}'")
	if err == nil && resNodes != nil && resNodes.ExitCode == 0 {
		status.Nodes = ParseSwarmNodes(resNodes.Output)
	}

	// 5. Configurar URLs de Dashboards
	targetHost := strings.TrimSpace(config.Host)
	if config.IsLocal() || targetHost == "" {
		targetHost = "localhost"
	}
	status.Dashboards["portainer"] = fmt.Sprintf("http://%s:9000", targetHost)
	status.Dashboards["dozzle"] = fmt.Sprintf("http://%s:8888", targetHost)
	status.Dashboards["traefik"] = fmt.Sprintf("http://%s:8080", targetHost)

	return status, nil
}

// enrichSwarmServicesWithInspect añade datos de exposición pública (Traefik) y dominio a los servicios.
func enrichSwarmServicesWithInspect(services []domain.SwarmServiceInfo, rawInspect string) {
	lines := strings.Split(rawInspect, "\n")
	type svcMeta struct {
		expose bool
		domain string
	}
	metaMap := make(map[string]svcMeta)
	for _, l := range lines {
		parts := strings.Split(strings.TrimSpace(l), "\t")
		if len(parts) >= 1 && parts[0] != "" {
			name := parts[0]
			enabled := false
			rule := ""
			if len(parts) >= 2 && parts[1] == "true" {
				enabled = true
			}
			if len(parts) >= 3 {
				rule = parts[2]
			}
			// Extraer dominio de Host(`...`) si existe
			domainName := ""
			if strings.Contains(rule, "Host(`") {
				start := strings.Index(rule, "Host(`") + 6
				end := strings.Index(rule[start:], "`)")
				if end != -1 {
					domainName = rule[start : start+end]
				}
			} else if strings.Contains(rule, "PathPrefix(`") {
				start := strings.Index(rule, "PathPrefix(`") + 12
				end := strings.Index(rule[start:], "`)")
				if end != -1 {
					domainName = rule[start : start+end]
				}
			}
			metaMap[name] = svcMeta{expose: enabled, domain: domainName}
		}
	}

	for i := range services {
		if meta, ok := metaMap[services[i].Name]; ok {
			services[i].Expose = meta.expose
			services[i].Domain = meta.domain
		}
	}
}

// ParseSwarmServices convierte la salida tabulada de `docker service ls` en struct domain.SwarmServiceInfo.
func ParseSwarmServices(raw string) []domain.SwarmServiceInfo {
	services := make([]domain.SwarmServiceInfo, 0)
	lines := strings.Split(raw, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 5 {
			continue
		}

		ports := ""
		if len(parts) >= 6 {
			ports = strings.TrimSpace(parts[5])
		}

		services = append(services, domain.SwarmServiceInfo{
			ID:       strings.TrimSpace(parts[0]),
			Name:     strings.TrimSpace(parts[1]),
			Mode:     strings.TrimSpace(parts[2]),
			Replicas: strings.TrimSpace(parts[3]),
			Image:    strings.TrimSpace(parts[4]),
			Ports:    ports,
		})
	}

	return services
}

// ParseSwarmNodes convierte la salida tabulada de `docker node ls` en struct domain.SwarmNodeInfo.
func ParseSwarmNodes(raw string) []domain.SwarmNodeInfo {
	nodes := make([]domain.SwarmNodeInfo, 0)
	lines := strings.Split(raw, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			continue
		}

		managerStatus := ""
		engineVersion := ""
		if len(parts) >= 5 {
			managerStatus = strings.TrimSpace(parts[4])
		}
		if len(parts) >= 6 {
			engineVersion = strings.TrimSpace(parts[5])
		}

		nodes = append(nodes, domain.SwarmNodeInfo{
			ID:            strings.TrimSpace(parts[0]),
			Hostname:      strings.TrimSpace(parts[1]),
			Status:        strings.TrimSpace(parts[2]),
			Availability:  strings.TrimSpace(parts[3]),
			ManagerStatus: managerStatus,
			EngineVersion: engineVersion,
		})
	}

	return nodes
}
