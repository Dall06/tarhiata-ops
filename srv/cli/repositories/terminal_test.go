package repositories

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/cli/domain"
)

func TestTerminalPresenter_PrintCard(t *testing.T) {
	tests := []struct {
		name     string
		card     domain.DashboardCard
		expected string
	}{
		{
			name: "renders standard card with unit",
			card: domain.DashboardCard{
				Title: "CPU",
				Value: "15",
				Unit:  "%",
				Color: "#3291FF",
			},
			expected: "[#3291FF] CPU: 15 %",
		},
		{
			name: "renders card without unit",
			card: domain.DashboardCard{
				Title: "Nodes",
				Value: "3",
				Unit:  "",
				Color: "#50E3C2",
			},
			expected: "[#50E3C2] Nodes: 3 ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			presenter := NewTerminalPresenterWithWriter(&buf)
			err := presenter.PrintCard(tt.card)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(buf.String(), tt.expected) {
				t.Errorf("expected output to contain %q, got %q", tt.expected, buf.String())
			}
		})
	}
}
