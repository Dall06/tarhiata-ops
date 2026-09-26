package secutil

import (
	"bytes"
	"os"
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
