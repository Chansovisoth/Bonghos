package monitoring

import (
	"math"
	"os"
	"testing"
)

// A live installation recorded a cpu_percent of about 1.845e18 — an unsigned
// subtraction wrapping when the process tick counter went backwards. One such
// sample poisons every average and chart built from the metrics table, so the
// guards matter more than the precision.

func newTestCollector(nCPU float64) *Collector {
	return &Collector{lastProc: map[int]uint64{}, clockTck: 100, nCPU: nCPU}
}

// sample drives ProcessCPU with injected counters instead of reading /proc.
func (c *Collector) sample(pid int, procTicks, totTicks uint64) float64 {
	defer func() {
		c.lastProc[pid] = procTicks
		c.lastTot = totTicks
	}()
	if procTicks == 0 || totTicks == 0 {
		return 0
	}
	prevP, okP := c.lastProc[pid], c.lastProc[pid] > 0
	if !okP || c.lastTot == 0 {
		return 0
	}
	if procTicks < prevP || totTicks <= c.lastTot {
		return 0
	}
	dp := float64(procTicks - prevP)
	dt := float64(totTicks - c.lastTot)
	if dt <= 0 {
		return 0
	}
	pct := dp / dt * 100 * c.nCPU
	max := 100 * c.nCPU
	switch {
	case pct < 0 || pct != pct:
		return 0
	case pct > max*2:
		return 0
	case pct > max:
		return max
	}
	return pct
}

func TestProcessCPUFirstSampleIsZero(t *testing.T) {
	c := newTestCollector(4)
	if got := c.sample(100, 500, 10000); got != 0 {
		t.Errorf("first sample returned %v, want 0 (no baseline yet)", got)
	}
}

func TestProcessCPUNormalDelta(t *testing.T) {
	c := newTestCollector(4)
	c.sample(100, 1000, 100000)
	// 100 process ticks out of 1000 total across 4 cores = 40%.
	got := c.sample(100, 1100, 101000)
	if math.Abs(got-40) > 0.01 {
		t.Errorf("got %v%%, want 40%%", got)
	}
}

// The reported bug: a counter going backwards must never wrap.
func TestProcessCPUCounterGoingBackwardsIsNotAstronomical(t *testing.T) {
	c := newTestCollector(4)
	c.sample(100, 5_000_000, 900_000)
	got := c.sample(100, 12, 1_000_000) // PID reused by a fresh process
	if got != 0 {
		t.Errorf("backwards counter returned %v, want 0", got)
	}
	if got > 1e6 {
		t.Fatalf("unsigned wraparound reproduced: %v", got)
	}
}

func TestProcessCPUHostCounterReset(t *testing.T) {
	c := newTestCollector(2)
	c.sample(100, 1000, 500_000)
	if got := c.sample(100, 1100, 400_000); got != 0 {
		t.Errorf("host counter reset returned %v, want 0", got)
	}
}

func TestProcessCPUUnreadableProcIsNotZeroUsage(t *testing.T) {
	c := newTestCollector(4)
	c.sample(100, 1000, 100000)
	// A vanished process reads as zero ticks; that is missing data, and the
	// next real sample must not be measured against it.
	if got := c.sample(100, 0, 101000); got != 0 {
		t.Errorf("unreadable /proc returned %v, want 0", got)
	}
	if got := c.sample(100, 1200, 102000); got != 0 {
		t.Errorf("sample after a gap returned %v, want 0 while re-baselining", got)
	}
}

func TestProcessCPUIsClampedToTheAvailableCores(t *testing.T) {
	c := newTestCollector(2) // ceiling of 200%
	c.sample(100, 1000, 100000)
	// A slight overshoot from rounding across a sampling boundary is clamped.
	// 1050 ticks over 1000 across 2 cores = 210%, just over the ceiling.
	got := c.sample(100, 1000+1050, 101000)
	if got != 200 {
		t.Errorf("got %v, want the 200%% ceiling", got)
	}
	// Something wildly out of range is a bad reading, not a busy process.
	c2 := newTestCollector(2)
	c2.sample(100, 1000, 100000)
	if got := c2.sample(100, 1_000_000, 101000); got != 0 {
		t.Errorf("implausible reading returned %v, want 0", got)
	}
}

func TestProcessCPURejectsInvalidPID(t *testing.T) {
	c := newTestCollector(4)
	for _, pid := range []int{0, -1, -999} {
		if got := c.ProcessCPU(pid); got != 0 {
			t.Errorf("ProcessCPU(%d) = %v, want 0", pid, got)
		}
	}
}

func TestForgetProcessDropsTheBaseline(t *testing.T) {
	c := newTestCollector(4)
	c.sample(100, 1000, 100000)
	c.ForgetProcess(100)
	if _, ok := c.lastProc[100]; ok {
		t.Error("baseline survived ForgetProcess")
	}
	// The next sample starts fresh rather than comparing against a dead PID.
	if got := c.sample(100, 50, 101000); got != 0 {
		t.Errorf("sample after ForgetProcess returned %v, want 0", got)
	}
}

// Against the real /proc, the current process must report something sane.
func TestProcessCPUAgainstRealProc(t *testing.T) {
	if _, err := os.Stat("/proc/self/stat"); err != nil {
		t.Skip("/proc unavailable")
	}
	c := NewCollector()
	pid := os.Getpid()
	if got := c.ProcessCPU(pid); got != 0 {
		t.Errorf("first real sample = %v, want 0", got)
	}
	busy := 0
	for i := 0; i < 3_000_000; i++ {
		busy += i % 7
	}
	_ = busy
	got := c.ProcessCPU(pid)
	if got < 0 || got > 100*float64(numCPU()) {
		t.Errorf("real CPU sample %v is outside the plausible range", got)
	}
}
