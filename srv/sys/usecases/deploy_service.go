package usecases

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type DeployServiceUseCase struct {
	ssh ports.SSHExecutor
}

func NewDeployServiceUseCase(ssh ports.SSHExecutor) ports.DeployServiceUseCase {
	return &DeployServiceUseCase{
		ssh: ssh,
	}
}

// DeployService orquesta la subida de archivos, descarga de la imagen y el despliegue dinámico.
func (uc *DeployServiceUseCase) Execute(service domain.CustomService, config domain.DeployConfig) error {
	// 1. Aprovisionar Imagen (Remotamente en el servidor)
	if err := uc.provisionImage(config); err != nil {
		return fmt.Errorf("error aprovisionando imagen: %w", err)
	}

	// 2. Crear directorio de trabajo en el servidor para este servicio
	workDir := fmt.Sprintf("/opt/tarhiata/services/%s", service.Name)
	uc.ssh.RunCommand(fmt.Sprintf("mkdir -p %s", workDir))

	// 3. Transferir archivos locales (ej. .env, secrets.json) del usuario al servidor
	if err := uc.transferFiles(service.Files, workDir); err != nil {
		return fmt.Errorf("error transfiriendo archivos env: %w", err)
	}

	// 3.5. Transferir archivos extra de configuración (Mounts)
	for _, m := range service.Mounts {
		content, err := os.ReadFile(m.LocalPath)
		if err != nil {
			return fmt.Errorf("no se pudo leer archivo de config local %s: %w", m.LocalPath, err)
		}
		// Guardarlos en una carpeta config/ dentro del workdir
		remoteConfDir := fmt.Sprintf("%s/configs", workDir)
		uc.ssh.RunCommand(fmt.Sprintf("mkdir -p %s", remoteConfDir))

		fileName := filepath.Base(m.LocalPath)
		remotePath := fmt.Sprintf("%s/%s", remoteConfDir, fileName)
		if err := uc.writeRemoteFile(remotePath, string(content)); err != nil {
			return fmt.Errorf("error transfiriendo config %s: %w", fileName, err)
		}
	}

	// 3.6. Propagar directorio de trabajo a nodos worker si la afinidad no es el manager
	targetNode := strings.TrimSpace(config.TargetNode)
	if targetNode != "" && targetNode != "manager" && targetNode != "master" {
		// Sincronizar archivos al worker vía rsync desde el manager
		syncCmd := fmt.Sprintf(
			"docker node ls --format '{{.Hostname}}' | while read node; do "+
				"if [ \"$node\" != \"$(hostname)\" ]; then "+
				"rsync -avz --rsync-path='mkdir -p %s && rsync' %s/ $node:%s/ 2>/dev/null || true; "+
				"fi; done", workDir, workDir, workDir)
		uc.ssh.RunCommand(syncCmd)
	}

	// 4. Generar e inyectar el Compose File dinámico (Con la magia de Traefik)
	composeContent := uc.generateCompose(service, config)
	if err := uc.writeRemoteFile(workDir+"/docker-compose.yml", composeContent); err != nil {
		return fmt.Errorf("error escribiendo compose: %w", err)
	}

	// 4.5. Pre-Deploy Hook (Migraciones de BD Automatizadas pre-swap)
	if strings.TrimSpace(service.PreDeployHook) != "" {
		hookCmd := strings.TrimSpace(service.PreDeployHook)
		targetImage := config.ImageSource
		if config.IsURL {
			targetImage = fmt.Sprintf("%s_local_image:latest", service.Name)
		}
		runHookCmd := fmt.Sprintf("docker run --rm --net tarhiata_internal --env-file %s/.env %s %s", workDir, targetImage, hookCmd)
		res, err := uc.ssh.RunCommand(runHookCmd)
		if err != nil || (res != nil && res.ExitCode != 0) {
			output := ""
			if res != nil {
				output = res.Output
			}
			return fmt.Errorf("falló el pre-deploy hook de migración ('%s'): %s", hookCmd, output)
		}
	}

	// 5. Desplegar el Stack en Swarm con autenticación de registry
	deployCmd := fmt.Sprintf("cd %s && docker stack deploy --with-registry-auth -c docker-compose.yml %s", workDir, service.Name)
	res, err := uc.ssh.RunCommand(deployCmd)
	if err != nil || res.ExitCode != 0 {
		return fmt.Errorf("error en docker stack deploy: %s", res.Output)
	}

	// 6. Exportar estado de referencias al VPS Host para sincronización multi-PC
	syncUC := NewSyncClusterStateUseCase(nil, uc.ssh)
	_ = syncUC.ExportStateToRemote()

	return nil
}

// provisionImage manda los comandos para bajar la imagen desde Hub o URL.
func (uc *DeployServiceUseCase) provisionImage(config domain.DeployConfig) error {
	if !config.IsURL {
		// Asumimos Docker Hub (Por ahora público. Aquí luego se inyecta docker login)
		res, err := uc.ssh.RunCommand(fmt.Sprintf("docker pull %s", config.ImageSource))
		if err != nil || res.ExitCode != 0 {
			return fmt.Errorf("falló docker pull: %s", res.Output)
		}
		return nil
	}

	// Es una URL (wget -> unzip -> docker load) procesado directo en el server
	cmd := fmt.Sprintf("wget -qO /tmp/img.zip %s && unzip -o /tmp/img.zip -d /tmp/img_ext && docker load -i /tmp/img_ext/*.tar && rm -rf /tmp/img.zip /tmp/img_ext", config.ImageSource)
	res, err := uc.ssh.RunCommand(cmd)
	if err != nil || res.ExitCode != 0 {
		return fmt.Errorf("falló descarga/carga desde URL: %s", res.Output)
	}
	return nil
}

// transferFiles lee de la máquina local y usa Base64 para escribirlos en el servidor vía SSH.
func (uc *DeployServiceUseCase) transferFiles(files []domain.ServiceFile, remoteDir string) error {
	for _, f := range files {
		content, err := os.ReadFile(f.LocalPath)
		if err != nil {
			return fmt.Errorf("no se pudo leer archivo local %s: %w", f.LocalPath, err)
		}

		remotePath := fmt.Sprintf("%s/%s", remoteDir, f.FileName)
		if err := uc.writeRemoteFile(remotePath, string(content)); err != nil {
			return err
		}
	}
	return nil
}

// writeRemoteFile usa base64 para evitar inyecciones raras de comillas o saltos de línea al mandar texto por SSH.
func (uc *DeployServiceUseCase) writeRemoteFile(remotePath, content string) error {
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	cmd := fmt.Sprintf("echo '%s' | base64 -d > %s", encoded, remotePath)
	res, err := uc.ssh.RunCommand(cmd)
	if err != nil || res.ExitCode != 0 {
		return fmt.Errorf("falló al escribir %s: %s", remotePath, res.Output)
	}
	return nil
}

// generateCompose construye el YAML on-the-fly con las etiquetas de enrutamiento de Traefik.
func (uc *DeployServiceUseCase) generateCompose(service domain.CustomService, config domain.DeployConfig) string {
	imageName := config.ImageSource
	if config.IsURL {
		imageName = fmt.Sprintf("%s_local_image:latest", service.Name) // Placeholder si la imagen es cargada por tar
	}

	compose := fmt.Sprintf("version: '3.8'\nservices:\n  %s:\n    image: %s\n", service.Name, imageName)

	if len(service.Files) > 0 {
		compose += "    env_file:\n"
		for _, f := range service.Files {
			compose += fmt.Sprintf("      - %s\n", f.FileName)
		}
	}

	if config.HealthcheckCmd != "" {
		escapedHealth := strings.ReplaceAll(config.HealthcheckCmd, `"`, `\"`)
		compose += "    healthcheck:\n"
		compose += fmt.Sprintf("      test: [\"CMD-SHELL\", \"%s\"]\n", escapedHealth)
		compose += "      interval: 30s\n"
		compose += "      timeout: 10s\n"
		compose += "      retries: 3\n"
	}

	if len(service.Mounts) > 0 {
		compose += "    volumes:\n"
		for _, m := range service.Mounts {
			fileName := filepath.Base(m.LocalPath)
			compose += fmt.Sprintf("      - ./configs/%s:%s\n", fileName, m.DestPath)
		}
	}

	compose += "    networks:\n      - tarhiata_public\n      - tarhiata_internal\n"

	// Despliegue & Node Affinity
	constraint := formatNodeConstraint(config.TargetNode)

	compose += "    deploy:\n"
	compose += "      update_config:\n"
	compose += "        order: start-first\n"
	compose += "        failure_action: rollback\n"
	compose += "        delay: 5s\n"
	compose += "        parallelism: 1\n"
	compose += "      rollback_config:\n"
	compose += "        order: stop-first\n"
	compose += "        parallelism: 1\n"
	compose += "      placement:\n        constraints:\n"
	compose += fmt.Sprintf("          - %s\n", constraint)

	// Etiquetas de Traefik (La magia PaaS)
	if config.Expose {
		compose += "      labels:\n"
		compose += "        - \"traefik.enable=true\"\n"

		rule := ""
		if config.Domain != "" {
			rule = fmt.Sprintf("Host(`%s`)", config.Domain)
		} else {
			// Si no hay dominio, enrutamos por un Path Prefix como fallback
			rule = fmt.Sprintf("PathPrefix(`/%s`)", service.Name)
			// (Se removió StripPrefix intencionalmente para forzar consistencia de rutas en el navegador)
		}

		compose += fmt.Sprintf("        - \"traefik.http.routers.%s.rule=%s\"\n", service.Name, rule)
		compose += fmt.Sprintf("        - \"traefik.http.services.%s.loadbalancer.server.port=%d\"\n", service.Name, config.Port)

		entrypoints := "web"
		if config.EnableSSL {
			entrypoints = "web,websecure"
		}
		compose += fmt.Sprintf("        - \"traefik.http.routers.%s.entrypoints=%s\"\n", service.Name, entrypoints)

		if config.EnableSSL {
			compose += fmt.Sprintf("        - \"traefik.http.routers.%s.tls.certresolver=leresolver\"\n", service.Name)
		}
	}

	compose += "\nnetworks:\n  tarhiata_public:\n    external: true\n  tarhiata_internal:\n    external: true\n"
	return compose
}

func formatNodeConstraint(node string) string {
	n := strings.TrimSpace(node)
	if n == "" || n == "manager" || n == "master" {
		return "node.role == manager"
	}
	if n == "worker" {
		return "node.role == worker"
	}
	if strings.Contains(n, "==") {
		return n
	}
	return fmt.Sprintf("node.hostname == %s", n)
}
