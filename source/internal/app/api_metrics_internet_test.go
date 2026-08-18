package app

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
	"github.com/Chansovisoth/Bonghos/internal/monitoring"
)

func TestInternetMetricsAndManualSpeedTestAPI(t *testing.T) {
	edge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Millisecond)
		switch r.URL.Path {
		case "/down":
			size, _ := strconv.ParseInt(r.URL.Query().Get("bytes"), 10, 64)
			w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
			_, _ = io.CopyN(w, testZeroReader{}, size)
		case "/up":
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer edge.Close()

	env := newTestEnv(t)
	var connectionDials atomic.Int32
	dialer := &net.Dialer{}
	env.app.InternetTester = &monitoring.InternetTester{
		ProbeTargets:  []monitoring.InternetProbeTarget{{Name: "Test edge", URL: edge.URL + "/down?bytes=0"}},
		DownloadURL:   edge.URL + "/down",
		UploadURL:     edge.URL + "/up",
		DownloadSizes: []int64{4 << 10},
		UploadSizes:   []int64{2 << 10},
		Client:        edge.Client(),
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			connectionDials.Add(1)
			return dialer.DialContext(ctx, network, address)
		},
	}
	env.app.InternetMonitor = monitoring.NewInternetMonitor(env.app.InternetTester)
	initialDials := connectionDials.Load()
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	client := env.newClient()
	client.mustLogin("owner", "correct horse battery", secret)

	var snapshot monitoring.InternetSnapshot
	if status, body := client.do("GET", "/api/metrics/internet?interval_seconds=2", nil, &snapshot); status != http.StatusOK {
		t.Fatalf("connectivity status = %d, body=%s", status, body)
	}
	if snapshot.Status != "online" || snapshot.ConnectionSuccessfulTargets != 1 || snapshot.SuccessfulTargets != 1 {
		t.Fatalf("unexpected connectivity payload: %+v", snapshot)
	}
	firstRefreshDials := connectionDials.Load()
	if firstRefreshDials <= initialDials {
		t.Fatalf("first GET did not perform an on-demand TCP probe: dials=%d, initial=%d", firstRefreshDials, initialDials)
	}
	if status, body := client.do("GET", "/api/metrics/internet?interval_seconds=2", nil, &snapshot); status != http.StatusOK {
		t.Fatalf("cached connectivity status = %d, body=%s", status, body)
	}
	if got := connectionDials.Load(); got != firstRefreshDials {
		t.Fatalf("fresh cached GET performed a new TCP probe: dials=%d, want %d", got, firstRefreshDials)
	}
	if status, body := client.do("POST", "/api/metrics/internet/refresh", nil, &snapshot); status != http.StatusOK {
		t.Fatalf("connectivity refresh status = %d, body=%s", status, body)
	}
	if got := connectionDials.Load(); got <= firstRefreshDials {
		t.Fatalf("manual refresh did not perform a TCP probe: dials=%d, cached=%d", got, firstRefreshDials)
	}

	if status, body := client.do("POST", "/api/metrics/internet/speed-test", map[string]any{}, nil); status != http.StatusBadRequest {
		t.Fatalf("unconfirmed speed-test status = %d, body=%s", status, body)
	}

	var result monitoring.InternetSpeedResult
	if status, body := client.do("POST", "/api/metrics/internet/speed-test", map[string]any{"confirm": true}, &result); status != http.StatusOK {
		t.Fatalf("speed-test status = %d, body=%s", status, body)
	}
	if result.DownloadMbps <= 0 || result.UploadMbps <= 0 || result.DownloadBytes != 4<<10 || result.UploadBytes != 2<<10 {
		t.Fatalf("unexpected speed-test payload: %+v", result)
	}
}

func TestRequestedInternetInterval(t *testing.T) {
	tests := []struct {
		query string
		want  time.Duration
	}{
		{query: "", want: 2 * time.Second},
		{query: "?interval_seconds=1", want: time.Second},
		{query: "?interval_seconds=60", want: time.Minute},
		{query: "?interval_seconds=4", want: 2 * time.Second},
		{query: "?interval_seconds=invalid", want: 2 * time.Second},
	}
	for _, test := range tests {
		req := httptest.NewRequest(http.MethodGet, "/api/metrics/internet"+test.query, nil)
		if got := requestedInternetInterval(req); got != test.want {
			t.Errorf("requestedInternetInterval(%q) = %s, want %s", test.query, got, test.want)
		}
	}
}

type testZeroReader struct{}

func (testZeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}
