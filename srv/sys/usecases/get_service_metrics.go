package usecases

import (
	"math"
	"math/rand"
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
	default: // "1h"
		count = 20
		step = 3 * time.Minute
	}

	// Verify if target VM is online via SSH connection check
	isOnline := false
	if config.Host != "" && uc.ssh != nil {
		if err := uc.ssh.Connect(config); err == nil {
			isOnline = true
			uc.ssh.Close()
		}
	}

	points := generatePoints(count, step, now, isOnline)

	return ServiceMetrics{
		ServiceName: serviceName,
		Range:       timeRange,
		Points:      points,
	}, nil
}
