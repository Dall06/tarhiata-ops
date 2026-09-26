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
	r, w, errPipe := os.Pipe()
	if errPipe != nil {
		t.Fatalf("failed to create pipe: %v", errPipe)
	}
	os.Stdout = w

	handler.RenderDashboard(nil)

	if errClose := w.Close(); errClose != nil {
		t.Fatalf("failed to close pipe writer: %v", errClose)
	}
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, errCopy := io.Copy(&buf, r); errCopy != nil {
		t.Fatalf("failed to copy buffer: %v", errCopy)
	}
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
	r, w, errPipe := os.Pipe()
	if errPipe != nil {
		t.Fatalf("failed to create pipe: %v", errPipe)
	}
	os.Stdout = w

	handler.RenderDashboard(config)

	if errClose := w.Close(); errClose != nil {
		t.Fatalf("failed to close pipe writer: %v", errClose)
	}
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, errCopy := io.Copy(&buf, r); errCopy != nil {
		t.Fatalf("failed to copy buffer: %v", errCopy)
	}
	output := buf.String()

	if output == "" {
		t.Errorf("expected dashboard output, got empty")
	}
}
