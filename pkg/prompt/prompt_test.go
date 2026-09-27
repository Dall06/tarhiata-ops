package prompt

import (
	"strings"
	"testing"
)

func TestConfirmWithReader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "lowercase s", input: "s\n", expected: true},
		{name: "lowercase si", input: "si\n", expected: true},
		{name: "uppercase Y", input: "Y\n", expected: true},
		{name: "lowercase yes", input: "yes\n", expected: true},
		{name: "lowercase n", input: "n\n", expected: false},
		{name: "empty newline", input: "\n", expected: false},
		{name: "random word", input: "cancel\n", expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := strings.NewReader(tc.input)
			got, err := ConfirmWithReader(r, "¿Deseas continuar?")
			if err != nil {
				t.Fatalf("ConfirmWithReader error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("ConfirmWithReader(%q) = %v; want %v", tc.input, got, tc.expected)
			}
		})
	}
}
