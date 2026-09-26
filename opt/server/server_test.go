package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestNewServer_TableDriven(t *testing.T) {
	tests := []struct {
		name                 string
		cfg                  Config
		expectedReadTimeout  time.Duration
		expectedWriteTimeout time.Duration
		expectedIdleTimeout  time.Duration
	}{
		{
			name: "Default timeouts applied when zero values provided",
			cfg: Config{
				Port: 8080,
			},
			expectedReadTimeout:  DefaultReadTimeout,
			expectedWriteTimeout: DefaultWriteTimeout,
			expectedIdleTimeout:  DefaultIdleTimeout,
		},
		{
			name: "Custom timeouts preserved when specified",
			cfg: Config{
				Port:         9090,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 7 * time.Second,
				IdleTimeout:  30 * time.Second,
			},
			expectedReadTimeout:  5 * time.Second,
			expectedWriteTimeout: 7 * time.Second,
			expectedIdleTimeout:  30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New(tt.cfg)
			if e == nil {
				t.Fatal("expected non-nil Echo instance")
			}
			if e.Server.ReadTimeout != tt.expectedReadTimeout {
				t.Errorf("expected ReadTimeout=%v, got %v", tt.expectedReadTimeout, e.Server.ReadTimeout)
			}
			if e.Server.WriteTimeout != tt.expectedWriteTimeout {
				t.Errorf("expected WriteTimeout=%v, got %v", tt.expectedWriteTimeout, e.Server.WriteTimeout)
			}
			if e.Server.IdleTimeout != tt.expectedIdleTimeout {
				t.Errorf("expected IdleTimeout=%v, got %v", tt.expectedIdleTimeout, e.Server.IdleTimeout)
			}

			// Validate perimeter middleware: security headers
			e.GET("/ping", func(c echo.Context) error {
				return c.String(http.StatusOK, "pong")
			})

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected 200 OK, got %d", rec.Code)
			}
			if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Errorf("expected X-Content-Type-Options: nosniff, got %s", rec.Header().Get("X-Content-Type-Options"))
			}
			if rec.Header().Get("X-Frame-Options") != "DENY" {
				t.Errorf("expected X-Frame-Options: DENY, got %s", rec.Header().Get("X-Frame-Options"))
			}
		})
	}
}

func TestStartGraceful_ShutdownLifecycle(t *testing.T) {
	e := New(Config{})
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "healthy")
	})

	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error, 1)
	go func() {
		errChan <- StartGracefulWithContext(ctx, e, "127.0.0.1:0", 2*time.Second)
	}()

	// Trigger graceful shutdown by canceling context
	cancel()

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("expected clean graceful shutdown without error, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for graceful shutdown to finish")
	}
}

