package sshclient

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

// Client es un cliente SSH genérico
type Client struct {
	conn *ssh.Client
}

// New crea una nueva instancia del cliente SSH
func New() *Client {
	return &Client{}
}

// Connect establece la conexión SSH segura leyendo la llave privada del disco.
func (c *Client) Connect(host, user, privateKeyPath string, port int) error {
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
	}

	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return fmt.Errorf("error conectando a %s: %w", address, err)
	}

	c.conn = conn
	return nil
}

// RunCommand ejecuta un comando de forma síncrona y devuelve la salida y el código de salida.
func (c *Client) RunCommand(cmd string) (string, int, error) {
	if c.conn == nil {
		return "", -1, fmt.Errorf("no hay una conexión SSH activa")
	}

	session, err := c.conn.NewSession()
	if err != nil {
		return "", -1, fmt.Errorf("no se pudo crear la sesión ssh: %w", err)
	}
	defer session.Close()

	out, err := session.CombinedOutput(cmd)
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*ssh.ExitError); ok {
			exitCode = exitError.ExitStatus()
		} else {
			exitCode = -1
		}
	}

	return string(out), exitCode, err
}

// InteractiveShell abre una consola PTY interactiva conectada a la terminal local del usuario.
func (c *Client) InteractiveShell() error {
	if c.conn == nil {
		return fmt.Errorf("no hay conexión activa")
	}

	session, err := c.conn.NewSession()
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

	if err := session.Shell(); err != nil {
		return fmt.Errorf("error iniciando shell interactiva: %w", err)
	}

	return session.Wait()
}

// InteractiveCommand ejecuta un comando específico pero manteniendo la terminal PTY interactiva conectada
func (c *Client) InteractiveCommand(cmd string) error {
	if c.conn == nil {
		return fmt.Errorf("no hay conexión activa")
	}

	session, err := c.conn.NewSession()
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
	if c.conn == nil {
		return false
	}
	_, _, err := c.conn.SendRequest("keepalive@pkg-sshclient", true, nil)
	return err == nil
}

// Close finaliza la conexión.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
