package modal

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
			name:     "contains openModal function",
			contains: "export function openModal",
		},
		{
			name:     "contains closeModal function",
			contains: "export function closeModal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := string(GetJSContent())
			if !strings.Contains(content, tt.contains) {
				t.Errorf("expected modal JS content to contain %q", tt.contains)
			}
		})
	}
}
