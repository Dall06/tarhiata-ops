package app

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestNewDashboardHandler(t *testing.T) {
	mockRepo := &mocks.MockConfigRepository{}
	handler := NewDashboardHandler(mockRepo)
	if handler == nil {
		t.Fatal("expected non-nil DashboardHandler")
	}
}

func TestRenderDashboard_NilConfig(t *testing.T) {
	mockRepo := &mocks.MockConfigRepository{}
	handler := NewDashboardHandler(mockRepo)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	handler.RenderDashboard(nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if output == "" {
		t.Errorf("expected dashboard output, got empty")
	}
}

func TestRenderDashboard_WithConfig(t *testing.T) {
	mockRepo := &mocks.MockConfigRepository{
		Services: []domain.SavedService{
			{Name: "app1"},
		},
		Databases: []domain.SavedDatabase{
			{Name: "db1"},
		},
	}
	handler := NewDashboardHandler(mockRepo)

	config := &domain.ServerConfig{
		Host:          "1.2.3.4",
		User:          "root",
		CloudProvider: "vultr",
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	handler.RenderDashboard(config)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if output == "" {
		t.Errorf("expected dashboard output, got empty")
	}
}
