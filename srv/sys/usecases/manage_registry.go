package usecases

import (
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type defaultManageRegistryAuthUseCase struct {
	repo    ports.ConfigRepository
	sshExec ports.SSHExecutor
}

func NewManageRegistryAuthUseCase(repo ports.ConfigRepository, sshExec ports.SSHExecutor) ports.ManageRegistryAuthUseCase {
	return &defaultManageRegistryAuthUseCase{
		repo:    repo,
		sshExec: sshExec,
	}
}

func (uc *defaultManageRegistryAuthUseCase) Save(cred domain.SavedRegistryCredential, config domain.ServerConfig) error {
	if strings.TrimSpace(cred.Server) == "" {
		cred.Server = "docker.io"
	}
	cred.Server = strings.ToLower(strings.TrimSpace(cred.Server))
	if strings.TrimSpace(cred.Username) == "" || strings.TrimSpace(cred.Password) == "" {
		return fmt.Errorf("el usuario y la contraseña/token del registry son requeridos")
	}

	// If SSH connection is available, perform docker login on the VPS
	if uc.sshExec != nil && config.Host != "" {
		loginCmd := fmt.Sprintf("docker login %s -u '%s' -p '%s'", cred.Server, cred.Username, cred.Password)
		res, err := uc.sshExec.RunCommand(loginCmd)
		if err != nil || (res != nil && res.ExitCode != 0) {
			output := ""
			if res != nil {
				output = res.Output
			}
			return fmt.Errorf("falló la autenticación 'docker login %s': %s", cred.Server, output)
		}
	}

	// Persist credentials in SQLite
	return uc.repo.SaveRegistryCredential(cred)
}

func (uc *defaultManageRegistryAuthUseCase) List() ([]domain.SavedRegistryCredential, error) {
	return uc.repo.GetRegistryCredentials()
}

func (uc *defaultManageRegistryAuthUseCase) Delete(server string, config domain.ServerConfig) error {
	server = strings.ToLower(strings.TrimSpace(server))
	if server == "" {
		return fmt.Errorf("el servidor de registry es requerido")
	}

	if uc.sshExec != nil && config.Host != "" {
		logoutCmd := fmt.Sprintf("docker logout %s || true", server)
		uc.sshExec.RunCommand(logoutCmd)
	}

	return uc.repo.DeleteRegistryCredential(server)
}
