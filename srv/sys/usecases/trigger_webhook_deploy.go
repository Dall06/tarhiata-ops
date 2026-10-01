package usecases

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Dall06/tarhiata-ops/pkg/validator"
	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

// ExtractCommitSHA busca el commit SHA en un payload de webhook de GitHub, GitLab o
// Gitea. Las 3 plataformas usan alguno de estos campos en sus eventos de "push";
// devuelve "" si no se encuentra ninguno (ej. payload genérico sin esos campos).
func ExtractCommitSHA(bodyBytes []byte) string {
	var payload struct {
		After       string `json:"after"`
		CheckoutSHA string `json:"checkout_sha"`
		HeadCommit  struct {
			ID string `json:"id"`
		} `json:"head_commit"`
	}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return ""
	}
	if payload.After != "" {
		return payload.After
	}
	if payload.CheckoutSHA != "" {
		return payload.CheckoutSHA
	}
	return payload.HeadCommit.ID
}

// TriggerWebhookDeployUseCase procesa eventos entrantes de Git Webhooks (GitHub, GitLab, Gitea) para auto-despliegue.
type TriggerWebhookDeployUseCase struct {
	repo     ports.ConfigRepository
	executor ports.SSHExecutor
}

func NewTriggerWebhookDeployUseCase(repo ports.ConfigRepository, executor ports.SSHExecutor) *TriggerWebhookDeployUseCase {
	return &TriggerWebhookDeployUseCase{
		repo:     repo,
		executor: executor,
	}
}

// VerifySignature verifica la firma HMAC SHA256 enviada por GitHub/Gitea contra un
// secreto guardado del lado del servidor. Sin secreto configurado no hay nada contra qué
// validar, así que se rechaza (fail-closed) en vez de permitir la petición.
func VerifySignature(secret string, body []byte, signatureHeader string) bool {
	if secret == "" {
		return false
	}
	signatureHeader = strings.TrimPrefix(signatureHeader, "sha256=")
	if signatureHeader == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)
	expectedHex := hex.EncodeToString(expectedMAC)

	return hmac.Equal([]byte(signatureHeader), []byte(expectedHex))
}

// Execute procesa la solicitud de auto-despliegue para el servicio indicado.
func (uc *TriggerWebhookDeployUseCase) Execute(serviceName, imageTag string, config domain.ServerConfig) (*domain.DeploymentRecord, error) {
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return nil, errors.New("nombre de servicio requerido")
	}

	// 1. Obtener servicio guardado en catálogo
	svc, err := uc.repo.GetService(serviceName)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo servicio: %w", err)
	}

	targetImage := imageTag
	if targetImage == "" && svc != nil && svc.ImageSource != "" {
		targetImage = svc.ImageSource
	}
	if targetImage == "" {
		targetImage = serviceName + ":latest"
	}

	// 2. Conectar SSH
	if err := uc.executor.Connect(config); err != nil {
		return nil, fmt.Errorf("error conectando SSH: %w", err)
	}
	defer func() {
		if clErr := uc.executor.Close(); clErr != nil {
			slog.Warn("trigger_webhook_deploy: error cerrando conexión SSH", "error", clErr)
		}
	}()

	// 3. Ejecutar actualización del servicio en Swarm
	quotedImage := validator.ShellQuote(targetImage)
	quotedService := validator.ShellQuote(serviceName)
	updateCmd := fmt.Sprintf("docker service update --image %s --force %s || docker service update --image %s --force %s || docker service update --image %s --force %s",
		quotedImage, quotedService,
		quotedImage, validator.ShellQuote("tarhiata-app-"+serviceName),
		quotedImage, validator.ShellQuote(serviceName+"_"+serviceName),
	)

	cmdRes, err := uc.executor.RunCommand(updateCmd)
	if err != nil {
		return nil, fmt.Errorf("falló actualización de servicio en Swarm: %w", err)
	}
	if cmdRes != nil && cmdRes.ExitCode != 0 {
		return nil, fmt.Errorf("docker service update retornó error %d: %s", cmdRes.ExitCode, cmdRes.Output)
	}

	// 4. Registrar en historial
	port := 80
	domainStr := ""
	expose := false
	if svc != nil {
		port = svc.Port
		domainStr = svc.Domain
		expose = svc.Expose
	}

	rec := domain.DeploymentRecord{
		ServiceName: serviceName,
		ImageTag:    targetImage,
		Port:        port,
		Domain:      domainStr,
		Expose:      expose,
		DeployedAt:  time.Now(),
		Status:      "success",
	}

	if errSave := uc.repo.SaveDeploymentRecord(rec); errSave != nil {
		// Loguear o continuar
	}

	return &rec, nil
}
