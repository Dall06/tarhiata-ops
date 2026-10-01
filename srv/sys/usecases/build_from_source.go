package usecases

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Dall06/tarhiata-ops/pkg/validator"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

type BuildFromSourceUseCase struct {
	ssh ports.SSHExecutor
}

func NewBuildFromSourceUseCase(ssh ports.SSHExecutor) *BuildFromSourceUseCase {
	return &BuildFromSourceUseCase{ssh: ssh}
}

// imageTagRegex exige un tag de imagen Docker válido (evita que un commit SHA
// inesperado o vacío produzca un tag inválido o, peor, inyecte algo al interpolarse).
var imageTagRegex = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.\-]{0,127}$`)

// Execute clona el repo del servicio (con el PAT si lo tiene), construye la imagen con
// el Dockerfile indicado y la publica en el registry privado interno. No despliega el
// servicio: el caller debe encadenar el redeploy (ej. TriggerWebhookDeployUseCase) con
// el tag que esta función devuelve, una vez que el build terminó bien.
func (uc *BuildFromSourceUseCase) Execute(svc domain.SavedService, commitRef string, config domain.ServerConfig, onLine func(line string)) (imageTag string, err error) {
	if strings.TrimSpace(svc.GitRepoURL) == "" {
		return "", fmt.Errorf("el servicio '%s' no tiene un GitRepoURL configurado", svc.Name)
	}

	tagRef := strings.TrimSpace(commitRef)
	if tagRef == "" {
		tagRef = "latest"
	}
	if !imageTagRegex.MatchString(tagRef) {
		return "", fmt.Errorf("commit/tag de referencia inválido: %q", commitRef)
	}

	branch := strings.TrimSpace(svc.GitBranch)
	if branch == "" {
		branch = "main"
	}
	dockerfilePath := strings.TrimSpace(svc.DockerfilePath)
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}

	emit := func(line string) {
		if onLine != nil {
			onLine(line)
		}
	}

	if err := uc.ssh.Connect(config); err != nil {
		return "", fmt.Errorf("error conectando SSH: %w", err)
	}
	defer func() {
		if clErr := uc.ssh.Close(); clErr != nil {
			emit(fmt.Sprintf("⚠️  aviso cerrando conexión SSH: %v", clErr))
		}
	}()

	workdir := fmt.Sprintf("/opt/tarhiata/builds/%s-%d", svc.Name, time.Now().UnixNano())
	cleanup := func() {
		if _, err := uc.ssh.RunCommand("rm -rf " + validator.ShellQuote(workdir)); err != nil {
			emit(fmt.Sprintf("⚠️  aviso limpiando workdir: %v", err))
		}
	}

	cloneURL := buildAuthenticatedCloneURL(svc.GitRepoURL, svc.GitAccessToken)
	emit(fmt.Sprintf("▶ Clonando %s (branch %s)...", svc.GitRepoURL, branch))
	cloneCmd := fmt.Sprintf("git clone --branch %s %s %s", validator.ShellQuote(branch), validator.ShellQuote(cloneURL), validator.ShellQuote(workdir))
	res, err := uc.ssh.RunCommandStreaming(cloneCmd, emit)
	if err != nil || res == nil || res.ExitCode != 0 {
		cleanup()
		return "", fmt.Errorf("falló git clone: %s", safeOutput(res, err))
	}

	if commitRef != "" {
		emit(fmt.Sprintf("▶ Posicionando en commit %s...", commitRef))
		checkoutCmd := fmt.Sprintf("cd %s && git checkout %s", validator.ShellQuote(workdir), validator.ShellQuote(commitRef))
		res, err = uc.ssh.RunCommandStreaming(checkoutCmd, emit)
		if err != nil || res == nil || res.ExitCode != 0 {
			cleanup()
			return "", fmt.Errorf("falló git checkout de %q: %s", commitRef, safeOutput(res, err))
		}
	}

	imageTag = fmt.Sprintf("%s/%s:%s", RegistryInternalAddress, svc.Name, tagRef)
	emit(fmt.Sprintf("▶ Construyendo imagen %s...", imageTag))
	buildCmd := fmt.Sprintf("cd %s && docker build -f %s -t %s .", validator.ShellQuote(workdir), validator.ShellQuote(dockerfilePath), validator.ShellQuote(imageTag))
	res, err = uc.ssh.RunCommandStreaming(buildCmd, emit)
	if err != nil || res == nil || res.ExitCode != 0 {
		cleanup()
		return "", fmt.Errorf("falló docker build: %s", safeOutput(res, err))
	}

	emit(fmt.Sprintf("▶ Publicando %s en el registry interno...", imageTag))
	pushCmd := fmt.Sprintf("docker push %s", validator.ShellQuote(imageTag))
	res, err = uc.ssh.RunCommandStreaming(pushCmd, emit)
	if err != nil || res == nil || res.ExitCode != 0 {
		cleanup()
		return "", fmt.Errorf("falló docker push: %s", safeOutput(res, err))
	}

	cleanup()
	emit("✅ Build completado")
	return imageTag, nil
}

// buildAuthenticatedCloneURL inyecta un PAT en una URL https de git como
// https://x-access-token:<token>@host/path, para repos privados. Si no hay token,
// devuelve la URL sin modificar (repo público).
func buildAuthenticatedCloneURL(rawURL, token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return rawURL
	}
	if rest, ok := strings.CutPrefix(rawURL, "https://"); ok {
		return "https://x-access-token:" + token + "@" + rest
	}
	return rawURL
}

func safeOutput(res *domain.CommandResult, err error) string {
	if res != nil && res.Output != "" {
		return res.Output
	}
	if err != nil {
		return err.Error()
	}
	return "sin detalle"
}
