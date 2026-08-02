package minecraft

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ----- console log parsing ---------------------------------------------------

var (
	// [12:34:56] [Server thread/INFO]: Alice joined the game
	reJoined = regexp.MustCompile(`\]:?\s+([A-Za-z0-9_]{1,16}) joined the game`)
	reLeft   = regexp.MustCompile(`\]:?\s+([A-Za-z0-9_]{1,16}) left the game`)
	// UUID of player Alice is 5d5c...
	reUUID = regexp.MustCompile(`UUID of player ([A-Za-z0-9_]{1,16}) is ([0-9a-fA-F-]{32,36})`)
	// There are 3 of a max of 20 players online: Alice, Bob, Carol
	reListHeader = regexp.MustCompile(`There are (\d+) of a max(?: of)? (\d+) players online:?\s*(.*)`)
	reDone       = regexp.MustCompile(`\]:?\s+Done \(([\d.]+)s\)!`)
)

// LogEvent is a parsed console event.
type LogEvent struct {
	Kind     string // joined | left | uuid | list | done
	Player   string
	UUID     string
	Online   []string
	Count    int
	Max      int
	StartSec float64
}

// ParseLogLine extracts a known event from one console line. Unknown formats
// return nil and must never crash player tracking.
func ParseLogLine(line string) *LogEvent {
	defer func() { recover() }() // absolute safety on hostile log content
	if m := reJoined.FindStringSubmatch(line); m != nil {
		return &LogEvent{Kind: "joined", Player: m[1]}
	}
	if m := reLeft.FindStringSubmatch(line); m != nil {
		return &LogEvent{Kind: "left", Player: m[1]}
	}
	if m := reUUID.FindStringSubmatch(line); m != nil {
		return &LogEvent{Kind: "uuid", Player: m[1], UUID: strings.ToLower(m[2])}
	}
	if m := reListHeader.FindStringSubmatch(line); m != nil {
		ev := &LogEvent{Kind: "list"}
		fmt.Sscanf(m[1], "%d", &ev.Count)
		fmt.Sscanf(m[2], "%d", &ev.Max)
		for _, name := range strings.Split(m[3], ",") {
			name = strings.TrimSpace(name)
			if validPlayerName(name) {
				ev.Online = append(ev.Online, name)
			}
		}
		return ev
	}
	if m := reDone.FindStringSubmatch(line); m != nil {
		ev := &LogEvent{Kind: "done"}
		fmt.Sscanf(m[1], "%f", &ev.StartSec)
		return ev
	}
	return nil
}

// ----- fixed command templates -----------------------------------------------

var playerNameRE = regexp.MustCompile(`^[A-Za-z0-9_]{1,16}$`)

func validPlayerName(name string) bool { return playerNameRE.MatchString(name) }

var ipRE = regexp.MustCompile(`^[0-9a-fA-F.:]{3,45}$`)

// sanitizeReason strips characters that could break out of a Minecraft command.
func sanitizeReason(reason string) string {
	reason = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r < 0x20 {
			return -1
		}
		return r
	}, reason)
	if len(reason) > 200 {
		reason = reason[:200]
	}
	return strings.TrimSpace(reason)
}

// PlayerCommand builds a Minecraft console command from a fixed template.
// Player-supplied data is validated; Linux shell strings are never involved.
func PlayerCommand(action, player, extra string) (string, error) {
	switch action {
	case "send_message":
		if !validPlayerName(player) {
			return "", errors.New("invalid player name")
		}
		msg := sanitizeReason(extra)
		if msg == "" {
			return "", errors.New("empty message")
		}
		return fmt.Sprintf("tell %s %s", player, msg), nil
	case "kick", "ban", "pardon", "whitelist_add", "whitelist_remove", "op", "deop":
		if !validPlayerName(player) {
			return "", errors.New("invalid player name")
		}
		switch action {
		case "kick":
			if r := sanitizeReason(extra); r != "" {
				return fmt.Sprintf("kick %s %s", player, r), nil
			}
			return "kick " + player, nil
		case "ban":
			if r := sanitizeReason(extra); r != "" {
				return fmt.Sprintf("ban %s %s", player, r), nil
			}
			return "ban " + player, nil
		case "pardon":
			return "pardon " + player, nil
		case "whitelist_add":
			return "whitelist add " + player, nil
		case "whitelist_remove":
			return "whitelist remove " + player, nil
		case "op":
			return "op " + player, nil
		case "deop":
			return "deop " + player, nil
		}
	case "ban_ip", "pardon_ip":
		if !ipRE.MatchString(player) {
			return "", errors.New("invalid IP address")
		}
		if action == "ban_ip" {
			if r := sanitizeReason(extra); r != "" {
				return fmt.Sprintf("ban-ip %s %s", player, r), nil
			}
			return "ban-ip " + player, nil
		}
		return "pardon-ip " + player, nil
	}
	return "", fmt.Errorf("unknown player action %q", action)
}

// ----- player administration files -------------------------------------------

type WhitelistEntry struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type OpEntry struct {
	UUID  string `json:"uuid"`
	Name  string `json:"name"`
	Level int    `json:"level"`
}

type BanEntry struct {
	UUID    string `json:"uuid,omitempty"`
	IP      string `json:"ip,omitempty"`
	Name    string `json:"name,omitempty"`
	Created string `json:"created,omitempty"`
	Source  string `json:"source,omitempty"`
	Expires string `json:"expires,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// AdminFiles reads standard Minecraft player administration JSON files.
// Missing files return empty slices, never errors.
type AdminFiles struct {
	Whitelist []WhitelistEntry `json:"whitelist"`
	Ops       []OpEntry        `json:"ops"`
	Banned    []BanEntry       `json:"banned_players"`
	BannedIPs []BanEntry       `json:"banned_ips"`
}

func ReadAdminFiles(serverDir string) *AdminFiles {
	a := &AdminFiles{}
	readJSON(filepath.Join(serverDir, "whitelist.json"), &a.Whitelist)
	readJSON(filepath.Join(serverDir, "ops.json"), &a.Ops)
	readJSON(filepath.Join(serverDir, "banned-players.json"), &a.Banned)
	readJSON(filepath.Join(serverDir, "banned-ips.json"), &a.BannedIPs)
	return a
}

func readJSON(path string, out any) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	json.Unmarshal(data, out) // tolerate malformed files silently
}

// ----- server.properties -----------------------------------------------------

// ReadProperties parses server.properties into an ordered map view.
func ReadProperties(serverDir string) (map[string]string, error) {
	data, err := os.ReadFile(filepath.Join(serverDir, "server.properties"))
	if err != nil {
		return nil, err
	}
	props := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			props[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return props, nil
}

// WriteProperty updates one key in server.properties preserving order,
// comments and unrelated formatting; writes atomically.
func WriteProperty(serverDir, key, value string) error {
	path := filepath.Join(serverDir, "server.properties")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	replaced := false
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "#") {
			continue
		}
		if k, _, ok := strings.Cut(t, "="); ok && strings.TrimSpace(k) == key {
			lines[i] = key + "=" + value
			replaced = true
		}
	}
	if !replaced {
		lines = append(lines, key+"="+value)
	}
	tmp, err := os.CreateTemp(serverDir, ".bonghos-props-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.WriteString(strings.Join(lines, "\n")); err != nil {
		tmp.Close()
		return err
	}
	tmp.Chmod(0o644)
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// WorldDir resolves the world directory from server.properties (level-name),
// defaulting to "world".
func WorldDir(serverDir string) string {
	props, err := ReadProperties(serverDir)
	if err != nil {
		return "world"
	}
	if lv := props["level-name"]; lv != "" && !strings.Contains(lv, "..") && !filepath.IsAbs(lv) {
		return lv
	}
	return "world"
}
