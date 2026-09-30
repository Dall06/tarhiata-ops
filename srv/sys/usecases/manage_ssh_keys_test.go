package usecases

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestManageSSHKeysUseCase(t *testing.T) {
	mockSSH := mocks.NewMockSSHExecutor()
	mockSSH.MockResponses["cat /root/.ssh/authorized_keys 2>/dev/null"] = &domain.CommandResult{
		Output:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC3 dev-master@vultr\nssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIL1 dev-teammate@laptop\n",
		ExitCode: 0,
	}

	uc := NewManageSSHKeysUseCase(mockSSH)
	cfg := domain.ServerConfig{Host: "157.230.12.8"}

	// 1. Probar listado
	keys, err := uc.ListKeys(cfg)
	if err != nil {
		t.Fatalf("Error listando llaves SSH: %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("Se esperaban 2 llaves SSH, obtenidas: %d", len(keys))
	}

	// 2. Probar adición de llave
	newKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINewKey dev-new@laptop"
	if err := uc.AddKey(cfg, newKey); err != nil {
		t.Fatalf("Error añadiendo llave SSH: %v", err)
	}

	// 3. Probar borrado de llave secundaria (dev-teammate)
	keys[1].IsVultrKey = false
	keys[1].Protected = false
	if err := uc.DeleteKey(cfg, keys[1].Fingerprint); err != nil {
		t.Fatalf("Error eliminando llave SSH secundaria: %v", err)
	}

	// 4. Probar que NO se puede eliminar la llave de Vultr
	keys[0].IsVultrKey = true
	keys[0].Protected = true
	mockSSH.MockResponses["cat /root/.ssh/authorized_keys 2>/dev/null"] = &domain.CommandResult{
		Output:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC3 dev-master@vultr\n",
		ExitCode: 0,
	}
	err = uc.DeleteKey(cfg, keys[0].Fingerprint)
	if err == nil {
		t.Fatalf("Se esperaba error al intentar eliminar la llave protegida de Vultr, pero se permitió")
	}
}

// TestManageSSHKeysUseCase_ListKeys_MarksVultrRegisteredKeys valida el flujo completo con
// el cliente Vultr inyectado (antes imposible de testear sin red real, al depender de un
// *http.Client concreto): una llave que SÍ está registrada en la cuenta de Vultr debe
// marcarse como IsVultrKey/Protected.
func TestManageSSHKeysUseCase_ListKeys_MarksVultrRegisteredKeys(t *testing.T) {
	mockSSH := mocks.NewMockSSHExecutor()
	mockSSH.MockResponses["cat /root/.ssh/authorized_keys 2>/dev/null"] = &domain.CommandResult{
		Output:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC3 dev-master@vultr\n",
		ExitCode: 0,
	}

	fake := &fakeVultrClient{sshKeys: []domain.VultrSSHKey{
		{ID: "1", Name: "master", SSHKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC3"},
	}}
	uc := NewManageSSHKeysUseCase(mockSSH).WithVultrClient(fake)

	keys, err := uc.ListKeys(domain.ServerConfig{Host: "1.2.3.4", VultrAPIToken: "test-token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 1 || !keys[0].IsVultrKey || !keys[0].Protected {
		t.Errorf("se esperaba 1 llave marcada como registrada en Vultr, got %+v", keys)
	}
}
