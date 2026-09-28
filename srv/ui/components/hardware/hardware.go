package hardware

import (
	_ "embed"
	"strings"
)

//go:embed hardware.js
var JSContent []byte

// GetJSContent devuelve el contenido JS del componente de hardware.
func GetJSContent() []byte {
	return JSContent
}

// StorageDevice describe una unidad de disco o partición del host.
type StorageDevice struct {
	Name       string `json:"name"`
	Size       string `json:"size"`
	Type       string `json:"type"`
	Rotational bool   `json:"rotational"`
	FSType     string `json:"fsType"`
	MountPoint string `json:"mountPoint"`
	Model      string `json:"model"`
}

// GPUDevice describe una tarjeta gráfica o acelerador detectado.
type GPUDevice struct {
	Model       string `json:"model"`
	Vendor      string `json:"vendor"`
	MemoryTotal string `json:"memoryTotal"`
	PCIAddress  string `json:"pciAddress"`
	Driver      string `json:"driver"`
}

// USBDevice describe un periférico USB conectado.
type USBDevice struct {
	Bus         string `json:"bus"`
	Device      string `json:"device"`
	ID          string `json:"id"`
	Description string `json:"description"`
}

// DisplayDevice describe una pantalla o conector de video.
type DisplayDevice struct {
	Connector  string `json:"connector"`
	Status     string `json:"status"`
	Resolution string `json:"resolution"`
}

// PCIDevice describe un controlador o adaptador en el bus PCI.
type PCIDevice struct {
	Address string `json:"address"`
	Class   string `json:"class"`
	Vendor  string `json:"vendor"`
	Device  string `json:"device"`
}

// HostDevicesResult agrupa todos los componentes de hardware detectados.
type HostDevicesResult struct {
	Storage  []StorageDevice `json:"storage"`
	GPUs     []GPUDevice     `json:"gpus"`
	USB      []USBDevice     `json:"usb"`
	Displays []DisplayDevice `json:"displays"`
	PCI      []PCIDevice     `json:"pci"`
}

// CountTotal devuelve el total consolidado de dispositivos detectados.
func (h HostDevicesResult) CountTotal() int {
	return len(h.Storage) + len(h.GPUs) + len(h.USB) + len(h.Displays) + len(h.PCI)
}

// FormatStorageTech devuelve 'HDD' o 'SSD / NVMe' dependiendo de si es rotacional.
func FormatStorageTech(rotational bool) string {
	if rotational {
		return "HDD"
	}
	return "SSD / NVMe"
}

// NormalizeDeviceName limpia espacios y caracteres extraños de nombres de hardware.
func NormalizeDeviceName(raw string) string {
	cleaned := strings.TrimSpace(raw)
	if cleaned == "" {
		return "Desconocido"
	}
	return cleaned
}
