package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockFlusherResponseWriter struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (m *mockFlusherResponseWriter) Flush() {
	m.flushed = true
}

func TestJSON(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		payload    any
	}{
		{
			name:       "status 200 with map",
			statusCode: http.StatusOK,
			payload:    map[string]string{"status": "ok"},
		},
		{
			name:       "status 201 with struct",
			statusCode: http.StatusCreated,
			payload:    struct{ ID int }{ID: 10},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			if err := JSON(rec, tc.statusCode, tc.payload); err != nil {
				t.Fatalf("JSON returned error: %v", err)
			}
			if rec.Code != tc.statusCode {
				t.Errorf("got status %d; want %d", rec.Code, tc.statusCode)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("got Content-Type %s; want application/json", ct)
			}
		})
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
	}{
		{
			name:       "400 bad request",
			statusCode: http.StatusBadRequest,
			message:    "invalid input",
		},
		{
			name:       "500 internal server error",
			statusCode: http.StatusInternalServerError,
			message:    "database connection failure",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			if err := Error(rec, tc.statusCode, tc.message); err != nil {
				t.Fatalf("Error returned error: %v", err)
			}
			if rec.Code != tc.statusCode {
				t.Errorf("got status %d; want %d", rec.Code, tc.statusCode)
			}
			var body map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode response body: %v", err)
			}
			if body["error"] != tc.message {
				t.Errorf("got message %q; want %q", body["error"], tc.message)
			}
		})
	}
}

func TestStreaming(t *testing.T) {
	rec := httptest.NewRecorder()
	mockWriter := &mockFlusherResponseWriter{ResponseRecorder: rec}

	flusher, err := SetupStreaming(mockWriter)
	if err != nil {
		t.Fatalf("SetupStreaming failed: %v", err)
	}

	if err := WriteEvent(mockWriter, flusher, "info", "compiling app"); err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	if err := WriteDone(mockWriter, flusher, map[string]string{"id": "app-1"}); err != nil {
		t.Fatalf("WriteDone failed: %v", err)
	}

	output := rec.Body.String()
	if !strings.Contains(output, `"t":"info"`) || !strings.Contains(output, `"m":"compiling app"`) {
		t.Errorf("output missing event info: %s", output)
	}
	if !strings.Contains(output, `"t":"done"`) || !strings.Contains(output, `"id":"app-1"`) {
		t.Errorf("output missing done payload: %s", output)
	}
}

func TestIsLoopback(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		expected bool
	}{
		{name: "ipv4 loopback without port", addr: "127.0.0.1", expected: true},
		{name: "ipv4 loopback with port", addr: "127.0.0.1:8080", expected: true},
		{name: "ipv6 loopback", addr: "::1", expected: true},
		{name: "ipv6 loopback with port", addr: "[::1]:9000", expected: true},
		{name: "localhost string", addr: "localhost:3000", expected: true},
		{name: "public ipv4", addr: "198.51.100.1:443", expected: false},
		{name: "lan ipv4", addr: "192.168.1.50:80", expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsLoopback(tc.addr)
			if got != tc.expected {
				t.Errorf("IsLoopback(%q) = %v; want %v", tc.addr, got, tc.expected)
			}
		})
	}
}
