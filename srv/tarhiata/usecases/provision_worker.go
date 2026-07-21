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
	managerSSH ports.SSHExecutor
}

func NewProvisionWorkerUseCase(ssh ports.SSHExecutor) ports.ProvisionWorkerUseCase {
	return &ProvisionWorkerUseCase{managerSSH: ssh}
}

func (uc *ProvisionWorkerUseCase) Execute(config domain.ServerConfig, nodeName string, labelType string) (string, error) {
	if config.DOAPIToken == "" {
		return "", fmt.Errorf("se requiere un Token de API de DigitalOcean configurado en el servidor para crear nodos automáticos")
	}

	fmt.Println("⏳ [1/5] Obteniendo Token de Swarm del Manager...")
	res, err := uc.managerSSH.RunCommand("docker swarm join-token worker -q")
	if err != nil || res.ExitCode != 0 {
		return "", fmt.Errorf("falló al obtener join-token: %s", res.Output)
	}
	joinToken := strings.TrimSpace(res.Output)
	managerIP := config.Host

	fmt.Printf("🏗️  [2/5] Provisionando nueva VM '%s' vía Terraform en DigitalOcean...\n", nodeName)
	workspace := filepath.Join(os.TempDir(), "tarhiata_tf_worker_"+nodeName)
	provisioner := repositories.NewDigitalOceanProvisioner(workspace)

	newIP, privKeyContent, err := provisioner.ProvisionNode(config.DOAPIToken, nodeName, "nyc1")
	if err != nil {
		return "", fmt.Errorf("falló provisionamiento terraform: %w", err)
	}

	fmt.Printf("✅ VM Creada! IP Pública: %s\n", newIP)
	fmt.Println("⏳ [3/5] Esperando a que Cloud-Init instale Docker en el nuevo nodo (Esto tomará un par de minutos)...")

	// Create temp private key file to SSH into the new worker
	tmpKeyPath := filepath.Join(os.TempDir(), "worker_key_"+nodeName)
	if err := os.WriteFile(tmpKeyPath, []byte(privKeyContent), 0600); err != nil {
		return "", fmt.Errorf("no se pudo escribir la llave ssh temporal: %w", err)
	}
	defer os.Remove(tmpKeyPath)

	workerSSH := repositories.NewCryptoSSHExecutor()

	// Reintentos de SSH (El nodo recién creado puede tardar en abrir el puerto 22)
	var connected bool
	for i := 0; i < 15; i++ {
		err := workerSSH.Connect(domain.ServerConfig{
			Host:       newIP,
			Port:       22,
			User:       "root",
			PrivateKey: tmpKeyPath,
		})
		if err == nil {
			connected = true
			break
		}
		time.Sleep(10 * time.Second)
	}

	if !connected {
		return "", fmt.Errorf("no se pudo conectar por SSH al nuevo nodo después de múltiples intentos")
	}
	defer workerSSH.Close()

	// Wait for cloud-init to finish
	_, err = workerSSH.RunCommand("cloud-init status --wait")
	if err != nil {
		return "", fmt.Errorf("error esperando a cloud-init: %w", err)
	}

	fmt.Println("🔗 [4/5] Uniendo el nuevo nodo al clúster de Docker Swarm...")
	joinCmd := fmt.Sprintf("docker swarm join --token %s %s:2377", joinToken, managerIP)
	resJoin, err := workerSSH.RunCommand(joinCmd)
	if err != nil || resJoin.ExitCode != 0 {
		return "", fmt.Errorf("falló swarm join en el worker: %s", resJoin.Output)
	}

	fmt.Println("🏷️  [5/5] Etiquetando el nodo para anclaje de recursos...")
	// Volvemos al Manager para etiquetar
	labelCmd := fmt.Sprintf("docker node update --label-add type=%s %s", labelType, nodeName)
	// Puede que Swarm tarde un segundito en registrar el nodo, hacemos reintentos simples
	for i := 0; i < 5; i++ {
		res, _ := uc.managerSSH.RunCommand(labelCmd)
		if res.ExitCode == 0 {
			break
		}
		time.Sleep(3 * time.Second)
	}

	fmt.Println("🎉 ¡Clúster expandido exitosamente!")
	return newIP, nil
}
