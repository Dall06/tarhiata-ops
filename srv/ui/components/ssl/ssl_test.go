package ssl

import (
	"testing"
	"time"
)

func TestCalculateRemainingDays(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		expiry   time.Time
		expected int
	}{
		{
			name:     "30 days future",
			expiry:   now.AddDate(0, 0, 30),
			expected: 30,
		},
		{
			name:     "already expired",
			expiry:   now.AddDate(0, 0, -5),
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateRemainingDays(tc.expiry, now)
			if got != tc.expected {
				t.Errorf("CalculateRemainingDays() = %d; want %d", got, tc.expected)
			}
		})
	}
}

func TestDetermineSSLStatus(t *testing.T) {
	tests := []struct {
		name          string
		hasCert       bool
		daysRemaining int
		expected      string
	}{
		{name: "no cert", hasCert: false, daysRemaining: 0, expected: "http_only"},
		{name: "expired", hasCert: true, daysRemaining: 0, expected: "expired"},
		{name: "expiring soon", hasCert: true, daysRemaining: 10, expected: "expiring_soon"},
		{name: "active valid", hasCert: true, daysRemaining: 60, expected: "active"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DetermineSSLStatus(tc.hasCert, tc.daysRemaining)
			if got != tc.expected {
				t.Errorf("DetermineSSLStatus(%v, %d) = %q; want %q", tc.hasCert, tc.daysRemaining, got, tc.expected)
			}
		})
	}
}
