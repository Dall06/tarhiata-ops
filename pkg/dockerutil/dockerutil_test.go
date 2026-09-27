package dockerutil

import (
	"testing"
)

func TestIsDockerError(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected bool
	}{
		{name: "no such container", output: "Error response from daemon: No such container: c1", expected: true},
		{name: "invalid service name", output: "invalid service name: ---", expected: true},
		{name: "successful output", output: "tarhiata-app-1\ntarhiata-app-2", expected: false},
		{name: "empty output", output: "", expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsDockerError(tc.output)
			if got != tc.expected {
				t.Errorf("IsDockerError(%q) = %v; want %v", tc.output, got, tc.expected)
			}
		})
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		expected string
	}{
		{name: "Host rule", rule: "Host(`api.example.com`)", expected: "api.example.com"},
		{name: "Host rule with headers", rule: "Host(`app.dev.io`) && Headers(`X-Auth`, `1`)", expected: "app.dev.io"},
		{name: "PathPrefix rule", rule: "PathPrefix(`/v1/health`)", expected: "/v1/health"},
		{name: "No matching rule", rule: "Method(`GET`)", expected: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractDomain(tc.rule)
			if got != tc.expected {
				t.Errorf("ExtractDomain(%q) = %q; want %q", tc.rule, got, tc.expected)
			}
		})
	}
}

func TestBuildSafeURI(t *testing.T) {
	tests := []struct {
		name        string
		engine      string
		serviceName string
		port        int
		expected    string
	}{
		{
			name:        "postgres uri",
			engine:      "postgres",
			serviceName: "tarhiata-db-postgres-prod",
			port:        5432,
			expected:    "postgres://admin:********@tarhiata-db-postgres-prod:5432/db",
		},
		{
			name:        "mongo uri",
			engine:      "mongodb",
			serviceName: "tarhiata-db-mongo-app",
			port:        27017,
			expected:    "mongodb://admin:********@tarhiata-db-mongo-app:27017/?authSource=admin",
		},
		{
			name:        "redis uri",
			engine:      "redis",
			serviceName: "tarhiata-db-cache",
			port:        6379,
			expected:    "redis://:********@tarhiata-db-cache:6379",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildSafeURI(tc.engine, tc.serviceName, tc.port)
			if got != tc.expected {
				t.Errorf("BuildSafeURI(%q, %q, %d) = %q; want %q", tc.engine, tc.serviceName, tc.port, got, tc.expected)
			}
		})
	}
}
