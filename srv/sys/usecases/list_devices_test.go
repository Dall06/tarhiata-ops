package usecases_test

import (
	"errors"
	"testing"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/usecases"
)

type mockDeviceSSHExecutor struct {
	connectErr error
	cmdResult  *domain.CommandResult
	cmdErr     error
	closeCalls int
}

func (m *mockDeviceSSHExecutor) Connect(config domain.ServerConfig) error {
	return m.connectErr
}

func (m *mockDeviceSSHExecutor) RunCommand(cmd string) (*domain.CommandResult, error) {
	if m.cmdErr != nil {
		return nil, m.cmdErr
	}
	return m.cmdResult, nil
}

func (m *mockDeviceSSHExecutor) RunCommandStreaming(cmd string, onLine func(string)) (*domain.CommandResult, error) {
	return m.RunCommand(cmd)
}

func (m *mockDeviceSSHExecutor) InteractiveShell() error {
	return nil
}

func (m *mockDeviceSSHExecutor) InteractiveCommand(cmd string) error {
	return nil
}

func (m *mockDeviceSSHExecutor) WriteRemoteFile(remotePath, content string) error {
	return nil
}

func (m *mockDeviceSSHExecutor) CheckConnection() bool {
	return true
}

func (m *mockDeviceSSHExecutor) Close() error {
	m.closeCalls++
	return nil
}

func TestListDevicesUseCase_Execute(t *testing.T) {
	sampleLinuxOutput := `===STORAGE===
NAME="sda" SIZE="50G" TYPE="disk" MOUNTPOINT="" MODEL="QEMU HARDDISK" ROTA="0" FSTYPE=""
NAME="sda1" SIZE="49G" TYPE="part" MOUNTPOINT="/" MODEL="" ROTA="0" FSTYPE="ext4"
NAME="sdb" SIZE="1000G" TYPE="disk" MOUNTPOINT="/mnt/data" MODEL="WDC WD10EZEX" ROTA="1" FSTYPE="xfs"

===GPU===
NVIDIA|NVIDIA GeForce RTX 4090|24576 MiB|535.129.03|00000000:01:00.0
00:02.0 VGA compatible controller: Red Hat, Inc. Virtio GPU (rev 01)

===USB===
Bus 001 Device 001: ID 1d6b:0002 Linux Foundation 2.0 root hub
Bus 001 Device 002: ID 80ee:0021 VirtualBox USB Tablet

===DISPLAYS===
HDMI-A-1|connected|1920x1080
DP-1|disconnected|

===PCI===
00:00.0 Host bridge: Intel Corporation 440FX - 82441FX PMC [Natoma] (rev 02)
00:01.0 ISA bridge: Intel Corporation 82371SB PIIX3 ISA [Natoma/Triton II]
`

	tests := []struct {
		name          string
		cfg           domain.ServerConfig
		mockExec      *mockDeviceSSHExecutor
		wantErr       bool
		expectedStore int
		expectedGPU   int
		expectedUSB   int
		expectedDisp  int
		expectedPCI   int
	}{
		{
			name: "inspección exitosa en linux con periféricos completos",
			cfg: domain.ServerConfig{
				Name: "vps-test",
				Host: "192.168.1.50",
			},
			mockExec: &mockDeviceSSHExecutor{
				cmdResult: &domain.CommandResult{
					Output: sampleLinuxOutput,
				},
			},
			wantErr:       false,
			expectedStore: 3,
			expectedGPU:   2,
			expectedUSB:   2,
			expectedDisp:  2,
			expectedPCI:   2,
		},
		{
			name: "error al conectar SSH",
			cfg: domain.ServerConfig{
				Name: "vps-offline",
				Host: "10.0.0.99",
			},
			mockExec: &mockDeviceSSHExecutor{
				connectErr: errors.New("connection timeout"),
			},
			wantErr: true,
		},
		{
			name: "error al ejecutar comando remoto",
			cfg: domain.ServerConfig{
				Name: "vps-bad-cmd",
				Host: "10.0.0.100",
			},
			mockExec: &mockDeviceSSHExecutor{
				cmdErr: errors.New("remote execution failed"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := usecases.NewListDevicesUseCase(tt.mockExec)
			res, err := uc.Execute(tt.cfg)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if res == nil {
				t.Fatal("Execute() devolvió nil res sin error")
			}

			if len(res.Storage) != tt.expectedStore {
				t.Errorf("len(Storage) = %d, want %d", len(res.Storage), tt.expectedStore)
			}

			if len(res.GPUs) != tt.expectedGPU {
				t.Errorf("len(GPUs) = %d, want %d", len(res.GPUs), tt.expectedGPU)
			}

			if len(res.USB) != tt.expectedUSB {
				t.Errorf("len(USB) = %d, want %d", len(res.USB), tt.expectedUSB)
			}

			if len(res.Displays) != tt.expectedDisp {
				t.Errorf("len(Displays) = %d, want %d", len(res.Displays), tt.expectedDisp)
			}

			if len(res.PCI) != tt.expectedPCI {
				t.Errorf("len(PCI) = %d, want %d", len(res.PCI), tt.expectedPCI)
			}

			// Validaciones detalladas de propiedades
			if res.Storage[0].Name != "sda" || res.Storage[0].Rotational != false {
				t.Errorf("Storage[0] inesperado: %+v", res.Storage[0])
			}
			if res.Storage[2].Rotational != true {
				t.Errorf("Storage[2] Rotational esperado true, obtenido false")
			}
			if res.GPUs[0].Model != "NVIDIA GeForce RTX 4090" || res.GPUs[0].MemoryTotal != "24576 MiB" {
				t.Errorf("GPU[0] inesperado: %+v", res.GPUs[0])
			}
			if res.Displays[0].Connector != "HDMI-A-1" || res.Displays[0].Status != "connected" || res.Displays[0].Resolution != "1920x1080" {
				t.Errorf("Display[0] inesperado: %+v", res.Displays[0])
			}
		})
	}
}

func TestParseHostDevicesOutput_DarwinAndEmpty(t *testing.T) {
	tests := []struct {
		name          string
		raw           string
		expectedStore int
		expectedGPU   int
		expectedDisp  int
	}{
		{
			name: "salida darwin macos",
			raw: `===STORAGE===
NAME="/dev/disk3s1" SIZE="500000K" TYPE="disk" MOUNTPOINT="/" MODEL="Apple-Disk" ROTA="0" FSTYPE="apfs"
===GPU===
GPU: Apple M1 Pro
Vendor: Apple (0x106b)
===DISPLAYS===
Built-in|connected|3024 x 1964 Retina
`,
			expectedStore: 1,
			expectedGPU:   1,
			expectedDisp:  1,
		},
		{
			name:          "salida vacia",
			raw:           "",
			expectedStore: 0,
			expectedGPU:   0,
			expectedDisp:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := usecases.ParseHostDevicesOutput(tt.raw)
			if parsed == nil {
				t.Fatal("ParseHostDevicesOutput retornó nil")
			}
			if len(parsed.Storage) != tt.expectedStore {
				t.Errorf("Storage count = %d, want %d", len(parsed.Storage), tt.expectedStore)
			}
			if len(parsed.GPUs) != tt.expectedGPU {
				t.Errorf("GPU count = %d, want %d", len(parsed.GPUs), tt.expectedGPU)
			}
			if len(parsed.Displays) != tt.expectedDisp {
				t.Errorf("Displays count = %d, want %d", len(parsed.Displays), tt.expectedDisp)
			}
		})
	}
}

// TestListDevicesUseCase_ClosesSSHConnection valida que Execute cierre la conexión SSH
// que abre, en vez de dejarla fugada en cada consulta de dispositivos del host.
func TestListDevicesUseCase_ClosesSSHConnection(t *testing.T) {
	exec := &mockDeviceSSHExecutor{cmdResult: &domain.CommandResult{Output: "", ExitCode: 0}}
	uc := usecases.NewListDevicesUseCase(exec)

	if _, err := uc.Execute(domain.ServerConfig{Name: "vps-test", Host: "192.168.1.50"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.closeCalls != 1 {
		t.Errorf("se esperaba 1 llamada a Close(), se registraron %d (fuga de conexión SSH)", exec.closeCalls)
	}
}
