package controllers

import (
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

// LoggingSSHExecutor envuelve un SSHExecutor y loguea absolutamente todos los
// comandos ejecutados y su output a un callback de streaming.
type LoggingSSHExecutor struct {
	inner   ports.SSHExecutor
	logFunc func(eventType, message string)
}

// NewLoggingSSHExecutor crea un wrapper de logging sobre un SSHExecutor existente.
func NewLoggingSSHExecutor(inner ports.SSHExecutor, logFunc func(eventType, message string)) *LoggingSSHExecutor {
	return &LoggingSSHExecutor{inner: inner, logFunc: logFunc}
}

func (l *LoggingSSHExecutor) Connect(config domain.ServerConfig) error {
	l.logFunc("log", fmt.Sprintf("🔗 Conectando SSH → %s@%s:%d", config.User, config.Host, config.Port))
	err := l.inner.Connect(config)
	if err != nil {
		l.logFunc("log", fmt.Sprintf("❌ Conexión SSH fallida: %v", err))
	} else {
		l.logFunc("log", fmt.Sprintf("✅ Conexión SSH establecida con %s", config.Host))
	}
	return err
}

func (l *LoggingSSHExecutor) RunCommand(cmd string) (*domain.CommandResult, error) {
	displayCmd := cmd
	if len(displayCmd) > 300 {
		displayCmd = displayCmd[:300] + "…"
	}
	l.logFunc("cmd", fmt.Sprintf("$ %s", displayCmd))

	result, err := l.inner.RunCommand(cmd)
	if result != nil && result.Output != "" {
		output := strings.TrimSpace(result.Output)
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				l.logFunc("out", trimmed)
			}
		}
	}
	if err != nil {
		l.logFunc("log", fmt.Sprintf("⚠️  Comando retornó error: %v", err))
	}
	return result, err
}

func (l *LoggingSSHExecutor) InteractiveShell() error {
	return l.inner.InteractiveShell()
}

func (l *LoggingSSHExecutor) InteractiveCommand(cmd string) error {
	return l.inner.InteractiveCommand(cmd)
}

func (l *LoggingSSHExecutor) WriteRemoteFile(remotePath, content string) error {
	l.logFunc("log", fmt.Sprintf("📄 Escribiendo archivo remoto: %s (%d bytes)", remotePath, len(content)))
	err := l.inner.WriteRemoteFile(remotePath, content)
	if err != nil {
		l.logFunc("log", fmt.Sprintf("❌ Error escribiendo %s: %v", remotePath, err))
	} else {
		l.logFunc("log", fmt.Sprintf("✅ Archivo %s escrito", remotePath))
	}
	return err
}

func (l *LoggingSSHExecutor) CheckConnection() bool {
	return l.inner.CheckConnection()
}

func (l *LoggingSSHExecutor) Close() error {
	return l.inner.Close()
}
