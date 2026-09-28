package topology

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
			name:     "contains renderTopologyServicesTable",
			contains: "export function renderTopologyServicesTable",
		},
		{
			name:     "contains loadServiceLinks",
			contains: "export async function loadServiceLinks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := string(GetJSContent())
			if !strings.Contains(content, tt.contains) {
				t.Errorf("expected topology JS content to contain %q", tt.contains)
			}
		})
	}
}
