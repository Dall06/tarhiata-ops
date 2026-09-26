package usecases

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestManageRegistryAuthUseCase_SaveAndList(t *testing.T) {
	repo := mocks.NewMockConfigRepository()
	uc := NewManageRegistryAuthUseCase(repo, nil)

	cred := domain.SavedRegistryCredential{
		Server:   "ghcr.io",
		Username: "myuser",
		Password: "ghp_secret_token",
	}

	err := uc.Save(cred, domain.ServerConfig{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	list, err := uc.List()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(list))
	}
	if list[0].Server != "ghcr.io" || list[0].Username != "myuser" {
		t.Errorf("unexpected credential: %+v", list[0])
	}
}

func TestManageRegistryAuthUseCase_Delete(t *testing.T) {
	repo := mocks.NewMockConfigRepository()
	uc := NewManageRegistryAuthUseCase(repo, nil)

	cred := domain.SavedRegistryCredential{
		Server:   "docker.io",
		Username: "dockeruser",
		Password: "password123",
	}
	if err := uc.Save(cred, domain.ServerConfig{}); err != nil {
		t.Fatalf("unexpected error saving registry credential: %v", err)
	}

	err := uc.Delete("docker.io", domain.ServerConfig{})
	if err != nil {
		t.Fatalf("expected no error deleting, got %v", err)
	}

	list, err := uc.List()
	if err != nil {
		t.Fatalf("unexpected error listing credentials: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 credentials after delete, got %d", len(list))
	}
}
