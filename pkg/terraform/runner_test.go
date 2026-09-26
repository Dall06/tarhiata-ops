package terraform

import (
	"os"
	"testing"
)

func TestDetectEngine(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Detect existing engine in system",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := DetectEngine()
			// If terraform or tofu is in PATH, it should return a non-empty string
			if path != "" {
				if _, err := os.Stat(path); err != nil {
					t.Fatalf("DetectEngine returned invalid path: %s", path)
				}
			}
		})
	}
}

func TestRunner_EngineName(t *testing.T) {
	tests := []struct {
		name     string
		execPath string
		expected string
	}{
		{
			name:     "Detect OpenTofu",
			execPath: "/usr/local/bin/tofu",
			expected: "OpenTofu",
		},
		{
			name:     "Detect Terraform",
			execPath: "/usr/local/bin/terraform",
			expected: "Terraform",
		},
		{
			name:     "Detect OpenTofu in custom path",
			execPath: "/home/user/.config/tarhiata/bin/tofu",
			expected: "OpenTofu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{
				workspace: "/tmp",
				execPath:  tt.execPath,
			}
			if got := r.EngineName(); got != tt.expected {
				t.Errorf("EngineName() = %v, want %v", got, tt.expected)
			}
		})
	}
}
