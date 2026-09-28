package domain

import (
	"testing"
)

func TestDecodeServerConfig(t *testing.T) {
	tests := []struct {
		name       string
		jsonInput  string
		wantName   string
		wantKey    string
		wantErr    bool
	}{
		{
			name:      "camelCase and key_path",
			jsonInput: `{"name":"prod-vps","host":"1.2.3.4","key_path":"/id_rsa"}`,
			wantName:  "prod-vps",
			wantKey:   "/id_rsa",
			wantErr:   false,
		},
		{
			name:      "invalid json",
			jsonInput: `{"name": invalid}`,
			wantName:  "",
			wantKey:   "",
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DecodeServerConfig([]byte(tc.jsonInput))
			if (err != nil) != tc.wantErr {
				t.Fatalf("DecodeServerConfig() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr {
				if got.Name != tc.wantName {
					t.Errorf("got.Name = %q, want %q", got.Name, tc.wantName)
				}
				if got.PrivateKey != tc.wantKey {
					t.Errorf("got.PrivateKey = %q, want %q", got.PrivateKey, tc.wantKey)
				}
			}
		})
	}
}

func TestDecodeSavedService(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		wantName  string
		wantImage string
		wantErr   bool
	}{
		{
			name:      "snake_case image_source",
			jsonInput: `{"name":"api-app","image_source":"myrepo/api:v1"}`,
			wantName:  "api-app",
			wantImage: "myrepo/api:v1",
			wantErr:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DecodeSavedService([]byte(tc.jsonInput))
			if (err != nil) != tc.wantErr {
				t.Fatalf("DecodeSavedService() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr {
				if got.Name != tc.wantName {
					t.Errorf("got.Name = %q, want %q", got.Name, tc.wantName)
				}
				if got.ImageSource != tc.wantImage {
					t.Errorf("got.ImageSource = %q, want %q", got.ImageSource, tc.wantImage)
				}
			}
		})
	}
}

func TestDecodeServiceLink(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		wantSrc   string
		wantTgt   string
		wantErr   bool
	}{
		{
			name:      "snake_case service link",
			jsonInput: `{"source_svc":"web","target_svc":"db","env_var_name":"DB_URL"}`,
			wantSrc:   "web",
			wantTgt:   "db",
			wantErr:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DecodeServiceLink([]byte(tc.jsonInput))
			if (err != nil) != tc.wantErr {
				t.Fatalf("DecodeServiceLink() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr {
				if got.SourceSvc != tc.wantSrc {
					t.Errorf("got.SourceSvc = %q, want %q", got.SourceSvc, tc.wantSrc)
				}
				if got.TargetSvc != tc.wantTgt {
					t.Errorf("got.TargetSvc = %q, want %q", got.TargetSvc, tc.wantTgt)
				}
			}
		})
	}
}

func TestDecodeSavedDatabase(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		wantName  string
		wantPort  int
		wantErr   bool
	}{
		{
			name:      "snake_case database",
			jsonInput: `{"name":"postgres","internal_port":5432}`,
			wantName:  "postgres",
			wantPort:  5432,
			wantErr:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DecodeSavedDatabase([]byte(tc.jsonInput))
			if (err != nil) != tc.wantErr {
				t.Fatalf("DecodeSavedDatabase() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr {
				if got.Name != tc.wantName {
					t.Errorf("got.Name = %q, want %q", got.Name, tc.wantName)
				}
				if got.InternalPort != tc.wantPort {
					t.Errorf("got.InternalPort = %d, want %d", got.InternalPort, tc.wantPort)
				}
			}
		})
	}
}
