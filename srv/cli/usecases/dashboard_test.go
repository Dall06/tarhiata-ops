package usecases

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	sysdomain "github.com/Dall06/tarhiata-ops/srv/sys/domain"
	sysmocks "github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestDashboardHandler_RenderDashboard(t *testing.T) {
	tests := []struct {
		name         string
		config       *sysdomain.ServerConfig
		services     []sysdomain.SavedService
		databases    []sysdomain.SavedDatabase
		expectedText []string
	}{
		{
			name:   "nil config shows offline status",
			config: nil,
			expectedText: []string{
				"Infrastructure",
				"OFFLINE",
				"Not configured",
			},
		},
		{
			name: "active config shows server metrics and stats",
			config: &sysdomain.ServerConfig{
				Host:          "1.2.3.4",
				User:          "root",
				CloudProvider: "vultr",
			},
			services: []sysdomain.SavedService{
				{Name: "app1"},
			},
			databases: []sysdomain.SavedDatabase{
				{Name: "db1"},
			},
			expectedText: []string{
				"Infrastructure",
				"ACTIVE",
				"1.2.3.4",
				"root",
				"vultr",
				"1 running",
				"1 online",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &sysmocks.MockConfigRepository{
				Services:  tt.services,
				Databases: tt.databases,
			}
			handler := NewDashboardHandler(mockRepo)

			oldStdout := os.Stdout
			r, w, errPipe := os.Pipe()
			if errPipe != nil {
				t.Fatalf("failed to create pipe: %v", errPipe)
			}
			os.Stdout = w

			handler.RenderDashboard(tt.config)

			if errClose := w.Close(); errClose != nil {
				t.Fatalf("failed to close pipe writer: %v", errClose)
			}
			os.Stdout = oldStdout

			var buf bytes.Buffer
			if _, errCopy := io.Copy(&buf, r); errCopy != nil {
				t.Fatalf("failed to copy buffer: %v", errCopy)
			}
			output := buf.String()

			for _, exp := range tt.expectedText {
				if !strings.Contains(output, exp) {
					t.Errorf("expected dashboard output to contain %q, got output:\n%s", exp, output)
				}
			}
		})
	}
}
