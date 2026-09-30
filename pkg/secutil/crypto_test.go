package secutil

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	testKey := bytes.Repeat([]byte("a"), 32)
	SetMasterKeyForTesting(testKey)
	defer ResetMasterKeyForTesting()

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple password",
			plaintext: "superSecretPassword123!",
		},
		{
			name:      "SSH private key",
			plaintext: "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAA=\n-----END OPENSSH PRIVATE KEY-----",
		},
		{
			name:      "cloud token",
			plaintext: "dop_v1_abcdef1234567890abcdef1234567890",
		},
		{
			name:      "unicode and emojis",
			plaintext: "clave_secreta_🇲🇽_tarhiata_2026",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("unexpected error encrypting: %v", err)
			}

			if tt.plaintext == "" {
				if encrypted != "" {
					t.Errorf("expected empty string for empty input, got %q", encrypted)
				}
				return
			}

			if !strings.HasPrefix(encrypted, EncryptedPrefix) {
				t.Errorf("expected prefix %s, got %s", EncryptedPrefix, encrypted)
			}

			decrypted, err := Decrypt(encrypted)
			if err != nil {
				t.Fatalf("unexpected error decrypting: %v", err)
			}

			if decrypted != tt.plaintext {
				t.Errorf("expected decrypted text to match original:\nexpected: %q\ngot:      %q", tt.plaintext, decrypted)
			}
		})
	}
}

func TestDecrypt_LegacyPlaintext(t *testing.T) {
	testKey := bytes.Repeat([]byte("b"), 32)
	SetMasterKeyForTesting(testKey)
	defer ResetMasterKeyForTesting()

	tests := []struct {
		name            string
		legacyPlaintext string
	}{
		{
			name:            "path to key file",
			legacyPlaintext: "~/.ssh/id_rsa",
		},
		{
			name:            "unencrypted password",
			legacyPlaintext: "postgres_unencrypted_123",
		},
		{
			name:            "empty string",
			legacyPlaintext: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Decrypt(tt.legacyPlaintext)
			if err != nil {
				t.Fatalf("unexpected error decrypting legacy plaintext: %v", err)
			}
			if result != tt.legacyPlaintext {
				t.Errorf("expected result to remain unchanged: got %q, want %q", result, tt.legacyPlaintext)
			}
		})
	}
}

func TestEncrypt_Idempotency(t *testing.T) {
	testKey := bytes.Repeat([]byte("c"), 32)
	SetMasterKeyForTesting(testKey)
	defer ResetMasterKeyForTesting()

	original := "my-secret-key"
	encrypted1, err := Encrypt(original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Calling Encrypt on already encrypted text should return it as-is
	encrypted2, err := Encrypt(encrypted1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if encrypted1 != encrypted2 {
		t.Errorf("expected Encrypt to be idempotent, got %q vs %q", encrypted1, encrypted2)
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := bytes.Repeat([]byte("1"), 32)
	key2 := bytes.Repeat([]byte("2"), 32)

	encrypted, err := EncryptWithKey("secret", key1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = DecryptWithKey(encrypted, key2)
	if err == nil {
		t.Fatal("expected decryption to fail with incorrect key")
	}
}

func TestGetMasterKey_EnvironmentVariable(t *testing.T) {
	ResetMasterKeyForTesting()
	defer ResetMasterKeyForTesting()

	os.Setenv("TARHIATA_SECRET_KEY", "custom-user-provided-secret-key-42")
	defer os.Unsetenv("TARHIATA_SECRET_KEY")

	key, err := GetMasterKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key) != 32 {
		t.Errorf("expected 32-byte key, got %d", len(key))
	}
}

// TestGetMasterKey_CorruptedFileFailsLoudly valida el flujo completo: si el archivo de
// clave maestra existe pero tiene un tamaño inválido (corrupción, escritura parcial tras
// un crash), GetMasterKey debe fallar con un error en vez de sobrescribirlo en silencio
// con una clave nueva, lo que volvería indescifrables todos los secretos ya cifrados.
func TestGetMasterKey_CorruptedFileFailsLoudly(t *testing.T) {
	ResetMasterKeyForTesting()
	defer ResetMasterKeyForTesting()
	os.Unsetenv("TARHIATA_SECRET_KEY")

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	keyDir := filepath.Join(tmpHome, ".config", "tarhiata")
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		t.Fatalf("failed to create key dir: %v", err)
	}
	keyPath := filepath.Join(keyDir, ".key")
	corrupted := []byte("esto-no-mide-32-bytes")
	if err := os.WriteFile(keyPath, corrupted, 0600); err != nil {
		t.Fatalf("failed to write corrupted key file: %v", err)
	}

	if _, err := GetMasterKey(); err == nil {
		t.Fatal("se esperaba un error por archivo de clave corrupto, no se generó ninguno")
	}

	// El archivo corrupto NO debió sobrescribirse con una clave nueva.
	after, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("unexpected error re-reading key file: %v", err)
	}
	if !bytes.Equal(after, corrupted) {
		t.Errorf("el archivo de clave se sobrescribió en silencio; se esperaba que quedara intacto para investigación manual")
	}
}

// TestGetMasterKey_MissingFileGeneratesNewKey confirma que el caso legítimo (primera
// ejecución, sin archivo de clave) sigue generando una clave nueva normalmente.
func TestGetMasterKey_MissingFileGeneratesNewKey(t *testing.T) {
	ResetMasterKeyForTesting()
	defer ResetMasterKeyForTesting()
	os.Unsetenv("TARHIATA_SECRET_KEY")

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	key, err := GetMasterKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key) != 32 {
		t.Errorf("expected 32-byte key, got %d", len(key))
	}

	keyPath := filepath.Join(tmpHome, ".config", "tarhiata", ".key")
	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("se esperaba que el archivo de clave quedara persistido: %v", err)
	}
}

func TestIsLocalHost(t *testing.T) {
	tests := []struct {
		name          string
		host          string
		cloudProvider string
		expected      bool
	}{
		{
			name:          "localhost host",
			host:          "localhost",
			cloudProvider: "",
			expected:      true,
		},
		{
			name:          "127.0.0.1 host",
			host:          "127.0.0.1",
			cloudProvider: "",
			expected:      true,
		},
		{
			name:          "::1 host",
			host:          "::1",
			cloudProvider: "",
			expected:      true,
		},
		{
			name:          "local string host",
			host:          "local",
			cloudProvider: "",
			expected:      true,
		},
		{
			name:          "cloudProvider local",
			host:          "my-vps",
			cloudProvider: "local",
			expected:      true,
		},
		{
			name:          "remote host ip",
			host:          "192.168.1.50",
			cloudProvider: "custom",
			expected:      false,
		},
		{
			name:          "remote vps hostname",
			host:          "vps.example.com",
			cloudProvider: "vultr",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsLocalHost(tt.host, tt.cloudProvider)
			if got != tt.expected {
				t.Errorf("IsLocalHost(%q, %q) = %v, expected %v", tt.host, tt.cloudProvider, got, tt.expected)
			}
		})
	}
}
