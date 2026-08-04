package usecases

import (
	"fmt"
	"strings"

	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type PruneSystemUseCase struct {
	sshExec ports.SSHExecutor
}

func NewPruneSystemUseCase(sshExec ports.SSHExecutor) *PruneSystemUseCase {
	return &PruneSystemUseCase{sshExec: sshExec}
}

func (uc *PruneSystemUseCase) Execute() (string, error) {
	res, err := uc.sshExec.RunCommand("docker system prune -af")
	if err != nil {
		return "", fmt.Errorf("error ejecutando docker system prune: %w", err)
	}
	output := strings.TrimSpace(res.Output)
	if output == "" {
		output = "✔ Docker system prune finalizado exitosamente. El sistema está limpio."
	}
	return output, nil
}
