package monitoring

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	cloudflareDownloadURL = "https://speed.cloudflare.com/__down"
	cloudflareUploadURL   = "https://speed.cloudflare.com/__up"
)

// InternetProbeTarget is a small HTTPS resource used to establish whether the
// Bonghos host can reach the public Internet. Targets are fixed by the build;
// accepting arbitrary URLs here would turn the authenticated API into an SSRF
// primitive.
type InternetProbeTarget struct {
	Name string
	URL  string
}

type InternetTargetResult struct {
	Name      string  `json:"name"`
	Reachable bool    `json:"reachable"`
	LatencyMS float64 `json:"latency_ms,omitempty"`
	Error     string  `json:"error,omitempty"`
}

// InternetSnapshot is the cached connectivity and diagnostic reading exposed
// to the Web UI. It is deliberately separate from Sample so remote networking
// can never stall the process/host metrics loop.
type InternetSnapshot struct {
	CollectedAt                 string                 `json:"collected_at"`
	Status                      string                 `json:"status"`
	ConnectionLatencyMS         float64                `json:"connection_latency_ms,omitempty"`
	ConnectionSuccessfulTargets int                    `json:"connection_successful_targets"`
	ConnectionTotalTargets      int                    `json:"connection_total_targets"`
	ConnectionTargets           []InternetTargetResult `json:"connection_targets"`
	ConsecutiveFailures         int                    `json:"consecutive_failures"`
	ReliabilitySuccessful       int                    `json:"reliability_successful"`
	ReliabilityTotal            int                    `json:"reliability_total"`
	DiagnosticsCollectedAt      string                 `json:"diagnostics_collected_at,omitempty"`
	DNSOK                       bool                   `json:"dns_ok"`
	DNSMS                       float64                `json:"dns_ms,omitempty"`
	LatencyMS                   float64                `json:"latency_ms,omitempty"`
	SuccessfulTargets           int                    `json:"successful_targets"`
	TotalTargets                int                    `json:"total_targets"`
	Targets                     []InternetTargetResult `json:"targets"`
}

type InternetConnectionSnapshot struct {
	CollectedAt       string
	LatencyMS         float64
	SuccessfulTargets int
	TotalTargets      int
	Targets           []InternetTargetResult
}

type InternetSpeedResult struct {
	TestedAt             string  `json:"tested_at"`
	Provider             string  `json:"provider"`
	LatencyMS            float64 `json:"latency_ms"`
	DownloadMbps         float64 `json:"download_mbps"`
	UploadMbps           float64 `json:"upload_mbps"`
	DownloadBytes        int64   `json:"download_bytes"`
	UploadBytes          int64   `json:"upload_bytes"`
	DurationMilliseconds int64   `json:"duration_ms"`
}

// InternetTester owns the fixed remote targets and transfer sizes. The fields
// are exported so tests can point it at a local httptest server without making
// real Internet requests.
type InternetTester struct {
	ProbeTargets  []InternetProbeTarget
	DownloadURL   string
	UploadURL     string
	DownloadSizes []int64
	UploadSizes   []int64
	Resolver      *net.Resolver
	Client        *http.Client
	DialContext   func(context.Context, string, string) (net.Conn, error)
}

func NewInternetTester() *InternetTester {
	dialer := &net.Dialer{Timeout: 1500 * time.Millisecond, KeepAlive: -1}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 8 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		DisableCompression:    true,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
	}
	return &InternetTester{
		ProbeTargets: []InternetProbeTarget{
			{Name: "Cloudflare", URL: cloudflareDownloadURL + "?bytes=0"},
			{Name: "Google", URL: "https://connectivitycheck.gstatic.com/generate_204"},
		},
		DownloadURL: cloudflareDownloadURL,
		UploadURL:   cloudflareUploadURL,
		// Roughly 53 MB total at most. The largest completed transfer is used
		// for the displayed estimate; smaller rounds warm up the connection.
		DownloadSizes: []int64{2 << 20, 10 << 20, 25 << 20},
		UploadSizes:   []int64{1 << 20, 5 << 20, 10 << 20},
		Resolver:      net.DefaultResolver,
		Client:        &http.Client{Transport: transport, Timeout: 45 * time.Second},
		DialContext:   dialer.DialContext,
	}
}

func (t *InternetTester) resolver() *net.Resolver {
	if t != nil && t.Resolver != nil {
		return t.Resolver
	}
	return net.DefaultResolver
}

func (t *InternetTester) client() *http.Client {
	if t != nil && t.Client != nil {
		return t.Client
	}
	return NewInternetTester().Client
}

func (t *InternetTester) dialContext() func(context.Context, string, string) (net.Conn, error) {
	if t != nil && t.DialContext != nil {
		return t.DialContext
	}
	dialer := &net.Dialer{Timeout: 1500 * time.Millisecond, KeepAlive: -1}
	return dialer.DialContext
}

func (t *InternetTester) probeTargets() []InternetProbeTarget {
	if t != nil && len(t.ProbeTargets) > 0 {
		return append([]InternetProbeTarget(nil), t.ProbeTargets...)
	}
	return NewInternetTester().ProbeTargets
}

// CheckConnection performs tiny TCP connection probes. Unlike Check, it does
// not perform TLS handshakes or transfer HTTP response bodies, so it can run at
// a game-dashboard cadence without multiplying heavier diagnostic traffic.
func (t *InternetTester) CheckConnection(ctx context.Context) InternetConnectionSnapshot {
	ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	targets := t.probeTargets()
	snapshot := InternetConnectionSnapshot{
		CollectedAt:  time.Now().UTC().Format(time.RFC3339Nano),
		TotalTargets: len(targets),
		Targets:      make([]InternetTargetResult, len(targets)),
	}
	type indexedResult struct {
		index  int
		result InternetTargetResult
	}
	results := make(chan indexedResult, len(targets))
	for index, target := range targets {
		go func() {
			results <- indexedResult{index: index, result: t.checkConnectionTarget(ctx, target)}
		}()
	}
	var latencyTotal float64
	for range targets {
		entry := <-results
		snapshot.Targets[entry.index] = entry.result
		if entry.result.Reachable {
			snapshot.SuccessfulTargets++
			latencyTotal += entry.result.LatencyMS
		}
	}
	if snapshot.SuccessfulTargets > 0 {
		snapshot.LatencyMS = latencyTotal / float64(snapshot.SuccessfulTargets)
	}
	return snapshot
}

func (t *InternetTester) checkConnectionTarget(ctx context.Context, target InternetProbeTarget) InternetTargetResult {
	result := InternetTargetResult{Name: target.Name}
	parsed, err := url.Parse(target.URL)
	if err != nil || parsed.Hostname() == "" {
		result.Error = "invalid_target"
		return result
	}
	port := parsed.Port()
	if port == "" {
		switch parsed.Scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		default:
			result.Error = "invalid_target"
			return result
		}
	}
	started := time.Now()
	connection, err := t.dialContext()(ctx, "tcp", net.JoinHostPort(parsed.Hostname(), port))
	result.LatencyMS = milliseconds(time.Since(started))
	if err != nil {
		result.Error = publicNetworkError(err)
		return result
	}
	_ = connection.Close()
	result.Reachable = true
	return result
}

func (t *InternetTester) Check(ctx context.Context) InternetSnapshot {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	targets := t.probeTargets()
	snapshot := InternetSnapshot{
		CollectedAt:            time.Now().UTC().Format(time.RFC3339Nano),
		DiagnosticsCollectedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Status:                 "offline",
		TotalTargets:           len(targets),
		Targets:                make([]InternetTargetResult, len(targets)),
	}

	if len(targets) > 0 {
		if parsed, err := url.Parse(targets[0].URL); err == nil && parsed.Hostname() != "" {
			started := time.Now()
			_, lookupErr := t.resolver().LookupHost(ctx, parsed.Hostname())
			snapshot.DNSMS = milliseconds(time.Since(started))
			snapshot.DNSOK = lookupErr == nil
		}
	}

	type indexedResult struct {
		index  int
		result InternetTargetResult
	}
	results := make(chan indexedResult, len(targets))
	for index, target := range targets {
		go func() {
			results <- indexedResult{index: index, result: t.checkTarget(ctx, target)}
		}()
	}
	var latencyTotal float64
	for range targets {
		entry := <-results
		snapshot.Targets[entry.index] = entry.result
		if entry.result.Reachable {
			snapshot.SuccessfulTargets++
			latencyTotal += entry.result.LatencyMS
		}
	}
	if snapshot.SuccessfulTargets > 0 {
		snapshot.LatencyMS = latencyTotal / float64(snapshot.SuccessfulTargets)
	}
	switch {
	case snapshot.TotalTargets > 0 && snapshot.SuccessfulTargets == snapshot.TotalTargets:
		snapshot.Status = "online"
	case snapshot.SuccessfulTargets > 0:
		snapshot.Status = "degraded"
	}
	return snapshot
}

func (t *InternetTester) checkTarget(ctx context.Context, target InternetProbeTarget) InternetTargetResult {
	result := InternetTargetResult{Name: target.Name}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.URL, nil)
	if err != nil {
		result.Error = "invalid_target"
		return result
	}
	setProbeHeaders(req)
	started := time.Now()
	resp, err := t.client().Do(req)
	if err != nil {
		result.Error = publicNetworkError(err)
		return result
	}
	defer resp.Body.Close()
	_, readErr := io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	result.LatencyMS = milliseconds(time.Since(started))
	if readErr != nil {
		result.Error = publicNetworkError(readErr)
		return result
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		result.Error = "http_" + strconv.Itoa(resp.StatusCode)
		return result
	}
	result.Reachable = true
	return result
}

const (
	DefaultInternetOfflineAfter = 3
	internetReliabilityWindow   = 10
)

// InternetMonitor owns one shared snapshot for the whole Bonghos process.
// Performance-page requests refresh it only when stale, so opening more
// dashboards never multiplies outbound connection, DNS, or HTTPS traffic.
type InternetMonitor struct {
	Tester       *InternetTester
	OfflineAfter int

	mu            sync.RWMutex
	snapshot      InternetSnapshot
	lastRefresh   time.Time
	failures      int
	reliability   []bool
	refreshMu     sync.Mutex
	quickRunMu    sync.Mutex
	diagnosticsMu sync.Mutex
}

func NewInternetMonitor(tester *InternetTester) *InternetMonitor {
	if tester == nil {
		tester = NewInternetTester()
	}
	return &InternetMonitor{
		Tester:       tester,
		OfflineAfter: DefaultInternetOfflineAfter,
		snapshot: InternetSnapshot{
			Status:                 "checking",
			ConnectionTotalTargets: len(tester.probeTargets()),
		},
	}
}

func (m *InternetMonitor) Refresh(ctx context.Context) InternetSnapshot {
	m.refreshMu.Lock()
	defer m.refreshMu.Unlock()
	return m.refresh(ctx)
}

func (m *InternetMonitor) RefreshIfStale(ctx context.Context, maxAge time.Duration) InternetSnapshot {
	if maxAge < time.Second {
		maxAge = time.Second
	}
	if m.fresh(maxAge) {
		return m.Snapshot()
	}
	m.refreshMu.Lock()
	defer m.refreshMu.Unlock()
	if m.fresh(maxAge) {
		return m.Snapshot()
	}
	return m.refresh(ctx)
}

func (m *InternetMonitor) fresh(maxAge time.Duration) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return !m.lastRefresh.IsZero() && time.Since(m.lastRefresh) < maxAge
}

func (m *InternetMonitor) refresh(ctx context.Context) InternetSnapshot {
	var checks sync.WaitGroup
	checks.Add(2)
	go func() {
		defer checks.Done()
		m.RefreshConnection(ctx)
	}()
	go func() {
		defer checks.Done()
		m.RefreshDiagnostics(ctx)
	}()
	checks.Wait()
	if ctx.Err() == nil {
		m.mu.Lock()
		m.lastRefresh = time.Now()
		m.mu.Unlock()
	}
	return m.Snapshot()
}

func (m *InternetMonitor) RefreshConnection(ctx context.Context) {
	m.quickRunMu.Lock()
	connection := m.Tester.CheckConnection(ctx)
	m.quickRunMu.Unlock()
	if ctx.Err() != nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshot.CollectedAt = connection.CollectedAt
	m.snapshot.ConnectionLatencyMS = connection.LatencyMS
	m.snapshot.ConnectionSuccessfulTargets = connection.SuccessfulTargets
	m.snapshot.ConnectionTotalTargets = connection.TotalTargets
	m.snapshot.ConnectionTargets = append([]InternetTargetResult(nil), connection.Targets...)
	reachable := connection.SuccessfulTargets > 0
	m.reliability = append(m.reliability, reachable)
	if len(m.reliability) > internetReliabilityWindow {
		m.reliability = append([]bool(nil), m.reliability[len(m.reliability)-internetReliabilityWindow:]...)
	}
	m.snapshot.ReliabilitySuccessful = 0
	for _, successful := range m.reliability {
		if successful {
			m.snapshot.ReliabilitySuccessful++
		}
	}
	m.snapshot.ReliabilityTotal = len(m.reliability)

	switch {
	case connection.TotalTargets > 0 && connection.SuccessfulTargets == connection.TotalTargets:
		m.failures = 0
		m.snapshot.Status = "online"
	case reachable:
		m.failures = 0
		m.snapshot.Status = "degraded"
	default:
		m.failures++
		offlineAfter := m.OfflineAfter
		if offlineAfter <= 0 {
			offlineAfter = DefaultInternetOfflineAfter
		}
		if m.failures >= offlineAfter {
			m.snapshot.Status = "offline"
		} else {
			m.snapshot.Status = "degraded"
		}
	}
	m.snapshot.ConsecutiveFailures = m.failures
}

func (m *InternetMonitor) RefreshDiagnostics(ctx context.Context) {
	m.diagnosticsMu.Lock()
	diagnostics := m.Tester.Check(ctx)
	m.diagnosticsMu.Unlock()
	if ctx.Err() != nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.snapshot.CollectedAt == "" {
		m.snapshot.CollectedAt = diagnostics.CollectedAt
	}
	m.snapshot.DiagnosticsCollectedAt = diagnostics.DiagnosticsCollectedAt
	m.snapshot.DNSOK = diagnostics.DNSOK
	m.snapshot.DNSMS = diagnostics.DNSMS
	m.snapshot.LatencyMS = diagnostics.LatencyMS
	m.snapshot.SuccessfulTargets = diagnostics.SuccessfulTargets
	m.snapshot.TotalTargets = diagnostics.TotalTargets
	m.snapshot.Targets = append([]InternetTargetResult(nil), diagnostics.Targets...)
}

func (m *InternetMonitor) Snapshot() InternetSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	snapshot := m.snapshot
	snapshot.ConnectionTargets = append([]InternetTargetResult(nil), m.snapshot.ConnectionTargets...)
	snapshot.Targets = append([]InternetTargetResult(nil), m.snapshot.Targets...)
	return snapshot
}

func (t *InternetTester) SpeedTest(ctx context.Context) (InternetSpeedResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	started := time.Now()
	result := InternetSpeedResult{TestedAt: time.Now().UTC().Format(time.RFC3339), Provider: "Cloudflare"}

	latencyTarget := InternetProbeTarget{Name: "Cloudflare", URL: withByteCount(t.downloadURL(), 0)}
	latency := t.checkTarget(ctx, latencyTarget)
	if !latency.Reachable {
		return result, fmt.Errorf("connectivity check failed: %s", humanNetworkError(latency.Error))
	}
	result.LatencyMS = latency.LatencyMS

	var lastDuration time.Duration
	for _, size := range t.downloadSizes() {
		duration, err := t.measureDownload(ctx, size)
		if err != nil {
			return result, fmt.Errorf("download test failed: %s", humanNetworkError(publicNetworkError(err)))
		}
		result.DownloadBytes += size
		lastDuration = duration
		result.DownloadMbps = megabitsPerSecond(size, duration)
	}
	if lastDuration <= 0 {
		return result, errors.New("download test produced no measurement")
	}

	for _, size := range t.uploadSizes() {
		duration, err := t.measureUpload(ctx, size)
		if err != nil {
			return result, fmt.Errorf("upload test failed: %s", humanNetworkError(publicNetworkError(err)))
		}
		result.UploadBytes += size
		result.UploadMbps = megabitsPerSecond(size, duration)
	}
	result.DurationMilliseconds = time.Since(started).Milliseconds()
	return result, nil
}

func (t *InternetTester) measureDownload(ctx context.Context, size int64) (time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, withByteCount(t.downloadURL(), size), nil)
	if err != nil {
		return 0, err
	}
	setProbeHeaders(req)
	started := time.Now()
	resp, err := t.client().Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	written, err := io.Copy(io.Discard, resp.Body)
	duration := time.Since(started)
	if err != nil {
		return 0, err
	}
	if written < size {
		return 0, errors.New("download response was incomplete")
	}
	return duration, nil
}

func (t *InternetTester) measureUpload(ctx context.Context, size int64) (time.Duration, error) {
	body := io.LimitReader(zeroReader{}, size)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.uploadURL(), body)
	if err != nil {
		return 0, err
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", "application/octet-stream")
	setProbeHeaders(req)
	started := time.Now()
	resp, err := t.client().Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, readErr := io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	duration := time.Since(started)
	if readErr != nil {
		return 0, readErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return duration, nil
}

func (t *InternetTester) downloadURL() string {
	if t != nil && strings.TrimSpace(t.DownloadURL) != "" {
		return t.DownloadURL
	}
	return cloudflareDownloadURL
}

func (t *InternetTester) uploadURL() string {
	if t != nil && strings.TrimSpace(t.UploadURL) != "" {
		return t.UploadURL
	}
	return cloudflareUploadURL
}

func (t *InternetTester) downloadSizes() []int64 {
	if t != nil && len(t.DownloadSizes) > 0 {
		return append([]int64(nil), t.DownloadSizes...)
	}
	return NewInternetTester().DownloadSizes
}

func (t *InternetTester) uploadSizes() []int64 {
	if t != nil && len(t.UploadSizes) > 0 {
		return append([]int64(nil), t.UploadSizes...)
	}
	return NewInternetTester().UploadSizes
}

func withByteCount(raw string, count int64) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	query := parsed.Query()
	query.Set("bytes", strconv.FormatInt(count, 10))
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func setProbeHeaders(req *http.Request) {
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Cache-Control", "no-store")
	req.Header.Set("User-Agent", "Bonghos internet monitor")
}

func milliseconds(duration time.Duration) float64 {
	return float64(duration.Microseconds()) / 1000
}

func megabitsPerSecond(bytes int64, duration time.Duration) float64 {
	if bytes <= 0 || duration <= 0 {
		return 0
	}
	return float64(bytes*8) / duration.Seconds() / 1_000_000
}

func publicNetworkError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "dns_failed"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return publicNetworkError(urlErr.Err)
	}
	return "connection_failed"
}

func humanNetworkError(code string) string {
	switch code {
	case "timeout":
		return "the request timed out"
	case "dns_failed":
		return "DNS lookup failed"
	case "connection_failed":
		return "the remote service could not be reached"
	case "invalid_target":
		return "the test target is invalid"
	default:
		if strings.HasPrefix(code, "http_") {
			return "the remote service returned " + strings.TrimPrefix(code, "http_")
		}
		return "the measurement could not be completed"
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}
