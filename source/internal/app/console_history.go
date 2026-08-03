package app

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const consoleHistoryLimit = 1000

func (a *App) appendConsoleHistory(line string) {
	a.consoleMu.Lock()
	defer a.consoleMu.Unlock()
	a.consoleHistory = append(a.consoleHistory, line)
	if len(a.consoleHistory) > consoleHistoryLimit {
		a.consoleHistory = append([]string(nil), a.consoleHistory[len(a.consoleHistory)-consoleHistoryLimit:]...)
	}
}

func (a *App) resetConsoleHistory() {
	a.consoleMu.Lock()
	a.consoleHistory = nil
	a.consoleMu.Unlock()
}

func (a *App) consoleHistorySnapshot(limit int) []string {
	a.consoleMu.Lock()
	defer a.consoleMu.Unlock()
	if limit > len(a.consoleHistory) {
		limit = len(a.consoleHistory)
	}
	if limit <= 0 {
		return []string{}
	}
	return append([]string(nil), a.consoleHistory[len(a.consoleHistory)-limit:]...)
}

func (a *App) handleConsoleHistory(w http.ResponseWriter, r *http.Request) {
	limit := consoleHistoryLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > consoleHistoryLimit {
		limit = consoleHistoryLimit
	}

	lines, source := a.consoleHistoryFromLog(limit)
	if len(lines) == 0 {
		lines = a.consoleHistorySnapshot(limit)
		source = "cache"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"lines":  lines,
		"limit":  limit,
		"source": source,
	})
}

func (a *App) consoleHistoryFromLog(limit int) ([]string, string) {
	inst, err := a.activeInstance()
	if err != nil {
		return nil, ""
	}
	logPath := filepath.Join(inst.AbsoluteDir(a.Home), "logs", "bonghos-console.log")
	lines, err := tailLines(logPath, limit)
	if err != nil {
		return nil, ""
	}
	return lines, "log"
}

func tailLines(path string, limit int) ([]string, error) {
	if limit <= 0 {
		return []string{}, nil
	}
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() == 0 {
		return []string{}, nil
	}

	const chunkSize int64 = 32 * 1024
	var data []byte
	offset := info.Size()
	newlines := 0
	for offset > 0 && newlines <= limit {
		n := chunkSize
		if offset < n {
			n = offset
		}
		offset -= n
		chunk := make([]byte, n)
		if _, err := f.ReadAt(chunk, offset); err != nil {
			return nil, err
		}
		newlines += bytes.Count(chunk, []byte{'\n'})
		data = append(chunk, data...)
	}

	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	if lines == nil {
		return []string{}, nil
	}
	return lines, nil
}
