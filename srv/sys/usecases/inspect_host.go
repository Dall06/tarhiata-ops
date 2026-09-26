package usecases

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

// InspectHostUseCase implementa ports.InspectHostUseCase.
type InspectHostUseCase struct {
	executor ports.SSHExecutor
}

// NewInspectHostUseCase instancia el caso de uso para telemetría y servicios del host.
func NewInspectHostUseCase(executor ports.SSHExecutor) ports.InspectHostUseCase {
	return &InspectHostUseCase{
		executor: executor,
	}
}

const hostMetricsCommand = `sh -c '
OS_NAME=$(uname -s)
if [ "$OS_NAME" = "Linux" ]; then
    PRETTY=$(grep -m1 "^PRETTY_NAME=" /etc/os-release 2>/dev/null | cut -d= -f2- | tr -d "\"")
    if [ -n "$PRETTY" ]; then
        OS_FULL="$PRETTY"
    else
        OS_FULL=$(uname -s -m)
    fi
else
    OS_FULL=$(uname -s -m)
fi
HOSTNAME=$(hostname)
UPTIME=$(uptime -p 2>/dev/null || uptime | sed "s/.*up \([^,]*\), .*/\1/")
LOAD=$(cat /proc/loadavg 2>/dev/null | awk "{print \$1, \$2, \$3}")
CORES=$(nproc 2>/dev/null || grep -c ^processor /proc/cpuinfo 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 1)

if [ "$OS_NAME" = "Darwin" ]; then
    MEM_TOTAL_BYTES=$(sysctl -n hw.memsize 2>/dev/null || echo 0)
    MEM_TOTAL_MB=$((MEM_TOTAL_BYTES / 1024 / 1024))
    MEM_USED_MB=$((MEM_TOTAL_MB / 2))
    DISK_LINE=$(df -k / 2>/dev/null | tail -n 1)
    DISK_TOTAL_MB=$(echo "$DISK_LINE" | awk "{print int(\$2/1024)}")
    DISK_USED_MB=$(echo "$DISK_LINE" | awk "{print int(\$3/1024)}")
    CPU_PCT=$(top -l 1 -n 0 2>/dev/null | awk "/CPU usage/ {print 100 - \$7}" | tr -d "%")
    LOAD=$(sysctl -n vm.loadavg 2>/dev/null | awk "{print \$2, \$3, \$4}")
else
    MEM_INFO=$(awk "/MemTotal:/ {total=\$2} /MemAvailable:/ {avail=\$2} END {used=total-avail; printf \"%d %d\", used/1024, total/1024}" /proc/meminfo 2>/dev/null)
    MEM_USED_MB=$(echo "$MEM_INFO" | awk "{print \$1}")
    MEM_TOTAL_MB=$(echo "$MEM_INFO" | awk "{print \$2}")
    DISK_LINE=$(df -m / 2>/dev/null | tail -n 1)
    DISK_USED_MB=$(echo "$DISK_LINE" | awk "{print \$3}")
    DISK_TOTAL_MB=$(echo "$DISK_LINE" | awk "{print \$2}")
    CPU_PCT=$(top -bn1 2>/dev/null | grep "Cpu" | awk -F"," "{for(i=1;i<=NF;i++) if(\$i ~ /id/) {split(\$i,a,\" \"); print 100-a[1]}}")
    if [ -z "$CPU_PCT" ]; then
        CPU_PCT=$(awk -v FS=" " "/^cpu /{print 100-(\$5*100/(\$2+\$3+\$4+\$5+\$6+\$7+\$8))}" /proc/stat 2>/dev/null)
    fi
    if [ -z "$CPU_PCT" ]; then
        CPU_PCT=$(vmstat 1 2 2>/dev/null | tail -1 | awk "{print 100-\$15}")
    fi
fi

echo "OS:$OS_FULL"
echo "HOST:$HOSTNAME"
echo "CORES:$CORES"
echo "CPU_PCT:$CPU_PCT"
echo "MEM_USED_MB:$MEM_USED_MB"
echo "MEM_TOTAL_MB:$MEM_TOTAL_MB"
echo "DISK_USED_MB:$DISK_USED_MB"
echo "DISK_TOTAL_MB:$DISK_TOTAL_MB"
echo "LOAD:$LOAD"
echo "UPTIME:$UPTIME"
'`

const hostServicesCommand = `sh -c '
if command -v systemctl >/dev/null 2>&1; then
    systemctl list-units --type=service --state=running --no-legend --plain 2>/dev/null
elif [ "$(uname -s)" = "Darwin" ]; then
    launchctl list 2>/dev/null | awk "NR>1 && \$1 != \"-\" {print \$3, \"loaded\", \"active\", \"running\", \$3}"
else
    service --status-all 2>&1 | awk "/\+/ {print \$4, \"loaded\", \"active\", \"running\", \$4}"
fi
'`

// Execute conecta al host y extrae métricas de rendimiento y servicios activos.
func (uc *InspectHostUseCase) Execute(config domain.ServerConfig) (*domain.HostInspection, error) {
	targetHost := strings.TrimSpace(config.Host)
	if config.IsLocal() && targetHost == "" {
		targetHost = "localhost"
	}
	serverName := strings.TrimSpace(config.Name)
	if serverName == "" {
		serverName = targetHost
	}

	if err := uc.executor.Connect(config); err != nil {
		return nil, fmt.Errorf("error conectando a %s: %w", targetHost, err)
	}

	inspection := &domain.HostInspection{
		ServerName: serverName,
		Host:       targetHost,
		IsLocal:    config.IsLocal(),
		Timestamp:  time.Now().UTC(),
		Services:   []domain.HostSystemService{},
	}

	// 1. Obtener Métricas de Hardware / Host
	resMetrics, err := uc.executor.RunCommand(hostMetricsCommand)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo métricas del host: %w", err)
	}
	if resMetrics != nil {
		inspection.Metrics = ParseHostMetrics(resMetrics.Output)
	}

	// 2. Obtener Lista de Servicios Activos del Sistema
	resServices, err := uc.executor.RunCommand(hostServicesCommand)
	if err == nil && resServices != nil {
		inspection.Services = ParseHostServices(resServices.Output)
	}

	return inspection, nil
}

// ExecuteMetricsOnly extrae exclusivamente las métricas de hardware sin incurrir en la sobrecarga de listar servicios.
func (uc *InspectHostUseCase) ExecuteMetricsOnly(config domain.ServerConfig) (*domain.HostMetrics, error) {
	targetHost := strings.TrimSpace(config.Host)
	if config.IsLocal() && targetHost == "" {
		targetHost = "localhost"
	}

	if err := uc.executor.Connect(config); err != nil {
		return nil, fmt.Errorf("error conectando a %s: %w", targetHost, err)
	}

	resMetrics, err := uc.executor.RunCommand(hostMetricsCommand)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo métricas del host: %w", err)
	}
	if resMetrics == nil {
		return &domain.HostMetrics{}, nil
	}

	metrics := ParseHostMetrics(resMetrics.Output)
	return &metrics, nil
}

// ExecuteServicesOnly extrae exclusivamente los servicios del sistema sin ejecutar telemetría de hardware.
func (uc *InspectHostUseCase) ExecuteServicesOnly(config domain.ServerConfig) ([]domain.HostSystemService, error) {
	targetHost := strings.TrimSpace(config.Host)
	if config.IsLocal() && targetHost == "" {
		targetHost = "localhost"
	}

	if err := uc.executor.Connect(config); err != nil {
		return nil, fmt.Errorf("error conectando a %s: %w", targetHost, err)
	}

	resServices, err := uc.executor.RunCommand(hostServicesCommand)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo servicios del host: %w", err)
	}
	if resServices == nil {
		return []domain.HostSystemService{}, nil
	}

	return ParseHostServices(resServices.Output), nil
}

// ParseHostMetrics convierte la salida estructurada de métricas a domain.HostMetrics.
func ParseHostMetrics(raw string) domain.HostMetrics {
	metrics := domain.HostMetrics{}
	lines := strings.Split(raw, "\n")

	var diskUsedMB, diskTotalMB int64

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "OS":
			metrics.OS = val
		case "HOST":
			metrics.Hostname = val
		case "CORES":
			cores, err := strconv.Atoi(val)
			if err == nil {
				metrics.CPUCores = cores
			}
		case "CPU_PCT":
			pct, err := strconv.ParseFloat(val, 64)
			if err == nil {
				metrics.CPUPercent = roundFloat(pct, 1)
			}
		case "MEM_USED_MB":
			memUsed, err := strconv.ParseInt(val, 10, 64)
			if err == nil {
				metrics.MemoryUsedMB = memUsed
			}
		case "MEM_TOTAL_MB":
			memTotal, err := strconv.ParseInt(val, 10, 64)
			if err == nil {
				metrics.MemoryTotalMB = memTotal
			}
		case "DISK_USED_MB":
			dUsed, err := strconv.ParseInt(val, 10, 64)
			if err == nil {
				diskUsedMB = dUsed
			}
		case "DISK_TOTAL_MB":
			dTotal, err := strconv.ParseInt(val, 10, 64)
			if err == nil {
				diskTotalMB = dTotal
			}
		case "LOAD":
			metrics.LoadAvg = val
		case "UPTIME":
			metrics.Uptime = val
		}
	}

	// Cálculo de porcentajes y conversión a GB
	if metrics.MemoryTotalMB > 0 {
		metrics.MemoryPercent = roundFloat((float64(metrics.MemoryUsedMB)/float64(metrics.MemoryTotalMB))*100.0, 1)
	}

	if diskTotalMB > 0 {
		metrics.DiskUsedGB = roundFloat(float64(diskUsedMB)/1024.0, 2)
		metrics.DiskTotalGB = roundFloat(float64(diskTotalMB)/1024.0, 2)
		metrics.DiskPercent = roundFloat((float64(diskUsedMB)/float64(diskTotalMB))*100.0, 1)
	}

	return metrics
}

// ParseHostServices procesa la salida de systemctl o launchctl y la mapea a domain.HostSystemService.
func ParseHostServices(raw string) []domain.HostSystemService {
	services := make([]domain.HostSystemService, 0)
	lines := strings.Split(raw, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		name := fields[0]
		load := fields[1]
		active := fields[2]
		sub := fields[3]
		desc := ""
		if len(fields) >= 5 {
			desc = strings.Join(fields[4:], " ")
		}

		services = append(services, domain.HostSystemService{
			Name:        name,
			LoadState:   load,
			ActiveState: active,
			SubState:    sub,
			Description: desc,
		})
	}

	return services
}

func roundFloat(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
