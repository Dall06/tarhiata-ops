package usecases

import (
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
)

// BootstrapperUseCase orquesta la inicialización de un servidor virgen.
type BootstrapperUseCase struct {
	ssh ports.SSHExecutor
}

// NewBootstrapperUseCase crea una nueva instancia del caso de uso.
func NewBootstrapperUseCase(ssh ports.SSHExecutor) ports.BootstrapperUseCase {
	return &BootstrapperUseCase{
		ssh: ssh,
	}
}

// InitServer prepara el servidor: instala dependencias, Docker,// InitServer prepara el servidor con la capa 1 y capa 2
func (uc *BootstrapperUseCase) InitServer(installObservability bool, acmeEmail string, installTS bool, tsAuthKey string, exposeObs bool) error {
	
	// 0. Liberar posibles bloqueos de apt por procesos de cloud-init atascados (ej. get-docker.sh sin noninteractive)
	fmt.Println("⏳ [Bootstrapper] Comprobando integridad del servidor y liberando dpkg locks si es necesario...")
	uc.ssh.RunCommand("export DEBIAN_FRONTEND=noninteractive; killall -9 apt apt-get dpkg 2>/dev/null; dpkg --configure -a 2>/dev/null")
	
	// 1. Configurar rotación de logs a nivel daemonr Docker
	if err := uc.ensureDockerInstalled(); err != nil {
		return fmt.Errorf("error asegurando docker: %w", err)
	}

	// 2. Configurar Docker Daemon (Log rotation)
	if err := uc.configureDockerDaemon(); err != nil {
		return fmt.Errorf("error configurando daemon.json: %w", err)
	}

	// 3. Inicializar Docker Swarm
	if err := uc.ensureSwarmActive(); err != nil {
		return fmt.Errorf("error inicializando swarm: %w", err)
	}

	// 4. Configurar Firewall (UFW) y asegurar aislamiento en VPS
	if err := uc.configureFirewall(); err != nil {
		return fmt.Errorf("error configurando firewall: %w", err)
	}

	// 5. Desplegar Traefik (El Proxy Inverso Global para la experiencia tipo Vercel)
	if err := uc.deployTraefik(acmeEmail); err != nil {
		return fmt.Errorf("error desplegando traefik: %w", err)
	}

	// 6. Opcional: Instalar Tailscale para VPN privada
	if installTS {
		if err := uc.InstallTailscale(tsAuthKey); err != nil {
			return fmt.Errorf("error instalando tailscale: %w", err)
		}
	}

	// 7. Opcional: Stack de Observabilidad
	if installObservability {
		if err := uc.DeployObservability(exposeObs); err != nil {
			return fmt.Errorf("error desplegando observabilidad: %w", err)
		}
	}

	return nil
}

// InstallTailscale instala y levanta Tailscale en el nodo
func (uc *BootstrapperUseCase) InstallTailscale(authKey string) error {
	res, _ := uc.ssh.RunCommand("command -v tailscale")
	if res.ExitCode != 0 {
		_, err := uc.ssh.RunCommand("curl -fsSL https://tailscale.com/install.sh | sh")
		if err != nil {
			return err
		}
	}
	
	if authKey != "" {
		res, err := uc.ssh.RunCommand("tailscale up --authkey=" + authKey)
		if err != nil || res.ExitCode != 0 {
			return fmt.Errorf("falló al levantar tailscale: %s", res.Output)
		}
	}
	return nil
}

func (uc *BootstrapperUseCase) ensureDockerInstalled() error {
	res, err := uc.ssh.RunCommand("command -v docker")
	if err == nil && res.ExitCode == 0 && strings.TrimSpace(res.Output) != "" {
		// Docker ya está instalado, saltamos este paso.
		return nil
	}

	// Ejecutar script oficial de instalación silenciosa para VPS
	installCmd := "curl -fsSL https://get.docker.com -o get-docker.sh && sh get-docker.sh"
	res, err = uc.ssh.RunCommand(installCmd)
	if err != nil || res.ExitCode != 0 {
		return fmt.Errorf("falló la instalación de docker: %s", res.Output)
	}
	return nil
}

func (uc *BootstrapperUseCase) configureDockerDaemon() error {
	// 1. Limitar el tamaño a nivel del demonio (Capa 1 de seguridad)
	daemonJSON := `{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "50m",
    "max-file": "3"
  }
}`
	cmd1 := fmt.Sprintf("mkdir -p /etc/docker && echo '%s' > /etc/docker/daemon.json && systemctl restart docker", daemonJSON)
	res1, err := uc.ssh.RunCommand(cmd1)
	if err != nil || res1.ExitCode != 0 {
		return fmt.Errorf("error configurando daemon.json: %s", res1.Output)
	}

	// 2. Tarea programada (Cron) para eliminar estrictamente logs de más de 48 horas (Capa 2)
	cronJob := `0 3 * * * root find /var/lib/docker/containers/ -name "*-json.log" -mtime +2 -exec truncate -s 0 {} \;`
	cmd2 := fmt.Sprintf("echo '%s' > /etc/cron.d/docker-log-cleanup && chmod 644 /etc/cron.d/docker-log-cleanup", cronJob)
	res2, err := uc.ssh.RunCommand(cmd2)
	if err != nil || res2.ExitCode != 0 {
		return fmt.Errorf("error configurando cronjob de 48h: %s", res2.Output)
	}

	return nil
}

func (uc *BootstrapperUseCase) ensureSwarmActive() error {
	res, err := uc.ssh.RunCommand("docker info | grep -i 'Swarm: active'")
	if err == nil && res.ExitCode == 0 && strings.TrimSpace(res.Output) != "" {
		// El clúster Swarm ya está encendido
		return nil
	}

	// Inicializar Swarm (Si el VPS tiene múltiples interfaces, Docker elegirá la default, 
	// pero podríamos pasarle --advertise-addr de la VLAN en el futuro si es necesario).
	res, err = uc.ssh.RunCommand("docker swarm init")
	if err != nil || res.ExitCode != 0 {
		return fmt.Errorf("falló docker swarm init: %s", res.Output)
	}
	return nil
}

func (uc *BootstrapperUseCase) configureFirewall() error {
	// 1. Validar que ufw exista (Ubuntu/Debian)
	res, _ := uc.ssh.RunCommand("command -v ufw")
	if res.ExitCode != 0 {
		_, err := uc.ssh.RunCommand("apt-get update && apt-get install -y ufw")
		if err != nil {
			return fmt.Errorf("no se pudo instalar ufw automáticamente")
		}
	}

	// 2. Reglas de Seguridad Base (VLANs/VPC awareness se agregará después dinámicamente)
	commands := []string{
		"ufw --force reset",          // Limpiar configuraciones basura
		"ufw default deny incoming",  // Bloquear absolutamente todo desde el exterior
		"ufw default allow outgoing", // Permitir que el servidor baje paquetes
		"ufw allow ssh",              // Dejar la puerta abierta para nuestro CLI
		"ufw allow 80/tcp",           // HTTP para Traefik
		"ufw allow 443/tcp",          // HTTPS para Traefik y SSL
		"ufw --force enable",         // Encender el escudo
	}

	for _, cmd := range commands {
		res, err := uc.ssh.RunCommand(cmd)
		if err != nil || res.ExitCode != 0 {
			return fmt.Errorf("falló el comando del firewall '%s': %s", cmd, res.Output)
		}
	}

	return nil
}

func (uc *BootstrapperUseCase) deployTraefik(acmeEmail string) error {
	// 1. Crear red pública para que los contenedores se comuniquen con Traefik
	res, _ := uc.ssh.RunCommand("docker network ls | grep tarhiata_public")
	if res.ExitCode != 0 {
		_, err := uc.ssh.RunCommand("docker network create --driver overlay tarhiata_public")
		if err != nil {
			return fmt.Errorf("falló al crear la red overlay de tarhiata: %w", err)
		}
	}

	// Configuración de Let's Encrypt dinámica
	acmeConfig := ""
	acmeVolume := ""
	if acmeEmail != "" {
		acmeConfig = fmt.Sprintf(`      - "--certificatesresolvers.leresolver.acme.tlschallenge=true"
      - "--certificatesresolvers.leresolver.acme.email=%s"
      - "--certificatesresolvers.leresolver.acme.storage=/letsencrypt/acme.json"`, acmeEmail)
		acmeVolume = "- \"traefik_certs:/letsencrypt\""
	}

	// 2. Archivo Docker Compose estándar para Traefik
	traefikCompose := fmt.Sprintf(`
version: '3.8'
services:
  traefik:
    image: traefik:v3.1
    environment:
      - DOCKER_API_VERSION=1.41
    command:
      - "--api.insecure=true"
      - "--providers.docker=true"
      - "--providers.docker.swarmMode=true"
      - "--providers.docker.exposedbydefault=false"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
%s
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - "/var/run/docker.sock:/var/run/docker.sock:ro"
      %s
    networks:
      - tarhiata_public
    deploy:
      placement:
        constraints:
          - node.role == manager

networks:
  tarhiata_public:
    external: true

volumes:
  traefik_certs:
`, acmeConfig, acmeVolume)

	// 3. Escribir el archivo en el servidor y desplegar
	// Usamos un 'cat' seguro vía heredoc a través del SSH
	writeCmd := fmt.Sprintf("cat << 'EOF' > /tmp/traefik-stack.yml\n%s\nEOF", traefikCompose)
	_, err := uc.ssh.RunCommand(writeCmd)
	if err != nil {
		return fmt.Errorf("falló al escribir traefik compose: %w", err)
	}

	res, err = uc.ssh.RunCommand("docker stack deploy -c /tmp/traefik-stack.yml tarhiata_proxy")
	if err != nil || res.ExitCode != 0 {
		return fmt.Errorf("falló al desplegar traefik: %s", res.Output)
	}

	return nil
}

// DeployObservability despliega Portainer (Dashboard) y Dozzle (Logs web ultra-ligeros)
func (uc *BootstrapperUseCase) DeployObservability(exposePublic bool) error {
	if exposePublic {
		// El usuario decidió exponerlo públicamente bajo su propio riesgo
		uc.ssh.RunCommand("ufw allow 9000/tcp && ufw allow 8888/tcp")
	} else {
		// Bloqueamos explícitamente el acceso público a estos puertos usando la cadena DOCKER-USER
		// ya que Docker bypassea UFW. Solo se podrá acceder por Tailscale (tailscale0) o localhost (lo).
		blockCmd := `EXT_IF=$(ip route get 8.8.8.8 | awk '{print $5; exit}') && iptables -I DOCKER-USER -i $EXT_IF -p tcp -m multiport --dports 9000,8888 -j DROP`
		uc.ssh.RunCommand(blockCmd)
	}

	compose := `version: '3.8'
services:
  portainer:
    image: portainer/portainer-ce:latest
    ports:
      - "9000:9000"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - portainer_data:/data
    deploy:
      placement:
        constraints: [node.role == manager]

  dozzle:
    image: amir20/dozzle:latest
    ports:
      - "8888:8080"
    environment:
      - DOZZLE_LEVEL=info
      - DOZZLE_TAIL_SIZE=300
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    deploy:
      placement:
        constraints: [node.role == manager]

volumes:
  portainer_data:
`
	
	writeCmd := fmt.Sprintf("cat << 'EOF' > /tmp/observability-stack.yml\n%s\nEOF", compose)
	if _, err := uc.ssh.RunCommand(writeCmd); err != nil {
		return fmt.Errorf("falló al escribir observability compose: %w", err)
	}

	res, err := uc.ssh.RunCommand("docker stack deploy -c /tmp/observability-stack.yml tarhiata_obs")
	if err != nil || res.ExitCode != 0 {
		return fmt.Errorf("falló al desplegar observabilidad: %s", res.Output)
	}

	return nil
}
