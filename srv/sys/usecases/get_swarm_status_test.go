package usecases

import (
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestParseSwarmServices(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCount int
		checkFirst    *domain.SwarmServiceInfo
	}{
		{
			name: "parses multiple swarm services",
			input: `40oqxskcccfl	tarhiata_obs_dozzle	replicated	1/1	amir20/dozzle:latest	
46o17m5madst	tarhiata_obs_portainer	replicated	1/1	portainer/portainer-ce:latest	
ujvr28huye2c	tarhiata_proxy_traefik	replicated	1/1	traefik:v3.1	*:80->80/tcp, *:443->443/tcp
`,
			expectedCount: 3,
			checkFirst: &domain.SwarmServiceInfo{
				ID:       "40oqxskcccfl",
				Name:     "tarhiata_obs_dozzle",
				Mode:     "replicated",
				Replicas: "1/1",
				Image:    "amir20/dozzle:latest",
				Ports:    "",
			},
		},
		{
			name:          "empty input returns empty slice",
			input:         "",
			expectedCount: 0,
			checkFirst:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSwarmServices(tt.input)
			if len(got) != tt.expectedCount {
				t.Fatalf("expected %d services, got %d", tt.expectedCount, len(got))
			}
			if tt.checkFirst != nil && len(got) > 0 {
				if got[0].Name != tt.checkFirst.Name {
					t.Errorf("expected %s, got %s", tt.checkFirst.Name, got[0].Name)
				}
				if got[0].Replicas != tt.checkFirst.Replicas {
					t.Errorf("expected %s, got %s", tt.checkFirst.Replicas, got[0].Replicas)
				}
			}
		})
	}
}

func TestParseSwarmNodes(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCount int
		checkFirst    *domain.SwarmNodeInfo
	}{
		{
			name: "parses single leader manager node",
			input: `n2ka3r33u9b243alwwsxrb353 *	master-test	Ready	Active	Leader	29.7.1
`,
			expectedCount: 1,
			checkFirst: &domain.SwarmNodeInfo{
				ID:            "n2ka3r33u9b243alwwsxrb353 *",
				Hostname:      "master-test",
				Status:        "Ready",
				Availability:  "Active",
				ManagerStatus: "Leader",
				EngineVersion: "29.7.1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSwarmNodes(tt.input)
			if len(got) != tt.expectedCount {
				t.Fatalf("expected %d nodes, got %d", tt.expectedCount, len(got))
			}
			if tt.checkFirst != nil && len(got) > 0 {
				if got[0].Hostname != tt.checkFirst.Hostname {
					t.Errorf("expected %s, got %s", tt.checkFirst.Hostname, got[0].Hostname)
				}
				if got[0].ManagerStatus != tt.checkFirst.ManagerStatus {
					t.Errorf("expected %s, got %s", tt.checkFirst.ManagerStatus, got[0].ManagerStatus)
				}
			}
		})
	}
}

func TestGetSwarmStatusUseCase_Execute(t *testing.T) {
	tests := []struct {
		name           string
		config         domain.ServerConfig
		mockResponses  map[string]*domain.CommandResult
		expectedActive bool
		expectedSvcs   int
	}{
		{
			name: "active swarm cluster",
			config: domain.ServerConfig{
				Host: "108.61.33.61",
			},
			mockResponses: map[string]*domain.CommandResult{
				"docker --version": {Output: "Docker version 29.7.1", ExitCode: 0},
				"LocalNodeState":   {Output: "active", ExitCode: 0},
				"service ls": {
					Output:   "1\ttarhiata_proxy_traefik\treplicated\t1/1\ttraefik:v3.1\t*:80->80/tcp\n",
					ExitCode: 0,
				},
				"node ls": {
					Output:   "node-1\tmaster-node\tReady\tActive\tLeader\t29.7.1\n",
					ExitCode: 0,
				},
			},
			expectedActive: true,
			expectedSvcs:   1,
		},
		{
			name: "inactive swarm cluster",
			config: domain.ServerConfig{
				Host: "10.0.0.1",
			},
			mockResponses: map[string]*domain.CommandResult{
				"docker --version": {Output: "Docker version 29.7.1", ExitCode: 0},
				"LocalNodeState":   {Output: "inactive", ExitCode: 0},
			},
			expectedActive: false,
			expectedSvcs:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := mocks.NewMockSSHExecutor()
			mockExec.MockResponses = tt.mockResponses

			uc := NewGetSwarmStatusUseCase(mockExec)
			res, err := uc.Execute(tt.config)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Active != tt.expectedActive {
				t.Errorf("expected active %v, got %v", tt.expectedActive, res.Active)
			}
			if len(res.Services) != tt.expectedSvcs {
				t.Errorf("expected %d services, got %d", tt.expectedSvcs, len(res.Services))
			}
		})
	}
}

func TestEnrichSwarmServicesWithInspect(t *testing.T) {
	tests := []struct {
		name           string
		initialSvcs    []domain.SwarmServiceInfo
		rawInspect     string
		expectedExpose bool
		expectedDomain string
	}{
		{
			name: "enriches service with host domain and expose true",
			initialSvcs: []domain.SwarmServiceInfo{
				{Name: "my-api"},
			},
			rawInspect:     "my-api\ttrue\tHost(`api.example.com`)\n",
			expectedExpose: true,
			expectedDomain: "api.example.com",
		},
		{
			name: "enriches service with path prefix and expose true",
			initialSvcs: []domain.SwarmServiceInfo{
				{Name: "web-app"},
			},
			rawInspect:     "web-app\ttrue\tPathPrefix(`/web-app`)\n",
			expectedExpose: true,
			expectedDomain: "/web-app",
		},
		{
			name: "service without traefik exposure",
			initialSvcs: []domain.SwarmServiceInfo{
				{Name: "worker-svc"},
			},
			rawInspect:     "worker-svc\tfalse\t\n",
			expectedExpose: false,
			expectedDomain: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcs := make([]domain.SwarmServiceInfo, len(tt.initialSvcs))
			copy(svcs, tt.initialSvcs)
			enrichSwarmServicesWithInspect(svcs, tt.rawInspect)

			if svcs[0].Expose != tt.expectedExpose {
				t.Errorf("expected Expose %v, got %v", tt.expectedExpose, svcs[0].Expose)
			}
			if svcs[0].Domain != tt.expectedDomain {
				t.Errorf("expected Domain '%s', got '%s'", tt.expectedDomain, svcs[0].Domain)
			}
		})
	}
}

func TestParseSwarmBundle(t *testing.T) {
	tests := []struct {
		name          string
		raw           string
		expectErr     bool
		expectedVer   string
		expectedState string
	}{
		{
			name: "parses complete valid bundle",
			raw: `===TARHIATA_DOCKER_VER===
Docker version 29.7.1
===TARHIATA_SWARM_STATE===
active
===TARHIATA_SERVICES===
svc-1	tarhiata-api	replicated	1/1	api:latest	*:80->80/tcp
===TARHIATA_INSPECT===
tarhiata-api	true	Host(` + "`" + `api.domain.com` + "`" + `)
===TARHIATA_NODES===
node-1	master	Ready	Active	Leader	29.7.1
===TARHIATA_END===`,
			expectErr:     false,
			expectedVer:   "Docker version 29.7.1",
			expectedState: "active",
		},
		{
			name: "parses bundle with inactive state",
			raw: `===TARHIATA_DOCKER_VER===
Docker version 28.0.0
===TARHIATA_SWARM_STATE===
inactive
===TARHIATA_SERVICES===
===TARHIATA_INSPECT===
===TARHIATA_NODES===
===TARHIATA_END===`,
			expectErr:     false,
			expectedVer:   "Docker version 28.0.0",
			expectedState: "inactive",
		},
		{
			name:          "returns error when delimiters are missing",
			raw:           "bash: docker: command not found",
			expectErr:     true,
			expectedVer:   "",
			expectedState: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := ParseSwarmBundle(tt.raw)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error parsing bundle, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error parsing bundle: %v", err)
			}
			if payload.DockerVersion != tt.expectedVer {
				t.Errorf("expected DockerVersion '%s', got '%s'", tt.expectedVer, payload.DockerVersion)
			}
			if payload.SwarmState != tt.expectedState {
				t.Errorf("expected SwarmState '%s', got '%s'", tt.expectedState, payload.SwarmState)
			}
		})
	}
}

func TestGetSwarmStatusUseCase_Execute_Bundled(t *testing.T) {
	tests := []struct {
		name           string
		config         domain.ServerConfig
		bundleOutput   string
		expectedActive bool
		expectedSvcs   int
		expectedNodes  int
	}{
		{
			name: "single trip active swarm bundled response",
			config: domain.ServerConfig{
				Host: "192.168.1.100",
			},
			bundleOutput: `===TARHIATA_DOCKER_VER===
Docker version 29.7.1
===TARHIATA_SWARM_STATE===
active
===TARHIATA_SERVICES===
svc-1	web-frontend	replicated	2/2	nginx:alpine	*:80->80/tcp
svc-2	tarhiata-db-postgres	replicated	1/1	postgres:16	
===TARHIATA_INSPECT===
web-frontend	true	Host(` + "`" + `web.mysite.com` + "`" + `)
===TARHIATA_NODES===
n1	node-manager	Ready	Active	Leader	29.7.1
n2	node-worker	Ready	Active		29.7.1
===TARHIATA_END===`,
			expectedActive: true,
			expectedSvcs:   2,
			expectedNodes:  2,
		},
		{
			name: "single trip inactive swarm bundled response",
			config: domain.ServerConfig{
				Host: "192.168.1.101",
			},
			bundleOutput: `===TARHIATA_DOCKER_VER===
Docker version 29.7.1
===TARHIATA_SWARM_STATE===
inactive
===TARHIATA_SERVICES===
===TARHIATA_INSPECT===
===TARHIATA_NODES===
===TARHIATA_END===`,
			expectedActive: false,
			expectedSvcs:   0,
			expectedNodes:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := mocks.NewMockSSHExecutor()
			mockExec.MockResponses = map[string]*domain.CommandResult{
				"===TARHIATA_DOCKER_VER===": {
					Output:   tt.bundleOutput,
					ExitCode: 0,
				},
			}

			uc := NewGetSwarmStatusUseCase(mockExec)
			status, err := uc.Execute(tt.config)
			if err != nil {
				t.Fatalf("unexpected error executing bundled swarm status: %v", err)
			}
			if status.Active != tt.expectedActive {
				t.Errorf("expected Active %v, got %v", tt.expectedActive, status.Active)
			}
			if len(status.Services) != tt.expectedSvcs {
				t.Errorf("expected %d services, got %d", tt.expectedSvcs, len(status.Services))
			}
			if len(status.Nodes) != tt.expectedNodes {
				t.Errorf("expected %d nodes, got %d", tt.expectedNodes, len(status.Nodes))
			}
			if tt.expectedActive && len(status.Services) > 0 {
				if !status.Services[0].Expose {
					t.Errorf("expected service 0 Expose to be true")
				}
				if status.Services[0].Domain != "web.mysite.com" {
					t.Errorf("expected Domain 'web.mysite.com', got '%s'", status.Services[0].Domain)
				}
			}
		})
	}
}


