package terminal

import (
	"testing"
)

func TestFormatPrompt(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		container string
		expected  string
	}{
		{
			name:      "host only",
			host:      "prod-vps",
			container: "",
			expected:  "root@prod-vps:~$",
		},
		{
			name:      "container specified",
			host:      "prod-vps",
			container: "traefik",
			expected:  "root@traefik:#",
		},
		{
			name:      "empty host and container",
			host:      "",
			container: "",
			expected:  "root@vps:~$",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatPrompt(tc.host, tc.container)
			if got != tc.expected {
				t.Errorf("FormatPrompt(%q, %q) = %q; want %q", tc.host, tc.container, got, tc.expected)
			}
		})
	}
}

func TestSanitizeTerminalCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "with spaces", input: "  docker ps  ", expected: "docker ps"},
		{name: "empty", input: "   ", expected: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeTerminalCommand(tc.input)
			if got != tc.expected {
				t.Errorf("SanitizeTerminalCommand(%q) = %q; want %q", tc.input, got, tc.expected)
			}
		})
	}
}
