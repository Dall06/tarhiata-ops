package secutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// EncryptedPrefix identifica un valor cifrado en reposo
	EncryptedPrefix = "enc:v1:"
	keyFileSize     = 32
)

var (
	masterKeyMu sync.RWMutex
	cachedKey   []byte
)

// SetMasterKeyForTesting permite inyectar una clave fija para pruebas determinísticas.
func SetMasterKeyForTesting(key []byte) {
	masterKeyMu.Lock()
	defer masterKeyMu.Unlock()
	cachedKey = key
}

// ResetMasterKeyForTesting restablece la clave en caché.
func ResetMasterKeyForTesting() {
	masterKeyMu.Lock()
	defer masterKeyMu.Unlock()
	cachedKey = nil
}

// GetMasterKey resuelve la clave de 32 bytes (256 bits):
// 1. Variable de entorno TARHIATA_SECRET_KEY (si está definida)
// 2. Archivo ~/.config/tarhiata/.key (se genera con permisos 0600 si no existe)
func GetMasterKey() ([]byte, error) {
	masterKeyMu.RLock()
	if len(cachedKey) == 32 {
		keyCopy := make([]byte, 32)
		copy(keyCopy, cachedKey)
		masterKeyMu.RUnlock()
		return keyCopy, nil
	}
	masterKeyMu.RUnlock()

	masterKeyMu.Lock()
	defer masterKeyMu.Unlock()

	if len(cachedKey) == 32 {
		keyCopy := make([]byte, 32)
		copy(keyCopy, cachedKey)
		return keyCopy, nil
	}

	// 1. Variable de entorno explícita
	if envKey := strings.TrimSpace(os.Getenv("TARHIATA_SECRET_KEY")); envKey != "" {
		hash := sha256.Sum256([]byte(envKey))
		cachedKey = hash[:]
		keyCopy := make([]byte, 32)
		copy(keyCopy, cachedKey)
		return keyCopy, nil
	}

	// 2. Archivo persistente ~/.config/tarhiata/.key
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("no se pudo determinar el directorio de usuario: %w", err)
	}

	keyDir := filepath.Join(homeDir, ".config", "tarhiata")
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return nil, fmt.Errorf("no se pudo crear directorio de claves %s: %w", keyDir, err)
	}

	keyPath := filepath.Join(keyDir, ".key")
	if data, err := os.ReadFile(keyPath); err == nil && len(data) == keyFileSize {
		cachedKey = data
		keyCopy := make([]byte, 32)
		copy(keyCopy, cachedKey)
		return keyCopy, nil
	}

	// Generar nueva clave criptográfica aleatoria de 32 bytes
	newKey := make([]byte, keyFileSize)
	if _, err := io.ReadFull(rand.Reader, newKey); err != nil {
		return nil, fmt.Errorf("error generando clave aleatoria: %w", err)
	}

	if err := os.WriteFile(keyPath, newKey, 0600); err != nil {
		return nil, fmt.Errorf("error persistiendo clave en %s: %w", keyPath, err)
	}

	cachedKey = newKey
	keyCopy := make([]byte, 32)
	copy(keyCopy, cachedKey)
	return keyCopy, nil
}

// Encrypt cifra una cadena en texto plano usando AES-256-GCM con la clave maestra.
// Si el texto ya está cifrado o vacío, lo retorna sin alteraciones.
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" || strings.HasPrefix(plaintext, EncryptedPrefix) {
		return plaintext, nil
	}

	key, err := GetMasterKey()
	if err != nil {
		return "", fmt.Errorf("falló al obtener clave de cifrado: %w", err)
	}

	return EncryptWithKey(plaintext, key)
}

// EncryptWithKey cifra una cadena usando una clave de 32 bytes provista explícitamente.
func EncryptWithKey(plaintext string, key []byte) (string, error) {
	if plaintext == "" || strings.HasPrefix(plaintext, EncryptedPrefix) {
		return plaintext, nil
	}
	if len(key) != 32 {
		return "", fmt.Errorf("la clave debe ser de exactamente 32 bytes, recibido %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("error creando cifrador aes: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("error creando modo gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("error generando nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return EncryptedPrefix + encoded, nil
}

// Decrypt descifra una cadena con prefijo enc:v1: usando la clave maestra.
// Si el valor no tiene el prefijo de cifrado (datos legados en texto plano), se retorna tal cual.
func Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" || !strings.HasPrefix(ciphertext, EncryptedPrefix) {
		return ciphertext, nil
	}

	key, err := GetMasterKey()
	if err != nil {
		return "", fmt.Errorf("falló al obtener clave para descifrado: %w", err)
	}

	return DecryptWithKey(ciphertext, key)
}

// DecryptWithKey descifra una cadena con prefijo enc:v1: usando una clave explícita de 32 bytes.
func DecryptWithKey(ciphertext string, key []byte) (string, error) {
	if ciphertext == "" || !strings.HasPrefix(ciphertext, EncryptedPrefix) {
		return ciphertext, nil
	}
	if len(key) != 32 {
		return "", fmt.Errorf("la clave debe ser de exactamente 32 bytes, recibido %d", len(key))
	}

	rawB64 := strings.TrimPrefix(ciphertext, EncryptedPrefix)
	data, err := base64.StdEncoding.DecodeString(rawB64)
	if err != nil {
		return "", fmt.Errorf("error decodificando base64 del valor cifrado: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("error creando cifrador aes: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("error creando modo gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("longitud de datos cifrados inválida")
	}

	nonce, sealed := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("error descifrando datos (clave inválida o datos corruptos): %w", err)
	}

	return string(plaintext), nil
}

// IsLocalHost determina si una configuración de host apunta al entorno local.
func IsLocalHost(host, cloudProvider string) bool {
	if cloudProvider == "local" {
		return true
	}
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "local" || h == "localhost" || h == "127.0.0.1" || h == "::1" {
		return true
	}
	return false
}

