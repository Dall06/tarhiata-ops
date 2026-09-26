package repositories

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Dall06/tarhiata-ops/pkg/sshclient"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

// CryptoSSHExecutor implementa la interfaz ports.SSHExecutor utilizando pkg/sshclient para hosts remotos y ejecución nativa para local.
type CryptoSSHExecutor struct {
	client  *sshclient.Client
	isLocal bool
	config  domain.ServerConfig
}

// NewCryptoSSHExecutor crea una nueva instancia del adaptador
func NewCryptoSSHExecutor() *CryptoSSHExecutor {
	return &CryptoSSHExecutor{
		client: sshclient.New(),
	}
}

// Connect establece la conexión SSH segura o inicializa el ejecutor local.
func (e *CryptoSSHExecutor) Connect(config domain.ServerConfig) error {
	e.config = config
	if config.IsLocal() {
		e.isLocal = true
		cmd := exec.Command("sh", "-c", "echo ok")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error verificando entorno local: %w (salida: %s)", err, string(out))
		}
		return nil
	}
	e.isLocal = false
	return e.client.Connect(config.Host, config.User, config.PrivateKey, config.Port)
}

// RunCommand ejecuta un comando de forma síncrona y captura la salida.
func (e *CryptoSSHExecutor) RunCommand(cmd string) (*domain.CommandResult, error) {
	if e.isLocal {
		execCmd := exec.Command("sh", "-c", cmd)
		out, err := execCmd.CombinedOutput()
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		}
		result := &domain.CommandResult{
			Output:   string(out),
			ExitCode: exitCode,
			Error:    err,
		}
		return result, nil
	}

	out, exitCode, err := e.client.RunCommand(cmd)
	result := &domain.CommandResult{
		Output:   out,
		ExitCode: exitCode,
		Error:    err,
	}
	return result, nil
}

// InteractiveShell abre una consola PTY interactiva conectada a la terminal del usuario.
func (e *CryptoSSHExecutor) InteractiveShell() error {
	if e.isLocal {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		cmd := exec.Command(shell)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return e.client.InteractiveShell()
}

// InteractiveCommand ejecuta un comando específico con stdio interactivo conectado.
func (e *CryptoSSHExecutor) InteractiveCommand(cmd string) error {
	if e.isLocal {
		execCmd := exec.Command("sh", "-c", cmd)
		execCmd.Stdin = os.Stdin
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		return execCmd.Run()
	}
	return e.client.InteractiveCommand(cmd)
}

// CheckConnection verifica si la conexión o el entorno local siguen vivos.
func (e *CryptoSSHExecutor) CheckConnection() bool {
	if e.isLocal {
		cmd := exec.Command("sh", "-c", "echo 1")
		if err := cmd.Run(); err != nil {
			return false
		}
		return true
	}
	return e.client.CheckConnection()
}

// Close finaliza la conexión.
func (e *CryptoSSHExecutor) Close() error {
	if e.isLocal {
		return nil
	}
	return e.client.Close()
}

// WriteRemoteFile escribe un archivo en el servidor remoto o localmente de forma segura.
func (e *CryptoSSHExecutor) WriteRemoteFile(remotePath, content string) error {
	if e.isLocal {
		dir := filepath.Dir(remotePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("falló al crear directorio local %s: %w", dir, err)
		}
		if err := os.WriteFile(remotePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("falló al escribir archivo local %s: %w", remotePath, err)
		}
		return nil
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	safePath := fmt.Sprintf("%q", remotePath)
	cmd := fmt.Sprintf("echo '%s' | base64 -d > %s", encoded, safePath)
	res, err := e.RunCommand(cmd)
	if err != nil {
		return fmt.Errorf("falló al escribir archivo %s: %w", remotePath, err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("falló al escribir archivo %s: %s", remotePath, res.Output)
	}
	return nil
}
