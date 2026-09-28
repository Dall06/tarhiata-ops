package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AlertSeverity representa el nivel de severidad para formatear colores e iconos.
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "INFO"
	SeveritySuccess  AlertSeverity = "SUCCESS"
	SeverityWarning  AlertSeverity = "WARNING"
	SeverityCritical AlertSeverity = "CRITICAL"
)

// WebhookConfig almacena la configuración de destinos para notificaciones salientes.
type WebhookConfig struct {
	DiscordURL    string `json:"discordUrl,omitempty"`
	TelegramToken string `json:"telegramToken,omitempty"`
	TelegramChat  string `json:"telegramChat,omitempty"`
	SlackURL      string `json:"slackUrl,omitempty"`
	GenericURL    string `json:"genericUrl,omitempty"`
	Enabled       bool   `json:"enabled"`
}

// AlertEvent encapsula un evento de alerta emitido por el sistema.
type AlertEvent struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Severity    AlertSeverity     `json:"severity"`
	ServerName  string            `json:"serverName,omitempty"`
	Resource    string            `json:"resource,omitempty"`
	Fields      map[string]string `json:"fields,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// Sender gestiona el despacho concurrente de alertas a canales externos.
type Sender struct {
	client *http.Client
}

// NewSender crea un nuevo despachador de alertas con timeout seguro.
func NewSender(timeout time.Duration) *Sender {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Sender{
		client: &http.Client{Timeout: timeout},
	}
}

// Dispatch envía la alerta a todos los canales configurados en WebhookConfig.
func (s *Sender) Dispatch(ctx context.Context, cfg WebhookConfig, event AlertEvent) []error {
	if !cfg.Enabled {
		return nil
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	var errs []error

	if strings.TrimSpace(cfg.DiscordURL) != "" {
		if err := s.SendDiscord(ctx, cfg.DiscordURL, event); err != nil {
			errs = append(errs, fmt.Errorf("discord: %w", err))
		}
	}

	if strings.TrimSpace(cfg.TelegramToken) != "" && strings.TrimSpace(cfg.TelegramChat) != "" {
		if err := s.SendTelegram(ctx, cfg.TelegramToken, cfg.TelegramChat, event); err != nil {
			errs = append(errs, fmt.Errorf("telegram: %w", err))
		}
	}

	if strings.TrimSpace(cfg.SlackURL) != "" {
		if err := s.SendSlack(ctx, cfg.SlackURL, event); err != nil {
			errs = append(errs, fmt.Errorf("slack: %w", err))
		}
	}

	if strings.TrimSpace(cfg.GenericURL) != "" {
		if err := s.SendGeneric(ctx, cfg.GenericURL, event); err != nil {
			errs = append(errs, fmt.Errorf("generic webhook: %w", err))
		}
	}

	return errs
}

// SendDiscord despacha un rich embed a un webhook de Discord.
func (s *Sender) SendDiscord(ctx context.Context, webhookURL string, event AlertEvent) error {
	color := 0x3b82f6 // Azul Info
	emoji := "ℹ️"
	switch event.Severity {
	case SeveritySuccess:
		color = 0x10b981 // Verde
		emoji = "✅"
	case SeverityWarning:
		color = 0xf59e0b // Naranja
		emoji = "⚠️"
	case SeverityCritical:
		color = 0xef4444 // Rojo
		emoji = "🚨"
	}

	type EmbedField struct {
		Name   string `json:"name"`
		Value  string `json:"value"`
		Inline bool   `json:"inline"`
	}

	fields := make([]EmbedField, 0, len(event.Fields)+2)
	if event.ServerName != "" {
		fields = append(fields, EmbedField{Name: "Servidor", Value: event.ServerName, Inline: true})
	}
	if event.Resource != "" {
		fields = append(fields, EmbedField{Name: "Recurso", Value: event.Resource, Inline: true})
	}
	for k, v := range event.Fields {
		fields = append(fields, EmbedField{Name: k, Value: v, Inline: true})
	}

	payload := map[string]interface{}{
		"username": "Tarhiata-Ops Monitor",
		"embeds": []map[string]interface{}{
			{
				"title":       fmt.Sprintf("%s [%s] %s", emoji, event.Severity, event.Title),
				"description": event.Description,
				"color":       color,
				"fields":      fields,
				"timestamp":   event.Timestamp.Format(time.RFC3339),
				"footer": map[string]string{
					"text": "Tarhiata Cloud Studio",
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord returned status %d", resp.StatusCode)
	}
	return nil
}

// SendTelegram envía un mensaje formateado en Markdown a un canal/chat de Telegram.
func (s *Sender) SendTelegram(ctx context.Context, botToken, chatID string, event AlertEvent) error {
	emoji := "ℹ️"
	switch event.Severity {
	case SeveritySuccess:
		emoji = "✅"
	case SeverityWarning:
		emoji = "⚠️"
	case SeverityCritical:
		emoji = "🚨"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s *[%s] %s*\n\n", emoji, event.Severity, escapeTelegramMarkdown(event.Title)))
	sb.WriteString(fmt.Sprintf("%s\n\n", escapeTelegramMarkdown(event.Description)))
	if event.ServerName != "" {
		sb.WriteString(fmt.Sprintf("🖥️ *Servidor:* `%s`\n", escapeTelegramMarkdown(event.ServerName)))
	}
	if event.Resource != "" {
		sb.WriteString(fmt.Sprintf("📦 *Recurso:* `%s`\n", escapeTelegramMarkdown(event.Resource)))
	}
	for k, v := range event.Fields {
		sb.WriteString(fmt.Sprintf("• *%s:* %s\n", escapeTelegramMarkdown(k), escapeTelegramMarkdown(v)))
	}
	sb.WriteString(fmt.Sprintf("\n⏱ _%s_", event.Timestamp.Format("2006-01-02 15:04:05 MST")))

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	form := url.Values{}
	form.Set("chat_id", chatID)
	form.Set("text", sb.String())
	form.Set("parse_mode", "Markdown")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram returned status %d", resp.StatusCode)
	}
	return nil
}

// SendSlack despacha una notificación en formato BlockKit a un incoming webhook de Slack.
func (s *Sender) SendSlack(ctx context.Context, webhookURL string, event AlertEvent) error {
	emoji := ":information_source:"
	switch event.Severity {
	case SeveritySuccess:
		emoji = ":white_check_mark:"
	case SeverityWarning:
		emoji = ":warning:"
	case SeverityCritical:
		emoji = ":rotating_light:"
	}

	headerText := fmt.Sprintf("%s [%s] %s", emoji, event.Severity, event.Title)
	blocks := []map[string]interface{}{
		{
			"type": "header",
			"text": map[string]string{
				"type": "plain_text",
				"text": headerText,
			},
		},
		{
			"type": "section",
			"text": map[string]string{
				"type": "mrkdwn",
				"text": event.Description,
			},
		},
	}

	payload := map[string]interface{}{
		"text":   headerText,
		"blocks": blocks,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}
	return nil
}

// SendGeneric envía el payload JSON estándar al webhook HTTP configurado.
func (s *Sender) SendGeneric(ctx context.Context, webhookURL string, event AlertEvent) error {
	bodyBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Tarhiata-Ops/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("generic webhook returned status %d", resp.StatusCode)
	}
	return nil
}

func escapeTelegramMarkdown(s string) string {
	r := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"`", "\\`",
	)
	return r.Replace(s)
}
