package nodes

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
			name:     "contains renderNodesTable",
			contains: "export function renderNodesTable",
		},
		{
			name:     "contains openWorkerModal",
			contains: "export function openWorkerModal",
		},
		{
			name:     "el reload de disponibilidad/expulsión de nodo tiene fallback a window.loadSwarmStatus",
			contains: "callbacks.onReloadStatus || window.loadSwarmStatus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := string(GetJSContent())
			if !strings.Contains(content, tt.contains) {
				t.Errorf("expected nodes JS content to contain %q", tt.contains)
			}
		})
	}
}
