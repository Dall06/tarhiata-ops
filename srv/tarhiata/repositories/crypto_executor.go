package repositories

import (
	"fmt"
	"os"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/domain"
	"golang.org/x/crypto/ssh"
)

// CryptoSSHExecutor implementa la interfaz ports.SSHExecutor
type CryptoSSHExecutor struct {
	client *ssh.Client
}

// NewCryptoSSHExecutor crea una nueva instancia del adaptador
func NewCryptoSSHExecutor() *CryptoSSHExecutor {
	return &CryptoSSHExecutor{}
}

// Connect establece la conexión SSH segura leyendo la llave privada del disco.
func (e *CryptoSSHExecutor) Connect(config domain.ServerConfig) error {
	key, err := os.ReadFile(config.PrivateKey)
	if err != nil {
		return fmt.Errorf("no se pudo leer la llave en %s: %w", config.PrivateKey, err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return fmt.Errorf("error al parsear la llave privada: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User: config.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Nota: Para un CLI P2P, InsecureIgnore es la norma inicial
	}

	address := fmt.Sprintf("%s:%d", config.Host, config.Port)
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return fmt.Errorf("error conectando a %s: %w", address, err)
	}

	e.client = client
	return nil
}

// RunCommand ejecuta un comando de forma síncrona y captura la salida (stdout + stderr).
func (e *CryptoSSHExecutor) RunCommand(cmd string) (*domain.CommandResult, error) {
	if e.client == nil {
		return nil, fmt.Errorf("no hay una conexión SSH activa")
	}

	session, err := e.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("no se pudo crear la sesión ssh: %w", err)
	}
	defer session.Close()

	out, err := session.CombinedOutput(cmd)

	result := &domain.CommandResult{
		Output: string(out),
	}

	if err != nil {
		result.Error = err
		if exitError, ok := err.(*ssh.ExitError); ok {
			result.ExitCode = exitError.ExitStatus()
		}
	}

	return result, nil
}

// InteractiveShell abre una consola PTY interactiva conectada a la terminal local del usuario.
func (e *CryptoSSHExecutor) InteractiveShell() error {
	if e.client == nil {
		return fmt.Errorf("no hay conexión activa")
	}

	session, err := e.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Conectamos las salidas y entradas de la sesión remota a nuestra terminal local
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	// Solicitamos un PTY (Pseudo-Terminal)
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm-256color", 80, 40, modes); err != nil {
		return fmt.Errorf("error solicitando pty: %w", err)
	}

	// Iniciamos la shell remota (Bash/Zsh)
	if err := session.Shell(); err != nil {
		return fmt.Errorf("error iniciando shell interactiva: %w", err)
	}

	// Bloqueamos el CLI hasta que el usuario escriba "exit" en el servidor remoto
	return session.Wait()
}

// InteractiveCommand ejecuta un comando específico pero manteniendo la terminal PTY interactiva conectada
func (e *CryptoSSHExecutor) InteractiveCommand(cmd string) error {
	if e.client == nil {
		return fmt.Errorf("no hay conexión activa")
	}

	session, err := e.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm-256color", 80, 40, modes); err != nil {
		return fmt.Errorf("error solicitando pty: %w", err)
	}

	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("error ejecutando comando interactivo: %w", err)
	}

	return nil
}

// CheckConnection hace un "ping" silencioso para la goroutine de estatus.
func (e *CryptoSSHExecutor) CheckConnection() bool {
	if e.client == nil {
		return false
	}
	_, _, err := e.client.SendRequest("keepalive@tarhiata-ops", true, nil)
	return err == nil
}

// Close finaliza la conexión.
func (e *CryptoSSHExecutor) Close() error {
	if e.client != nil {
		return e.client.Close()
	}
	return nil
}
