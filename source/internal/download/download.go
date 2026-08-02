// Package download implements the SSRF-protected server-side archive
// downloader. The Linux host performs the download; the browser only creates
// the operation and monitors progress.
package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var (
	ErrScheme         = errors.New("only https:// URLs are allowed (http:// requires an explicit Owner setting)")
	ErrCredentials    = errors.New("URLs with embedded usernames or passwords are rejected")
	ErrBlockedAddr    = errors.New("destination address is blocked (private, loopback, link-local or metadata range)")
	ErrTooManyHops    = errors.New("too many redirects")
	ErrTooLarge       = errors.New("download exceeds the maximum archive size")
	ErrDiskSpace      = errors.New("not enough free disk space for this download")
	ErrHostNotAllowed = errors.New("host is not on the trusted download allowlist")
)

// Options configures one download.
type Options struct {
	AllowInsecureHTTP bool
	TrustedHosts      []string // empty = any public host
	MaxBytes          int64
	FreeSpaceReserve  int64 // bytes to keep free on the target filesystem
	ConnectTimeout    time.Duration
	HeaderTimeout     time.Duration
	IdleTimeout       time.Duration
	TotalTimeout      time.Duration
	MaxRedirects      int
}

func DefaultOptions() Options {
	return Options{
		MaxBytes:         64 << 30,
		FreeSpaceReserve: 1 << 30,
		ConnectTimeout:   15 * time.Second,
		HeaderTimeout:    30 * time.Second,
		IdleTimeout:      60 * time.Second,
		TotalTimeout:     6 * time.Hour,
		MaxRedirects:     5,
	}
}

// ValidateURL performs the static checks (scheme, credentials, allowlist).
func ValidateURL(raw string, o Options) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	switch u.Scheme {
	case "https":
	case "http":
		if !o.AllowInsecureHTTP {
			return nil, ErrScheme
		}
	case "file", "ftp", "ssh", "data", "javascript", "gopher":
		return nil, fmt.Errorf("%w (scheme %q rejected)", ErrScheme, u.Scheme)
	default:
		return nil, fmt.Errorf("%w (scheme %q rejected)", ErrScheme, u.Scheme)
	}
	if u.User != nil {
		return nil, ErrCredentials
	}
	if u.Hostname() == "" {
		return nil, errors.New("URL has no host")
	}
	if len(o.TrustedHosts) > 0 {
		ok := false
		for _, h := range o.TrustedHosts {
			if strings.EqualFold(h, u.Hostname()) ||
				strings.HasSuffix(strings.ToLower(u.Hostname()), "."+strings.ToLower(h)) {
				ok = true
				break
			}
		}
		if !ok {
			return nil, ErrHostNotAllowed
		}
	}
	return u, nil
}

// blockedIP reports whether an address must never be contacted.
func blockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	// Cloud metadata endpoints and other special ranges.
	blocked := []string{
		"169.254.0.0/16", // link-local incl. 169.254.169.254 metadata
		"100.64.0.0/10",  // CGNAT
		"192.0.0.0/24",   // IETF protocol assignments
		"198.18.0.0/15",  // benchmarking
		"0.0.0.0/8",
		"fd00::/8", "fc00::/7", // unique local
		"::1/128",
	}
	for _, cidr := range blocked {
		if _, n, err := net.ParseCIDR(cidr); err == nil && n.Contains(ip) {
			return true
		}
	}
	return false
}

// safeDialer resolves and vets the destination address at connect time.
// This closes DNS-rebinding gaps: the checked IP is the connected IP.
func safeDialer(o Options) *net.Dialer {
	return &net.Dialer{
		Timeout: o.ConnectTimeout,
		Control: func(network, address string, c syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if blockedIP(ip) {
				return ErrBlockedAddr
			}
			return nil
		},
	}
}

// client builds an HTTP client that revalidates every redirect hop.
func client(o Options) *http.Client {
	transport := &http.Transport{
		DialContext:           safeDialer(o).DialContext,
		ResponseHeaderTimeout: o.HeaderTimeout,
		IdleConnTimeout:       o.IdleTimeout,
		DisableCompression:    true,
		Proxy:                 nil, // never use environment proxies for pack downloads
	}
	return &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= o.MaxRedirects {
				return ErrTooManyHops
			}
			if _, err := ValidateURL(req.URL.String(), o); err != nil {
				return err
			}
			// never forward Authorization to a different host
			if len(via) > 0 && req.URL.Host != via[0].URL.Host {
				req.Header.Del("Authorization")
				req.Header.Del("Cookie")
			}
			return nil
		},
	}
}

// Result describes a completed download.
type Result struct {
	Path          string
	Bytes         int64
	FinalHost     string
	ContentLength int64 // -1 when the source did not provide a total size
}

// RedactURL removes sensitive query parameters for logging.
func RedactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "(unparseable url)"
	}
	q := u.Query()
	for k := range q {
		lk := strings.ToLower(k)
		if strings.Contains(lk, "token") || strings.Contains(lk, "key") ||
			strings.Contains(lk, "secret") || strings.Contains(lk, "sig") ||
			strings.Contains(lk, "auth") || strings.Contains(lk, "password") {
			q.Set(k, "REDACTED")
		}
	}
	u.RawQuery = q.Encode()
	u.User = nil
	return u.String()
}

// Fetch streams the URL to destDir/download.partial then renames it to
// download.complete. progress receives (downloaded, total|-1).
func Fetch(ctx context.Context, rawURL, destDir string, o Options,
	progress func(done, total int64)) (*Result, error) {

	u, err := ValidateURL(rawURL, o)
	if err != nil {
		return nil, err
	}
	if o.TotalTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.TotalTimeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Bonghos/1.0 (+https://github.com/Chansovisoth/Bonghos)")

	resp, err := client(o).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote returned %s", resp.Status)
	}

	total := resp.ContentLength // may be -1
	if total > 0 && o.MaxBytes > 0 && total > o.MaxBytes {
		return nil, ErrTooLarge
	}
	if err := checkSpace(destDir, total, o); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, err
	}
	partial := filepath.Join(destDir, "download.partial")
	f, err := os.OpenFile(partial, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}

	var done int64
	buf := make([]byte, 1<<20)
	lastSpaceCheck := time.Now()
	for {
		select {
		case <-ctx.Done():
			f.Close()
			os.Remove(partial)
			return nil, ctx.Err()
		default:
		}
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if o.MaxBytes > 0 && done+int64(n) > o.MaxBytes {
				f.Close()
				os.Remove(partial)
				return nil, ErrTooLarge
			}
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				os.Remove(partial)
				return nil, werr
			}
			done += int64(n)
			if progress != nil {
				progress(done, total)
			}
			if time.Since(lastSpaceCheck) > 10*time.Second {
				lastSpaceCheck = time.Now()
				if err := checkSpace(destDir, 0, o); err != nil {
					f.Close()
					os.Remove(partial)
					return nil, err
				}
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			os.Remove(partial)
			return nil, rerr
		}
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	if total > 0 && done != total {
		os.Remove(partial)
		return nil, fmt.Errorf("incomplete download: got %d of %d bytes", done, total)
	}
	final := filepath.Join(destDir, "download.complete")
	if err := os.Rename(partial, final); err != nil {
		return nil, err
	}
	finalHost := u.Hostname()
	if resp.Request != nil && resp.Request.URL != nil {
		finalHost = resp.Request.URL.Hostname()
	}
	return &Result{Path: final, Bytes: done, FinalHost: finalHost, ContentLength: total}, nil
}

func checkSpace(dir string, incoming int64, o Options) error {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return nil // cannot determine; do not block
	}
	free := int64(st.Bavail) * st.Bsize
	need := o.FreeSpaceReserve
	if incoming > 0 {
		need += incoming
	}
	if free < need {
		return ErrDiskSpace
	}
	return nil
}
