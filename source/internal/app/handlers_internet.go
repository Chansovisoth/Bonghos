package app

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/monitoring"
)

func (a *App) internetTester() *monitoring.InternetTester {
	if a.InternetTester != nil {
		return a.InternetTester
	}
	return monitoring.NewInternetTester()
}

func (a *App) internetMonitor() *monitoring.InternetMonitor {
	if a.InternetMonitor != nil {
		return a.InternetMonitor
	}
	a.InternetMonitor = monitoring.NewInternetMonitor(a.internetTester())
	return a.InternetMonitor
}

func (a *App) handleMetricsInternet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.internetMonitor().RefreshIfStale(r.Context(), requestedInternetInterval(r)))
}

func requestedInternetInterval(r *http.Request) time.Duration {
	seconds, err := strconv.Atoi(r.URL.Query().Get("interval_seconds"))
	if err != nil {
		seconds = 2
	}
	switch seconds {
	case 1, 2, 3, 5, 10, 30, 60:
	default:
		seconds = 2
	}
	return time.Duration(seconds) * time.Second
}

func (a *App) handleRefreshMetricsInternet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.internetMonitor().Refresh(r.Context()))
}

func (a *App) handleInternetSpeedTest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if err := readJSON(r, &req, 1<<10); err != nil || !req.Confirm {
		writeErr(w, http.StatusBadRequest, errors.New("Internet speed test requires explicit confirmation"))
		return
	}
	if !a.internetSpeedMu.TryLock() {
		writeErr(w, http.StatusConflict, errors.New("an Internet speed test is already running"))
		return
	}
	defer a.internetSpeedMu.Unlock()

	result, err := a.internetTester().SpeedTest(r.Context())
	if err != nil {
		if a.Logf != nil {
			a.Logf("Internet speed test failed: %v", err)
		}
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	u := currentUser(r)
	a.audit(u.ID, u.Username, "internet_speed_test", result.Provider,
		fmt.Sprintf("download=%.1fMbps upload=%.1fMbps", result.DownloadMbps, result.UploadMbps), remoteIP(r))
	writeJSON(w, http.StatusOK, result)
}
