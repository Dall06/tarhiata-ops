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
		// Evasiones que antes pasaban la blocklist y ahora deben bloquearse.
		{name: "rm con espacios extra evade el match exacto anterior", input: "rm   -rf    /", expected: false},
		{name: "rm con tabs/newlines internos", input: "rm -rf\t/", expected: false},
		{name: "rm -fr (orden de flags invertido)", input: "rm -fr /", expected: false},
		{name: "rm con flags largos", input: "rm --recursive --force /", expected: false},
		{name: "poweroff", input: "poweroff", expected: false},
		{name: "halt", input: "halt", expected: false},
		{name: "init 6", input: "init 6", expected: false},
		{name: "dd of=/dev/sda", input: "dd of=/dev/sda", expected: false},
		{name: "redirección directa a un disco", input: "echo x > /dev/sda1", expected: false},
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

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "cadena simple", input: "hola", expected: "'hola'"},
		{name: "comilla simple embebida", input: "it's", expected: `'it'\''s'`},
		{name: "metacaracteres de shell no se expanden al usarse", input: "$(rm -rf /); echo `id`", expected: `'$(rm -rf /); echo ` + "`id`" + `'`},
		{name: "cadena vacia", input: "", expected: "''"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ShellQuote(tc.input)
			if got != tc.expected {
				t.Errorf("ShellQuote(%q) = %q; want %q", tc.input, got, tc.expected)
			}
		})
	}
}
