package repositories

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/pkg/sshclient"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

func TestCryptoSSHExecutor_LocalExecution(t *testing.T) {
	exec := NewCryptoSSHExecutor()

	localCfg := domain.ServerConfig{
		Host: "localhost",
	}

	if err := exec.Connect(localCfg); err != nil {
		t.Fatalf("unexpected error connecting to local: %v", err)
	}

	if !exec.CheckConnection() {
		t.Error("expected CheckConnection to be true for local")
	}

	// Test RunCommand
	tests := []struct {
		name         string
		cmd          string
		expectedOut  string
		expectedCode int
	}{
		{
			name:         "echo test",
			cmd:          "echo hello_tarhiata",
			expectedOut:  "hello_tarhiata",
			expectedCode: 0,
		},
		{
			name:         "command with non-zero exit code",
			cmd:          "exit 42",
			expectedOut:  "",
			expectedCode: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := exec.RunCommand(tt.cmd)
			if err != nil && tt.expectedCode == 0 {
				t.Errorf("unexpected error running command: %v", err)
			}
			if res.ExitCode != tt.expectedCode {
				t.Errorf("expected exit code %d, got %d", tt.expectedCode, res.ExitCode)
			}
			if tt.expectedOut != "" && !strings.Contains(res.Output, tt.expectedOut) {
				t.Errorf("expected output to contain %q, got %q", tt.expectedOut, res.Output)
			}
		})
	}

	// Test WriteRemoteFile in local mode
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "sub", "test.txt")
	content := "tarhiata_local_file_content"

	if err := exec.WriteRemoteFile(targetFile, content); err != nil {
		t.Fatalf("unexpected error writing local file: %v", err)
	}

	readBack, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("unexpected error reading written file: %v", err)
	}
	if string(readBack) != content {
		t.Errorf("expected content %q, got %q", content, string(readBack))
	}

	if err := exec.Close(); err != nil {
		t.Errorf("unexpected error closing executor: %v", err)
	}
}

func TestCryptoSSHExecutor_PooledExecution(t *testing.T) {
	dialCount := 0
	sshclient.GlobalPool.SetDialer(func(host, user, privateKeyPath string, port int) (*sshclient.Client, error) {
		dialCount++
		c := sshclient.New()
		c.SetMockConnected(true)
		return c, nil
	})

	remoteCfg := domain.ServerConfig{
		Host: "192.168.1.100",
		User: "root",
		Port: 22,
	}

	exec1 := NewCryptoSSHExecutor()
	if err := exec1.Connect(remoteCfg); err != nil {
		t.Fatalf("unexpected error connecting exec1: %v", err)
	}

	if !exec1.CheckConnection() {
		t.Error("expected CheckConnection to be true")
	}

	exec2 := NewCryptoSSHExecutor()
	if err := exec2.Connect(remoteCfg); err != nil {
		t.Fatalf("unexpected error connecting exec2: %v", err)
	}

	if dialCount != 1 {
		t.Errorf("expected 1 dial for pooled connections, got %d", dialCount)
	}

	if err := exec1.Close(); err != nil {
		t.Errorf("unexpected error on exec1 Close: %v", err)
	}
	if err := exec2.Close(); err != nil {
		t.Errorf("unexpected error on exec2 Close: %v", err)
	}
}

// TestCryptoSSHExecutor_RunCommand_PropagatesTransportError valida el flujo completo:
// una caída real de transporte SSH (sesión no disponible) debe propagarse como el error
// de Go de RunCommand, no quedar silenciada (antes SIEMPRE se devolvía nil ahí, dejando
// mensajes de error vacíos en decenas de usecases ante una caída de conexión real).
func TestCryptoSSHExecutor_RunCommand_PropagatesTransportError(t *testing.T) {
	sshclient.GlobalPool.SetDialer(func(host, user, privateKeyPath string, port int) (*sshclient.Client, error) {
		c := sshclient.New()
		c.SetMockConnected(true) // conectado a nivel de pool, pero sin *ssh.Client real subyacente
		return c, nil
	})

	exec := NewCryptoSSHExecutor()
	if err := exec.Connect(domain.ServerConfig{Host: "192.168.1.200", User: "root", Port: 22}); err != nil {
		t.Fatalf("unexpected error connecting: %v", err)
	}
	defer exec.Close()

	res, err := exec.RunCommand("echo hi")
	if err == nil {
		t.Fatal("se esperaba un error de transporte real, se obtuvo nil")
	}
	if res == nil || res.ExitCode != -1 {
		t.Errorf("se esperaba ExitCode -1 para un fallo de transporte, obtenido: %+v", res)
	}
}

// TestCryptoSSHExecutor_LocalExecution_NonZeroExitDoesNotPropagateAsGoError confirma
// que el comportamiento existente (exit code != 0 no es un error de Go) no se rompió.
func TestCryptoSSHExecutor_LocalExecution_NonZeroExitDoesNotPropagateAsGoError(t *testing.T) {
	exec := NewCryptoSSHExecutor()
	if err := exec.Connect(domain.ServerConfig{Host: "localhost"}); err != nil {
		t.Fatalf("unexpected error connecting to local: %v", err)
	}
	defer exec.Close()

	res, err := exec.RunCommand("exit 7")
	if err != nil {
		t.Errorf("un exit code != 0 no debería propagarse como error de Go, obtenido: %v", err)
	}
	if res.ExitCode != 7 {
		t.Errorf("expected exit code 7, got %d", res.ExitCode)
	}
}

// TestCryptoSSHExecutor_RunCommandStreaming_Local valida el flujo completo en modo
// local: cada línea producida por el comando debe llegar a onLine a medida que se
// produce (no solo al final en el Output acumulado).
func TestCryptoSSHExecutor_RunCommandStreaming_Local(t *testing.T) {
	exec := NewCryptoSSHExecutor()
	if err := exec.Connect(domain.ServerConfig{Host: "localhost"}); err != nil {
		t.Fatalf("unexpected error connecting to local: %v", err)
	}
	defer exec.Close()

	var gotLines []string
	res, err := exec.RunCommandStreaming("printf 'uno\\ndos\\ntres\\n'", func(line string) {
		gotLines = append(gotLines, line)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", res.ExitCode)
	}
	want := []string{"uno", "dos", "tres"}
	if len(gotLines) != len(want) {
		t.Fatalf("se esperaban %d líneas, got %v", len(want), gotLines)
	}
	for i, w := range want {
		if gotLines[i] != w {
			t.Errorf("línea %d: got %q, want %q", i, gotLines[i], w)
		}
	}
}
