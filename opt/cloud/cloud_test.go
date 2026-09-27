package cloud

import (
	"fmt"
	"strings"
	"testing"
)

func TestTemplatesRendering(t *testing.T) {
	tests := []struct {
		name       string
		template   string
		args       []any
		wantSubstr []string
	}{
		{
			name:     "vultr template generation",
			template: vultrTerraformTemplate,
			args:     []any{"node-1", "node-1", "vc2-1c-1gb", "mex", "node-1", "node-1"},
			wantSubstr: []string{
				"vultr_instance",
				"tarhiata-key-node-1",
				"tarhiata-swarm-node-1",
				"vc2-1c-1gb",
				"mex",
			},
		},
		{
			name:     "digitalocean template generation",
			template: doTerraformTemplate,
			args:     []any{"worker-do-1", "worker-do-1", "worker-do-1", "nyc1", "s-1vcpu-1gb"},
			wantSubstr: []string{
				"digitalocean_droplet",
				"tarhiata-key-worker-do-1",
				"tarhiata-swarm-worker-do-1",
				"s-1vcpu-1gb",
				"nyc1",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rendered := fmt.Sprintf(tc.template, tc.args...)
			for _, sub := range tc.wantSubstr {
				if !strings.Contains(rendered, sub) {
					t.Errorf("rendered template missing expected substr %q", sub)
				}
			}
		})
	}
}
