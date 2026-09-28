package telemetry

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
			name:     "contains refreshServerTelemetry",
			contains: "export async function refreshServerTelemetry",
		},
		{
			name:     "contains deactivateInitialSkeletons",
			contains: "export function deactivateInitialSkeletons",
		},
		{
			name:     "contains processSwarmStatus",
			contains: "export function processSwarmStatus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := string(GetJSContent())
			if !strings.Contains(content, tt.contains) {
				t.Errorf("expected telemetry JS content to contain %q", tt.contains)
			}
		})
	}
}
