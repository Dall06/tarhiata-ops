package usecases

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/repositories"
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
	if config.DOAPIToken == "" {
		return "", fmt.Errorf("se requiere un Token de API de DigitalOcean configurado en el servidor para crear nodos automáticos")
	}

	fmt.Println("⏳ [1/6] Obteniendo Token de Swarm del Manager...")
	res, err := uc.managerSSH.RunCommand("docker swarm join-token worker -q")
	if err != nil || res.ExitCode != 0 {
		return "", fmt.Errorf("falló al obtener join-token: %s", res.Output)
	}
	joinToken := strings.TrimSpace(res.Output)
	managerIP := config.Host

	fmt.Printf("🏗️  [2/6] Provisionando VM '%s' vía Terraform (Idempotente)...\n", nodeName)
	var provisioner ports.Provisioner
	homeDir, _ := os.UserHomeDir()
	if uc.Provisioner != nil {
		provisioner = uc.Provisioner
	} else {
		workspace := filepath.Join(homeDir, ".config", "tarhiata", "terraform", "worker_"+nodeName)
		provisioner = repositories.NewDigitalOceanProvisioner(workspace)
	}

	newIP, privKeyContent, err := provisioner.ProvisionNode(config.DOAPIToken, nodeName, "nyc1")
	if err != nil {
		return newIP, fmt.Errorf("falló provisionamiento terraform: %w", err)
	}

	fmt.Printf("✅ VM Confirmada en IP: %s\n", newIP)

	// Implementar Rollback (GAP 2: Gestión de Orfandad)
	setupSuccess := false
	defer func() {
		if !setupSuccess {
			fmt.Println("⚠️ Ocurrió un error en la configuración. Ejecutando ROLLBACK (terraform destroy) para evitar costos fantasma...")
			provisioner.DestroyNode(config.DOAPIToken, nodeName)
		}
	}()

	// Guardar la llave privada de forma persistente (GAP 2)
	keyDir := filepath.Join(homeDir, ".ssh")
	os.MkdirAll(keyDir, 0700)
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
	} else {
		workerSSH = repositories.NewCryptoSSHExecutor()
	}

	var connected bool
	for i := 0; i < 15; i++ {
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
	// Join Swarm (Falla silenciosamente si ya es parte del swarm, lo cual es correcto)
	joinCmd := fmt.Sprintf("docker swarm join --token %s %s:2377", joinToken, managerIP)
	workerSSH.RunCommand(joinCmd)

	fmt.Println("🏷️  [6/6] Etiquetando el nodo para anclaje de recursos...")
	// Volvemos al Manager para etiquetar
	labelCmd := fmt.Sprintf("docker node update --label-add type=%s %s", labelType, nodeName)
	for i := 0; i < 5; i++ {
		res, _ := uc.managerSSH.RunCommand(labelCmd)
		if res.ExitCode == 0 {
			break
		}
		time.Sleep(3 * time.Second)
	}

	fmt.Println("🎉 ¡Nodo provisionado y asegurado exitosamente!")
	setupSuccess = true
	return newIP, nil
}
