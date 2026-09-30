package usecases

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/pkg/validator"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
	"github.com/Dall06/tarhiata-ops/srv/sys/repositories"
)

type ManageSSHKeysUseCase struct {
	sshExecutor ports.SSHExecutor
	vultrClient ports.VultrClient
}

func NewManageSSHKeysUseCase(ssh ports.SSHExecutor) *ManageSSHKeysUseCase {
	return &ManageSSHKeysUseCase{
		sshExecutor: ssh,
		vultrClient: repositories.NewVultrHTTPClient(),
	}
}

// WithVultrClient permite inyectar un cliente Vultr falso para pruebas unitarias.
func (uc *ManageSSHKeysUseCase) WithVultrClient(c ports.VultrClient) *ManageSSHKeysUseCase {
	uc.vultrClient = c
	return uc
}

// computeFingerprint calcula la huella MD5 (estándar ssh-keygen) de una llave SSH pública en formato base64.
func computeFingerprint(keyBase64 string) string {
	raw, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return ""
	}
	hash := md5.Sum(raw)
	var hexParts []string
	for _, b := range hash {
		hexParts = append(hexParts, fmt.Sprintf("%02x", b))
	}
	return strings.Join(hexParts, ":")
}

// computeSHA256Fingerprint calcula la huella SHA256 (estándar OpenSSH moderno).
func computeSHA256Fingerprint(keyBase64 string) string {
	raw, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(raw)
	return "SHA256:" + strings.TrimRight(base64.StdEncoding.EncodeToString(hash[:]), "=")
}

// ListKeys lee /root/.ssh/authorized_keys del VPS y lo contrasta con la API de Vultr.
func (uc *ManageSSHKeysUseCase) ListKeys(cfg domain.ServerConfig) ([]domain.SSHKeyInfo, error) {
	sshExec := uc.sshExecutor
	if sshExec == nil {
		exec := repositories.NewCryptoSSHExecutor()
		if err := exec.Connect(cfg); err != nil {
			return nil, fmt.Errorf("error conectando por SSH al VPS: %w", err)
		}
		defer exec.Close()
		sshExec = exec
	}

	// 1. Obtener llaves registradas en la cuenta de Vultr vía Vultr API v2
	vultrKeysMap := make(map[string]bool)
	if cfg.VultrAPIToken != "" {
		if vKeys, err := uc.vultrClient.GetSSHKeys(cfg.VultrAPIToken); err == nil {
			for _, vk := range vKeys {
				parts := strings.Fields(strings.TrimSpace(vk.SSHKey))
				if len(parts) >= 2 {
					fpMD5 := computeFingerprint(parts[1])
					fpSHA := computeSHA256Fingerprint(parts[1])
					if fpMD5 != "" {
						vultrKeysMap[fpMD5] = true
					}
					if fpSHA != "" {
						vultrKeysMap[fpSHA] = true
					}
					// También mapear por el contenido crudo de la llave
					vultrKeysMap[parts[1]] = true
				}
			}
		}
	}

	// 2. Leer /root/.ssh/authorized_keys en el VPS
	res, err := sshExec.RunCommand("cat /root/.ssh/authorized_keys 2>/dev/null")
	if err != nil || res.ExitCode != 0 {
		return nil, fmt.Errorf("no se pudo leer /root/.ssh/authorized_keys: %s", res.Output)
	}

	lines := strings.Split(res.Output, "\n")
	var keyInfos []domain.SSHKeyInfo

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		parts := strings.Fields(trimmed)
		if len(parts) < 2 {
			continue
		}

		keyType := parts[0]
		keyBase64 := parts[1]
		comment := "Llave sin comentario"
		if len(parts) >= 3 {
			comment = strings.Join(parts[2:], " ")
		}

		fpMD5 := computeFingerprint(keyBase64)
		fpSHA := computeSHA256Fingerprint(keyBase64)

		isVultr := vultrKeysMap[fpMD5] || vultrKeysMap[fpSHA] || vultrKeysMap[keyBase64] || strings.Contains(strings.ToLower(comment), "vultr")

		keyInfos = append(keyInfos, domain.SSHKeyInfo{
			Fingerprint: fpMD5,
			Comment:     comment,
			Type:        keyType,
			KeyContent:  trimmed,
			IsVultrKey:  isVultr,
			Protected:   isVultr,
		})
	}

	return keyInfos, nil
}

// AddKey añade una nueva llave SSH pública a /root/.ssh/authorized_keys en el VPS.
func (uc *ManageSSHKeysUseCase) AddKey(cfg domain.ServerConfig, publicKey string) error {
	publicKey = strings.TrimSpace(publicKey)
	if publicKey == "" || (!strings.HasPrefix(publicKey, "ssh-") && !strings.HasPrefix(publicKey, "ecdsa-")) {
		return fmt.Errorf("la llave SSH pública proporcionada no tiene un formato válido (debe iniciar con ssh-rsa, ssh-ed25519, etc.)")
	}

	sshExec := uc.sshExecutor
	if sshExec == nil {
		exec := repositories.NewCryptoSSHExecutor()
		if err := exec.Connect(cfg); err != nil {
			return fmt.Errorf("error conectando por SSH al VPS: %w", err)
		}
		defer exec.Close()
		sshExec = exec
	}

	// Verificar si ya existe para evitar duplicados
	resCheck, errCheck := sshExec.RunCommand("cat /root/.ssh/authorized_keys 2>/dev/null")
	if errCheck == nil && resCheck != nil && strings.Contains(resCheck.Output, publicKey) {
		return fmt.Errorf("la llave SSH ya se encuentra registrada en authorized_keys")
	}

	cmd := fmt.Sprintf("mkdir -p /root/.ssh && chmod 700 /root/.ssh && echo %s >> /root/.ssh/authorized_keys && chmod 600 /root/.ssh/authorized_keys", validator.ShellQuote(publicKey))
	res, err := sshExec.RunCommand(cmd)
	if err != nil || res == nil || res.ExitCode != 0 {
		out := ""
		if res != nil {
			out = res.Output
		}
		if out == "" && err != nil {
			out = err.Error()
		}
		return fmt.Errorf("error añadiendo llave SSH: %s", out)
	}

	return nil
}

// DeleteKey elimina una llave SSH por huella digital o contenido, BLOQUEANDO la eliminación si está registrada en Vultr.
func (uc *ManageSSHKeysUseCase) DeleteKey(cfg domain.ServerConfig, targetIdentifier string) error {
	targetIdentifier = strings.TrimSpace(targetIdentifier)
	if targetIdentifier == "" {
		return fmt.Errorf("se requiere una huella digital o identificador de llave para eliminar")
	}

	keys, err := uc.ListKeys(cfg)
	if err != nil {
		return err
	}

	var keyToDelete *domain.SSHKeyInfo
	for _, k := range keys {
		if k.Fingerprint == targetIdentifier || k.KeyContent == targetIdentifier || strings.Contains(k.Comment, targetIdentifier) {
			keyToDelete = &k
			break
		}
	}

	if keyToDelete == nil {
		return fmt.Errorf("no se encontró ninguna llave SSH que coincida con '%s'", targetIdentifier)
	}

	// REG LÓGICA DE SEGURIDAD: Bloquear eliminación si está registrada en Vultr
	if keyToDelete.IsVultrKey || keyToDelete.Protected {
		return fmt.Errorf("⛔ ACCESO DENEGADO: La llave SSH '%s' (%s) está registrada en Vultr como llave maestra de cuenta. Por seguridad, no se puede eliminar para evitar dejar el servidor inaccesible", keyToDelete.Comment, keyToDelete.Fingerprint)
	}

	// Reescribir authorized_keys excluyendo la llave seleccionada
	var updatedLines []string
	for _, k := range keys {
		if k.Fingerprint != keyToDelete.Fingerprint && k.KeyContent != keyToDelete.KeyContent {
			updatedLines = append(updatedLines, k.KeyContent)
		}
	}

	newContent := strings.Join(updatedLines, "\n") + "\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(newContent))

	sshExec := uc.sshExecutor
	if sshExec == nil {
		exec := repositories.NewCryptoSSHExecutor()
		if err := exec.Connect(cfg); err != nil {
			return fmt.Errorf("error conectando por SSH al VPS: %w", err)
		}
		defer exec.Close()
		sshExec = exec
	}

	cmd := fmt.Sprintf("echo '%s' | base64 -d > /root/.ssh/authorized_keys && chmod 600 /root/.ssh/authorized_keys", encoded)
	res, err := sshExec.RunCommand(cmd)
	if err != nil || res == nil || res.ExitCode != 0 {
		out := ""
		if res != nil {
			out = res.Output
		}
		if out == "" && err != nil {
			out = err.Error()
		}
		return fmt.Errorf("error al actualizar /root/.ssh/authorized_keys: %s", out)
	}

	return nil
}
