package usecases

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

// JoinExistingWorkerUseCase une un servidor que YA existe en el fleet (con o sin Docker
// instalado) como nodo worker del clúster Swarm de otro servidor del fleet. A diferencia
// de ProvisionWorkerUseCase, no crea infraestructura nueva vía Terraform: el caller ya
// conectó ambas conexiones SSH (manager y worker) antes de llamar a Execute.
type JoinExistingWorkerUseCase struct {
	managerSSH ports.SSHExecutor
	workerSSH  ports.SSHExecutor
}

func NewJoinExistingWorkerUseCase(managerSSH, workerSSH ports.SSHExecutor) *JoinExistingWorkerUseCase {
	return &JoinExistingWorkerUseCase{managerSSH: managerSSH, workerSSH: workerSSH}
}

// Execute une el worker al clúster del manager y lo etiqueta. managerHost es el host/IP
// del manager (para el comando "docker swarm join"); nodeName es el nombre con el que se
// intentará etiquetar el nodo en Swarm (normalmente el hostname real del worker).
func (uc *JoinExistingWorkerUseCase) Execute(managerHost, nodeName, labelType string) error {
	fmt.Println("⏳ [1/5] Obteniendo Token de Swarm del Manager...")
	res, err := uc.managerSSH.RunCommand("docker swarm join-token worker -q")
	if err != nil || res == nil || res.ExitCode != 0 {
		out := ""
		if res != nil {
			out = res.Output
		}
		return fmt.Errorf("falló al obtener join-token (¿el manager ya tiene Swarm activo?): %s", out)
	}
	joinToken := strings.TrimSpace(res.Output)

	fmt.Println("🐋 [2/5] Asegurando que Docker esté instalado en el worker...")
	// Reusa InitServerUseCase.ensureDockerInstalled: a diferencia del worker auto-
	// provisionado (que ya trae Docker vía cloud-init), un VPS existente puede no tenerlo.
	if err := (&InitServerUseCase{ssh: uc.workerSSH}).ensureDockerInstalled(); err != nil {
		return fmt.Errorf("error instalando docker en el worker: %w", err)
	}

	fmt.Println("🔗 [3/5] Abriendo puertos de clúster Swarm (manager y worker)...")
	if _, errUfw := uc.managerSSH.RunCommand("ufw allow 2377/tcp && ufw allow 7946/tcp && ufw allow 7946/udp && ufw allow 4789/udp"); errUfw != nil {
		slog.Debug("join_existing_worker: aviso al abrir puertos swarm en manager", "error", errUfw)
	}
	if _, errWkrUfw := uc.workerSSH.RunCommand("systemctl start docker || service docker start && ufw allow 2377/tcp && ufw allow 7946/tcp && ufw allow 7946/udp && ufw allow 4789/udp"); errWkrUfw != nil {
		slog.Debug("join_existing_worker: aviso al encender docker y abrir puertos swarm en worker", "error", errWkrUfw)
	}

	// Permitir que el worker haga pull del registry Docker privado interno (sin TLS,
	// solo alcanzable dentro de la red overlay) para build-from-source.
	daemonJSON := fmt.Sprintf(`{"insecure-registries": ["%s"]}`, RegistryInternalAddress)
	daemonCmd := fmt.Sprintf("mkdir -p /etc/docker && echo '%s' > /etc/docker/daemon.json && systemctl restart docker", daemonJSON)
	if _, errDaemon := uc.workerSSH.RunCommand(daemonCmd); errDaemon != nil {
		slog.Debug("join_existing_worker: aviso al configurar insecure-registries en worker", "error", errDaemon)
	}

	fmt.Println("🚀 [4/5] Uniendo el worker al clúster Swarm...")
	joinCmd := fmt.Sprintf("docker swarm join --token %s %s:2377", joinToken, managerHost)
	joinRes, joinErr := uc.workerSSH.RunCommand(joinCmd)
	if joinErr != nil || (joinRes != nil && joinRes.ExitCode != 0) {
		out := ""
		if joinRes != nil {
			out = joinRes.Output
		}
		fmt.Printf("⚠️  Aviso Swarm Join en Worker (puede estar ya unido): %v (out: %s)\n", joinErr, out)
	}

	actualHostname := nodeName
	if resHost, errHost := uc.workerSSH.RunCommand("hostname"); errHost == nil && resHost != nil && strings.TrimSpace(resHost.Output) != "" {
		actualHostname = strings.TrimSpace(resHost.Output)
	}

	fmt.Println("🏷️  [5/5] Etiquetando el nodo para anclaje de recursos...")
	labeled := false
	for i := 0; i < 15 && !labeled; i++ {
		for _, nameCandidate := range []string{nodeName, actualHostname} {
			if nameCandidate == "" {
				continue
			}
			if res, errExec := uc.managerSSH.RunCommand(fmt.Sprintf("docker node update --label-add type=%s %s", labelType, nameCandidate)); errExec == nil && res != nil && res.ExitCode == 0 {
				labeled = true
				fmt.Printf("✅ Nodo '%s' etiquetado con 'type=%s'!\n", nameCandidate, labelType)
				break
			}
		}
		if labeled {
			break
		}
		time.Sleep(3 * time.Second)
	}
	if !labeled {
		slog.Warn("join_existing_worker: no se pudo etiquetar el nodo automáticamente, puede etiquetarse manualmente desde Nodos del Clúster", "node", nodeName)
	}

	return nil
}
