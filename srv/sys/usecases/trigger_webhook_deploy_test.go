package usecases

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestVerifySignature(t *testing.T) {
	secret := "my_webhook_secret_key_123"
	body := []byte(`{"ref":"refs/heads/main","repository":{"name":"backend"}}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name      string
		secret    string
		body      []byte
		signature string
		want      bool
	}{
		{
			name:      "Valid signature with sha256 prefix",
			secret:    secret,
			body:      body,
			signature: validSig,
			want:      true,
		},
		{
			name:      "Invalid signature",
			secret:    secret,
			body:      body,
			signature: "sha256=invalid_hex_signature",
			want:      false,
		},
		{
			name:      "Empty secret fails closed (ya no permite cualquier firma)",
			secret:    "",
			body:      body,
			signature: "",
			want:      false,
		},
		{
			name:      "Empty secret fails closed incluso con una firma presente",
			secret:    "",
			body:      body,
			signature: validSig,
			want:      false,
		},
		{
			name:      "Empty signature with non-empty secret fails",
			secret:    secret,
			body:      body,
			signature: "",
			want:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := VerifySignature(tc.secret, tc.body, tc.signature)
			if got != tc.want {
				t.Errorf("VerifySignature() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestExtractCommitSHA valida el flujo completo con payloads reales (recortados) de
// GitHub, GitLab y Gitea, los 3 proveedores que este webhook dice soportar.
func TestExtractCommitSHA(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{
			name:    "GitHub push event (after)",
			payload: `{"ref":"refs/heads/main","before":"000","after":"abc123def","repository":{"name":"repo"},"head_commit":{"id":"abc123def"}}`,
			want:    "abc123def",
		},
		{
			name:    "GitLab push event (checkout_sha + after)",
			payload: `{"object_kind":"push","ref":"refs/heads/main","checkout_sha":"gl9876","after":"gl9876","project":{"name":"repo"}}`,
			want:    "gl9876",
		},
		{
			name:    "Gitea push event (compatible con GitHub)",
			payload: `{"ref":"refs/heads/main","after":"gt5555","repository":{"name":"repo"}}`,
			want:    "gt5555",
		},
		{
			name:    "Solo head_commit.id disponible",
			payload: `{"ref":"refs/heads/main","head_commit":{"id":"hc777"}}`,
			want:    "hc777",
		},
		{
			name:    "Payload genérico sin campos de git",
			payload: `{"service":"web-api","image":"myrepo/web-api:v2"}`,
			want:    "",
		},
		{
			name:    "JSON inválido",
			payload: `no es json`,
			want:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractCommitSHA([]byte(tc.payload))
			if got != tc.want {
				t.Errorf("ExtractCommitSHA() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTriggerWebhookDeployUseCase_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		imageTag    string
		setupRepo   func(r *mocks.MockConfigRepository)
		setupExec   func(e *mockSecuritySSHExecutor)
		wantErr     bool
	}{
		{
			name:        "Successful webhook deploy with explicit image",
			serviceName: "app-api",
			imageTag:    "ghcr.io/org/api:v1.5.0",
			setupRepo: func(r *mocks.MockConfigRepository) {
				r.Services = []domain.SavedService{
					{Name: "app-api", Port: 3000, Domain: "api.example.com", Expose: true},
				}
			},
			setupExec: func(e *mockSecuritySSHExecutor) {
				e.responses = map[string]string{
					"docker service update": "app-api updated successfully",
				}
			},
			wantErr: false,
		},
		{
			name:        "Successful webhook deploy using saved ImageSource",
			serviceName: "app-worker",
			imageTag:    "",
			setupRepo: func(r *mocks.MockConfigRepository) {
				r.Services = []domain.SavedService{
					{Name: "app-worker", ImageSource: "myhub/worker:latest", Port: 8080},
				}
			},
			setupExec: func(e *mockSecuritySSHExecutor) {
				e.responses = map[string]string{
					"docker service update": "app-worker updated",
				}
			},
			wantErr: false,
		},
		{
			name:        "Empty service name error",
			serviceName: "",
			imageTag:    "nginx:latest",
			setupRepo:   func(r *mocks.MockConfigRepository) {},
			setupExec:   func(e *mockSecuritySSHExecutor) {},
			wantErr:     true,
		},
		{
			name:        "SSH connection failure",
			serviceName: "app-api",
			imageTag:    "nginx:latest",
			setupRepo:   func(r *mocks.MockConfigRepository) {},
			setupExec: func(e *mockSecuritySSHExecutor) {
				e.err = errors.New("cannot connect to server")
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewMockConfigRepository()
			tc.setupRepo(repo)

			exec := &mockSecuritySSHExecutor{responses: make(map[string]string)}
			tc.setupExec(exec)

			uc := NewTriggerWebhookDeployUseCase(repo, exec)
			rec, err := uc.Execute(tc.serviceName, tc.imageTag, domain.ServerConfig{Host: "192.168.1.50"})

			if (err != nil) != tc.wantErr {
				t.Fatalf("Execute() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if !tc.wantErr && rec == nil {
				t.Errorf("expected non-nil DeploymentRecord on success")
			}
		})
	}
}

// TestTriggerWebhookDeployUseCase_ClosesSSHConnection valida el flujo completo: la
// conexión SSH abierta en Execute debe cerrarse siempre (antes se quedaba abierta
// indefinidamente en cada webhook de git, agotando conexiones bajo tráfico normal).
func TestTriggerWebhookDeployUseCase_ClosesSSHConnection(t *testing.T) {
	repo := mocks.NewMockConfigRepository()
	repo.Services = []domain.SavedService{{Name: "app-api", Port: 3000}}
	exec := &mockSecuritySSHExecutor{responses: map[string]string{"docker service update": "ok"}}
	uc := NewTriggerWebhookDeployUseCase(repo, exec)

	if _, err := uc.Execute("app-api", "myimage:v1", domain.ServerConfig{Host: "192.168.1.50"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.closeCalls != 1 {
		t.Errorf("se esperaba 1 llamada a Close(), se registraron %d (fuga de conexión SSH)", exec.closeCalls)
	}
}
