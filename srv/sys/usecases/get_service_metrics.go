package usecases

import (
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type MetricPoint = domain.MetricPoint
type ServiceMetrics = domain.ServiceMetrics

type GetServiceMetricsUseCase struct {
	repo ports.ConfigRepository
	ssh  ports.SSHExecutor
}

func NewGetServiceMetricsUseCase(repo ports.ConfigRepository, ssh ports.SSHExecutor) *GetServiceMetricsUseCase {
	return &GetServiceMetricsUseCase{repo: repo, ssh: ssh}
}

func generatePoints(count int, step time.Duration, now time.Time, isOnline bool) []MetricPoint {
	points := make([]MetricPoint, count)
	if !isOnline {
		for i := 0; i < count; i++ {
			t := now.Add(-time.Duration(count-1-i) * step)
			points[i] = MetricPoint{
				Timestamp: t.Format("15:04"),
				CPU:       0.0,
				Memory:    0.0,
				Network:   0.0,
				Disk:      0.0,
			}
		}
		return points
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	baseCPU := 2.5 + float64(r.Intn(15))
	baseMem := 120.0 + float64(r.Intn(200))

	for i := 0; i < count; i++ {
		t := now.Add(-time.Duration(count-1-i) * step)
		variation := math.Sin(float64(i)*0.4) * 3.0
		cpu := math.Max(0.5, math.Min(98.0, baseCPU+variation+float64(r.Intn(5))))
		mem := math.Max(30.0, baseMem+(variation*5.0)+float64(r.Intn(10)))
		net := math.Max(1.0, 15.0+math.Cos(float64(i)*0.5)*10.0)
		disk := math.Max(0.1, 2.0+math.Sin(float64(i)*0.8)*1.5)

		points[i] = MetricPoint{
			Timestamp: t.Format("15:04"),
			CPU:       math.Round(cpu*10) / 10,
			Memory:    math.Round(mem*10) / 10,
			Network:   math.Round(net*10) / 10,
			Disk:      math.Round(disk*10) / 10,
		}
	}
	return points
}

type DockerContainerStats struct {
	Name string
	CPU  float64
	Mem  float64
	Net  float64
	Disk float64
}

func parseHumanBytes(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return 0.0
	}
	s = strings.ToUpper(s)

	var valStr string
	var unit string
	for i, r := range s {
		if (r < '0' || r > '9') && r != '.' && r != '-' {
			valStr = s[:i]
			unit = strings.TrimSpace(s[i:])
			break
		}
	}
	if valStr == "" {
		valStr = s
	}

	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0.0
	}

	switch unit {
	case "B":
		return val / 1024.0
	case "KB", "KIB":
		return val
	case "MB", "MIB":
		return val * 1024.0
	case "GB", "GIB":
		return val * 1024.0 * 1024.0
	case "TB", "TIB":
		return val * 1024.0 * 1024.0 * 1024.0
	default:
		return val / 1024.0
	}
}

func parseDockerStatsLine(line string) (DockerContainerStats, error) {
	parts := strings.Split(line, "\t")
	if len(parts) < 5 {
		return DockerContainerStats{}, fmt.Errorf("invalid stats line")
	}
	name := strings.TrimSpace(parts[0])

	cpuStr := strings.TrimSuffix(strings.TrimSpace(parts[1]), "%")
	cpu, err := strconv.ParseFloat(cpuStr, 64)
	if err != nil {
		return DockerContainerStats{}, fmt.Errorf("invalid cpu: %w", err)
	}

	memStr := strings.Split(parts[2], " / ")[0]
	memKB := parseHumanBytes(memStr)

	netStr := strings.Split(parts[3], " / ")[0]
	netKB := parseHumanBytes(netStr)

	diskStr := strings.Split(parts[4], " / ")[0]
	diskKB := parseHumanBytes(diskStr)

	return DockerContainerStats{
		Name: name,
		CPU:  cpu,
		Mem:  memKB / 1024.0,
		Net:  netKB,
		Disk: diskKB / 1024.0,
	}, nil
}

type netIOSample struct {
	cumulativeKB float64
	at           time.Time
}

var (
	netIOCacheMu sync.Mutex
	netIOCache   = map[string]netIOSample{}
)

// networkRateKBs convierte el contador acumulado de NetIO de "docker stats" (bytes desde
// que arrancó el contenedor) en una tasa KB/s real, comparando contra la lectura anterior
// cacheada por proceso. La primera lectura de una clave no tiene con qué comparar y
// devuelve 0.
// ponytail: caché global en memoria por proceso; no persiste entre reinicios ni se
// comparte entre instancias de la app en réplica. Suficiente para un dashboard de un solo
// proceso; si se necesita multi-instancia, mover a un store compartido.
func networkRateKBs(key string, cumulativeKB float64, now time.Time) float64 {
	netIOCacheMu.Lock()
	defer netIOCacheMu.Unlock()

	prev, ok := netIOCache[key]
	netIOCache[key] = netIOSample{cumulativeKB: cumulativeKB, at: now}

	if !ok {
		return 0
	}
	elapsed := now.Sub(prev.at).Seconds()
	if elapsed <= 0 || cumulativeKB < prev.cumulativeKB {
		// Reinicio del contenedor (el contador volvió a 0) o muestra fuera de orden.
		return 0
	}
	return (cumulativeKB - prev.cumulativeKB) / elapsed
}

func (uc *GetServiceMetricsUseCase) Execute(serviceName string, timeRange string, config domain.ServerConfig) (ServiceMetrics, error) {
	if timeRange == "" {
		timeRange = "1h"
	}

	now := time.Now()
	var count int
	var step time.Duration

	switch timeRange {
	case "15m":
		count = 15
		step = 1 * time.Minute
	case "6h":
		count = 24
		step = 15 * time.Minute
	case "24h":
		count = 24
		step = 1 * time.Hour
	default:
		count = 20
		step = 3 * time.Minute
	}

	isOnline := false
	var realStats *DockerContainerStats

	if config.Host != "" && uc.ssh != nil {
		if err := uc.ssh.Connect(config); err == nil {
			isOnline = true

			res, cmdErr := uc.ssh.RunCommand("docker stats --no-stream --format '{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}' 2>/dev/null")
			if cmdErr == nil && res != nil && res.Output != "" {
				lines := strings.Split(strings.TrimSpace(res.Output), "\n")

				var totalCPU, totalMem, totalNet, totalDisk float64
				var matchedCount int

				for _, line := range lines {
					if strings.TrimSpace(line) == "" {
						continue
					}
					stats, pErr := parseDockerStatsLine(line)
					if pErr != nil {
						continue
					}

					if serviceName == "all" || serviceName == "" || strings.Contains(strings.ToLower(stats.Name), strings.ToLower(serviceName)) {
						totalCPU += stats.CPU
						totalMem += stats.Mem
						totalNet += stats.Net
						totalDisk += stats.Disk
						matchedCount++
					}
				}

				if matchedCount > 0 {
					netRateKey := config.Host + "|" + serviceName
					realStats = &DockerContainerStats{
						CPU:  totalCPU,
						Mem:  totalMem,
						Net:  networkRateKBs(netRateKey, totalNet, now),
						Disk: totalDisk,
					}
				}
			}

			closeErr := uc.ssh.Close()
			if closeErr != nil {
				slog.Warn("error cerrando conexion ssh en métricas", "error", closeErr)
			}
		}
	}

	points := generatePoints(count, step, now, isOnline)

	if realStats != nil && len(points) > 0 {
		lastIdx := len(points) - 1
		points[lastIdx].CPU = math.Round(realStats.CPU*10) / 10
		points[lastIdx].Memory = math.Round(realStats.Mem*10) / 10
		points[lastIdx].Network = math.Round(realStats.Net*10) / 10
		points[lastIdx].Disk = math.Round(realStats.Disk*10) / 10
	}

	return ServiceMetrics{
		ServiceName: serviceName,
		Range:       timeRange,
		Points:      points,
	}, nil
}
