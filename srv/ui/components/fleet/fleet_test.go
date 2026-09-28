package fleet

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
			name:     "contains renderFleetDirectory",
			contains: "export function renderFleetDirectory",
		},
		{
			name:     "contains switchActiveServer",
			contains: "export async function switchActiveServer",
		},
		{
			name:     "contains testAllServers",
			contains: "export async function testAllServers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := string(GetJSContent())
			if !strings.Contains(content, tt.contains) {
				t.Errorf("expected fleet JS content to contain %q", tt.contains)
			}
		})
	}
}
