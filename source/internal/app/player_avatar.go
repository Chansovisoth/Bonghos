package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const maxPlayerAvatarBytes = 512 * 1024

var (
	playerNameRE        = regexp.MustCompile(`^[A-Za-z0-9_]{1,16}$`)
	playerAvatarBaseURL = "https://minotar.net/helm"
	playerAvatarClient  = &http.Client{Timeout: 5 * time.Second}
)

func (a *App) handlePlayerAvatar(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" {
		username = "MHF_Steve"
	}
	if !playerNameRE.MatchString(username) {
		writeErr(w, http.StatusBadRequest, errors.New("invalid player username"))
		return
	}
	size := playerAvatarSize(r.URL.Query().Get("size"))
	body, contentType, err := fetchPlayerAvatar(r.Context(), username, size)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func playerAvatarSize(raw string) int {
	size := 64
	if parsed, err := strconv.Atoi(raw); err == nil {
		size = parsed
	}
	if size < 16 {
		return 16
	}
	if size > 128 {
		return 128
	}
	return size
}

func fetchPlayerAvatar(ctx context.Context, username string, size int) ([]byte, string, error) {
	endpoint := fmt.Sprintf("%s/%s/%d.png", strings.TrimRight(playerAvatarBaseURL, "/"), url.PathEscape(username), size)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Bonghos/1.0 (+https://github.com/Chansovisoth/Bonghos)")
	resp, err := playerAvatarClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("player avatar service returned %s", resp.Status)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return nil, "", errors.New("player avatar service returned a non-image response")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPlayerAvatarBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(body) > maxPlayerAvatarBytes {
		return nil, "", errors.New("player avatar response too large")
	}
	return body, contentType, nil
}
