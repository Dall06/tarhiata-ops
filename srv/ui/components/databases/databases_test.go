package databases

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
			name:     "contains renderDatabaseCards",
			contains: "export function renderDatabaseCards",
		},
		{
			name:     "contains triggerDatabaseBackup",
			contains: "export async function triggerDatabaseBackup",
		},
		{
			name:     "contains openDeployDBModal",
			contains: "export function openDeployDBModal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := string(GetJSContent())
			if !strings.Contains(content, tt.contains) {
				t.Errorf("expected databases JS content to contain %q", tt.contains)
			}
		})
	}
}
