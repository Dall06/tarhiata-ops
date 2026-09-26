package domain

import "time"

// StorageDevice representa una unidad o partición de almacenamiento conectada al host.
type StorageDevice struct {
	Name       string `json:"name"`       // Ej: "sda", "nvme0n1", "vda"
	Size       string `json:"size"`       // Ej: "50G", "1TB"
	Type       string `json:"type"`       // Ej: "disk", "part", "rom"
	MountPoint string `json:"mountPoint"` // Ej: "/", "/data", "[SWAP]"
	Model      string `json:"model"`      // Ej: "Samsung SSD 980" o "QEMU HARDDISK"
	Rotational bool   `json:"rotational"` // false = SSD / NVMe, true = HDD
	FSType     string `json:"fsType"`     // Ej: "ext4", "xfs", "btrfs"
}

// GPUDevice representa una tarjeta o acelerador gráfico conectado al bus del host.
type GPUDevice struct {
	Model       string `json:"model"`                 // Ej: "NVIDIA RTX 4090", "Virtio GPU", "UHD Graphics 630"
	Vendor      string `json:"vendor"`                // Ej: "NVIDIA Corporation", "Intel Corporation", "Advanced Micro Devices"
	MemoryTotal string `json:"memoryTotal,omitempty"` // Ej: "24576 MiB"
	Driver      string `json:"driver,omitempty"`      // Ej: "nvidia", "i915", "virtio-pci"
	PCIAddress  string `json:"pciAddress,omitempty"`  // Ej: "00:02.0"
}

// USBDevice representa un periférico o controlador conectado a los puertos USB del host.
type USBDevice struct {
	Bus          string `json:"bus"`                    // Ej: "001"
	Device       string `json:"device"`                 // Ej: "002"
	ID           string `json:"id"`                     // Ej: "1d6b:0002" (VendorID:ProductID)
	Description  string `json:"description"`            // Ej: "Linux Foundation 2.0 root hub"
	Manufacturer string `json:"manufacturer,omitempty"` // Ej: "Kingston", "Logitech"
}

// DisplayDevice representa una salida o pantalla física/virtual conectada al host (HDMI, DP, etc.).
type DisplayDevice struct {
	Connector  string `json:"connector"`            // Ej: "HDMI-1", "DP-1", "Virtual-1"
	Status     string `json:"status"`               // "connected" | "disconnected"
	Resolution string `json:"resolution,omitempty"` // Ej: "1920x1080", "2560x1440"
}

// PCIDevice representa un bus, puente o tarjeta en la topología PCI del host.
type PCIDevice struct {
	Address string `json:"address"` // Ej: "00:00.0"
	Class   string `json:"class"`   // Ej: "Host bridge", "Ethernet controller"
	Vendor  string `json:"vendor"`  // Ej: "Intel Corporation"
	Device  string `json:"device"`  // Ej: "82540EM Gigabit Ethernet Controller"
}

// HostDevices agrupa todos los componentes y periféricos de hardware detectados en el servidor.
type HostDevices struct {
	ServerName string          `json:"serverName"`
	Host       string          `json:"host"`
	IsLocal    bool            `json:"isLocal"`
	Timestamp  time.Time       `json:"timestamp"`
	Storage    []StorageDevice `json:"storage"`
	GPUs       []GPUDevice     `json:"gpus"`
	USB        []USBDevice     `json:"usb"`
	Displays   []DisplayDevice `json:"displays"`
	PCI        []PCIDevice     `json:"pci"`
}
