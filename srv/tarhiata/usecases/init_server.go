package usecases

import (
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
)

// InitServerUseCase orquesta la inicialización de un servidor virgen.
type InitServerUseCase struct {
	ssh ports.SSHExecutor
}

// NewInitServerUseCase crea una nueva instancia del caso de uso.
func NewInitServerUseCase(ssh ports.SSHExecutor) ports.InitServerUseCase {
	return &InitServerUseCase{
		ssh: ssh,
	}
}

// InitServer prepara el servidor: instala dependencias, Docker,// InitServer prepara el servidor con la capa 1 y capa 2
func (uc *InitServerUseCase) Execute(acmeEmail string) error {

	// 0. Liberar posibles bloqueos de apt por procesos de cloud-init atascados (ej. get-docker.sh sin noninteractive)
	fmt.Println("⏳ [Bootstrapper] Comprobando integridad del servidor y liberando dpkg locks si es necesario...")
	uc.ssh.RunCommand("export DEBIAN_FRONTEND=noninteractive; killall -9 apt apt-get dpkg 2>/dev/null; dpkg --configure -a 2>/dev/null")

	// 0.5. Hardening de SSH y prevención de Fuerza Bruta
	if err := uc.hardenSSH(); err != nil {
		return fmt.Errorf("error asegurando SSH: %w", err)
	}
	if err := uc.installFail2Ban(); err != nil {
		return fmt.Errorf("error instalando fail2ban: %w", err)
	}

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

	return nil
}

func (uc *InitServerUseCase) hardenSSH() error {
	fmt.Println("🔒 [Bootstrapper] Asegurando servicio SSH...")
	// Deshabilitar root login con contraseña, pero dejar PasswordAuthentication intacto por si es un servidor BYO
	cmd := `sed -i 's/^#*PermitRootLogin.*/PermitRootLogin prohibit-password/' /etc/ssh/sshd_config && systemctl restart ssh || systemctl restart sshd`
	_, err := uc.ssh.RunCommand(cmd)
	return err
}

func (uc *InitServerUseCase) installFail2Ban() error {
	fmt.Println("🛡️  [Bootstrapper] Instalando Fail2Ban para prevenir fuerza bruta...")

	// Solo instala fail2ban si no existe
	res, _ := uc.ssh.RunCommand("command -v fail2ban-server")
	if res.ExitCode != 0 {
		_, err := uc.ssh.RunCommand("export DEBIAN_FRONTEND=noninteractive; apt-get update && apt-get install -y fail2ban")
		if err != nil {
			return err
		}
	}

	// Configurar bantime a 15 minutos (900s) para sshd
	jailConfig := `[sshd]
enabled = true
port = ssh
filter = sshd
logpath = /var/log/auth.log
maxretry = 3
bantime = 900
findtime = 600
`
	writeCmd := fmt.Sprintf("cat << 'EOF' > /etc/fail2ban/jail.local\n%s\nEOF\nsystemctl restart fail2ban", jailConfig)
	_, err := uc.ssh.RunCommand(writeCmd)
	return err
}

func (uc *InitServerUseCase) ensureDockerInstalled() error {
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

func (uc *InitServerUseCase) configureDockerDaemon() error {
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

func (uc *InitServerUseCase) ensureSwarmActive() error {
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

func (uc *InitServerUseCase) configureFirewall() error {
	// 1. Validar que ufw exista (Ubuntu/Debian)
	res, _ := uc.ssh.RunCommand("command -v ufw")
	if res.ExitCode != 0 {
		_, err := uc.ssh.RunCommand("apt-get update && apt-get install -y ufw")
		if err != nil {
			return fmt.Errorf("no se pudo instalar ufw automáticamente")
		}
	}

	// 2. Reglas de Seguridad Base (Respetando servicios existentes)
	commands := []string{
		"ufw allow 80/tcp",  // HTTP para Traefik
		"ufw allow 443/tcp", // HTTPS para Traefik y SSL
		"CURRENT_SSH_PORT=$(echo $SSH_CLIENT | awk '{print $3}'); if [ -n \"$CURRENT_SSH_PORT\" ]; then ufw allow $CURRENT_SSH_PORT/tcp; else ufw allow ssh; fi", // Puerto SSH dinámico
		"ufw --force enable", // Encender el escudo
	}

	for _, cmd := range commands {
		res, err := uc.ssh.RunCommand(cmd)
		if err != nil || res.ExitCode != 0 {
			return fmt.Errorf("falló el comando del firewall '%s': %s", cmd, res.Output)
		}
	}

	return nil
}

func (uc *InitServerUseCase) deployTraefik(acmeEmail string) error {
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
