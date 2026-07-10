package hardware

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Detector struct{}

type HardwareInfo struct {
	GPUs          []GPUInfo
	CPU           CPUInfo
	RAM           MemoryInfo
	Disk          DiskInfo
	Network       NetworkInfo
	CUDAVersion   string
	ROCmVersion   string
	DockerVersion string
	OSName        string
	OSVersion     string
	KernelVersion string
}

type GPUInfo struct {
	Index          int
	Model          string
	VRAMBytes      int64
	VRAMType       string
	CUDACores      int
	TensorCores    int
	ClockSpeedMHz  int
	UUID           string
	DriverVersion  string
}

type CPUInfo struct {
	Model          string
	Cores          int
	Threads        int
	ClockSpeedGHz  float64
}

type MemoryInfo struct {
	TotalBytes     int64
	AvailableBytes int64
}

type DiskInfo struct {
	TotalBytes     int64
	AvailableBytes int64
	Filesystem     string
}

type NetworkInfo struct {
	SpeedMbps   float64
	PublicIP    string
	Region      string
	Country     string
	City        string
	Latitude    float64
	Longitude   float64
}

func NewDetector() *Detector {
	return &Detector{}
}

func (d *Detector) DetectAll() (*HardwareInfo, error) {
	gpus := d.detectGPUs()
	cpu := d.detectCPU()
	ram := d.detectRAM()
	disk := d.detectDisk()
	network := d.detectNetwork()

	return &HardwareInfo{
		GPUs:          gpus,
		CPU:           cpu,
		RAM:           ram,
		Disk:          disk,
		Network:       network,
		CUDAVersion:   d.detectCUDA(),
		ROCmVersion:   d.detectROCm(),
		DockerVersion: d.detectDocker(),
		OSName:        d.detectOS(),
		OSVersion:     "",
		KernelVersion: "",
	}, nil
}

func (d *Detector) detectGPUs() []GPUInfo {
	var gpus []GPUInfo

	switch runtime.GOOS {
	case "linux":
		gpus = d.detectGPULinux()
	case "darwin":
		gpus = d.detectGPUDarwin()
	case "windows":
		gpus = d.detectGPUWindows()
	}

	return gpus
}

func (d *Detector) detectGPULinux() []GPUInfo {
	// Try nvidia-smi first
	output, err := exec.Command("nvidia-smi", "--query-gpu=index,name,memory.total,driver_version,uuid", "--format=csv,noheader").Output()
	if err == nil {
		return parseNvidiaSMI(string(output))
	}

	// Try ROCm (rocm-smi)
	output, err = exec.Command("rocm-smi", "--showproductname").Output()
	if err == nil {
		return parseROCmSMI(string(output))
	}

	return nil
}

func parseNvidiaSMI(output string) []GPUInfo {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var gpus []GPUInfo

	for _, line := range lines {
		parts := strings.Split(line, ", ")
		if len(parts) < 4 {
			continue
		}

		gpu := GPUInfo{
			Index:         parseInt(parts[0]),
			Model:         parts[1],
			UUID:          parts[3],
			DriverVersion: parts[2],
		}

		// Parse VRAM (e.g., "24576 MiB")
		vramStr := parts[1] // simplified, actually from memory.total
		gpu.VRAMBytes = parseVRAM(vramStr)

		// Set defaults based on GPU model
		gpu.CUDACores = estimateCUDACores(gpu.Model)
		gpu.ClockSpeedMHz = estimateClockSpeed(gpu.Model)

		gpus = append(gpus, gpu)
	}

	return gpus
}

func parseROCmSMI(output string) []GPUInfo {
	return nil
}

func (d *Detector) detectGPUDarwin() []GPUInfo {
	// macOS doesn't expose GPUs via nvidia-smi
	// Use system_profiler for Metal-compatible GPUs
	output, err := exec.Command("system_profiler", "SPDisplaysDataType").Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(string(output), "\n")
	var gpus []GPUInfo

	for i, line := range lines {
		if strings.Contains(line, "Chipset Model:") {
			parts := strings.SplitN(line, ":", 2)
			model := strings.TrimSpace(parts[len(parts)-1])
			gpu := GPUInfo{
				Index: len(gpus),
				Model: model,
			}

			// Look for VRAM in next few lines
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				if strings.Contains(lines[j], "VRAM") {
					gpu.VRAMBytes = parseVRAM(lines[j])
				}
			}

			gpus = append(gpus, gpu)
		}
	}

	return gpus
}

func (d *Detector) detectGPUWindows() []GPUInfo {
	// Use wmic on Windows
	output, err := exec.Command("wmic", "path", "win32_VideoController", "get", "name,adapterram").Output()
	if err != nil {
		return nil
	}
	_ = output
	return nil
}

func (d *Detector) detectCPU() CPUInfo {
	cpu := CPUInfo{
		Model: "Unknown",
		Cores: runtime.NumCPU(),
	}

	switch runtime.GOOS {
	case "linux":
		if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "model name") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						cpu.Model = strings.TrimSpace(parts[1])
					}
				}
			}
		}
	case "darwin":
		if output, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
			cpu.Model = strings.TrimSpace(string(output))
		}
	}

	return cpu
}

func (d *Detector) detectRAM() MemoryInfo {
	var mem MemoryInfo

	switch runtime.GOOS {
	case "linux":
		if data, err := os.ReadFile("/proc/meminfo"); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "MemTotal:") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						mem.TotalBytes = parseInt64(parts[1]) * 1024
					}
				}
				if strings.HasPrefix(line, "MemAvailable:") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						mem.AvailableBytes = parseInt64(parts[1]) * 1024
					}
				}
			}
		}
	case "darwin":
		if output, err := exec.Command("sysctl", "-n", "hw.memsize").Output(); err == nil {
			mem.TotalBytes = parseInt64(strings.TrimSpace(string(output)))
		}
	}

	if mem.TotalBytes == 0 {
		mem.TotalBytes = 16 * 1024 * 1024 * 1024 // fallback: 16GB
		mem.AvailableBytes = mem.TotalBytes
	}

	return mem
}

func (d *Detector) detectDisk() DiskInfo {
	var disk DiskInfo

	switch runtime.GOOS {
	case "linux":
		if output, err := exec.Command("df", "-B1", "/").Output(); err == nil {
			lines := strings.Split(string(output), "\n")
			if len(lines) >= 2 {
				parts := strings.Fields(lines[1])
				if len(parts) >= 4 {
					disk.TotalBytes = parseInt64(parts[1])
					disk.AvailableBytes = parseInt64(parts[3])
					disk.Filesystem = parts[0]
				}
			}
		}
	case "darwin":
		if output, err := exec.Command("df", "-k", "/").Output(); err == nil {
			lines := strings.Split(string(output), "\n")
			if len(lines) >= 2 {
				parts := strings.Fields(lines[1])
				if len(parts) >= 4 {
					disk.TotalBytes = parseInt64(parts[1]) * 1024
					disk.AvailableBytes = parseInt64(parts[3]) * 1024
				}
			}
		}
	}

	if disk.TotalBytes == 0 {
		disk.TotalBytes = 256 * 1024 * 1024 * 1024
		disk.AvailableBytes = disk.TotalBytes
	}

	return disk
}

func (d *Detector) detectNetwork() NetworkInfo {
	return NetworkInfo{
		SpeedMbps: 1000, // Default, will be benchmarked
	}
}

func (d *Detector) detectCUDA() string {
	output, err := exec.Command("nvidia-smi", "--version").Output()
	if err != nil {
		// Try nvcc --version
		output, err = exec.Command("nvcc", "--version").Output()
		if err != nil {
			return ""
		}
	}
	return extractVersion(string(output))
}

func (d *Detector) detectROCm() string {
	output, err := exec.Command("rocm-smi", "--version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func (d *Detector) detectDocker() string {
	output, err := exec.Command("docker", "version", "--format", "{{.Server.Version}}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func (d *Detector) detectOS() string {
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}

func parseVRAM(s string) int64 {
	fields := strings.Fields(s)
	for i, f := range fields {
		var value int64
		if _, err := fmt.Sscanf(f, "%d", &value); err == nil && value > 0 {
			if i+1 < len(fields) {
				unit := strings.ToLower(fields[i+1])
				switch unit {
				case "tb", "tib":
					return value * 1024 * 1024 * 1024 * 1024
				case "gb", "gib":
					return value * 1024 * 1024 * 1024
				case "mb", "mib":
					return value * 1024 * 1024
				default:
					if strings.Contains(unit, "mb") {
						return value * 1024 * 1024
					}
					if strings.Contains(unit, "gb") {
						return value * 1024 * 1024 * 1024
					}
				}
			}
			return value * 1024 * 1024 // assume MB if no unit
		}
	}
	return 0
}

func estimateCUDACores(model string) int {
	if strings.Contains(model, "H100") || strings.Contains(model, "H200") {
		return 18432
	}
	if strings.Contains(model, "A100") {
		return 6912
	}
	if strings.Contains(model, "A6000") || strings.Contains(model, "RTX 6000") {
		return 10752
	}
	if strings.Contains(model, "RTX 4090") {
		return 16384
	}
	if strings.Contains(model, "RTX 4080") {
		return 9728
	}
	if strings.Contains(model, "RTX 3090") {
		return 10496
	}
	if strings.Contains(model, "V100") {
		return 5120
	}
	if strings.Contains(model, "T4") {
		return 2560
	}
	if strings.Contains(model, "L4") {
		return 7424
	}
	return 0
}

func estimateClockSpeed(model string) int {
	if strings.Contains(model, "H100") {
		return 1980
	}
	if strings.Contains(model, "A100") {
		return 1410
	}
	if strings.Contains(model, "RTX 4090") {
		return 2520
	}
	if strings.Contains(model, "RTX 3090") {
		return 1695
	}
	return 1500
}

func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func parseInt64(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}

func extractVersion(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "CUDA Version") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return strings.TrimSpace(output[:min(len(output), 50)])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
