package osterminal

import (
	"strings"
	"testing"
)

func TestBuildSSHCommand(t *testing.T) {
	tests := []struct {
		name       string
		user       string
		host       string
		port       int
		key        string
		wantPrefix string
		wantSubstr string
	}{
		{
			name:       "Standard remote host default port 22",
			user:       "root",
			host:       "108.61.33.61",
			port:       22,
			key:        "~/.ssh/id_rsa",
			wantPrefix: "ssh -o StrictHostKeyChecking=accept-new",
			wantSubstr: "-i ~/.ssh/id_rsa root@108.61.33.61",
		},
		{
			name:       "Custom port and user",
			user:       "ubuntu",
			host:       "staging.example.com",
			port:       2222,
			key:        "/path/to/key.pem",
			wantPrefix: "ssh -o StrictHostKeyChecking=accept-new",
			wantSubstr: "-p 2222 ubuntu@staging.example.com",
		},
		{
			name:       "Empty user defaults to root",
			user:       "",
			host:       "192.168.1.100",
			port:       22,
			key:        "",
			wantPrefix: "ssh -o StrictHostKeyChecking=accept-new",
			wantSubstr: "root@192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildSSHCommand(tt.user, tt.host, tt.port, tt.key)
			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("BuildSSHCommand() prefix = %v, wantPrefix %v", got, tt.wantPrefix)
			}
			if !strings.Contains(got, tt.wantSubstr) {
				t.Errorf("BuildSSHCommand() = %v, want substring %v", got, tt.wantSubstr)
			}
		})
	}
}

func TestOpenBrowserInvalidURL(t *testing.T) {
	err := OpenBrowser("ftp://invalid-scheme.com")
	if err == nil {
		t.Error("OpenBrowser with ftp scheme should return error")
	}
}

