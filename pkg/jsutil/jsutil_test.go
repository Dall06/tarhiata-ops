package jsutil

import (
	"strings"
	"testing"
)

func TestGetJSContent(t *testing.T) {
	tests := []struct {
		name     string
		contains string
	}{
		{
			name:     "contains escapeHtml function",
			contains: "export function escapeHtml",
		},
		{
			name:     "contains getGaugeColor function",
			contains: "export function getGaugeColor",
		},
		{
			name:     "contains formatFileSize function",
			contains: "export function formatFileSize",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := string(GetJSContent())
			if !strings.Contains(content, tt.contains) {
				t.Errorf("expected jsutil JS content to contain %q", tt.contains)
			}
		})
	}
}
