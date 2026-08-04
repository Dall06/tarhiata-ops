package usecases

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type FileItem struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"isDir"`
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
}

type ManageVolumesUseCase struct {
	repo ports.ConfigRepository
	ssh  ports.SSHExecutor
}

func NewManageVolumesUseCase(repo ports.ConfigRepository, ssh ports.SSHExecutor) *ManageVolumesUseCase {
	return &ManageVolumesUseCase{repo: repo, ssh: ssh}
}

const BaseDataPath = "/opt/data"

// sanitizePath evita vulnerabilidades de Directory Traversal (../)
func sanitizePath(userPath string) (string, error) {
	cleaned := filepath.Clean(userPath)
	if !strings.HasPrefix(cleaned, BaseDataPath) {
		return "", fmt.Errorf("acceso denegado: el camino '%s' está fuera de %s", userPath, BaseDataPath)
	}
	return cleaned, nil
}

// ListVolumes devuelve la lista de carpetas principales registradas en /opt/data
func (uc *ManageVolumesUseCase) ListVolumes(config domain.ServerConfig) ([]string, error) {
	if uc.ssh == nil {
		return nil, fmt.Errorf("ejecutor SSH no configurado")
	}
	if err := uc.ssh.Connect(config); err != nil {
		return nil, fmt.Errorf("error al conectar por SSH: %w", err)
	}
	defer uc.ssh.Close()

	cmd := fmt.Sprintf("mkdir -p %s && ls -1 %s", BaseDataPath, BaseDataPath)
	res, err := uc.ssh.RunCommand(cmd)
	if err != nil || res.ExitCode != 0 {
		return []string{}, nil
	}

	lines := strings.Split(strings.TrimSpace(res.Output), "\n")
	var volumes []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			volumes = append(volumes, l)
		}
	}
	return volumes, nil
}

// ListVolumeFiles lista los archivos y subcarpetas dentro de un volumen o subdirectorio
func (uc *ManageVolumesUseCase) ListVolumeFiles(targetPath string, config domain.ServerConfig) ([]FileItem, error) {
	if targetPath == "" || targetPath == "/" {
		targetPath = BaseDataPath
	}
	cleanPath, err := sanitizePath(targetPath)
	if err != nil {
		return nil, err
	}

	if uc.ssh == nil {
		return nil, fmt.Errorf("ejecutor SSH no configurado")
	}
	if err := uc.ssh.Connect(config); err != nil {
		return nil, fmt.Errorf("error de conexión SSH: %w", err)
	}
	defer uc.ssh.Close()

	// Formato de salida de ls en el servidor: "is_dir|size|mtime|filename"
	cmd := fmt.Sprintf("mkdir -p %s && for f in %s/*; do [ -e \"$f\" ] || continue; if [ -d \"$f\" ]; then echo \"1|0|$(stat -c %%Y \"$f\" 2>/dev/null || date +%%s)|$(basename \"$f\")\"; else echo \"0|$(stat -c %%s \"$f\" 2>/dev/null || echo 0)|$(stat -c %%Y \"$f\" 2>/dev/null || date +%%s)|$(basename \"$f\")\"; fi; done", cleanPath, cleanPath)
	res, err := uc.ssh.RunCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("error al listar archivos: %w", err)
	}

	var items []FileItem
	lines := strings.Split(strings.TrimSpace(res.Output), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		parts := strings.SplitN(l, "|", 4)
		if len(parts) < 4 {
			continue
		}
		isDir := parts[0] == "1"
		size, _ := strconv.ParseInt(parts[1], 10, 64)
		timestamp, _ := strconv.ParseInt(parts[2], 10, 64)
		name := parts[3]

		modTime := time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")

		items = append(items, FileItem{
			Name:    name,
			Path:    filepath.Join(cleanPath, name),
			IsDir:   isDir,
			Size:    size,
			ModTime: modTime,
		})
	}
	return items, nil
}

// ReadFileContent lee el texto de un archivo dentro de /opt/data/
func (uc *ManageVolumesUseCase) ReadFileContent(targetPath string, config domain.ServerConfig) (string, error) {
	cleanPath, err := sanitizePath(targetPath)
	if err != nil {
		return "", err
	}

	if uc.ssh == nil {
		return "", fmt.Errorf("ejecutor SSH no configurado")
	}
	if err := uc.ssh.Connect(config); err != nil {
		return "", fmt.Errorf("error de conexión SSH: %w", err)
	}
	defer uc.ssh.Close()

	cmd := fmt.Sprintf("head -c 200000 %s", cleanPath)
	res, err := uc.ssh.RunCommand(cmd)
	if err != nil || res.ExitCode != 0 {
		return "", fmt.Errorf("error al leer archivo '%s': %s", cleanPath, res.Output)
	}
	return res.Output, nil
}

// WriteFileContent guarda/sobreescribe un archivo de texto en el servidor
func (uc *ManageVolumesUseCase) WriteFileContent(targetPath string, content string, config domain.ServerConfig) error {
	cleanPath, err := sanitizePath(targetPath)
	if err != nil {
		return err
	}

	if uc.ssh == nil {
		return fmt.Errorf("ejecutor SSH no configurado")
	}
	if err := uc.ssh.Connect(config); err != nil {
		return fmt.Errorf("error de conexión SSH: %w", err)
	}
	defer uc.ssh.Close()

	// Crear el directorio padre si no existe
	parentDir := filepath.Dir(cleanPath)
	mkdirCmd := fmt.Sprintf("mkdir -p %s", parentDir)
	uc.ssh.RunCommand(mkdirCmd)

	// Inyectar contenido en base64 para evitar problemas de comillas en bash
	b64Data := fmt.Sprintf("%x", content) // hex encoding
	writeCmd := fmt.Sprintf("echo '%s' | xxd -r -p > %s", b64Data, cleanPath)
	res, err := uc.ssh.RunCommand(writeCmd)
	if err != nil || res.ExitCode != 0 {
		// Fallback simple si xxd no estuviese presente
		escaped := strings.ReplaceAll(content, `'`, `'\''`)
		fallbackCmd := fmt.Sprintf("cat << 'EOF_TARHIATA_FILE' > %s\n%s\nEOF_TARHIATA_FILE", cleanPath, escaped)
		res2, err2 := uc.ssh.RunCommand(fallbackCmd)
		if err2 != nil || res2.ExitCode != 0 {
			return fmt.Errorf("error al escribir archivo: %s", res.Output)
		}
	}
	return nil
}

// DownloadFile devuelve los bytes de un archivo para descarga directa
func (uc *ManageVolumesUseCase) DownloadFile(targetPath string, config domain.ServerConfig) ([]byte, string, error) {
	cleanPath, err := sanitizePath(targetPath)
	if err != nil {
		return nil, "", err
	}

	if uc.ssh == nil {
		return nil, "", fmt.Errorf("ejecutor SSH no configurado")
	}
	if err := uc.ssh.Connect(config); err != nil {
		return nil, "", fmt.Errorf("error de conexión SSH: %w", err)
	}
	defer uc.ssh.Close()

	cmd := fmt.Sprintf("base64 %s 2>/dev/null || cat %s", cleanPath, cleanPath)
	res, err := uc.ssh.RunCommand(cmd)
	if err != nil || res.ExitCode != 0 {
		return nil, "", fmt.Errorf("error al descargar archivo '%s': %s", cleanPath, res.Output)
	}

	// Decodificar base64 si corresponde
	raw := strings.TrimSpace(res.Output)
	filename := filepath.Base(cleanPath)
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err == nil {
		return decoded, filename, nil
	}
	return []byte(raw), filename, nil
}

// DeleteFile elimina un archivo o carpeta dentro de /opt/data
func (uc *ManageVolumesUseCase) DeleteFile(targetPath string, config domain.ServerConfig) error {
	cleanPath, err := sanitizePath(targetPath)
	if err != nil {
		return err
	}
	if cleanPath == BaseDataPath {
		return fmt.Errorf("no es posible eliminar el directorio raíz %s", BaseDataPath)
	}

	if uc.ssh == nil {
		return fmt.Errorf("ejecutor SSH no configurado")
	}
	if err := uc.ssh.Connect(config); err != nil {
		return fmt.Errorf("error de conexión SSH: %w", err)
	}
	defer uc.ssh.Close()

	cmd := fmt.Sprintf("rm -rf %s", cleanPath)
	res, err := uc.ssh.RunCommand(cmd)
	if err != nil || res.ExitCode != 0 {
		return fmt.Errorf("error al eliminar '%s': %s", cleanPath, res.Output)
	}
	return nil
}
