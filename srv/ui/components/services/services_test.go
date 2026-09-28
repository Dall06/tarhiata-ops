package services

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
			name:     "contains renderAppCards",
			contains: "export function renderAppCards",
		},
		{
			name:     "contains openDeployModal",
			contains: "export function openDeployModal",
		},
		{
			name:     "contains restartServiceOrContainer",
			contains: "export async function restartServiceOrContainer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := string(GetJSContent())
			if !strings.Contains(content, tt.contains) {
				t.Errorf("expected services JS content to contain %q", tt.contains)
			}
		})
	}
}
