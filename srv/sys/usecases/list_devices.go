package usecases

import (
	"bufio"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Dall06/tarhiata-ops/srv/sys/domain"
	"github.com/Dall06/tarhiata-ops/srv/sys/ports"
)

// ListDevicesUseCase implementa ports.ListDevicesUseCase.
type ListDevicesUseCase struct {
	executor ports.SSHExecutor
}

// NewListDevicesUseCase crea una nueva instancia del caso de uso para listar hardware y dispositivos.
func NewListDevicesUseCase(executor ports.SSHExecutor) ports.ListDevicesUseCase {
	return &ListDevicesUseCase{
		executor: executor,
	}
}

const hostDevicesCommand = `sh -c '
echo "===STORAGE==="
if command -v lsblk >/dev/null 2>&1; then
    lsblk -P -o NAME,SIZE,TYPE,MOUNTPOINT,MODEL,ROTA,FSTYPE 2>/dev/null
elif [ "$(uname -s)" = "Darwin" ]; then
    df -k 2>/dev/null | awk "NR>1 && \$1 ~ /^\/dev/ {print \"NAME=\\\"\" \$1 \"\\\" SIZE=\\\"\" \$2 \"K\\\" TYPE=\\\"disk\\\" MOUNTPOINT=\\\"\" \$9 \"\\\" MODEL=\\\"Darwin-Disk\\\" ROTA=\\\"0\\\" FSTYPE=\\\"apfs\\\"\"}"
else
    df -k 2>/dev/null | awk "NR>1 && \$1 ~ /^\/dev/ {print \"NAME=\\\"\" \$1 \"\\\" SIZE=\\\"\" \$2 \"K\\\" TYPE=\\\"disk\\\" MOUNTPOINT=\\\"\" \$6 \"\\\" MODEL=\\\"Linux-Disk\\\" ROTA=\\\"0\\\" FSTYPE=\\\"ext4\\\"\"}"
fi

echo "===GPU==="
if command -v nvidia-smi >/dev/null 2>&1; then
    nvidia-smi --query-gpu=name,memory.total,driver_version,pci.bus_id --format=csv,noheader 2>/dev/null | while IFS=, read -r gname gmem gdrv gpci; do
        echo "NVIDIA|$gname|$gmem|$gdrv|$gpci"
    done
fi
if command -v lspci >/dev/null 2>&1; then
    lspci 2>/dev/null | grep -iE "vga|3d|display|nvidia|amd|graphics"
elif [ "$(uname -s)" = "Darwin" ]; then
    system_profiler SPDisplaysDataType 2>/dev/null | awk -F": " "/Chipset Model:/ {print \"GPU: \" \$2} /Vendor:/ {print \"Vendor: \" \$2}"
fi

echo "===USB==="
if command -v lsusb >/dev/null 2>&1; then
    lsusb 2>/dev/null
elif [ -d /sys/bus/usb/devices ]; then
    for d in /sys/bus/usb/devices/*; do
        if [ -f "$d/idVendor" ] && [ -f "$d/idProduct" ]; then
            v=$(cat "$d/idVendor" 2>/dev/null)
            p=$(cat "$d/idProduct" 2>/dev/null)
            prod=$(cat "$d/product" 2>/dev/null)
            mfg=$(cat "$d/manufacturer" 2>/dev/null)
            b=$(basename "$d")
            echo "Bus $b Device 000: ID $v:$p $mfg $prod"
        fi
    done
fi

echo "===DISPLAYS==="
for s in /sys/class/drm/*/status; do
    if [ -f "$s" ]; then
        conn=$(basename $(dirname "$s") | sed "s/^card[0-9]*-//")
        stat=$(cat "$s" 2>/dev/null)
        mode_file="$(dirname "$s")/modes"
        res=""
        if [ -f "$mode_file" ]; then
            res=$(head -n 1 "$mode_file" 2>/dev/null)
        fi
        echo "$conn|$stat|$res"
    fi
done
if [ "$(uname -s)" = "Darwin" ]; then
    system_profiler SPDisplaysDataType 2>/dev/null | awk -F": " "/Resolution:/ {print \"Built-in|connected|\" \$2}"
fi

echo "===PCI==="
if command -v lspci >/dev/null 2>&1; then
    lspci 2>/dev/null | head -n 40
fi
'
`

// Execute ejecuta la inspección remota de hardware y parsea los dispositivos conectados.
func (uc *ListDevicesUseCase) Execute(config domain.ServerConfig) (*domain.HostDevices, error) {
	targetHost := strings.TrimSpace(config.Host)
	if config.IsLocal() && targetHost == "" {
		targetHost = "localhost"
	}
	serverName := strings.TrimSpace(config.Name)
	if serverName == "" {
		serverName = targetHost
	}

	if err := uc.executor.Connect(config); err != nil {
		return nil, fmt.Errorf("falló conexión SSH con servidor '%s': %w", serverName, err)
	}

	cmdRes, err := uc.executor.RunCommand(hostDevicesCommand)
	if err != nil {
		return nil, fmt.Errorf("falló al inspeccionar dispositivos del host: %w", err)
	}

	rawOutput := ""
	if cmdRes != nil {
		rawOutput = cmdRes.Output
	}

	devices := ParseHostDevicesOutput(rawOutput)
	devices.ServerName = serverName
	devices.Host = targetHost
	devices.IsLocal = config.IsLocal()
	devices.Timestamp = time.Now()

	slog.Info("dispositivos del host detectados exitosamente",
		"server", serverName,
		"storage_count", len(devices.Storage),
		"gpu_count", len(devices.GPUs),
		"usb_count", len(devices.USB),
		"displays_count", len(devices.Displays),
		"pci_count", len(devices.PCI),
	)

	return devices, nil
}

// ParseHostDevicesOutput parsea el texto con secciones emitido por el script de detección.
func ParseHostDevicesOutput(output string) *domain.HostDevices {
	devices := &domain.HostDevices{
		Storage:  make([]domain.StorageDevice, 0),
		GPUs:     make([]domain.GPUDevice, 0),
		USB:      make([]domain.USBDevice, 0),
		Displays: make([]domain.DisplayDevice, 0),
		PCI:      make([]domain.PCIDevice, 0),
	}

	currentSection := ""
	scanner := bufio.NewScanner(strings.NewReader(output))

	var currentDarwinGPU *domain.GPUDevice

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "===") && strings.HasSuffix(line, "===") {
			currentSection = strings.Trim(line, "=")
			continue
		}

		switch currentSection {
		case "STORAGE":
			dev := parseStorageLine(line)
			if dev != nil {
				devices.Storage = append(devices.Storage, *dev)
			}

		case "GPU":
			if strings.HasPrefix(line, "GPU: ") {
				if currentDarwinGPU != nil {
					devices.GPUs = append(devices.GPUs, *currentDarwinGPU)
				}
				currentDarwinGPU = &domain.GPUDevice{
					Model:  strings.TrimPrefix(line, "GPU: "),
					Vendor: "Apple",
				}
				continue
			}
			if strings.HasPrefix(line, "Vendor: ") && currentDarwinGPU != nil {
				currentDarwinGPU.Vendor = strings.TrimPrefix(line, "Vendor: ")
				continue
			}

			gpu := parseGPULine(line)
			if gpu != nil {
				devices.GPUs = append(devices.GPUs, *gpu)
			}

		case "USB":
			usb := parseUSBLine(line)
			if usb != nil {
				devices.USB = append(devices.USB, *usb)
			}

		case "DISPLAYS":
			disp := parseDisplayLine(line)
			if disp != nil {
				devices.Displays = append(devices.Displays, *disp)
			}

		case "PCI":
			pci := parsePCILine(line)
			if pci != nil {
				devices.PCI = append(devices.PCI, *pci)
			}
		}
	}

	if currentDarwinGPU != nil {
		devices.GPUs = append(devices.GPUs, *currentDarwinGPU)
	}

	if scanErr := scanner.Err(); scanErr != nil {
		slog.Warn("list_devices: advertencia durante escaneo de salida", "error", scanErr)
	}

	return devices
}

func parseStorageLine(line string) *domain.StorageDevice {
	if !strings.Contains(line, "NAME=") {
		return nil
	}

	pairs := parseKeyValuePairs(line)
	name := pairs["NAME"]
	if name == "" {
		return nil
	}

	size := pairs["SIZE"]
	devType := pairs["TYPE"]
	if devType == "loop" || strings.HasPrefix(name, "loop") {
		return nil
	}
	mountPoint := pairs["MOUNTPOINT"]
	model := pairs["MODEL"]
	rotaStr := pairs["ROTA"]
	fsType := pairs["FSTYPE"]

	rotational := rotaStr == "1"

	return &domain.StorageDevice{
		Name:       name,
		Size:       size,
		Type:       devType,
		MountPoint: mountPoint,
		Model:      model,
		Rotational: rotational,
		FSType:     fsType,
	}
}

func parseGPULine(line string) *domain.GPUDevice {
	// 1. Caso NVIDIA format: NVIDIA|name|memory|driver|pci
	if strings.HasPrefix(line, "NVIDIA|") {
		parts := strings.Split(line, "|")
		if len(parts) >= 5 {
			return &domain.GPUDevice{
				Vendor:      "NVIDIA Corporation",
				Model:       strings.TrimSpace(parts[1]),
				MemoryTotal: strings.TrimSpace(parts[2]),
				Driver:      strings.TrimSpace(parts[3]),
				PCIAddress:  strings.TrimSpace(parts[4]),
			}
		}
	}

	// 2. Caso lspci: 00:02.0 VGA compatible controller: Red Hat, Inc. Virtio GPU (rev 01)
	colonIdx := strings.Index(line, ":")
	if colonIdx == -1 {
		return nil
	}

	// Parte antes de ':' puede tener la dirección PCI y el tipo de controlador
	leftPart := strings.TrimSpace(line[:colonIdx])
	descPart := strings.TrimSpace(line[colonIdx+1:])

	// Si hay un segundo ':' (como en "00:02.0 VGA compatible controller: ...")
	if secondColon := strings.Index(descPart, ":"); secondColon != -1 && strings.Contains(leftPart, ".") {
		leftPart = leftPart + ":" + strings.TrimSpace(descPart[:secondColon])
		descPart = strings.TrimSpace(descPart[secondColon+1:])
	}

	firstSpace := strings.Index(leftPart, " ")
	pciAddr := ""
	if firstSpace != -1 {
		pciAddr = leftPart[:firstSpace]
	}

	vendor := "Unknown"
	model := descPart

	if strings.Contains(descPart, "NVIDIA") {
		vendor = "NVIDIA Corporation"
	}
	if strings.Contains(descPart, "Intel") {
		vendor = "Intel Corporation"
	}
	if strings.Contains(descPart, "Advanced Micro Devices") || strings.Contains(descPart, "AMD") {
		vendor = "AMD / ATI"
	}
	if strings.Contains(descPart, "Red Hat") || strings.Contains(descPart, "Virtio") {
		vendor = "Red Hat (Virtio)"
	}
	if strings.Contains(descPart, "VirtualBox") || strings.Contains(descPart, "InnoTek") {
		vendor = "InnoTek / VirtualBox"
	}
	if strings.Contains(descPart, "VMware") {
		vendor = "VMware"
	}

	return &domain.GPUDevice{
		Vendor:     vendor,
		Model:      model,
		PCIAddress: pciAddr,
	}
}

func parseUSBLine(line string) *domain.USBDevice {
	// Formato estándar lsusb: "Bus 001 Device 002: ID 1d6b:0002 Linux Foundation 2.0 root hub"
	if !strings.HasPrefix(line, "Bus ") {
		return nil
	}

	parts := strings.Fields(line)
	if len(parts) < 6 {
		return nil
	}

	bus := parts[1]
	device := strings.TrimSuffix(parts[3], ":")

	id := ""
	descStart := 4
	for i, part := range parts {
		if part == "ID" && i+1 < len(parts) {
			id = parts[i+1]
			descStart = i + 2
			break
		}
	}

	description := ""
	if descStart < len(parts) {
		description = strings.Join(parts[descStart:], " ")
	}

	return &domain.USBDevice{
		Bus:         bus,
		Device:      device,
		ID:          id,
		Description: description,
	}
}

func parseDisplayLine(line string) *domain.DisplayDevice {
	// Formato: Connector|Status|Resolution
	parts := strings.Split(line, "|")
	if len(parts) < 2 {
		return nil
	}

	connector := strings.TrimSpace(parts[0])
	status := strings.TrimSpace(parts[1])
	resolution := ""
	if len(parts) >= 3 {
		resolution = strings.TrimSpace(parts[2])
	}

	if connector == "" {
		return nil
	}

	return &domain.DisplayDevice{
		Connector:  connector,
		Status:     status,
		Resolution: resolution,
	}
}

func parsePCILine(line string) *domain.PCIDevice {
	// Formato lspci: "00:00.0 Host bridge: Intel Corporation 440FX - 82441FX PMC [Natoma] (rev 02)"
	firstSpace := strings.Index(line, " ")
	if firstSpace == -1 {
		return nil
	}

	address := line[:firstSpace]
	remainder := strings.TrimSpace(line[firstSpace+1:])

	colonIdx := strings.Index(remainder, ":")
	if colonIdx == -1 {
		return &domain.PCIDevice{
			Address: address,
			Device:  remainder,
		}
	}

	class := strings.TrimSpace(remainder[:colonIdx])
	deviceInfo := strings.TrimSpace(remainder[colonIdx+1:])

	vendor := ""
	device := deviceInfo
	if devParts := strings.SplitN(deviceInfo, " ", 2); len(devParts) > 1 {
		vendor = devParts[0]
		device = devParts[1]
	}

	return &domain.PCIDevice{
		Address: address,
		Class:   class,
		Vendor:  vendor,
		Device:  device,
	}
}

func parseKeyValuePairs(line string) map[string]string {
	result := make(map[string]string)
	var currentKey strings.Builder
	var currentValue strings.Builder
	inQuotes := false
	buildingKey := true

	for i := 0; i < len(line); i++ {
		char := line[i]
		if char == '=' && buildingKey && !inQuotes {
			buildingKey = false
			continue
		}
		if char == '"' {
			inQuotes = !inQuotes
			if !inQuotes {
				key := strings.TrimSpace(currentKey.String())
				if key != "" {
					result[key] = currentValue.String()
				}
				currentKey.Reset()
				currentValue.Reset()
				buildingKey = true
			}
			continue
		}
		if buildingKey {
			if char != ' ' {
				currentKey.WriteByte(char)
			}
			continue
		}
		if inQuotes {
			currentValue.WriteByte(char)
		}
	}

	return result
}
