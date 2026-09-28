package toast

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
			name:     "contains showToast function",
			contains: "export function showToast",
		},
		{
			name:     "contains toast-container selector",
			contains: "toastContainer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := string(GetJSContent())
			if !strings.Contains(content, tt.contains) {
				t.Errorf("expected toast JS content to contain %q", tt.contains)
			}
		})
	}
}
