package usecases

import (
	"errors"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
)

func TestDrainNodeUseCase_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		nodeID       string
		availability string
		setupExec    func(e *mockSecuritySSHExecutor)
		wantErr      bool
		wantAvail    string
		wantTasks    int
	}{
		{
			name:         "Drain node successfully with 0 running tasks",
			nodeID:       "node-worker-1",
			availability: "drain",
			setupExec: func(e *mockSecuritySSHExecutor) {
				e.responses = map[string]string{
					"docker node update": "node-worker-1",
					"docker node ps":     "0",
				}
			},
			wantErr:   false,
			wantAvail: "drain",
			wantTasks: 0,
		},
		{
			name:         "Drain node with 3 migrating tasks",
			nodeID:       "node-worker-2",
			availability: "drain",
			setupExec: func(e *mockSecuritySSHExecutor) {
				e.responses = map[string]string{
					"docker node update": "node-worker-2",
					"docker node ps":     "3",
				}
			},
			wantErr:   false,
			wantAvail: "drain",
			wantTasks: 3,
		},
		{
			name:         "Activate node successfully",
			nodeID:       "node-worker-1",
			availability: "active",
			setupExec: func(e *mockSecuritySSHExecutor) {
				e.responses = map[string]string{
					"docker node update": "node-worker-1",
					"docker node ps":     "0",
				}
			},
			wantErr:   false,
			wantAvail: "active",
			wantTasks: 0,
		},
		{
			name:         "Invalid availability string",
			nodeID:       "node-worker-1",
			availability: "destroy",
			setupExec:    func(e *mockSecuritySSHExecutor) {},
			wantErr:      true,
		},
		{
			name:         "Empty node ID",
			nodeID:       "",
			availability: "drain",
			setupExec:    func(e *mockSecuritySSHExecutor) {},
			wantErr:      true,
		},
		{
			name:         "SSH execution error",
			nodeID:       "node-worker-1",
			availability: "drain",
			setupExec: func(e *mockSecuritySSHExecutor) {
				e.err = errors.New("SSH network down")
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exec := &mockSecuritySSHExecutor{responses: make(map[string]string)}
			tc.setupExec(exec)

			uc := NewDrainNodeUseCase(exec)
			res, err := uc.Execute(tc.nodeID, tc.availability, domain.ServerConfig{Host: "192.168.1.10"})

			if (err != nil) != tc.wantErr {
				t.Fatalf("Execute() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if !tc.wantErr {
				if res == nil {
					t.Fatalf("expected non-nil DrainNodeResult on success")
				}
				if res.Availability != tc.wantAvail {
					t.Errorf("expected Availability %s, got %s", tc.wantAvail, res.Availability)
				}
				if res.RunningTasks != tc.wantTasks {
					t.Errorf("expected RunningTasks %d, got %d", tc.wantTasks, res.RunningTasks)
				}
			}
		})
	}
}

// TestDrainNodeUseCase_ClosesSSHConnection valida que Execute cierre la conexión SSH
// que abre, en vez de dejarla fugada tras cada operación de drenado/activación de nodo.
func TestDrainNodeUseCase_ClosesSSHConnection(t *testing.T) {
	exec := &mockSecuritySSHExecutor{responses: map[string]string{
		"docker node update": "node-worker-1",
		"docker node ps":     "0",
	}}
	uc := NewDrainNodeUseCase(exec)

	if _, err := uc.Execute("node-worker-1", "drain", domain.ServerConfig{Host: "192.168.1.10"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.closeCalls != 1 {
		t.Errorf("se esperaba 1 llamada a Close(), se registraron %d (fuga de conexión SSH)", exec.closeCalls)
	}
}
