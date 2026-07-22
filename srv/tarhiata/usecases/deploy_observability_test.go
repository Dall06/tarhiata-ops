package usecases

import (
	"strings"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/tarhiata/tests/mocks"
)

func TestDeployObservability_ExecutePersistent(t *testing.T) {
	tests := []struct {
		name              string
		exposePublic      bool
		deployType        string
		expectCmdContains []string
	}{
		{
			name:         "Single-node public access",
			exposePublic: true,
			deployType:   "single-node",
			expectCmdContains: []string{
				"ufw allow 9000/tcp",
				"ufw allow 3001/tcp",
				`--constraint "node.role == manager"`, // from init-perms-obs
				"mkdir -p /host/opt/tarhiata/obs/config",
				"docker stack deploy -c /tmp/obs-persist-stack.yml tarhiata_obs",
				`constraints: ["node.role == manager"]`, // from stack deploy
			},
		},
		{
			name:         "Multi-node private access (blocked mesh)",
			exposePublic: false,
			deployType:   "multi-node",
			expectCmdContains: []string{
				"iptables -I DOCKER-USER -i $EXT_IF -p tcp -m multiport --dports 9000,3001 -j DROP",
				`--constraint "node.labels.type == obs"`,   // from init-perms-obs
				`constraints: ["node.labels.type == obs"]`, // from stack deploy
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSSH := mocks.NewMockSSHExecutor()
			uc := NewDeployObservabilityUseCase(mockSSH)

			err := uc.ExecutePersistent(tt.exposePublic, tt.deployType)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			allCmds := strings.Join(mockSSH.CommandsExecuted, " ||| ")
			for _, expected := range tt.expectCmdContains {
				if !strings.Contains(allCmds, expected) {
					t.Errorf("expected command to contain '%s', but it wasn't found in: %s", expected, allCmds)
				}
			}
		})
	}
}
