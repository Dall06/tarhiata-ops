package apiclient

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
			name:     "contains setupSecurityInterceptor",
			contains: "setupSecurityInterceptor",
		},
		{
			name:     "contains apiFetch",
			contains: "apiFetch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := string(GetJSContent())
			if !strings.Contains(content, tt.contains) {
				t.Errorf("expected JS content to contain %q", tt.contains)
			}
		})
	}
}
