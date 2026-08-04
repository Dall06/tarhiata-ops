package usecases

import (
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
	"golang.org/x/crypto/bcrypt"
)

type DeployObservabilityUseCase struct {
	ssh ports.SSHExecutor
}

func NewDeployObservabilityUseCase(ssh ports.SSHExecutor) ports.DeployObservabilityUseCase {
	return &DeployObservabilityUseCase{ssh: ssh}
}

// Execute despliega Portainer (Dashboard) y Dozzle (Logs web ultra-ligeros)
func (uc *DeployObservabilityUseCase) Execute(exposePublic bool) error {
	var middlewaresDef, portainerMiddleware, dozzleMiddleware string
	if !exposePublic {
		middlewaresDef = "- \"traefik.http.middlewares.vpn-allowlist.ipallowlist.sourcerange=100.64.0.0/10,127.0.0.1/32\""
		portainerMiddleware = "- \"traefik.http.routers.portainer.middlewares=vpn-allowlist\""
		dozzleMiddleware = "- \"traefik.http.routers.dozzle.middlewares=vpn-allowlist\""
	}

	compose := fmt.Sprintf(`version: '3.8'
services:
  portainer:
    image: portainer/portainer-ce:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - portainer_data:/data
    networks:
      - tarhiata_internal
    deploy:
      labels:
        - "traefik.enable=true"
        - "traefik.http.routers.portainer.rule=Host(`+"`"+`portainer.tarhiata.local`+"`"+`)"
        - "traefik.http.routers.portainer.entrypoints=web"
        - "traefik.http.services.portainer.loadbalancer.server.port=9000"
        %s
        %s
      placement:
        constraints: [node.role == manager]

  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_LEVEL=info
      - DOZZLE_TAIL_SIZE=300
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    networks:
      - tarhiata_internal
    deploy:
      labels:
        - "traefik.enable=true"
        - "traefik.http.routers.dozzle.rule=Host(`+"`"+`dozzle.tarhiata.local`+"`"+`)"
        - "traefik.http.routers.dozzle.entrypoints=web"
        - "traefik.http.services.dozzle.loadbalancer.server.port=8080"
        %s
      placement:
        constraints: [node.role == manager]

networks:
  tarhiata_internal:
    external: true

volumes:
  portainer_data:
`, middlewaresDef, portainerMiddleware, dozzleMiddleware)

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

func (uc *DeployObservabilityUseCase) ExecutePersistent(exposePublic bool, deployType string, grafanaPassword string) error {
	return uc.ExecutePersistentWithVolume(exposePublic, deployType, grafanaPassword, "/opt/data/obs")
}

func (uc *DeployObservabilityUseCase) ExecutePersistentWithVolume(exposePublic bool, deployType, grafanaPassword, volumePath string) error {
	if volumePath == "" {
		volumePath = "/opt/data/obs"
	}
	volumePath = strings.TrimSuffix(volumePath, "/")

	constraint := `"node.role == manager"`
	if deployType == "multi-node" {
		constraint = `"node.labels.type == obs"`
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(grafanaPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("falló al encriptar contraseña para portainer: %w", err)
	}
	portainerPasswordHash := strings.ReplaceAll(string(hash), "$", "$$")

	// 1. Crear directorios y permisos directamente por SSH en el volumen especificado
	mkdirCmd := fmt.Sprintf("mkdir -p %s/config %s/data/loki %s/data/grafana %s/data/portainer && chown -R 472:472 %s/data/grafana && chown -R 10001:10001 %s/data/loki",
		volumePath, volumePath, volumePath, volumePath, volumePath, volumePath)
	uc.ssh.RunCommand(mkdirCmd)

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

	prometheusConfig := `global:
  scrape_interval: 15s
scrape_configs:
  - job_name: 'cadvisor'
    static_configs:
      - targets: ['cadvisor:8080']
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']`

	grafanaDatasource := `apiVersion: 1
datasources:
  - name: Loki
    type: loki
    access: proxy
    url: http://loki:3100
    isDefault: true
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090`

	writeConfigCmd := fmt.Sprintf(`docker service create --name write-configs-obs --restart-condition none --constraint %s --mount type=bind,source=/,destination=/host alpine sh -c "cat << 'EOF' > /host%s/config/loki.yaml
%s
EOF
cat << 'EOF' > /host%s/config/promtail.yaml
%s
EOF
cat << 'EOF' > /host%s/config/prometheus.yaml
%s
EOF
cat << 'EOF' > /host%s/config/grafana-datasources.yaml
%s
EOF
"`, constraint, volumePath, lokiConfig, volumePath, promtailConfig, volumePath, prometheusConfig, volumePath, grafanaDatasource)

	uc.ssh.RunCommand(writeConfigCmd)
	uc.ssh.RunCommand("docker service rm write-configs-obs")

	// 4. Stack Compose
	var middlewaresDef, portainerMiddleware, grafanaMiddleware string
	if !exposePublic {
		middlewaresDef = "- \"traefik.http.middlewares.vpn-allowlist.ipallowlist.sourcerange=100.64.0.0/10,127.0.0.1/32\""
		portainerMiddleware = "- \"traefik.http.routers.portainer.middlewares=vpn-allowlist\""
		grafanaMiddleware = "- \"traefik.http.routers.grafana.middlewares=vpn-allowlist\""
	}

	compose := fmt.Sprintf(`version: '3.8'
services:
  portainer:
    image: portainer/portainer-ce:latest
    command: ["--admin-password", "%%s"]
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - %s/data/portainer:/data
    networks:
      - tarhiata_internal
    deploy:
      labels:
        - "traefik.enable=true"
        - "traefik.http.routers.portainer.rule=Host(` + "`" + `portainer.tarhiata.local` + "`" + `)"
        - "traefik.http.routers.portainer.entrypoints=web"
        - "traefik.http.services.portainer.loadbalancer.server.port=9000"
        %%s
        %%s
      placement:
        constraints: [node.role == manager]

  cadvisor:
    image: gcr.io/cadvisor/cadvisor:v0.47.2
    volumes:
      - /:/rootfs:ro
      - /var/run:/var/run:ro
      - /sys:/sys:ro
      - /var/lib/docker/:/var/lib/docker:ro
      - /dev/disk/:/dev/disk:ro
    networks:
      - tarhiata_internal
    deploy:
      mode: global

  prometheus:
    image: prom/prometheus:v2.48.1
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
    volumes:
      - %s/config/prometheus.yaml:/etc/prometheus/prometheus.yml
      - %s/data/prometheus:/prometheus
    networks:
      - tarhiata_internal
    ports:
      - "9090:9090"
    deploy:
      placement:
        constraints: [%%s]

  loki:
    image: grafana/loki:2.9.2
    command: -config.file=/etc/loki/local-config.yaml
    volumes:
      - %s/config/loki.yaml:/etc/loki/local-config.yaml
      - %s/data/loki:/loki
    networks:
      - tarhiata_internal
    deploy:
      placement:
        constraints: [%%s]

  promtail:
    image: grafana/promtail:2.9.2
    volumes:
      - /var/lib/docker/containers:/var/lib/docker/containers:ro
      - %s/config/promtail.yaml:/etc/promtail/config.yml
    command: -config.file=/etc/promtail/config.yml
    networks:
      - tarhiata_internal
    deploy:
      mode: global

  grafana:
    image: grafana/grafana:10.2.2
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=%%s
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - %s/data/grafana:/var/lib/grafana
      - %s/config/grafana-datasources.yaml:/etc/grafana/provisioning/datasources/datasources.yaml
    networks:
      - tarhiata_internal
    deploy:
      labels:
        - "traefik.enable=true"
        - "traefik.http.routers.grafana.rule=Host(` + "`" + `grafana.tarhiata.local` + "`" + `)"
        - "traefik.http.routers.grafana.entrypoints=web"
        - "traefik.http.services.grafana.loadbalancer.server.port=3000"
        %%s
      placement:
        constraints: [%%s]

networks:
  tarhiata_internal:
    external: true
`, volumePath, volumePath, volumePath, volumePath, volumePath, volumePath, volumePath, volumePath)

	writeCmd := fmt.Sprintf("cat << 'EOF' > /tmp/obs-persist-stack.yml\n%s\nEOF", fmt.Sprintf(compose, portainerPasswordHash, middlewaresDef, portainerMiddleware, constraint, constraint, grafanaPassword, grafanaMiddleware, constraint))
	if _, err := uc.ssh.RunCommand(writeCmd); err != nil {
		return fmt.Errorf("falló al escribir observability compose: %w", err)
	}

	res, err := uc.ssh.RunCommand("docker stack deploy -c /tmp/obs-persist-stack.yml tarhiata_obs")
	if err != nil || res.ExitCode != 0 {
		return fmt.Errorf("falló al desplegar observabilidad: %s", res.Output)
	}

	return nil
}
