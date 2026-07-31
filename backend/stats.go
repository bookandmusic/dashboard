package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type CPUStats struct {
	Usage   float64   `json:"usage"`
	Cores   int       `json:"cores"`
	Loadavg []float64 `json:"loadavg"`
}

type MemoryStats struct {
	Total     uint64  `json:"total"`
	Available uint64  `json:"available"`
	Usage     float64 `json:"usage"`
}

type DiskInfo struct {
	Name      string  `json:"name"`
	Mount     string  `json:"mount"`
	Total     uint64  `json:"total"`
	Available uint64  `json:"available"`
	Usage     float64 `json:"usage"`
}

type NetworkStats struct {
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
	RxRate  uint64 `json:"rx_rate"`
	TxRate  uint64 `json:"tx_rate"`
}

type Stats struct {
	CPU      CPUStats     `json:"cpu"`
	Memory   MemoryStats  `json:"memory"`
	Disks    []DiskInfo   `json:"disks"`
	Network  NetworkStats `json:"network"`
	Uptime   int64        `json:"uptime"`
	Hostname string       `json:"hostname"`
}

// cpuTimes 是 /proc/stat 首行的累计计数器。
type cpuTimes struct {
	total, idle uint64
}

func parseStat(content string) (cpuTimes, int, error) {
	var t cpuTimes
	cores := 0
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch {
		case fields[0] == "cpu" && len(fields) >= 9:
			var vals [8]uint64
			for i := 0; i < 8; i++ {
				v, err := strconv.ParseUint(fields[i+1], 10, 64)
				if err != nil {
					return t, 0, fmt.Errorf("parse stat cpu field %d: %w", i, err)
				}
				vals[i] = v
			}
			for _, v := range vals {
				t.total += v
			}
			t.idle = vals[3] + vals[4] // idle + iowait
		case strings.HasPrefix(fields[0], "cpu") && len(fields[0]) > 3:
			cores++
		}
	}
	if t.total == 0 {
		return t, 0, fmt.Errorf("parse stat: no aggregate cpu line")
	}
	return t, cores, nil
}

func parseMeminfo(content string) (total, available uint64, err error) {
	found := 0
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		var dst *uint64
		switch fields[0] {
		case "MemTotal:":
			dst = &total
		case "MemAvailable:":
			dst = &available
		default:
			continue
		}
		kb, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parse meminfo %s: %w", fields[0], err)
		}
		*dst = kb * 1024
		found++
		if found == 2 {
			return total, available, nil
		}
	}
	return 0, 0, fmt.Errorf("parse meminfo: MemTotal/MemAvailable not found")
}

func parseNetDev(content string) (rx, tx uint64, err error) {
	matched := false
	for _, line := range strings.Split(content, "\n") {
		iface, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		name := strings.TrimSpace(iface)
		if name == "lo" {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) < 9 {
			return 0, 0, fmt.Errorf("parse net/dev %s: too few fields", name)
		}
		r, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parse net/dev %s rx: %w", name, err)
		}
		x, err := strconv.ParseUint(fields[8], 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parse net/dev %s tx: %w", name, err)
		}
		rx += r
		tx += x
		matched = true
	}
	if !matched {
		return 0, 0, fmt.Errorf("parse net/dev: no interfaces found")
	}
	return rx, tx, nil
}

func parseUptime(content string) (int64, error) {
	fields := strings.Fields(content)
	if len(fields) == 0 {
		return 0, fmt.Errorf("parse uptime: empty")
	}
	f, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("parse uptime: %w", err)
	}
	return int64(f), nil
}

func parseLoadavg(content string) ([]float64, error) {
	fields := strings.Fields(content)
	if len(fields) < 3 {
		return nil, fmt.Errorf("parse loadavg: too few fields")
	}
	out := make([]float64, 3)
	for i := 0; i < 3; i++ {
		f, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			return nil, fmt.Errorf("parse loadavg: %w", err)
		}
		out[i] = f
	}
	return out, nil
}

// Collector 从 procPath（默认 /proc，容器内为挂载的宿主机路径）采集系统指标。
type Collector struct {
	procPath string
	hostRoot string
	now      func() time.Time

	mu      sync.RWMutex
	current *Stats

	prevCPU  cpuTimes
	prevRx   uint64
	prevTx   uint64
	prevAt   time.Time
	primed   bool
}

func NewCollector(procPath, hostRoot string) *Collector {
	return &Collector{procPath: procPath, hostRoot: hostRoot, now: time.Now}
}

func (c *Collector) readFile(name string) (string, error) {
	data, err := os.ReadFile(filepath.Join(c.procPath, name))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Sample 读取一次指标并更新缓存。CPU 使用率与网络速率需要两次采样才能计算，
// 首次采样时这两项为 0。
func (c *Collector) Sample() error {
	statContent, err := c.readFile("stat")
	if err != nil {
		return fmt.Errorf("sample: %w", err)
	}
	memContent, err := c.readFile("meminfo")
	if err != nil {
		return fmt.Errorf("sample: %w", err)
	}
	netContent, err := c.readFile("net/dev")
	if err != nil {
		return fmt.Errorf("sample: %w", err)
	}
	uptimeContent, err := c.readFile("uptime")
	if err != nil {
		return fmt.Errorf("sample: %w", err)
	}
	loadavgContent, err := c.readFile("loadavg")
	if err != nil {
		return fmt.Errorf("sample: %w", err)
	}

	cpu, cores, err := parseStat(statContent)
	if err != nil {
		return fmt.Errorf("sample: %w", err)
	}
	memTotal, memAvail, err := parseMeminfo(memContent)
	if err != nil {
		return fmt.Errorf("sample: %w", err)
	}
	rx, tx, err := parseNetDev(netContent)
	if err != nil {
		return fmt.Errorf("sample: %w", err)
	}
	uptime, err := parseUptime(uptimeContent)
	if err != nil {
		return fmt.Errorf("sample: %w", err)
	}
	loadavg, err := parseLoadavg(loadavgContent)
	if err != nil {
		return fmt.Errorf("sample: %w", err)
	}

	now := c.now()
	s := &Stats{
		CPU:    CPUStats{Cores: cores, Loadavg: loadavg},
		Memory: MemoryStats{Total: memTotal, Available: memAvail},
		Network: NetworkStats{
			RxBytes: rx,
			TxBytes: tx,
		},
		Uptime: uptime,
	}
	if s.Memory.Total > 0 {
		s.Memory.Usage = round2(float64(memTotal-memAvail) / float64(memTotal) * 100)
	}

	if c.primed {
		// 计数器回绕/接口重置时跳过本次差值，避免 uint64 下溢
		if cpu.total >= c.prevCPU.total && cpu.idle >= c.prevCPU.idle {
			dTotal := cpu.total - c.prevCPU.total
			dIdle := cpu.idle - c.prevCPU.idle
			if dTotal > 0 {
				s.CPU.Usage = round2(float64(dTotal-dIdle) / float64(dTotal) * 100)
			}
		}
		if dt := now.Sub(c.prevAt).Seconds(); dt > 0 {
			if rx >= c.prevRx {
				s.Network.RxRate = uint64(float64(rx-c.prevRx) / dt)
			}
			if tx >= c.prevTx {
				s.Network.TxRate = uint64(float64(tx-c.prevTx) / dt)
			}
		}
	}
	c.prevCPU, c.prevRx, c.prevTx, c.prevAt = cpu, rx, tx, now
	c.primed = true

	s.Disks = c.sampleDisks()
	if hostname, err := os.Hostname(); err == nil {
		s.Hostname = hostname
	}

	c.mu.Lock()
	c.current = s
	c.mu.Unlock()
	return nil
}

// pseudoFS 是需要排除的系统/伪文件系统（只统计真实数据卷）。
// overlay 为容器根，与宿主机数据盘重复，一并排除。
var pseudoFS = map[string]bool{
	"proc": true, "sysfs": true, "tmpfs": true, "devtmpfs": true, "devpts": true,
	"cgroup": true, "cgroup2": true, "pstore": true, "securityfs": true, "debugfs": true,
	"tracefs": true, "configfs": true, "bpf": true, "autofs": true, "hugetlbfs": true,
	"mqueue": true, "efivarfs": true, "binfmt_misc": true, "fusectl": true,
	"rpc_pipefs": true, "sunrpc": true, "nsfs": true, "selinuxfs": true, "ramfs": true,
	"overlay": true,
}

// sampleDisks 枚举挂载点并统计每块数据卷的占用。
// 按设备去重（同一块盘的多个 bind mount 只留一个），跳过文件挂载与伪文件系统；
// statfs 失败的自动跳过（容器部署时挂载点可能不在本命名空间）。
// 设置 hostRoot 后可对 hostRoot+挂载点 统计（配合卷挂载使用）。
func (c *Collector) sampleDisks() []DiskInfo {
	data, err := os.ReadFile(filepath.Join(c.procPath, "mounts"))
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var disks []DiskInfo
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		device, mount, fstype := fields[0], fields[1], fields[2]
		if pseudoFS[fstype] || seen[device] {
			continue
		}
		target := mount
		if c.hostRoot != "" {
			target = filepath.Join(c.hostRoot, mount)
		}
		if fi, err := os.Stat(target); err != nil || !fi.IsDir() {
			continue
		}
		var st syscall.Statfs_t
		if err := syscall.Statfs(target, &st); err != nil {
			continue
		}
		bsize := uint64(st.Bsize)
		total := st.Blocks * bsize
		avail := st.Bavail * bsize
		if total == 0 {
			continue
		}
		seen[device] = true
		disks = append(disks, DiskInfo{
			Name:      diskName(device),
			Mount:     mount,
			Total:     total,
			Available: avail,
			Usage:     round2(float64(total-avail) / float64(total) * 100),
		})
	}
	return disks
}

// diskName 把设备路径清理为简短显示名：/dev/sda1 → sda1，/dev/mapper/foo → foo。
func diskName(device string) string {
	d := strings.TrimPrefix(device, "/dev/")
	d = strings.TrimPrefix(d, "mapper/")
	return d
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}

// Current 返回最近一次采样的快照；从未采样时返回零值。
func (c *Collector) Current() *Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.current == nil {
		return &Stats{CPU: CPUStats{Loadavg: []float64{0, 0, 0}}}
	}
	return c.current
}

// Run 启动采样循环：先快速采两次（间隔 1s）让速率立即可用，之后按 interval 采样。
func (c *Collector) Run(interval time.Duration) {
	if err := c.Sample(); err != nil {
		log.Printf("stats prime sample failed: %v", err)
	}
	time.Sleep(time.Second)
	if err := c.Sample(); err != nil {
		log.Printf("stats prime sample failed: %v", err)
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		if err := c.Sample(); err != nil {
			log.Printf("stats sample failed: %v", err)
		}
	}
}
