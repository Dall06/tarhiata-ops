package volumes

import (
	"testing"
)

func TestSanitizeVolumePath(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		base     string
		expected string
	}{
		{
			name:     "valid subpath",
			raw:      "/opt/data/traefik/acme.json",
			base:     "/opt/data",
			expected: "/opt/data/traefik/acme.json",
		},
		{
			name:     "path traversal attempt",
			raw:      "/opt/data/../../etc/passwd",
			base:     "/opt/data",
			expected: "/opt/data",
		},
		{
			name:     "empty base defaults to /opt/data",
			raw:      "/opt/data/app",
			base:     "",
			expected: "/opt/data/app",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeVolumePath(tc.raw, tc.base)
			if got != tc.expected {
				t.Errorf("SanitizeVolumePath(%q, %q) = %q; want %q", tc.raw, tc.base, got, tc.expected)
			}
		})
	}
}
