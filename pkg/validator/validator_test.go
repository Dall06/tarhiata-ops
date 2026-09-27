package validator

import (
	"testing"
)

func TestIsIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "valid simple identifier", input: "api-backend", expected: true},
		{name: "valid alphanumeric with dot", input: "web.service_1", expected: true},
		{name: "empty string", input: "", expected: false},
		{name: "injection characters with spaces", input: "api; rm -rf /", expected: false},
		{name: "special characters", input: "app$(whoami)", expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsIdentifier(tc.input)
			if got != tc.expected {
				t.Errorf("IsIdentifier(%q) = %v; want %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestIsNodeID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "valid swarm node id", input: "246813579bdfhjlnprtvxz", expected: true},
		{name: "valid id with hyphen", input: "node-worker-01", expected: true},
		{name: "empty id", input: "", expected: false},
		{name: "id with invalid characters", input: "node/worker", expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsNodeID(tc.input)
			if got != tc.expected {
				t.Errorf("IsNodeID(%q) = %v; want %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestIsSafeCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "safe docker ps", input: "docker ps", expected: true},
		{name: "safe logs command", input: "docker logs my-app", expected: true},
		{name: "destructive rm root", input: "rm -rf /", expected: false},
		{name: "destructive reboot", input: "sudo reboot now", expected: false},
		{name: "destructive fork bomb", input: ":(){ :|:& };:", expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsSafeCommand(tc.input)
			if got != tc.expected {
				t.Errorf("IsSafeCommand(%q) = %v; want %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestNormalizeRegion(t *testing.T) {
	tests := []struct {
		name      string
		provider  string
		region    string
		expected  string
		shouldErr bool
	}{
		{name: "vultr mexico alias", provider: "vultr", region: "cdmx", expected: "mex", shouldErr: false},
		{name: "vultr amsterdam alias", provider: "vultr", region: "ams2", expected: "ams", shouldErr: false},
		{name: "do nyc alias", provider: "digitalocean", region: "newyork", expected: "nyc1", shouldErr: false},
		{name: "do custom region", provider: "do", region: "sfo3", expected: "sfo3", shouldErr: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeRegion(tc.provider, tc.region)
			if (err != nil) != tc.shouldErr {
				t.Fatalf("NormalizeRegion(%q, %q) error = %v; shouldErr = %v", tc.provider, tc.region, err, tc.shouldErr)
			}
			if got != tc.expected {
				t.Errorf("got %q; want %q", got, tc.expected)
			}
		})
	}
}
