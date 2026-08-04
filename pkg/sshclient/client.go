package sshclient

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// Client es un cliente SSH genérico
type Client struct {
	mu   sync.RWMutex
	conn *ssh.Client
}

// New crea una nueva instancia del cliente SSH
func New() *Client {
	return &Client{}
}

// Connect establece la conexión SSH segura leyendo la llave privada del disco.
func (c *Client) Connect(host, user, privateKeyPath string, port int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return fmt.Errorf("no se pudo leer la llave en %s: %w", privateKeyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return fmt.Errorf("error al parsear la llave privada: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return fmt.Errorf("error conectando a %s: %w", address, err)
	}

	if c.conn != nil {
		_ = c.conn.Close()
	}

	c.conn = conn
	return nil
}

func (c *Client) RunCommand(cmd string) (string, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	return c.RunCommandWithContext(ctx, cmd)
}

// RunCommandWithContext ejecuta un comando con soporte para cancelación vía Context
func (c *Client) RunCommandWithContext(ctx context.Context, cmd string) (string, int, error) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return "", -1, fmt.Errorf("no hay una conexión SSH activa")
	}

	session, err := conn.NewSession()
	if err != nil {
		return "", -1, fmt.Errorf("no se pudo crear la sesión ssh: %w", err)
	}

	type result struct {
		out  string
		code int
		err  error
	}
	done := make(chan result, 1)

	go func() {
		out, err := session.CombinedOutput(cmd)
		exitCode := 0
		if err != nil {
			if exitError, ok := err.(*ssh.ExitError); ok {
				exitCode = exitError.ExitStatus()
			} else {
				exitCode = -1
			}
		}
		done <- result{string(out), exitCode, err}
	}()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGTERM)
		_ = session.Close() // Fuerza a CombinedOutput a retornar un error y desbloquear la goroutine
		return "", -1, ctx.Err()
	case res := <-done:
		_ = session.Close()
		return res.out, res.code, res.err
	}
}

// InteractiveShell abre una consola PTY interactiva conectada a la terminal local del usuario.
func (c *Client) InteractiveShell() error {
	return c.InteractiveShellWithDimensions(120, 40)
}

// InteractiveShellWithDimensions abre una consola PTY con ancho y alto personalizados.
func (c *Client) InteractiveShellWithDimensions(width, height int) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("no hay conexión activa")
	}

	session, err := conn.NewSession()
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

	if width <= 0 {
		width = 120
	}
	if height <= 0 {
		height = 40
	}

	if err := session.RequestPty("xterm-256color", height, width, modes); err != nil {
		return fmt.Errorf("error solicitando pty: %w", err)
	}

	if err := session.Shell(); err != nil {
		return fmt.Errorf("error iniciando shell interactiva: %w", err)
	}

	return session.Wait()
}

// InteractiveCommand ejecuta un comando específico pero manteniendo la terminal PTY interactiva conectada
func (c *Client) InteractiveCommand(cmd string) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("no hay conexión activa")
	}

	session, err := conn.NewSession()
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

// CheckConnection hace un "ping" silencioso para verificar si la conexión sigue viva.
func (c *Client) CheckConnection() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return false
	}
	_, _, err := c.conn.SendRequest("keepalive@pkg-sshclient", true, nil)
	return err == nil
}

// Close finaliza la conexión.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}
