package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSender_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		serverCode  int
		sendFunc    func(s *Sender, ctx context.Context, url string, ev AlertEvent) error
		expectError bool
	}{
		{
			name:       "Discord success",
			serverCode: http.StatusOK,
			sendFunc: func(s *Sender, ctx context.Context, url string, ev AlertEvent) error {
				return s.SendDiscord(ctx, url, ev)
			},
			expectError: false,
		},
		{
			name:       "Discord error status 400",
			serverCode: http.StatusBadRequest,
			sendFunc: func(s *Sender, ctx context.Context, url string, ev AlertEvent) error {
				return s.SendDiscord(ctx, url, ev)
			},
			expectError: true,
		},
		{
			name:       "Slack success",
			serverCode: http.StatusOK,
			sendFunc: func(s *Sender, ctx context.Context, url string, ev AlertEvent) error {
				return s.SendSlack(ctx, url, ev)
			},
			expectError: false,
		},
		{
			name:       "Slack error status 500",
			serverCode: http.StatusInternalServerError,
			sendFunc: func(s *Sender, ctx context.Context, url string, ev AlertEvent) error {
				return s.SendSlack(ctx, url, ev)
			},
			expectError: true,
		},
		{
			name:       "Generic Webhook success",
			serverCode: http.StatusOK,
			sendFunc: func(s *Sender, ctx context.Context, url string, ev AlertEvent) error {
				return s.SendGeneric(ctx, url, ev)
			},
			expectError: false,
		},
		{
			name:       "Generic Webhook error status 404",
			serverCode: http.StatusNotFound,
			sendFunc: func(s *Sender, ctx context.Context, url string, ev AlertEvent) error {
				return s.SendGeneric(ctx, url, ev)
			},
			expectError: true,
		},
	}

	event := AlertEvent{
		Title:       "Test Alert",
		Description: "High CPU detected on swarm-worker-1",
		Severity:    SeverityCritical,
		ServerName:  "vps-prod-01",
		Resource:    "tarhiata-app-backend",
		Fields:      map[string]string{"CPU": "94.2%", "RAM": "82.1%"},
		Timestamp:   time.Now(),
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.serverCode)
			}))
			defer server.Close()

			sender := NewSender(3 * time.Second)
			err := tc.sendFunc(sender, context.Background(), server.URL, event)
			if (err != nil) != tc.expectError {
				t.Fatalf("sendFunc() error = %v, expectError = %v", err, tc.expectError)
			}
		})
	}
}

func TestSender_Dispatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender(3 * time.Second)

	cfg := WebhookConfig{
		Enabled:    true,
		DiscordURL: server.URL,
		SlackURL:   server.URL,
		GenericURL: server.URL,
	}

	ev := AlertEvent{
		Title:       "System Healthy",
		Description: "All nodes active",
		Severity:    SeveritySuccess,
	}

	errs := sender.Dispatch(context.Background(), cfg, ev)
	if len(errs) != 0 {
		t.Fatalf("Dispatch() unexpected errors: %v", errs)
	}

	// Disabled should return immediately with no errors
	cfg.Enabled = false
	errsDisabled := sender.Dispatch(context.Background(), cfg, ev)
	if len(errsDisabled) != 0 {
		t.Fatalf("Dispatch() when disabled should return empty error list")
	}
}
