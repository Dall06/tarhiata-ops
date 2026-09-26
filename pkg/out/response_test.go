package out

import (
	"encoding/json"
	"testing"
)

func TestResponseConstructors_TableDriven(t *testing.T) {
	tests := []struct {
		name            string
		build           func() Response
		expectedSuccess bool
		checkData       func(t *testing.T, r Response)
	}{
		{
			name: "OK constructor with string data",
			build: func() Response {
				return OK("test-data")
			},
			expectedSuccess: true,
			checkData: func(t *testing.T, r Response) {
				if r.Data != "test-data" {
					t.Errorf("expected data 'test-data', got %v", r.Data)
				}
				if r.CalledAt.IsZero() {
					t.Error("expected CalledAt to be non-zero")
				}
			},
		},
		{
			name: "Msg constructor with status message",
			build: func() Response {
				return Msg("service restarted successfully")
			},
			expectedSuccess: true,
			checkData: func(t *testing.T, r Response) {
				if r.Message != "service restarted successfully" {
					t.Errorf("expected message 'service restarted successfully', got %s", r.Message)
				}
				if r.Data != nil {
					t.Errorf("expected nil data, got %v", r.Data)
				}
			},
		},
		{
			name: "Fail constructor with failure message",
			build: func() Response {
				return Fail("connection timeout")
			},
			expectedSuccess: false,
			checkData: func(t *testing.T, r Response) {
				if r.Message != "connection timeout" {
					t.Errorf("expected message 'connection timeout', got %s", r.Message)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tt.build()
			if resp.Success != tt.expectedSuccess {
				t.Errorf("expected Success=%v, got %v", tt.expectedSuccess, resp.Success)
			}
			tt.checkData(t, resp)

			dataBytes, errMarshal := json.Marshal(resp)
			if errMarshal != nil {
				t.Fatalf("failed to marshal Response to JSON: %v", errMarshal)
			}
			if len(dataBytes) == 0 {
				t.Error("expected non-empty JSON bytes")
			}
		})
	}
}
