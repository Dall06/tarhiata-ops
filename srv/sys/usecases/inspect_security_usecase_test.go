package usecases

import (
	"errors"
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

type mockSecuritySSHExecutor struct {
	responses  map[string]string
	err        error
	closeCalls int
}

func (m *mockSecuritySSHExecutor) Connect(cfg domain.ServerConfig) error { return m.err }
func (m *mockSecuritySSHExecutor) Close() error                          { m.closeCalls++; return nil }
func (m *mockSecuritySSHExecutor) RunCommand(cmd string) (*domain.CommandResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	for k, v := range m.responses {
		if strings.Contains(cmd, k) {
			return &domain.CommandResult{Output: v, ExitCode: 0}, nil
		}
	}
	return &domain.CommandResult{Output: "", ExitCode: 0}, nil
}
func (m *mockSecuritySSHExecutor) InteractiveShell() error             { return nil }
func (m *mockSecuritySSHExecutor) InteractiveCommand(cmd string) error { return nil }
func (m *mockSecuritySSHExecutor) WriteRemoteFile(r, c string) error   { return nil }
func (m *mockSecuritySSHExecutor) CheckConnection() bool               { return true }

func TestInspectSecurityUseCase_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		ufwOutput     string
		f2bOutput     string
		execErr       error
		wantUFWActive bool
		wantRulesLen  int
		wantF2BActive bool
		wantJailsLen  int
		wantBannedLen int
	}{
		{
			name: "Active UFW and Active Fail2Ban with rules and banned IPs",
			ufwOutput: `Status: active

     To                         Action      From
     --                         ------      ----
[ 1] 22/tcp                     ALLOW IN    Anywhere
[ 2] 80/tcp                     ALLOW IN    Anywhere
[ 3] 443/tcp                    ALLOW IN    Anywhere
[ 4] 8080/tcp                   DENY IN     Anywhere`,
			f2bOutput: "Status\n" +
				"|- Number of jail:      1\n" +
				"`- Jail list:   sshd\n" +
				"Status for the jail: sshd\n" +
				"|- Filter\n" +
				"|  |- Currently failed: 2\n" +
				"|  |- Total failed:     45\n" +
				"`- Actions\n" +
				"   |- Currently banned: 2\n" +
				"   |- Total banned:     10\n" +
				"   `- Banned IP list:   198.51.100.1 203.0.113.5",
			wantUFWActive: true,
			wantRulesLen:  4,
			wantF2BActive: true,
			wantJailsLen:  1,
			wantBannedLen: 2,
		},
		{
			name:          "Inactive UFW and Not Installed Fail2Ban",
			ufwOutput:     "Status: inactive\n",
			f2bOutput:     "NOT_INSTALLED",
			wantUFWActive: false,
			wantRulesLen:  0,
			wantF2BActive: false,
			wantJailsLen:  0,
			wantBannedLen: 0,
		},
		{
			name:          "SSH Command Error fallback",
			execErr:       errors.New("connection failed"),
			wantUFWActive: false,
			wantRulesLen:  0,
			wantF2BActive: false,
			wantJailsLen:  0,
			wantBannedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &mockSecuritySSHExecutor{
				responses: map[string]string{
					"ufw":      tt.ufwOutput,
					"fail2ban": tt.f2bOutput,
				},
				err: tt.execErr,
			}
			uc := NewInspectSecurityUseCase(exec)
			res, err := uc.Execute()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.UFWActive != tt.wantUFWActive {
				t.Errorf("UFWActive = %v, want %v", res.UFWActive, tt.wantUFWActive)
			}
			if len(res.UFWRules) != tt.wantRulesLen {
				t.Errorf("UFWRules len = %d, want %d", len(res.UFWRules), tt.wantRulesLen)
			}
			if res.Fail2Ban.Active != tt.wantF2BActive {
				t.Errorf("Fail2Ban Active = %v, want %v", res.Fail2Ban.Active, tt.wantF2BActive)
			}
			if len(res.Fail2Ban.Jails) != tt.wantJailsLen {
				t.Errorf("Jails len = %d, want %d", len(res.Fail2Ban.Jails), tt.wantJailsLen)
			}
			if len(res.Fail2Ban.BannedIPs) != tt.wantBannedLen {
				t.Errorf("BannedIPs len = %d, want %d", len(res.Fail2Ban.BannedIPs), tt.wantBannedLen)
			}
		})
	}
}
