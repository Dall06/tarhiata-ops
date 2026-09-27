package osterminal

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// BuildSSHCommand construye la línea de comando de SSH nativo para conectarse a un host.
func BuildSSHCommand(user, host string, port int, privateKey string) string {
	if strings.TrimSpace(user) == "" {
		user = "root"
	}
	parts := []string{"ssh", "-o", "StrictHostKeyChecking=accept-new"}
	if strings.TrimSpace(privateKey) != "" {
		parts = append(parts, "-i", strings.TrimSpace(privateKey))
	}
	if port > 0 && port != 22 {
		parts = append(parts, "-p", fmt.Sprintf("%d", port))
	}
	parts = append(parts, fmt.Sprintf("%s@%s", user, strings.TrimSpace(host)))
	return strings.Join(parts, " ")
}

// OpenNativeTerminal abre una ventana real del emulador de terminal del sistema operativo
// y ejecuta directamente el comando (ej. ssh o subshell local).
func OpenNativeTerminal(command string) error {
	switch runtime.GOOS {
	case "darwin":
		escaped := strings.ReplaceAll(command, "\"", "\\\"")
		script := fmt.Sprintf("tell application \"Terminal\" to do script \"%s\"\ntell application \"Terminal\" to activate", escaped)
		cmd := exec.Command("osascript", "-e", script)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("error al abrir Terminal en macOS: %w", err)
		}
		return nil

	case "linux":
		terminals := []struct {
			bin  string
			args []string
		}{
			{"x-terminal-emulator", []string{"-e", command}},
			{"gnome-terminal", []string{"--", "sh", "-c", command}},
			{"konsole", []string{"-e", command}},
			{"alacritty", []string{"-e", "sh", "-c", command}},
			{"xfce4-terminal", []string{"-e", command}},
			{"xterm", []string{"-e", command}},
		}
		for _, t := range terminals {
			if path, err := exec.LookPath(t.bin); err == nil && path != "" {
				cmd := exec.Command(path, t.args...)
				if err := cmd.Start(); err != nil {
					return fmt.Errorf("error al iniciar terminal %s: %w", t.bin, err)
				}
				return nil
			}
		}
		return fmt.Errorf("no se encontró ningún emulador de terminal compatible instalado")

	case "windows":
		cmd := exec.Command("cmd.exe", "/c", "start", "cmd.exe", "/k", command)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("error al abrir consola en Windows: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("sistema operativo no soportado para apertura automática de terminal: %s", runtime.GOOS)
	}
}

// OpenBrowser abre una URL en el navegador web predeterminado del sistema operativo.
func OpenBrowser(rawURL string) error {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return fmt.Errorf("osterminal: url inválida (debe iniciar con http:// o https://): %s", rawURL)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("osterminal: fallo al abrir navegador para %s: %w", rawURL, err)
	}
	return nil
}

