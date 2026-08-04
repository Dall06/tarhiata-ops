package usecases

import (
	"testing"

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
