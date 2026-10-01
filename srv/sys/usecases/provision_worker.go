package usecases

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Dall06/tarhiata-ops/opt/cloud"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
	"github.com/Dall06/tarhiata-ops/srv/sys/repositories"
)

type ProvisionWorkerUseCase struct {
	managerSSH       ports.SSHExecutor
	Provisioner      ports.Provisioner
	WorkerSSHFactory func() ports.SSHExecutor
}

func NewProvisionWorkerUseCase(ssh ports.SSHExecutor) ports.ProvisionWorkerUseCase {
	return &ProvisionWorkerUseCase{managerSSH: ssh}
}

func (uc *ProvisionWorkerUseCase) Execute(config domain.ServerConfig, nodeName string, labelType string) (string, error) {
	return uc.ExecuteWithPlanAndRegion(config, nodeName, labelType, "", "")
}

func (uc *ProvisionWorkerUseCase) ExecuteWithRegion(config domain.ServerConfig, nodeName string, labelType string, requestedRegion string) (string, error) {
	return uc.ExecuteWithPlanAndRegion(config, nodeName, labelType, "", requestedRegion)
}

func (uc *ProvisionWorkerUseCase) ExecuteWithPlanAndRegion(config domain.ServerConfig, nodeName string, labelType string, requestedPlan string, requestedRegion string) (string, error) {
	token := config.VultrAPIToken
	if token == "" {
		token = config.DOAPIToken
	}
	if token == "" {
		return "", fmt.Errorf("se requiere un Token de API de Vultr configurado en el servidor para crear nodos automáticos")
	}

	fmt.Println("⏳ [1/6] Obteniendo Token de Swarm del Manager...")
	res, err := uc.managerSSH.RunCommand("docker swarm join-token worker -q")
	if err != nil || res.ExitCode != 0 {
		return "", fmt.Errorf("falló al obtener join-token: %s", res.Output)
	}
	joinToken := strings.TrimSpace(res.Output)
	managerIP := config.Host

	fmt.Printf("🏗️  [2/6] Provisionando VM '%s' (Plan: %s) en Vultr vía Terraform...\n", nodeName, requestedPlan)
	var provisioner ports.Provisioner
	homeDir, errHome := os.UserHomeDir()
	if errHome != nil {
		homeDir = os.TempDir()
	}
	var activeToken string
	var region string
	if uc.Provisioner != nil {
		provisioner = uc.Provisioner
		activeToken = "mock"
	}
	if uc.Provisioner == nil {
		workspace := filepath.Join(homeDir, ".config", "tarhiata", "terraform", "worker_"+nodeName)
		if config.CloudProvider == "digitalocean" && config.DOAPIToken != "" {
			provisioner = cloud.NewProvisioner("digitalocean", workspace)
			activeToken = config.DOAPIToken
			region = "nyc1"
			if requestedRegion != "" {
				region = requestedRegion
			}
		}
		if config.CloudProvider != "digitalocean" || config.DOAPIToken == "" {
			provisioner = cloud.NewProvisioner("vultr", workspace)
			activeToken = token
			region = "mex"
			if requestedRegion != "" {
				region = requestedRegion
			}
		}
	}

	provRes, err := provisioner.ProvisionNode(activeToken, nodeName, region, requestedPlan)
	if err != nil {
		return provRes.PublicIP, fmt.Errorf("falló provisionamiento terraform: %w", err)
	}
	newIP := provRes.PublicIP
	privKeyContent := provRes.PrivateKey

	fmt.Printf("✅ VM Confirmada en IP: %s\n", newIP)

	// Implementar Rollback (GAP 2: Gestión de Orfandad)
	setupSuccess := false
	defer func() {
		if !setupSuccess {
			fmt.Println("⚠️ Ocurrió un error en la configuración. Ejecutando ROLLBACK (terraform destroy) para evitar costos fantasma...")
			if errDestroy := provisioner.DestroyNode(activeToken, nodeName); errDestroy != nil {
				slog.Warn("rollback: fallo al destruir nodo huérfano", "node", nodeName, "error", errDestroy)
			}
		}
	}()

	// Guardar la llave privada de forma persistente (GAP 2)
	keyDir := filepath.Join(homeDir, ".ssh")
	if errMk := os.MkdirAll(keyDir, 0700); errMk != nil {
		slog.Warn("fallo creando directorio de llaves SSH", "dir", keyDir, "error", errMk)
	}
	keyPath := filepath.Join(keyDir, "tarhiata_worker_"+nodeName+".pem")

	// Solo re-escribimos la llave si no existe o si Terraform la acaba de crear (simplificado: siempre intentamos escribirla si tenemos contenido)
	if privKeyContent != "" {
		if err := os.WriteFile(keyPath, []byte(privKeyContent), 0600); err != nil {
			return newIP, fmt.Errorf("no se pudo guardar la llave ssh persistente: %w", err)
		}
	}

	fmt.Println("⏳ [3/6] Conectando por SSH al Worker (Reintentos si está arrancando)...")
	var workerSSH ports.SSHExecutor
	if uc.WorkerSSHFactory != nil {
		workerSSH = uc.WorkerSSHFactory()
	}
	if uc.WorkerSSHFactory == nil {
		workerSSH = repositories.NewCryptoSSHExecutor()
	}

	var connected bool
	for i := 0; i < 30; i++ { // Tolerancia de hasta 5 minutos para encendido de VM e inicio de SSH
		err := workerSSH.Connect(domain.ServerConfig{
			Host:       newIP,
			Port:       22,
			User:       "root",
			PrivateKey: keyPath,
		})
		if err == nil {
			connected = true
			break
		}
		time.Sleep(10 * time.Second)
	}

	if !connected {
		return newIP, fmt.Errorf("no se pudo conectar por SSH al nuevo nodo después de múltiples intentos")
	}
	defer workerSSH.Close()

	fmt.Println("⏳ [4/6] Esperando a que Cloud-Init termine (Instalación de Docker)...")
	_, err = workerSSH.RunCommand("cloud-init status --wait")
	if err != nil {
		return newIP, fmt.Errorf("error esperando a cloud-init: %w", err)
	}

	fmt.Println("🔗 [5/6] Asegurando red y clúster Swarm...")
	// Asegurar que el Firewall UFW del Manager permita los puertos del clúster Swarm
	resUfw, errUfw := uc.managerSSH.RunCommand("ufw allow 2377/tcp && ufw allow 7946/tcp && ufw allow 7946/udp && ufw allow 4789/udp")
	if errUfw != nil {
		slog.Debug("aviso al abrir puertos swarm en manager", "error", errUfw)
	}
	if resUfw != nil && resUfw.ExitCode != 0 {
		slog.Debug("código de salida al abrir puertos swarm en manager", "exitCode", resUfw.ExitCode)
	}

	// Asegurar que el Worker tenga Docker arriba y sus puertos de Swarm abiertos
	resWkrUfw, errWkrUfw := workerSSH.RunCommand("systemctl start docker || service docker start && ufw allow 2377/tcp && ufw allow 7946/tcp && ufw allow 7946/udp && ufw allow 4789/udp")
	if errWkrUfw != nil {
		slog.Debug("aviso al encender docker y abrir puertos swarm en worker", "error", errWkrUfw)
	}
	if resWkrUfw != nil && resWkrUfw.ExitCode != 0 {
		slog.Debug("código de salida al encender docker en worker", "exitCode", resWkrUfw.ExitCode)
	}

	// Permitir que el worker haga pull del registry Docker privado interno (sin TLS,
	// solo alcanzable dentro de la red overlay) para build-from-source.
	daemonJSON := fmt.Sprintf(`{"insecure-registries": ["%s"]}`, RegistryInternalAddress)
	daemonCmd := fmt.Sprintf("mkdir -p /etc/docker && echo '%s' > /etc/docker/daemon.json && systemctl restart docker", daemonJSON)
	if _, errDaemon := workerSSH.RunCommand(daemonCmd); errDaemon != nil {
		slog.Debug("aviso al configurar insecure-registries en worker", "error", errDaemon)
	}

	joinCmd := fmt.Sprintf("docker swarm join --token %s %s:2377", joinToken, managerIP)
	joinRes, joinErr := workerSSH.RunCommand(joinCmd)
	if joinErr != nil || (joinRes != nil && joinRes.ExitCode != 0) {
		outStr := ""
		if joinRes != nil { outStr = joinRes.Output }
		fmt.Printf("⚠️  Aviso Swarm Join en Worker (puede estar ya unido): %v (out: %s)\n", joinErr, outStr)
	}

	// Obtenemos el hostname real del nodo worker
	actualHostname := nodeName
	resHost, errHost := workerSSH.RunCommand("hostname")
	if errHost == nil && resHost != nil && strings.TrimSpace(resHost.Output) != "" {
		actualHostname = strings.TrimSpace(resHost.Output)
	}

	fmt.Println("🏷️  [6/6] Etiquetando el nodo para anclaje de recursos...")
	labeled := false
	for i := 0; i < 15; i++ {
		// 1. Intentar por nodeName o actualHostname
		for _, nameCandidate := range []string{nodeName, actualHostname} {
			if nameCandidate == "" { continue }
			res, errExec := uc.managerSSH.RunCommand(fmt.Sprintf("docker node update --label-add type=%s %s", labelType, nameCandidate))
			if errExec == nil && res != nil && res.ExitCode == 0 {
				labeled = true
				fmt.Printf("✅ Nodo '%s' etiquetado con 'type=%s'!\n", nameCandidate, labelType)
				break
			}
		}
		if labeled { break }

		// 2. Buscar por ID de nodo worker en el clúster
		resList, errList := uc.managerSSH.RunCommand("docker node ls --format '{{.ID}} {{.Hostname}}'")
		if errList == nil && resList != nil && resList.Output != "" {
			for _, line := range strings.Split(resList.Output, "\n") {
				line = strings.TrimSpace(line)
				if line == "" { continue }
				parts := strings.Fields(line)
				nID := parts[0]
				nHost := ""
				if len(parts) > 1 { nHost = parts[1] }

				if strings.EqualFold(nHost, nodeName) || strings.EqualFold(nHost, actualHostname) || strings.HasPrefix(nHost, "worker") {
					res, errExec := uc.managerSSH.RunCommand(fmt.Sprintf("docker node update --label-add type=%s %s", labelType, nID))
					if errExec == nil && res != nil && res.ExitCode == 0 {
						labeled = true
						fmt.Printf("✅ Nodo '%s' (%s) etiquetado con 'type=%s'!\n", nHost, nID, labelType)
						break
					}
				}
			}
			if labeled { break }
		}
		time.Sleep(3 * time.Second)
	}

	fmt.Println("🎉 ¡Nodo provisionado y asegurado exitosamente!")
	setupSuccess = true
	return newIP, nil
}
