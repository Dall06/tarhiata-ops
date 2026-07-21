package usecases

import (
	"fmt"
	"github.com/Dall06/tarhiata-ops/srv/tarhiata/ports"
)

type DeployObservabilityUseCase struct {
	ssh ports.SSHExecutor
}

func NewDeployObservabilityUseCase(ssh ports.SSHExecutor) ports.DeployObservabilityUseCase {
	return &DeployObservabilityUseCase{ssh: ssh}
}

// Execute despliega Portainer (Dashboard) y Dozzle (Logs web ultra-ligeros)
func (uc *DeployObservabilityUseCase) Execute(exposePublic bool) error {
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

func (uc *DeployObservabilityUseCase) ExecutePersistent(exposePublic bool) error {
	if exposePublic {
		uc.ssh.RunCommand("ufw allow 9000/tcp && ufw allow 3001/tcp")
	} else {
		blockCmd := `EXT_IF=$(ip route get 8.8.8.8 | awk '{print $5; exit}') && iptables -I DOCKER-USER -i $EXT_IF -p tcp -m multiport --dports 9000,3001 -j DROP`
		uc.ssh.RunCommand(blockCmd)
	}

	// 1. Crear directorios para los configs y data
	uc.ssh.RunCommand("mkdir -p /opt/tarhiata/obs/config /opt/tarhiata/obs/data/loki /opt/tarhiata/obs/data/grafana /opt/tarhiata/obs/data/portainer")

	// Permisos para Grafana (uid 472) y Loki (uid 10001)
	uc.ssh.RunCommand("chown -R 472:472 /opt/tarhiata/obs/data/grafana && chown -R 10001:10001 /opt/tarhiata/obs/data/loki")

	// 2. Escribir Loki Config
	lokiConfig := `auth_enabled: false
server:
  http_listen_port: 3100
ingester:
  lifecycler:
    ring:
      kvstore:
        store: inmemory
      replication_factor: 1
  chunk_idle_period: 5m
  chunk_retain_period: 30s
schema_config:
  configs:
  - from: 2020-10-24
    store: boltdb-shipper
    object_store: filesystem
    schema: v11
    index:
      prefix: index_
      period: 24h
storage_config:
  boltdb_shipper:
    active_index_directory: /loki/index
    cache_location: /loki/index_cache
    shared_store: filesystem
  filesystem:
    directory: /loki/chunks
compactor:
  working_directory: /loki/compactor
  shared_store: filesystem
limits_config:
  reject_old_samples: true
  reject_old_samples_max_age: 168h`

	uc.ssh.RunCommand(fmt.Sprintf("cat << 'EOF' > /opt/tarhiata/obs/config/loki.yaml\n%s\nEOF", lokiConfig))

	// 3. Escribir Promtail Config
	promtailConfig := `server:
  http_listen_port: 9080
  grpc_listen_port: 0
positions:
  filename: /tmp/positions.yaml
clients:
  - url: http://loki:3100/loki/api/v1/push
scrape_configs:
- job_name: containers
  static_configs:
  - targets:
      - localhost
    labels:
      job: containerlogs
      __path__: /var/lib/docker/containers/*/*log
  pipeline_stages:
  - json:
      expressions:
        output: log
        stream: stream
        attrs:
  - json:
      expressions:
        tag:
      source: attrs
  - regex:
      expression: (?P<image_name>(?:[^|]*[^|])).(?P<container_name>(?:[^|]*[^|])).(?P<image_id>(?:[^|]*[^|])).(?P<container_id>(?:[^|]*[^|]))
      source: tag
  - timestamp:
      format: RFC3339Nano
      source: time
  - labels:
      tag:
      stream:
      image_name:
      container_name:
      image_id:
      container_id:
  - output:
      source: output`

	uc.ssh.RunCommand(fmt.Sprintf("cat << 'EOF' > /opt/tarhiata/obs/config/promtail.yaml\n%s\nEOF", promtailConfig))

	// 4. Stack Compose
	compose := `version: '3.8'
services:
  portainer:
    image: portainer/portainer-ce:latest
    ports:
      - "9000:9000"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /opt/tarhiata/obs/data/portainer:/data
    deploy:
      placement:
        constraints: [node.role == manager]

  loki:
    image: grafana/loki:2.9.2
    ports:
      - "3100:3100"
    command: -config.file=/etc/loki/local-config.yaml
    volumes:
      - /opt/tarhiata/obs/config/loki.yaml:/etc/loki/local-config.yaml
      - /opt/tarhiata/obs/data/loki:/loki
    deploy:
      placement:
        constraints: [node.role == manager]

  promtail:
    image: grafana/promtail:2.9.2
    volumes:
      - /var/lib/docker/containers:/var/lib/docker/containers:ro
      - /opt/tarhiata/obs/config/promtail.yaml:/etc/promtail/config.yml
    command: -config.file=/etc/promtail/config.yml
    deploy:
      mode: global

  grafana:
    image: grafana/grafana:10.2.2
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - /opt/tarhiata/obs/data/grafana:/var/lib/grafana
    deploy:
      placement:
        constraints: [node.role == manager]
`

	writeCmd := fmt.Sprintf("cat << 'EOF' > /tmp/obs-persist-stack.yml\n%s\nEOF", compose)
	if _, err := uc.ssh.RunCommand(writeCmd); err != nil {
		return fmt.Errorf("falló al escribir observability compose: %w", err)
	}

	res, err := uc.ssh.RunCommand("docker stack deploy -c /tmp/obs-persist-stack.yml tarhiata_obs")
	if err != nil || res.ExitCode != 0 {
		return fmt.Errorf("falló al desplegar observabilidad: %s", res.Output)
	}

	// 5. Configurar Data Source en Grafana (automático mediante API si quisieramos, pero por ahora lo dejamos a mano o le damos instrucciones al usuario)
	return nil
}
