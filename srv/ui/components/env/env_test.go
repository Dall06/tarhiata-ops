package env

import (
	"testing"
)

func TestParseEnvFile(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected map[string]string
	}{
		{
			name: "standard env",
			raw:  "PORT=8080\nHOST=localhost\n# comment\nDB_PASS=\"secret123\"",
			expected: map[string]string{
				"PORT":    "8080",
				"HOST":    "localhost",
				"DB_PASS": "secret123",
			},
		},
		{
			name:     "empty env",
			raw:      "",
			expected: map[string]string{},
		},
		{
			name: "single quotes",
			raw:  "API_KEY='xyz-999'",
			expected: map[string]string{
				"API_KEY": "xyz-999",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseEnvFile(tc.raw)
			if len(got) != len(tc.expected) {
				t.Fatalf("ParseEnvFile() len = %d; want %d", len(got), len(tc.expected))
			}
			for k, wantVal := range tc.expected {
				if got[k] != wantVal {
					t.Errorf("key %q = %q; want %q", k, got[k], wantVal)
				}
			}
		})
	}
}

func TestFormatEnvFile(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected string
	}{
		{
			name: "alphabetical formatting",
			input: map[string]string{
				"PORT": "8080",
				"HOST": "localhost",
			},
			expected: "HOST=localhost\nPORT=8080",
		},
		{
			name:     "empty map",
			input:    map[string]string{},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatEnvFile(tc.input)
			if got != tc.expected {
				t.Errorf("FormatEnvFile() = %q; want %q", got, tc.expected)
			}
		})
	}
}
