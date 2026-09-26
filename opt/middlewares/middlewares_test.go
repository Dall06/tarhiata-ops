package middlewares

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRequestLogger_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		handler        echo.HandlerFunc
		expectedStatus int
		expectError    bool
	}{
		{
			name: "Successful request logged at 200",
			handler: func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "Client error logged at 400",
			handler: func(c echo.Context) error {
				return c.String(http.StatusBadRequest, "bad request")
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    false,
		},
		{
			name: "Server error returned with error",
			handler: func(c echo.Context) error {
				return errors.New("falla interna")
			},
			expectedStatus: http.StatusOK,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			loggerMw := RequestLogger()(tt.handler)
			err := loggerMw(c)

			if (err != nil) != tt.expectError {
				t.Fatalf("expected error=%v, got err=%v", tt.expectError, err)
			}
			if !tt.expectError && rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestSecurityPerimeter_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		targetURL      string
		expectedStatus int
		blocked        bool
	}{
		{
			name:           "Valid API path is allowed through",
			targetURL:      "/api/services",
			expectedStatus: http.StatusOK,
			blocked:        false,
		},
		{
			name:           "Path traversal with dot-dot-slash in path blocked with 400",
			targetURL:      "/api/volumes/read?path=../../etc/shadow",
			expectedStatus: http.StatusBadRequest,
			blocked:        true,
		},
		{
			name:           "Path traversal with encoded dots in query blocked with 400",
			targetURL:      "/api/volumes/read?path=%2e%2e/opt/data",
			expectedStatus: http.StatusBadRequest,
			blocked:        true,
		},
		{
			name:           "Direct /etc/passwd path blocked with 400",
			targetURL:      "/api/volumes/read?path=/etc/passwd",
			expectedStatus: http.StatusBadRequest,
			blocked:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, tt.targetURL, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			dummyHandler := func(ctx echo.Context) error {
				return ctx.String(http.StatusOK, "allowed")
			}

			mw := SecurityPerimeter()(dummyHandler)
			err := mw(c)
			if err != nil {
				t.Fatalf("unexpected handler error: %v", err)
			}

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}
