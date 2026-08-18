package monitoring

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestInternetTesterChecksConnectivityAndMeasuresTransfers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Millisecond)
		switch r.URL.Path {
		case "/down":
			size, _ := strconv.ParseInt(r.URL.Query().Get("bytes"), 10, 64)
			w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
			_, _ = io.CopyN(w, zeroReader{}, size)
		case "/up":
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tester := &InternetTester{
		ProbeTargets:  []InternetProbeTarget{{Name: "Local edge", URL: server.URL + "/down?bytes=0"}},
		DownloadURL:   server.URL + "/down",
		UploadURL:     server.URL + "/up",
		DownloadSizes: []int64{8 << 10, 16 << 10},
		UploadSizes:   []int64{4 << 10, 8 << 10},
		Client:        server.Client(),
	}

	snapshot := tester.Check(context.Background())
	if snapshot.Status != "online" || !snapshot.DNSOK || snapshot.SuccessfulTargets != 1 || snapshot.TotalTargets != 1 {
		t.Fatalf("unexpected connectivity snapshot: %+v", snapshot)
	}
	if len(snapshot.Targets) != 1 || !snapshot.Targets[0].Reachable || snapshot.Targets[0].LatencyMS < 0 {
		t.Fatalf("unexpected target result: %+v", snapshot.Targets)
	}
	connection := tester.CheckConnection(context.Background())
	if connection.SuccessfulTargets != 1 || connection.TotalTargets != 1 || connection.LatencyMS < 0 {
		t.Fatalf("unexpected TCP connection snapshot: %+v", connection)
	}

	result, err := tester.SpeedTest(context.Background())
	if err != nil {
		t.Fatalf("SpeedTest: %v", err)
	}
	if result.DownloadMbps <= 0 || result.UploadMbps <= 0 {
		t.Fatalf("non-positive speed result: %+v", result)
	}
	if result.DownloadBytes != 24<<10 || result.UploadBytes != 12<<10 {
		t.Fatalf("unexpected transfer totals: %+v", result)
	}
	if result.Provider != "Cloudflare" || result.TestedAt == "" {
		t.Fatalf("missing result metadata: %+v", result)
	}
}

func TestInternetMonitorSmoothsCompleteFailuresAndRecovers(t *testing.T) {
	var failing atomic.Bool
	failing.Store(true)
	tester := &InternetTester{
		ProbeTargets: []InternetProbeTarget{{Name: "Test edge", URL: "https://edge.test/probe"}},
		DialContext: func(context.Context, string, string) (net.Conn, error) {
			if failing.Load() {
				return nil, errors.New("test connection failure")
			}
			client, server := net.Pipe()
			_ = server.Close()
			return client, nil
		},
	}
	monitor := NewInternetMonitor(tester)
	for attempt := 1; attempt <= 2; attempt++ {
		monitor.RefreshConnection(context.Background())
		snapshot := monitor.Snapshot()
		if snapshot.Status != "degraded" || snapshot.ConsecutiveFailures != attempt {
			t.Fatalf("attempt %d snapshot = %+v, want degraded", attempt, snapshot)
		}
	}
	monitor.RefreshConnection(context.Background())
	if snapshot := monitor.Snapshot(); snapshot.Status != "offline" || snapshot.ConsecutiveFailures != 3 {
		t.Fatalf("third failed snapshot = %+v, want offline", snapshot)
	}

	failing.Store(false)
	monitor.RefreshConnection(context.Background())
	snapshot := monitor.Snapshot()
	if snapshot.Status != "online" || snapshot.ConsecutiveFailures != 0 {
		t.Fatalf("recovered snapshot = %+v, want online", snapshot)
	}
	if snapshot.ReliabilitySuccessful != 1 || snapshot.ReliabilityTotal != 4 {
		t.Fatalf("reliability = %d/%d, want 1/4", snapshot.ReliabilitySuccessful, snapshot.ReliabilityTotal)
	}
}

func TestInternetMonitorRefreshesSharedSnapshotOnlyWhenStale(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	var connectionDials atomic.Int32
	dialer := &net.Dialer{}
	tester := &InternetTester{
		ProbeTargets: []InternetProbeTarget{{Name: "Local edge", URL: server.URL}},
		Client:       server.Client(),
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			connectionDials.Add(1)
			return dialer.DialContext(ctx, network, address)
		},
	}
	monitor := NewInternetMonitor(tester)

	first := monitor.RefreshIfStale(context.Background(), time.Minute)
	if first.Status != "online" || connectionDials.Load() != 1 {
		t.Fatalf("first refresh = %+v, dials=%d", first, connectionDials.Load())
	}
	second := monitor.RefreshIfStale(context.Background(), time.Minute)
	if second.CollectedAt != first.CollectedAt || connectionDials.Load() != 1 {
		t.Fatalf("fresh snapshot was not reused: first=%q second=%q dials=%d", first.CollectedAt, second.CollectedAt, connectionDials.Load())
	}
	monitor.Refresh(context.Background())
	if connectionDials.Load() != 2 {
		t.Fatalf("forced refresh dials=%d, want 2", connectionDials.Load())
	}
}

func TestInternetMonitorDoesNotCacheCancelledRefresh(t *testing.T) {
	tester := &InternetTester{
		ProbeTargets: []InternetProbeTarget{{Name: "Cancelled", URL: "https://edge.test/probe"}},
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return nil, ctx.Err()
		},
	}
	monitor := NewInternetMonitor(tester)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	snapshot := monitor.Refresh(ctx)
	if snapshot.Status != "checking" || snapshot.CollectedAt != "" || monitor.fresh(time.Minute) {
		t.Fatalf("cancelled refresh changed or cached the shared snapshot: %+v", snapshot)
	}
}

func TestInternetTesterReportsDegradedWithoutLeakingNetworkErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ok" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "provider detail that must not be exposed", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	tester := &InternetTester{
		ProbeTargets: []InternetProbeTarget{
			{Name: "Available", URL: server.URL + "/ok"},
			{Name: "Blocked", URL: server.URL + "/blocked"},
		},
		Client: server.Client(),
	}
	snapshot := tester.Check(context.Background())
	if snapshot.Status != "degraded" || snapshot.SuccessfulTargets != 1 {
		t.Fatalf("unexpected degraded snapshot: %+v", snapshot)
	}
	if got := snapshot.Targets[1].Error; got != "http_503" {
		t.Fatalf("public error = %q, want http_503", got)
	}
	if snapshot.Targets[1].Reachable {
		t.Fatal("failed target was marked reachable")
	}
}

func TestInternetTesterHonorsCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tester := &InternetTester{
		ProbeTargets: []InternetProbeTarget{{Name: "Cancelled", URL: server.URL}},
		Client:       server.Client(),
	}
	snapshot := tester.Check(ctx)
	if snapshot.Status != "offline" || snapshot.Targets[0].Error != "connection_failed" {
		t.Fatalf("unexpected cancelled snapshot: %+v", snapshot)
	}
}
