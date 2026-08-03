// Package monitoring collects baseline process and host metrics from /proc
// without requiring any Minecraft-side mod. Resident memory is reported as
// RSS, never mislabeled as Java heap.
package monitoring

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Sample is one metrics snapshot.
type Sample struct {
	CollectedAt    string  `json:"collected_at"`
	CPUPercent     float64 `json:"cpu_percent"`
	RSSBytes       int64   `json:"rss_bytes"` // resident memory (NOT Java heap)
	HostMemTotal   int64   `json:"host_mem_total"`
	HostMemAvail   int64   `json:"host_mem_avail"`
	Load1          float64 `json:"load1"`
	DiskTotal      int64   `json:"disk_total"`
	DiskFree       int64   `json:"disk_free"`
	ServerDirBytes int64   `json:"server_dir_bytes"`
	BackupDirBytes int64   `json:"backup_dir_bytes"`
	OnlinePlayers  int     `json:"online_players"`
	UptimeSeconds  int64   `json:"uptime_seconds"`
	JavaPID        int     `json:"java_pid"`
}

// Collector tracks CPU tick deltas per PID between samples.
type Collector struct {
	lastProc map[int]uint64
	lastTot  uint64
	clockTck float64
	nCPU     float64
}

func NewCollector() *Collector {
	return &Collector{lastProc: map[int]uint64{}, clockTck: 100, nCPU: float64(numCPU())}
}

func numCPU() int {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return 1
	}
	n := strings.Count(string(data), "processor\t")
	if n < 1 {
		return 1
	}
	return n
}

// ProcessCPU returns CPU percent of pid since the previous call.
//
// Both counters are unsigned and monotonic only while the same process keeps
// running. A restarted server reuses the collector with a fresh, smaller tick
// count, and an unguarded subtraction wraps to about 1.8e19 — a single poisoned
// sample that then skews every average and chart drawn from the metrics table.
// Anything that is not a sane forward delta returns zero instead.
func (c *Collector) ProcessCPU(pid int) float64 {
	if pid <= 0 {
		return 0
	}
	procTicks := readProcTicks(pid)
	totTicks := readTotalTicks()
	defer func() {
		c.lastProc[pid] = procTicks
		c.lastTot = totTicks
	}()

	// A zero reading means /proc was unreadable or the process has gone. Do
	// not treat it as "used no CPU"; it is an absence of data.
	if procTicks == 0 || totTicks == 0 {
		return 0
	}
	prevP, okP := c.lastProc[pid], c.lastProc[pid] > 0
	if !okP || c.lastTot == 0 {
		return 0 // first sample for this PID: nothing to compare against
	}
	// Counters going backwards means the PID was reused or the host counter
	// reset. Re-baseline silently rather than reporting nonsense.
	if procTicks < prevP || totTicks <= c.lastTot {
		return 0
	}

	dp := float64(procTicks - prevP)
	dt := float64(totTicks - c.lastTot)
	if dt <= 0 {
		return 0
	}
	pct := dp / dt * 100 * c.nCPU

	// A process cannot use more than every core. Small overshoots happen from
	// rounding across a sampling boundary, so clamp rather than discard, but
	// treat anything wildly out of range as a bad reading.
	max := 100 * c.nCPU
	switch {
	case pct < 0 || pct != pct: // negative, or NaN from a zero delta
		return 0
	case pct > max*2:
		return 0 // implausible: something reset underneath us
	case pct > max:
		return max
	}
	return pct
}

// ForgetProcess drops the cached tick baseline for a PID. The supervisor calls
// this when a server stops so the next start is measured from its own zero
// rather than against a dead process's counters.
func (c *Collector) ForgetProcess(pid int) {
	if pid > 0 {
		delete(c.lastProc, pid)
	}
}

func readProcTicks(pid int) uint64 {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0
	}
	// fields after the (comm) which may contain spaces
	s := string(data)
	idx := strings.LastIndex(s, ")")
	if idx < 0 {
		return 0
	}
	fields := strings.Fields(s[idx+1:])
	if len(fields) < 13 {
		return 0
	}
	utime, _ := strconv.ParseUint(fields[11], 10, 64)
	stime, _ := strconv.ParseUint(fields[12], 10, 64)
	return utime + stime
}

func readTotalTicks() uint64 {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "cpu ") {
			var tot uint64
			for _, f := range strings.Fields(line)[1:] {
				v, _ := strconv.ParseUint(f, 10, 64)
				tot += v
			}
			return tot
		}
	}
	return 0
}

// ProcessRSS returns the resident set size of pid in bytes.
func ProcessRSS(pid int) int64 {
	if pid <= 0 {
		return 0
	}
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			f := strings.Fields(line)
			if len(f) >= 2 {
				kb, _ := strconv.ParseInt(f[1], 10, 64)
				return kb << 10
			}
		}
	}
	return 0
}

// HostMemory returns (totalBytes, availableBytes).
func HostMemory() (int64, int64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	var total, avail int64
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		kb, _ := strconv.ParseInt(f[1], 10, 64)
		switch f[0] {
		case "MemTotal:":
			total = kb << 10
		case "MemAvailable:":
			avail = kb << 10
		}
	}
	return total, avail
}

// LoadAvg returns the 1-minute load average.
func LoadAvg() float64 {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0
	}
	f := strings.Fields(string(data))
	if len(f) < 1 {
		return 0
	}
	v, _ := strconv.ParseFloat(f[0], 64)
	return v
}

// DiskUsage returns (total, free) bytes for the filesystem containing path.
func DiskUsage(path string) (int64, int64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0
	}
	return int64(st.Blocks) * st.Bsize, int64(st.Bavail) * st.Bsize
}

// StoreSample inserts a sample into the metrics table.
func StoreSample(db *sql.DB, instanceID int64, s *Sample) {
	db.Exec(`INSERT INTO metrics (collected_at, instance_id, cpu_percent, rss_bytes,
		host_mem_total, host_mem_avail, load1, disk_total, disk_free,
		server_dir_bytes, backup_dir_bytes, online_players)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		s.CollectedAt, instanceID, s.CPUPercent, s.RSSBytes,
		s.HostMemTotal, s.HostMemAvail, s.Load1, s.DiskTotal, s.DiskFree,
		s.ServerDirBytes, s.BackupDirBytes, s.OnlinePlayers)
}

// Prune removes samples older than retention.
func Prune(db *sql.DB, retention time.Duration) {
	cutoff := time.Now().UTC().Add(-retention).Format(time.RFC3339)
	db.Exec(`DELETE FROM metrics WHERE collected_at < ?`, cutoff)
}

// History returns recent samples (ascending time), capped.
func History(db *sql.DB, since time.Time, limit int) ([]Sample, error) {
	rows, err := db.Query(`SELECT collected_at, cpu_percent, rss_bytes, host_mem_total,
		host_mem_avail, load1, disk_total, disk_free, server_dir_bytes,
		backup_dir_bytes, online_players
		FROM metrics WHERE collected_at >= ? ORDER BY collected_at ASC LIMIT ?`,
		since.UTC().Format(time.RFC3339), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Sample
	for rows.Next() {
		var s Sample
		if err := rows.Scan(&s.CollectedAt, &s.CPUPercent, &s.RSSBytes, &s.HostMemTotal,
			&s.HostMemAvail, &s.Load1, &s.DiskTotal, &s.DiskFree,
			&s.ServerDirBytes, &s.BackupDirBytes, &s.OnlinePlayers); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
