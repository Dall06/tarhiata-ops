package hardware

import (
	"testing"
)

func TestHostDevicesResult_CountTotal(t *testing.T) {
	tests := []struct {
		name     string
		result   HostDevicesResult
		expected int
	}{
		{
			name:     "empty result",
			result:   HostDevicesResult{},
			expected: 0,
		},
		{
			name: "populated devices",
			result: HostDevicesResult{
				Storage:  []StorageDevice{{Name: "sda", Size: "500GB"}},
				GPUs:     []GPUDevice{{Model: "RTX 3080"}},
				USB:      []USBDevice{{ID: "1234:5678"}},
				Displays: []DisplayDevice{{Connector: "HDMI-1"}},
				PCI:      []PCIDevice{{Address: "00:01.0"}},
			},
			expected: 5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.result.CountTotal()
			if got != tc.expected {
				t.Errorf("CountTotal() = %d; want %d", got, tc.expected)
			}
		})
	}
}

func TestFormatStorageTech(t *testing.T) {
	tests := []struct {
		name       string
		rotational bool
		expected   string
	}{
		{name: "HDD rotational", rotational: true, expected: "HDD"},
		{name: "SSD non-rotational", rotational: false, expected: "SSD / NVMe"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatStorageTech(tc.rotational)
			if got != tc.expected {
				t.Errorf("FormatStorageTech(%v) = %s; want %s", tc.rotational, got, tc.expected)
			}
		})
	}
}

func TestNormalizeDeviceName(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected string
	}{
		{name: "empty raw", raw: "   ", expected: "Desconocido"},
		{name: "trimmed raw", raw: "  NVIDIA RTX 4090  ", expected: "NVIDIA RTX 4090"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeDeviceName(tc.raw)
			if got != tc.expected {
				t.Errorf("NormalizeDeviceName(%q) = %q; want %q", tc.raw, got, tc.expected)
			}
		})
	}
}
