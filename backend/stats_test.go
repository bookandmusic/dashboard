package main

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const stat1 = `cpu  1000 0 500 8000 500 0 0 0 0 0
cpu0 250 0 125 2000 125 0 0 0 0 0
cpu1 250 0 125 2000 125 0 0 0 0 0
cpu2 250 0 125 2000 125 0 0 0 0 0
cpu3 250 0 125 2000 125 0 0 0 0 0
intr 12345
ctxt 67890
`

const stat2 = `cpu  2000 0 1000 9000 500 0 0 0 0 0
cpu0 500 0 250 2250 125 0 0 0 0 0
cpu1 500 0 250 2250 125 0 0 0 0 0
cpu2 500 0 250 2250 125 0 0 0 0 0
cpu3 500 0 250 2250 125 0 0 0 0 0
intr 12345
ctxt 67890
`

const meminfoFixture = `MemTotal:       1000000 kB
MemFree:         100000 kB
MemAvailable:    600000 kB
Buffers:          50000 kB
`

const netdev1 = `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo:  999999     100    0    0    0     0          0         0   999999     100    0    0    0     0       0          0
  eth0: 1000000    2000    0    0    0     0          0         0   500000    1500    0    0    0     0       0          0
`

const netdev2 = `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo:  1999999    200    0    0    0     0          0         0  1999999     200    0    0    0     0       0          0
  eth0: 1002048    2010    0    0    0     0          0         0   501024    1510    0    0    0     0       0          0
`

const uptimeFixture = "5085257.47 30477728.86\n"
const loadavgFixture = "0.50 0.75 1.00 1/500 12345\n"

// 一个真实数据卷（/，ext4）+ 若干应被过滤的伪文件系统
const mountsFixture = `/dev/root / ext4 rw,reliread 0 0
proc /proc proc rw,nosuid,nodev,noexec,relatime 0 0
tmpfs /dev/shm tmpfs rw,nosuid,nodev 0 0
sysfs /sys sysfs rw,nosuid,nodev,noexec,relatime 0 0
`

func writeProcFiles(t *testing.T, dir, stat, meminfo, netdev, uptime, loadavg string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "net"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"stat":     stat,
		"meminfo":  meminfo,
		"net/dev":  netdev,
		"uptime":   uptime,
		"loadavg":  loadavg,
		"mounts":   mountsFixture,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCollectorSample(t *testing.T) {
	dir := t.TempDir()
	writeProcFiles(t, dir, stat1, meminfoFixture, netdev1, uptimeFixture, loadavgFixture)

	c := NewCollector(dir, "")
	t0 := time.Now()
	c.now = func() time.Time { return t0 }

	if err := c.Sample(); err != nil {
		t.Fatalf("Sample: %v", err)
	}
	s := c.Current()

	if s.CPU.Cores != 4 {
		t.Errorf("CPU.Cores = %d, want 4", s.CPU.Cores)
	}
	if s.CPU.Usage != 0 {
		t.Errorf("first sample CPU.Usage = %v, want 0 (not primed)", s.CPU.Usage)
	}
	if len(s.CPU.Loadavg) != 3 || s.CPU.Loadavg[0] != 0.5 || s.CPU.Loadavg[2] != 1.0 {
		t.Errorf("CPU.Loadavg = %v", s.CPU.Loadavg)
	}
	if s.Memory.Total != 1000000*1024 {
		t.Errorf("Memory.Total = %d", s.Memory.Total)
	}
	if s.Memory.Available != 600000*1024 {
		t.Errorf("Memory.Available = %d", s.Memory.Available)
	}
	if math.Abs(s.Memory.Usage-40.0) > 0.01 {
		t.Errorf("Memory.Usage = %v, want 40.0", s.Memory.Usage)
	}
	if s.Network.RxBytes != 1000000 || s.Network.TxBytes != 500000 {
		t.Errorf("Network bytes = %d/%d, lo must be excluded", s.Network.RxBytes, s.Network.TxBytes)
	}
	if s.Uptime != 5085257 {
		t.Errorf("Uptime = %d, want 5085257", s.Uptime)
	}
	if len(s.Disks) != 1 || s.Disks[0].Mount != "/" {
		t.Fatalf("Disks = %+v, want single / mount (pseudo fs filtered)", s.Disks)
	}
	if s.Disks[0].Name != "root" {
		t.Errorf("Disks[0].Name = %q, want root (/dev/root cleaned)", s.Disks[0].Name)
	}
	if s.Disks[0].Total == 0 || s.Disks[0].Usage < 0 || s.Disks[0].Usage > 100 {
		t.Errorf("Disks[0] = %+v", s.Disks[0])
	}

	writeProcFiles(t, dir, stat2, meminfoFixture, netdev2, uptimeFixture, loadavgFixture)
	c.now = func() time.Time { return t0.Add(2 * time.Second) }

	if err := c.Sample(); err != nil {
		t.Fatalf("Sample 2: %v", err)
	}
	s = c.Current()

	// dTotal=2500, dIdle=1000 → 60%
	if math.Abs(s.CPU.Usage-60.0) > 0.01 {
		t.Errorf("CPU.Usage = %v, want 60.0", s.CPU.Usage)
	}
	// delta rx=2048 over 2s → 1024 B/s; delta tx=1024 → 512 B/s
	if s.Network.RxRate != 1024 {
		t.Errorf("Network.RxRate = %d, want 1024", s.Network.RxRate)
	}
	if s.Network.TxRate != 512 {
		t.Errorf("Network.TxRate = %d, want 512", s.Network.TxRate)
	}
	if s.Network.RxBytes != 1002048 {
		t.Errorf("Network.RxBytes = %d", s.Network.RxBytes)
	}
}

func TestCollectorCurrentBeforeSample(t *testing.T) {
	c := NewCollector("/nonexistent", "")
	s := c.Current()
	if s == nil {
		t.Fatal("Current() must never return nil")
	}
	if s.Uptime != 0 || s.Memory.Total != 0 {
		t.Errorf("expected zero stats, got %+v", s)
	}
}

func TestCollectorSampleMissingProc(t *testing.T) {
	c := NewCollector("/nonexistent-proc-path", "")
	if err := c.Sample(); err == nil {
		t.Error("expected error for missing proc path")
	}
}
