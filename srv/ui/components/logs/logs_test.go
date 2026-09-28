package logs

import (
	"testing"
)

func TestFilterLogs(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		query    string
		expected int
	}{
		{
			name:     "matching query",
			raw:      "INFO: started\nWARN: high memory\nERROR: failed to connect\nINFO: ended",
			query:    "error",
			expected: 1,
		},
		{
			name:     "empty query returns all",
			raw:      "line1\nline2\nline3",
			query:    "",
			expected: 3,
		},
		{
			name:     "no match",
			raw:      "line1\nline2",
			query:    "notfound",
			expected: 0,
		},
		{
			name:     "empty raw",
			raw:      "",
			query:    "test",
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FilterLogs(tc.raw, tc.query)
			if len(got) != tc.expected {
				t.Errorf("FilterLogs() len = %d; want %d", len(got), tc.expected)
			}
		})
	}
}

func TestSanitizeTailLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{name: "valid 50", input: "50", expected: 50},
		{name: "invalid text", input: "abc", expected: 100},
		{name: "zero", input: "0", expected: 100},
		{name: "too large capped at 2000", input: "9999", expected: 2000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeTailLines(tc.input)
			if got != tc.expected {
				t.Errorf("SanitizeTailLines(%q) = %d; want %d", tc.input, got, tc.expected)
			}
		})
	}
}
