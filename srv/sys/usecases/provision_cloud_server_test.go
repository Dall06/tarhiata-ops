package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestProvisionCloudServerUseCase_Execute(t *testing.T) {
	tests := []struct {
		name          string
		req           ports.ProvisionCloudRequest
		mockIP        string
		mockPrivKey   string
		mockProvErr   error
		existingSrv   *domain.ServerConfig
		connectResult *domain.ConnectionResult
		expectErr     bool
		expectedName  string
	}{
		{
			name: "Error when API token is empty",
			req: ports.ProvisionCloudRequest{
				Name:     "cloud-srv",
				APIToken: "",
			},
			expectErr: true,
		},
		{
			name: "Error when server name already exists",
			req: ports.ProvisionCloudRequest{
				Name:     "existing-srv",
				APIToken: "token-123",
			},
			existingSrv: &domain.ServerConfig{
				Name: "existing-srv",
				Host: "1.2.3.4",
			},
			expectErr: true,
		},
		{
			name: "Error when provisioner fails",
			req: ports.ProvisionCloudRequest{
				Name:     "failing-srv",
				APIToken: "token-123",
				Provider: "vultr",
			},
			mockProvErr: errors.New("cloud quota exceeded"),
			expectErr:   true,
		},
		{
			name: "Successful provision and connection probe",
			req: ports.ProvisionCloudRequest{
				Name:        "new-vps",
				Provider:    "vultr",
				APIToken:    "valid-token",
				Region:      "mex",
				SetAsActive: true,
			},
			mockIP:      "198.51.100.25",
			mockPrivKey: "-----BEGIN RSA PRIVATE KEY-----\nMIIE...\n-----END RSA PRIVATE KEY-----",
			connectResult: &domain.ConnectionResult{
				Name:         "new-vps",
				Connected:    true,
				TargetHost:   "198.51.100.25",
				OS:           "Linux x86_64",
				DockerActive: true,
			},
			expectErr:    false,
			expectedName: "new-vps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mocks.MockConfigRepository{}
			if tt.existingSrv != nil {
				if err := repo.SaveServerConfig(*tt.existingSrv); err != nil {
					t.Fatalf("unexpected error saving server config: %v", err)
				}
			}

			sshMock := &mocks.MockSSHExecutor{}
			connectUC := NewConnectServerUseCase(sshMock)

			uc := NewProvisionCloudServerUseCase(repo, connectUC)
			uc.WithRetryConfig(1, 1*time.Millisecond)

			mockProv := mocks.NewMockProvisioner()
			mockProv.MockIP = tt.mockIP
			mockProv.MockPrivKey = tt.mockPrivKey
			mockProv.MockError = tt.mockProvErr

			uc.WithProvisionerFactory(func(provider, workspace string) ports.Provisioner {
				return mockProv
			})

			res, err := uc.Execute(tt.req)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Execute() error = %v, expectErr %v", err, tt.expectErr)
			}

			if !tt.expectErr {
				if res == nil {
					t.Fatal("Execute() returned nil result on success")
				}
				if res.Name != tt.expectedName {
					t.Errorf("Execute() result Name = %s, want %s", res.Name, tt.expectedName)
				}
				saved, err := repo.GetServerConfigByName(tt.expectedName)
				if err != nil || saved == nil {
					t.Errorf("Server %s was not saved to repository", tt.expectedName)
				}
			}
		})
	}
}
