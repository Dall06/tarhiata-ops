package store

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
			name:     "contains state export",
			contains: "export const state",
		},
		{
			name:     "contains servers property",
			contains: "servers:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := string(GetJSContent())
			if !strings.Contains(content, tt.contains) {
				t.Errorf("expected state JS content to contain %q", tt.contains)
			}
		})
	}
}
