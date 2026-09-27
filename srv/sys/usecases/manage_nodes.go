package usecases

import (
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type NodeInfo = domain.NodeInfo

type ManageNodesUseCase struct {
	repo ports.ConfigRepository
	ssh  ports.SSHExecutor
}

func NewManageNodesUseCase(repo ports.ConfigRepository, ssh ports.SSHExecutor) *ManageNodesUseCase {
	return &ManageNodesUseCase{repo: repo, ssh: ssh}
}

func parseNodeRole(mgrStatus string) (string, bool) {
	statusLower := strings.ToLower(mgrStatus)
	if strings.Contains(statusLower, "leader") {
		return "manager", true
	}
	if strings.Contains(statusLower, "reachable") {
		return "manager", false
	}
	return "worker", false
}

// ListNodes obtiene la lista detallada de nodos del clúster Docker Swarm
func (uc *ManageNodesUseCase) ListNodes(config domain.ServerConfig) ([]NodeInfo, error) {
	if uc.ssh == nil {
		return nil, fmt.Errorf("ejecutor SSH no configurado")
	}
	if err := uc.ssh.Connect(config); err != nil {
		return nil, fmt.Errorf("error de conexión SSH: %w", err)
	}
	defer uc.ssh.Close()

	res, err := uc.ssh.RunCommand("docker node ls --format '{{.ID}}|{{.Hostname}}|{{.Status}}|{{.Availability}}|{{.ManagerStatus}}|{{.EngineVersion}}'")
	if err != nil || res == nil || res.ExitCode != 0 {
		out := ""
		if res != nil {
			out = res.Output
		}
		if out == "" && err != nil {
			out = err.Error()
		}
		if strings.Contains(strings.ToLower(out), "not a swarm manager") || strings.Contains(strings.ToLower(out), "cannot connect to the docker daemon") {
			return []NodeInfo{}, nil
		}
		return nil, fmt.Errorf("error al listar nodos Swarm: %s", out)
	}

	var nodes []NodeInfo
	lines := strings.Split(strings.TrimSpace(res.Output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) >= 4 {
			id := parts[0]
			hostname := parts[1]
			status := parts[2]
			availability := parts[3]
			mgrStatus := ""
			if len(parts) >= 5 {
				mgrStatus = parts[4]
			}
			engineVer := ""
			if len(parts) >= 6 {
				engineVer = parts[5]
			}

			role, isLeader := parseNodeRole(mgrStatus)

			// Obtener etiquetas del nodo
			labels := uc.getNodeLabels(id)

			nodes = append(nodes, NodeInfo{
				ID:            id,
				Hostname:      hostname,
				Role:          role,
				Status:        status,
				Availability:  availability,
				IsLeader:      isLeader,
				EngineVersion: engineVer,
				Labels:        labels,
			})
		}
	}

	return nodes, nil
}

func (uc *ManageNodesUseCase) getNodeLabels(nodeID string) map[string]string {
	labels := make(map[string]string)
	res, err := uc.ssh.RunCommand(fmt.Sprintf("docker node inspect %s --format '{{json .Spec.Labels}}'", nodeID))
	if err == nil && res != nil && res.ExitCode == 0 && res.Output != "" && res.Output != "null\n" {
		out := strings.TrimSpace(res.Output)
		out = strings.TrimPrefix(out, "{")
		out = strings.TrimSuffix(out, "}")
		pairs := strings.Split(out, ",")
		for _, pair := range pairs {
			kv := strings.Split(pair, ":")
			if len(kv) == 2 {
				k := strings.Trim(strings.TrimSpace(kv[0]), `"`)
				v := strings.Trim(strings.TrimSpace(kv[1]), `"`)
				if k != "" {
					labels[k] = v
				}
			}
		}
	}
	return labels
}

// UpdateNodeAvailability cambia el estado del nodo ("active", "pause", "drain")
func (uc *ManageNodesUseCase) UpdateNodeAvailability(nodeID, availability string, config domain.ServerConfig) error {
	availability = strings.ToLower(strings.TrimSpace(availability))
	if availability != "active" && availability != "pause" && availability != "drain" {
		return fmt.Errorf("disponibilidad inválida. Opciones permitidas: active, pause, drain")
	}

	if uc.ssh == nil {
		return fmt.Errorf("ejecutor SSH no configurado")
	}
	if err := uc.ssh.Connect(config); err != nil {
		return fmt.Errorf("error de conexión SSH: %w", err)
	}
	defer uc.ssh.Close()

	res, err := uc.ssh.RunCommand(fmt.Sprintf("docker node update --availability %s %s", availability, nodeID))
	if err != nil || res == nil || res.ExitCode != 0 {
		out := ""
		if res != nil {
			out = res.Output
		}
		if out == "" && err != nil {
			out = err.Error()
		}
		return fmt.Errorf("error al actualizar disponibilidad del nodo: %s", out)
	}
	return nil
}

// SetNodeRole promueve ("manager") o degrada ("worker") un nodo Swarm
func (uc *ManageNodesUseCase) SetNodeRole(nodeID, role string, config domain.ServerConfig) error {
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "manager" && role != "worker" {
		return fmt.Errorf("rol inválido. Opciones: manager, worker")
	}

	if uc.ssh == nil {
		return fmt.Errorf("ejecutor SSH no configurado")
	}
	if err := uc.ssh.Connect(config); err != nil {
		return fmt.Errorf("error de conexión SSH: %w", err)
	}
	defer uc.ssh.Close()

	cmd := fmt.Sprintf("docker node promote %s", nodeID)
	if role == "worker" {
		cmd = fmt.Sprintf("docker node demote %s", nodeID)
	}

	res, err := uc.ssh.RunCommand(cmd)
	if err != nil || res == nil || res.ExitCode != 0 {
		out := ""
		if res != nil {
			out = res.Output
		}
		if out == "" && err != nil {
			out = err.Error()
		}
		return fmt.Errorf("error al cambiar rol del nodo: %s", out)
	}
	return nil
}

// AddNodeLabel agrega o actualiza una etiqueta de afinidad en un nodo
func (uc *ManageNodesUseCase) AddNodeLabel(nodeID, key, value string, config domain.ServerConfig) error {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if key == "" {
		return fmt.Errorf("la clave de la etiqueta no puede estar vacía")
	}

	if uc.ssh == nil {
		return fmt.Errorf("ejecutor SSH no configurado")
	}
	if err := uc.ssh.Connect(config); err != nil {
		return fmt.Errorf("error de conexión SSH: %w", err)
	}
	defer uc.ssh.Close()

	res, err := uc.ssh.RunCommand(fmt.Sprintf("docker node update --label-add %s=%s %s", key, value, nodeID))
	if err != nil || res == nil || res.ExitCode != 0 {
		out := ""
		if res != nil {
			out = res.Output
		}
		if out == "" && err != nil {
			out = err.Error()
		}
		return fmt.Errorf("error al agregar etiqueta al nodo: %s", out)
	}
	return nil
}

// RemoveNodeLabel elimina una etiqueta de un nodo
func (uc *ManageNodesUseCase) RemoveNodeLabel(nodeID, key string, config domain.ServerConfig) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("la clave de la etiqueta no puede estar vacía")
	}

	if uc.ssh == nil {
		return fmt.Errorf("ejecutor SSH no configurado")
	}
	if err := uc.ssh.Connect(config); err != nil {
		return fmt.Errorf("error de conexión SSH: %w", err)
	}
	defer uc.ssh.Close()

	res, err := uc.ssh.RunCommand(fmt.Sprintf("docker node update --label-rm %s %s", key, nodeID))
	if err != nil || res == nil || res.ExitCode != 0 {
		out := ""
		if res != nil {
			out = res.Output
		}
		if out == "" && err != nil {
			out = err.Error()
		}
		return fmt.Errorf("error al eliminar etiqueta del nodo: %s", out)
	}
	return nil
}
