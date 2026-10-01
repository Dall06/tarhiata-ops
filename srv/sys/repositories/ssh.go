package repositories

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Dall06/tarhiata-ops/pkg/secutil"
	"github.com/Dall06/tarhiata-ops/pkg/sshclient"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"golang.org/x/crypto/ssh"
)

// CryptoSSHExecutor implementa la interfaz ports.SSHExecutor utilizando pkg/sshclient para hosts remotos y ejecución nativa para local.
type CryptoSSHExecutor struct {
	client  *sshclient.Client
	isLocal bool
	config  domain.ServerConfig
	poolKey string
	pooled  bool
}

// NewCryptoSSHExecutor crea una nueva instancia del adaptador con pooling activo por defecto.
func NewCryptoSSHExecutor() *CryptoSSHExecutor {
	return &CryptoSSHExecutor{
		pooled: true,
	}
}

// NewUnpooledCryptoSSHExecutor crea un ejecutor con conexión dedicada sin pasar por el pool.
func NewUnpooledCryptoSSHExecutor() *CryptoSSHExecutor {
	return &CryptoSSHExecutor{
		client: sshclient.New(),
		pooled: false,
	}
}

// Connect establece la conexión SSH segura o inicializa el ejecutor local.
func (e *CryptoSSHExecutor) Connect(config domain.ServerConfig) error {
	e.config = config
	if secutil.IsLocalHost(config.Host, config.CloudProvider) {
		e.isLocal = true
		cmd := exec.Command("sh", "-c", "echo ok")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error verificando entorno local: %w (salida: %s)", err, string(out))
		}
		return nil
	}
	e.isLocal = false
	if e.pooled {
		if e.poolKey != "" {
			sshclient.GlobalPool.Release(e.poolKey, e.client)
			e.poolKey = ""
		}
		key := sshclient.PoolKey(config.Host, config.User, config.PrivateKey, config.Port)
		client, err := sshclient.GlobalPool.Get(config.Host, config.User, config.PrivateKey, config.Port)
		if err != nil {
			return err
		}
		e.client = client
		e.poolKey = key
		return nil
	}
	if e.client == nil {
		e.client = sshclient.New()
	}
	return e.client.Connect(config.Host, config.User, config.PrivateKey, config.Port)
}

// RunCommand ejecuta un comando de forma síncrona y captura la salida. El error de Go
// que devuelve solo representa un fallo real de transporte/ejecución (no se pudo lanzar
// el comando, sesión SSH caída, timeout); un exit code distinto de cero del comando en sí
// se reporta únicamente en CommandResult.ExitCode/Error, igual que antes, para no romper
// a los ~130 llamadores que ya tratan un exit code != 0 como una falla esperada y no una
// caída de transporte.
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
		if _, isExitErr := err.(*exec.ExitError); err != nil && !isExitErr {
			return result, err
		}
		return result, nil
	}

	if e.client == nil {
		return &domain.CommandResult{
			Output:   "",
			ExitCode: -1,
			Error:    fmt.Errorf("cliente ssh no conectado"),
		}, fmt.Errorf("cliente ssh no conectado")
	}

	out, exitCode, err := e.client.RunCommand(cmd)
	result := &domain.CommandResult{
		Output:   out,
		ExitCode: exitCode,
		Error:    err,
	}
	if _, isExitErr := err.(*ssh.ExitError); err != nil && !isExitErr {
		return result, err
	}
	return result, nil
}

// RunCommandStreaming ejecuta un comando entregando cada línea de salida a onLine a
// medida que se produce. En modo local usa un pipe sobre exec.Command; en modo remoto
// delega en pkg/sshclient.Client.RunCommandStreaming.
func (e *CryptoSSHExecutor) RunCommandStreaming(cmd string, onLine func(line string)) (*domain.CommandResult, error) {
	if e.isLocal {
		execCmd := exec.Command("sh", "-c", cmd)
		pr, pw := io.Pipe()
		execCmd.Stdout = pw
		execCmd.Stderr = pw

		var fullOutput strings.Builder
		scanDone := make(chan struct{})
		go func() {
			defer close(scanDone)
			scanner := bufio.NewScanner(pr)
			scanner.Buffer(make([]byte, 64*1024), 1024*1024)
			for scanner.Scan() {
				line := scanner.Text()
				fullOutput.WriteString(line)
				fullOutput.WriteString("\n")
				if onLine != nil {
					onLine(line)
				}
			}
			if scanErr := scanner.Err(); scanErr != nil {
				slog.Debug("error escaneando salida en streaming local", "error", scanErr)
			}
		}()

		startErr := execCmd.Start()
		var runErr error
		if startErr != nil {
			runErr = startErr
		} else {
			runErr = execCmd.Wait()
		}
		if clErr := pw.Close(); clErr != nil {
			slog.Debug("falló cierre del pipe writer en streaming local", "error", clErr)
		}
		<-scanDone

		exitCode := 0
		if runErr != nil {
			if exitErr, ok := runErr.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		}
		result := &domain.CommandResult{
			Output:   fullOutput.String(),
			ExitCode: exitCode,
			Error:    runErr,
		}
		if _, isExitErr := runErr.(*exec.ExitError); runErr != nil && !isExitErr {
			return result, runErr
		}
		return result, nil
	}

	if e.client == nil {
		return &domain.CommandResult{
			Output:   "",
			ExitCode: -1,
			Error:    fmt.Errorf("cliente ssh no conectado"),
		}, fmt.Errorf("cliente ssh no conectado")
	}

	out, exitCode, err := e.client.RunCommandStreaming(cmd, onLine)
	result := &domain.CommandResult{
		Output:   out,
		ExitCode: exitCode,
		Error:    err,
	}
	if _, isExitErr := err.(*ssh.ExitError); err != nil && !isExitErr {
		return result, err
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
	if e.client == nil {
		return fmt.Errorf("cliente ssh no conectado")
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
	if e.client == nil {
		return fmt.Errorf("cliente ssh no conectado")
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
	if e.client == nil {
		return false
	}
	return e.client.CheckConnection()
}

// Close finaliza la conexión o la devuelve al pool si está activa.
func (e *CryptoSSHExecutor) Close() error {
	if e.isLocal {
		return nil
	}
	if e.pooled && e.poolKey != "" {
		sshclient.GlobalPool.Release(e.poolKey, e.client)
		e.client = nil
		e.poolKey = ""
		return nil
	}
	if e.client != nil {
		err := e.client.Close()
		e.client = nil
		return err
	}
	return nil
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
