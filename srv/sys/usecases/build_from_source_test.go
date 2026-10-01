package usecases

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestBuildFromSourceUseCase_Execute(t *testing.T) {
	baseSvc := domain.SavedService{
		Name:       "web-api",
		SourceType: "git",
		GitRepoURL: "https://github.com/org/repo.git",
		GitBranch:  "main",
	}
	cfg := domain.ServerConfig{Host: "1.2.3.4"}

	t.Run("happy path: clone, checkout, build y push en orden, tag correcto", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewBuildFromSourceUseCase(mockSSH)

		var lines []string
		tag, err := uc.Execute(baseSvc, "abc1234", cfg, func(line string) { lines = append(lines, line) })
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantTag := RegistryInternalAddress + "/web-api:abc1234"
		if tag != wantTag {
			t.Errorf("tag inesperado: got %q, want %q", tag, wantTag)
		}

		joined := strings.Join(mockSSH.CommandsExecuted, " ||| ")
		for _, must := range []string{"git clone --branch 'main'", "git checkout 'abc1234'", "docker build -f 'Dockerfile' -t '" + wantTag + "'", "docker push '" + wantTag + "'", "rm -rf"} {
			if !strings.Contains(joined, must) {
				t.Errorf("se esperaba el fragmento %q en los comandos ejecutados: %s", must, joined)
			}
		}
		if len(lines) == 0 {
			t.Error("se esperaban líneas de progreso emitidas via onLine")
		}
	})

	t.Run("sin GitRepoURL falla antes de tocar SSH", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewBuildFromSourceUseCase(mockSSH)

		if _, err := uc.Execute(domain.SavedService{Name: "x"}, "abc", cfg, nil); err == nil {
			t.Fatal("se esperaba error por falta de GitRepoURL")
		}
		if len(mockSSH.CommandsExecuted) != 0 {
			t.Errorf("no debió ejecutarse ningún comando SSH: %v", mockSSH.CommandsExecuted)
		}
	})

	t.Run("commit/tag de referencia con metacaracteres se rechaza", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewBuildFromSourceUseCase(mockSSH)

		if _, err := uc.Execute(baseSvc, "abc; rm -rf / #", cfg, nil); err == nil {
			t.Fatal("se esperaba error por commit ref inválido")
		}
	})

	t.Run("PAT se inyecta en la URL de clone pero no en texto plano reconocible como comando separado", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewBuildFromSourceUseCase(mockSSH)
		svc := baseSvc
		svc.GitAccessToken = "ghp_secret"

		if _, err := uc.Execute(svc, "abc1234", cfg, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var cloneCmd string
		for _, c := range mockSSH.CommandsExecuted {
			if strings.HasPrefix(c, "git clone") {
				cloneCmd = c
			}
		}
		want := "https://x-access-token:ghp_secret@github.com/org/repo.git"
		if !strings.Contains(cloneCmd, want) {
			t.Errorf("se esperaba el PAT inyectado en la URL de clone: %s", cloneCmd)
		}
	})

	t.Run("clone fallido no intenta build ni push, y limpia el workdir", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		mockSSH.MockResponses["git clone"] = &domain.CommandResult{Output: "repository not found", ExitCode: 128}
		uc := NewBuildFromSourceUseCase(mockSSH)

		if _, err := uc.Execute(baseSvc, "abc1234", cfg, nil); err == nil || !strings.Contains(err.Error(), "repository not found") {
			t.Fatalf("se esperaba error de clone, got: %v", err)
		}
		for _, c := range mockSSH.CommandsExecuted {
			if strings.HasPrefix(c, "docker build") || strings.HasPrefix(c, "docker push") {
				t.Errorf("no debió intentarse build/push tras un clone fallido: %s", c)
			}
		}
		var cleaned bool
		for _, c := range mockSSH.CommandsExecuted {
			if strings.HasPrefix(c, "rm -rf") {
				cleaned = true
			}
		}
		if !cleaned {
			t.Error("se esperaba limpieza del workdir tras el fallo")
		}
	})

	t.Run("DockerfilePath personalizado se respeta", func(t *testing.T) {
		mockSSH := mocks.NewMockSSHExecutor()
		uc := NewBuildFromSourceUseCase(mockSSH)
		svc := baseSvc
		svc.DockerfilePath = "docker/Dockerfile.prod"

		if _, err := uc.Execute(svc, "abc1234", cfg, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var found bool
		for _, c := range mockSSH.CommandsExecuted {
			if strings.Contains(c, fmt.Sprintf("-f %s", "'docker/Dockerfile.prod'")) {
				found = true
			}
		}
		if !found {
			t.Errorf("no se usó el DockerfilePath personalizado: %v", mockSSH.CommandsExecuted)
		}
	})
}
