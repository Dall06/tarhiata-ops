package usecases

import (
	"testing"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestGetServiceMetricsUseCase_Execute(t *testing.T) {
	ssh := mocks.NewMockSSHExecutor()
	uc := NewGetServiceMetricsUseCase(nil, ssh)

	metrics, err := uc.Execute("web-app", "1h", domain.ServerConfig{Host: "1.2.3.4"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metrics.ServiceName != "web-app" {
		t.Errorf("expected service name 'web-app', got: %s", metrics.ServiceName)
	}

	if len(metrics.Points) == 0 {
		t.Fatalf("expected metric points, got 0")
	}

	for _, pt := range metrics.Points {
		if pt.CPU < 0 || pt.Memory < 0 {
			t.Errorf("invalid metric point values: %+v", pt)
		}
	}
}

func TestParseHumanBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{name: "kilobytes", input: "512KiB", expected: 512},
		{name: "megabytes a KB", input: "1MiB", expected: 1024},
		{name: "gigabytes a KB", input: "1GiB", expected: 1024 * 1024},
		{name: "bytes a KB", input: "2048B", expected: 2},
		{name: "guion es cero", input: "-", expected: 0},
		{name: "vacio es cero", input: "", expected: 0},
		{name: "sin unidad asume bytes", input: "1024", expected: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseHumanBytes(tc.input)
			if got != tc.expected {
				t.Errorf("parseHumanBytes(%q) = %v; want %v", tc.input, got, tc.expected)
			}
		})
	}
}

// TestParseDockerStatsLine_DiskUnitConversion valida el flujo completo del parseo de una
// línea real de "docker stats": Mem y Disk deben quedar en MB (dividiendo KB entre 1024),
// nunca en KB crudo, para que el dashboard no muestre el disco ~1024x inflado.
func TestParseDockerStatsLine_DiskUnitConversion(t *testing.T) {
	line := "web-app\t12.50%\t256MiB / 512MiB\t10MiB / 5MiB\t100MiB / 50MiB"

	stats, err := parseDockerStatsLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.CPU != 12.5 {
		t.Errorf("expected CPU 12.5, got %v", stats.CPU)
	}
	if stats.Mem != 256 {
		t.Errorf("expected Mem 256 MB, got %v", stats.Mem)
	}
	if stats.Disk != 100 {
		t.Errorf("expected Disk 100 MB (no 102400 KB sin convertir), got %v", stats.Disk)
	}
}

// TestNetworkRateKBs valida el flujo completo del cálculo de tasa de red: "docker stats"
// entrega un contador acumulado (NetIO desde que arrancó el contenedor), no una tasa; la
// primera lectura no tiene con qué comparar, la segunda debe dar KB/s reales, y un
// contador que retrocede (reinicio del contenedor) no debe producir una tasa negativa.
func TestNetworkRateKBs(t *testing.T) {
	key := "TestNetworkRateKBs-key"
	t0 := time.Now()

	first := networkRateKBs(key, 1000, t0)
	if first != 0 {
		t.Errorf("primera lectura: se esperaba tasa 0 (sin punto de comparación), got %v", first)
	}

	second := networkRateKBs(key, 1500, t0.Add(5*time.Second))
	if second != 100 {
		t.Errorf("segunda lectura: se esperaba (1500-1000)/5s = 100 KB/s, got %v", second)
	}

	reset := networkRateKBs(key, 200, t0.Add(6*time.Second))
	if reset != 0 {
		t.Errorf("contador reiniciado (menor al anterior): se esperaba tasa 0, got %v", reset)
	}
}
