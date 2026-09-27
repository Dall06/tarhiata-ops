package sshexec

import (
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

func TestRunLocal(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		wantSubstr  string
		wantSuccess bool
	}{
		{
			name:        "echo hello",
			command:     "echo 'hello tarhiata'",
			wantSubstr:  "hello tarhiata",
			wantSuccess: true,
		},
		{
			name:        "exit 0 with output",
			command:     "printf 'status:ok'",
			wantSubstr:  "status:ok",
			wantSuccess: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := RunLocal(tc.command)
			if err != nil && tc.wantSuccess {
				t.Fatalf("RunLocal(%q) unexpected error: %v", tc.command, err)
			}
			if !strings.Contains(res.Output, tc.wantSubstr) {
				t.Errorf("RunLocal(%q) output = %q; want substr %q", tc.command, res.Output, tc.wantSubstr)
			}
			if tc.wantSuccess && res.ExitCode != 0 {
				t.Errorf("RunLocal(%q) exitCode = %d; want 0", tc.command, res.ExitCode)
			}
		})
	}
}

func TestRun_LocalRouting(t *testing.T) {
	cfg := domain.ServerConfig{
		Host:          "127.0.0.1",
		CloudProvider: "local",
	}

	res, err := Run(cfg, "echo 'routed local'")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !strings.Contains(res.Output, "routed local") {
		t.Errorf("expected output to contain 'routed local', got: %s", res.Output)
	}
}
