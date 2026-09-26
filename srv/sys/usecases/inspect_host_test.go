package usecases

import (
	"errors"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/tests/mocks"
)

func TestParseHostMetrics(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected domain.HostMetrics
	}{
		{
			name: "valid linux output",
			input: `OS:Linux x86_64
HOST:master-test
CORES:4
CPU_PCT:25.5
MEM_USED_MB:2048
MEM_TOTAL_MB:8192
DISK_USED_MB:10240
DISK_TOTAL_MB:40960
LOAD:0.52 0.48 0.35
UPTIME:up 3 days, 4 hours
`,
			expected: domain.HostMetrics{
				OS:            "Linux x86_64",
				Hostname:      "master-test",
				CPUCores:      4,
				CPUPercent:    25.5,
				MemoryUsedMB:  2048,
				MemoryTotalMB: 8192,
				MemoryPercent: 25.0,
				DiskUsedGB:    10.0,
				DiskTotalGB:   40.0,
				DiskPercent:   25.0,
				LoadAvg:       "0.52 0.48 0.35",
				Uptime:        "up 3 days, 4 hours",
			},
		},
		{
			name:     "empty output",
			input:    "",
			expected: domain.HostMetrics{},
		},
		{
			name: "partial and malformed lines",
			input: `MALFORMED_LINE
OS:Darwin arm64
CORES:not_a_number
`,
			expected: domain.HostMetrics{
				OS: "Darwin arm64",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseHostMetrics(tt.input)

			if got.OS != tt.expected.OS {
				t.Errorf("expected OS %s, got %s", tt.expected.OS, got.OS)
			}
			if got.Hostname != tt.expected.Hostname {
				t.Errorf("expected Hostname %s, got %s", tt.expected.Hostname, got.Hostname)
			}
			if got.CPUCores != tt.expected.CPUCores {
				t.Errorf("expected CPUCores %d, got %d", tt.expected.CPUCores, got.CPUCores)
			}
			if got.CPUPercent != tt.expected.CPUPercent {
				t.Errorf("expected CPUPercent %f, got %f", tt.expected.CPUPercent, got.CPUPercent)
			}
			if got.MemoryUsedMB != tt.expected.MemoryUsedMB {
				t.Errorf("expected MemoryUsedMB %d, got %d", tt.expected.MemoryUsedMB, got.MemoryUsedMB)
			}
			if got.MemoryPercent != tt.expected.MemoryPercent {
				t.Errorf("expected MemoryPercent %f, got %f", tt.expected.MemoryPercent, got.MemoryPercent)
			}
			if got.DiskUsedGB != tt.expected.DiskUsedGB {
				t.Errorf("expected DiskUsedGB %f, got %f", tt.expected.DiskUsedGB, got.DiskUsedGB)
			}
			if got.DiskPercent != tt.expected.DiskPercent {
				t.Errorf("expected DiskPercent %f, got %f", tt.expected.DiskPercent, got.DiskPercent)
			}
		})
	}
}

func TestParseHostServices(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCount int
		checkFirst    *domain.HostSystemService
	}{
		{
			name: "standard systemctl list output",
			input: `docker.service loaded active running Docker Application Container Engine
cron.service loaded active running Regular background program processing daemon
ssh.service loaded active running OpenBSD Secure Shell server
`,
			expectedCount: 3,
			checkFirst: &domain.HostSystemService{
				Name:        "docker.service",
				LoadState:   "loaded",
				ActiveState: "active",
				SubState:    "running",
				Description: "Docker Application Container Engine",
			},
		},
		{
			name:          "empty input",
			input:         "",
			expectedCount: 0,
			checkFirst:    nil,
		},
		{
			name: "line with less than four fields ignored",
			input: `short line only
valid.service loaded active running Description text
`,
			expectedCount: 1,
			checkFirst: &domain.HostSystemService{
				Name:        "valid.service",
				LoadState:   "loaded",
				ActiveState: "active",
				SubState:    "running",
				Description: "Description text",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseHostServices(tt.input)
			if len(got) != tt.expectedCount {
				t.Fatalf("expected %d services, got %d", tt.expectedCount, len(got))
			}
			if tt.checkFirst != nil && len(got) > 0 {
				if got[0].Name != tt.checkFirst.Name {
					t.Errorf("expected Name %s, got %s", tt.checkFirst.Name, got[0].Name)
				}
				if got[0].ActiveState != tt.checkFirst.ActiveState {
					t.Errorf("expected ActiveState %s, got %s", tt.checkFirst.ActiveState, got[0].ActiveState)
				}
				if got[0].SubState != tt.checkFirst.SubState {
					t.Errorf("expected SubState %s, got %s", tt.checkFirst.SubState, got[0].SubState)
				}
				if got[0].Description != tt.checkFirst.Description {
					t.Errorf("expected Description %s, got %s", tt.checkFirst.Description, got[0].Description)
				}
			}
		})
	}
}

func TestInspectHostUseCase_Execute(t *testing.T) {
	tests := []struct {
		name          string
		config        domain.ServerConfig
		connectErr    error
		mockResponses map[string]*domain.CommandResult
		expectErr     bool
		expectedName  string
		expectedSvc   int
	}{
		{
			name: "successful inspection",
			config: domain.ServerConfig{
				Name: "prod-vps",
				Host: "1.2.3.4",
			},
			connectErr: nil,
			mockResponses: map[string]*domain.CommandResult{
				"MEM_USED_MB": {
					Output:   "OS:Linux x86_64\nHOST:prod-node\nCORES:2\nCPU_PCT:15.0\nMEM_USED_MB:1000\nMEM_TOTAL_MB:2000\nDISK_USED_MB:5000\nDISK_TOTAL_MB:20000\nLOAD:0.10 0.20 0.15\nUPTIME:up 1 day\n",
					ExitCode: 0,
				},
				"systemctl": {
					Output:   "docker.service loaded active running Docker Engine\nssh.service loaded active running SSH Server\n",
					ExitCode: 0,
				},
			},
			expectErr:    false,
			expectedName: "prod-vps",
			expectedSvc:  2,
		},
		{
			name: "connection error",
			config: domain.ServerConfig{
				Name: "unreachable-vps",
				Host: "10.0.0.99",
			},
			connectErr: errors.New("timeout"),
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := mocks.NewMockSSHExecutor()
			mockExec.ConnectError = tt.connectErr
			mockExec.MockResponses = tt.mockResponses

			uc := NewInspectHostUseCase(mockExec)
			result, err := uc.Execute(tt.config)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ServerName != tt.expectedName {
				t.Errorf("expected ServerName %s, got %s", tt.expectedName, result.ServerName)
			}
			if len(result.Services) != tt.expectedSvc {
				t.Errorf("expected %d services, got %d", tt.expectedSvc, len(result.Services))
			}
			if result.Metrics.CPUCores != 2 {
				t.Errorf("expected 2 cores, got %d", result.Metrics.CPUCores)
			}
		})
	}
}

func TestInspectHostUseCase_ExecuteMetricsOnly(t *testing.T) {
	mockExec := mocks.NewMockSSHExecutor()
	mockExec.MockResponses = map[string]*domain.CommandResult{
		"sh -c": {
			Output:   "OS:Linux\nHOST:metrics-vps\nCORES:4\nCPU_PCT:12.5\nMEM_USED_MB:1024\nMEM_TOTAL_MB:4096\nDISK_USED_MB:5000\nDISK_TOTAL_MB:20000\nLOAD:0.1 0.2 0.3\nUPTIME:up 1 day\n",
			ExitCode: 0,
		},
	}

	uc := NewInspectHostUseCase(mockExec)
	metrics, err := uc.ExecuteMetricsOnly(domain.ServerConfig{Host: "metrics-vps"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metrics.Hostname != "metrics-vps" {
		t.Errorf("expected Hostname 'metrics-vps', got '%s'", metrics.Hostname)
	}
	if metrics.CPUCores != 4 {
		t.Errorf("expected 4 cores, got %d", metrics.CPUCores)
	}
}

func TestInspectHostUseCase_ExecuteServicesOnly(t *testing.T) {
	mockExec := mocks.NewMockSSHExecutor()
	mockExec.MockResponses = map[string]*domain.CommandResult{
		"sh -c": {
			Output:   "docker.service loaded active running Docker Application Container Engine\n",
			ExitCode: 0,
		},
	}

	uc := NewInspectHostUseCase(mockExec)
	services, err := uc.ExecuteServicesOnly(domain.ServerConfig{Host: "services-vps"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}
	if services[0].Name != "docker.service" {
		t.Errorf("expected 'docker.service', got '%s'", services[0].Name)
	}
}

