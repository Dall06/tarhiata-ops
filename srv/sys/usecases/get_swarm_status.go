package usecases

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/Dall06/tarhiata-ops/pkg/dockerutil"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

// SwarmBundleCommand es el script Bash empaquetado que ejecuta todas las consultas de Docker Swarm en un único viaje SSH.
const SwarmBundleCommand = `echo "===TARHIATA_DOCKER_VER==="
docker --version 2>/dev/null
echo "===TARHIATA_SWARM_STATE==="
docker info --format '{{.Swarm.LocalNodeState}}' 2>/dev/null
echo "===TARHIATA_SERVICES==="
docker service ls --format '{{.ID}}\t{{.Name}}\t{{.Mode}}\t{{.Replicas}}\t{{.Image}}\t{{.Ports}}' 2>/dev/null
echo "===TARHIATA_INSPECT==="
svcs=$(docker service ls -q 2>/dev/null)
if [ -n "$svcs" ]; then
    docker service inspect $svcs --format '{{.Spec.Name}}\t{{index .Spec.Labels "traefik.enable"}}\t{{index .Spec.Labels (printf "traefik.http.routers.%s.rule" .Spec.Name)}}' 2>/dev/null
fi
echo "===TARHIATA_NODES==="
docker node ls --format '{{.ID}}\t{{.Hostname}}\t{{.Status}}\t{{.Availability}}\t{{.ManagerStatus}}\t{{.EngineVersion}}' 2>/dev/null
echo "===TARHIATA_END==="`

// SwarmBundlePayload contiene las secciones parseadas del comando unificado de Swarm.
type SwarmBundlePayload struct {
	DockerVersion string
	SwarmState    string
	ServicesRaw   string
	InspectRaw    string
	NodesRaw      string
}

func extractSection(raw, startMarker, endMarker string) string {
	startIdx := strings.Index(raw, startMarker)
	if startIdx == -1 {
		return ""
	}
	content := raw[startIdx+len(startMarker):]
	endIdx := strings.Index(content, endMarker)
	if endIdx == -1 {
		return strings.TrimSpace(content)
	}
	return strings.TrimSpace(content[:endIdx])
}

// ParseSwarmBundle divide la salida delimitada en sus 5 secciones.
func ParseSwarmBundle(raw string) (SwarmBundlePayload, error) {
	if !strings.Contains(raw, "===TARHIATA_DOCKER_VER===") {
		return SwarmBundlePayload{}, fmt.Errorf("salida no contiene delimitadores de Tarhiata Swarm")
	}

	payload := SwarmBundlePayload{
		DockerVersion: extractSection(raw, "===TARHIATA_DOCKER_VER===", "===TARHIATA_SWARM_STATE==="),
		SwarmState:    extractSection(raw, "===TARHIATA_SWARM_STATE===", "===TARHIATA_SERVICES==="),
		ServicesRaw:   extractSection(raw, "===TARHIATA_SERVICES===", "===TARHIATA_INSPECT==="),
		InspectRaw:    extractSection(raw, "===TARHIATA_INSPECT===", "===TARHIATA_NODES==="),
		NodesRaw:      extractSection(raw, "===TARHIATA_NODES===", "===TARHIATA_END==="),
	}

	return payload, nil
}

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
	defer func() {
		if clErr := uc.executor.Close(); clErr != nil {
			slog.Warn("get_swarm_status: error cerrando conexión SSH", "host", config.Host, "error", clErr)
		}
	}()

	status := &domain.SwarmStatus{
		Services:   []domain.SwarmServiceInfo{},
		Nodes:      []domain.SwarmNodeInfo{},
		Dashboards: make(map[string]string),
	}

	// Configurar URLs de Dashboards
	targetHost := strings.TrimSpace(config.Host)
	if IsLocal(config) || targetHost == "" {
		targetHost = "localhost"
	}
	status.Dashboards["portainer"] = fmt.Sprintf("http://%s:9000", targetHost)
	status.Dashboards["dozzle"] = fmt.Sprintf("http://%s:8888", targetHost)
	status.Dashboards["traefik"] = fmt.Sprintf("http://%s:8080", targetHost)

	// Intento 1: Ejecución Bundled en un único viaje SSH
	resBundle, errBundle := uc.executor.RunCommand(SwarmBundleCommand)
	if errBundle == nil && resBundle != nil && resBundle.ExitCode == 0 && strings.Contains(resBundle.Output, "===TARHIATA_DOCKER_VER===") {
		payload, errParse := ParseSwarmBundle(resBundle.Output)
		if errParse == nil {
			status.DockerVersion = payload.DockerVersion
			if strings.TrimSpace(payload.SwarmState) != "active" {
				status.Active = false
				status.NodeState = "inactive"
				return status, nil
			}

			status.Active = true
			status.NodeState = "active"

			if payload.ServicesRaw != "" {
				status.Services = ParseSwarmServices(payload.ServicesRaw)
				if payload.InspectRaw != "" {
					enrichSwarmServicesWithInspect(status.Services, payload.InspectRaw)
				}
			}

			if payload.NodesRaw != "" {
				status.Nodes = ParseSwarmNodes(payload.NodesRaw)
			}

			return status, nil
		}
	}

	// Fallback de contingencia: Invocaciones individuales si el bundle no estuviera presente o soportado
	resDocker, err := uc.executor.RunCommand("docker --version")
	if err != nil || resDocker == nil || resDocker.ExitCode != 0 {
		status.Active = false
		status.NodeState = "inactive"
		return status, nil
	}
	status.DockerVersion = strings.TrimSpace(resDocker.Output)

	resSwarm, err := uc.executor.RunCommand("docker info --format '{{.Swarm.LocalNodeState}}'")
	if err != nil || resSwarm == nil || strings.TrimSpace(resSwarm.Output) != "active" {
		status.Active = false
		status.NodeState = "inactive"
		return status, nil
	}
	status.Active = true
	status.NodeState = "active"

	resServices, err := uc.executor.RunCommand("docker service ls --format '{{.ID}}\t{{.Name}}\t{{.Mode}}\t{{.Replicas}}\t{{.Image}}\t{{.Ports}}'")
	if err == nil && resServices != nil && resServices.ExitCode == 0 {
		status.Services = ParseSwarmServices(resServices.Output)
		resInspect, errInsp := uc.executor.RunCommand("docker service inspect $(docker service ls -q) --format '{{.Spec.Name}}\t{{index .Spec.Labels \"traefik.enable\"}}\t{{index .Spec.Labels (printf \"traefik.http.routers.%s.rule\" .Spec.Name)}}' 2>/dev/null")
		if errInsp == nil && resInspect != nil && resInspect.ExitCode == 0 {
			enrichSwarmServicesWithInspect(status.Services, resInspect.Output)
		}
	}

	resNodes, err := uc.executor.RunCommand("docker node ls --format '{{.ID}}\t{{.Hostname}}\t{{.Status}}\t{{.Availability}}\t{{.ManagerStatus}}\t{{.EngineVersion}}'")
	if err == nil && resNodes != nil && resNodes.ExitCode == 0 {
		status.Nodes = ParseSwarmNodes(resNodes.Output)
	}

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
			domainName := dockerutil.ExtractDomain(rule)
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

func extractDomainFromRule(rule string) string {
	return dockerutil.ExtractDomain(rule)
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
