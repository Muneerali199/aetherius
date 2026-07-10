package metrics

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/aetherius/platform/agent/internal/heartbeat"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type Collector struct{}

func NewCollector() heartbeat.MetricsCollector {
	return &Collector{}
}

func (c *Collector) GetGPUUtilization() ([]float64, error) {
	if runtime.GOOS == "linux" {
		return c.getNvidiaGPUUtil()
	}
	return nil, nil
}

func (c *Collector) GetGPUTemps() ([]float64, error) {
	if runtime.GOOS == "linux" {
		return c.getNvidiaGPUTemps()
	}
	return nil, nil
}

func (c *Collector) GetVRAMUsed() ([]int64, error) {
	if runtime.GOOS == "linux" {
		return c.getNvidiaVRAMUsed()
	}
	return nil, nil
}

func (c *Collector) GetCPUUtilization() (float64, error) {
	pct, err := cpu.Percent(0, false)
	if err != nil || len(pct) == 0 {
		return 0, err
	}
	return pct[0] / 100.0, nil
}

func (c *Collector) GetRAMUsedGB() (int64, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return int64(v.Used / (1024 * 1024 * 1024)), nil
}

func (c *Collector) GetDiskUsedGB() (int64, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return 0, err
	}
	for _, p := range partitions {
		if p.Mountpoint == "/" || p.Mountpoint == "/System/Volumes/Data" {
			usage, err := disk.Usage(p.Mountpoint)
			if err != nil {
				return 0, err
			}
			return int64(usage.Used / (1024 * 1024 * 1024)), nil
		}
	}
	return 0, nil
}

func (c *Collector) GetNetworkRXBytes() (int64, error) {
	io, err := net.IOCounters(false)
	if err != nil || len(io) == 0 {
		return 0, err
	}
	return int64(io[0].BytesRecv), nil
}

func (c *Collector) GetNetworkTXBytes() (int64, error) {
	io, err := net.IOCounters(false)
	if err != nil || len(io) == 0 {
		return 0, err
	}
	return int64(io[0].BytesSent), nil
}

func (c *Collector) GetLoadAverage() (float64, error) {
	avg, err := load.Avg()
	if err != nil {
		return 0, err
	}
	return avg.Load1, nil
}

func (c *Collector) GetUptimeSeconds() (int64, error) {
	uptime, err := host.Uptime()
	if err != nil {
		return 0, err
	}
	return int64(uptime), nil
}

func (c *Collector) GetRunningContainers() ([]string, error) {
	output, err := exec.Command("docker", "ps", "--format", "{{.ID}}").Output()
	if err != nil {
		return nil, nil
	}
	containers := strings.Fields(string(output))
	return containers, nil
}

func (c *Collector) getNvidiaValue(query string) []string {
	data, err := exec.Command("nvidia-smi", "--query-gpu="+query, "--format=csv,noheader,nounits").Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	return lines
}

func (c *Collector) getNvidiaGPUUtil() ([]float64, error) {
	vals := c.getNvidiaValue("utilization.gpu")
	var result []float64
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
			result = append(result, f)
		}
	}
	return result, nil
}

func (c *Collector) getNvidiaGPUTemps() ([]float64, error) {
	vals := c.getNvidiaValue("temperature.gpu")
	var result []float64
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
			result = append(result, f)
		}
	}
	return result, nil
}

func (c *Collector) getNvidiaVRAMUsed() ([]int64, error) {
	vals := c.getNvidiaValue("memory.used")
	var result []int64
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
			result = append(result, int64(f)*1024*1024)
		}
	}
	return result, nil
}
