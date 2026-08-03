package supervisor

import (
	"strings"
	"testing"
)

func TestSanitizeLineStripsANSIColour(t *testing.T) {
	// A typical coloured server log line straight off the PTY.
	in := "\x1b[32m[12:00:00] [Server thread/INFO]:\x1b[0m Done (18.431s)!"
	got := SanitizeLine(in)
	want := "[12:00:00] [Server thread/INFO]: Done (18.431s)!"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	if strings.Contains(got, "\x1b") {
		t.Error("escape character survived")
	}
}

func TestSanitizeLineHandlesEveryEscapeFamily(t *testing.T) {
	cases := map[string]string{
		"\x1b[1;31mred\x1b[0m":         "red",      // CSI
		"\x1b]0;window title\x07text":  "text",     // OSC ending in BEL
		"\x1b]0;title\x1b\\text":       "text",     // OSC ending in ST
		"\x1bPsome dcs\x1b\\after":     "after",    // DCS
		"\x1b(Bplain":                  "plain",    // charset selection
		"\x1b=alt":                     "alt",      // two-byte escape
		"\x1b[2Kcleared":               "cleared",  // erase line
		"before\x1b[":                  "before",   // unterminated CSI
		"tab\there":                    "tab here", // tab becomes a space
		"a\x00b\x07c":                  "abc",      // C0 controls
		"plain text with no escapes":   "plain text with no escapes",
		"§aMinecraft §bcolour §ccodes": "§aMinecraft §bcolour §ccodes", // preserved
	}
	for in, want := range cases {
		if got := SanitizeLine(in); got != want {
			t.Errorf("SanitizeLine(%q) = %q, want %q", in, got, want)
		}
	}
}

// Progress bars redraw with carriage returns; a terminal shows only the final
// state, so that is what should be stored and displayed.
func TestSanitizeLineKeepsOnlyTheFinalCarriageReturnSegment(t *testing.T) {
	in := "Downloading 10%\rDownloading 50%\rDownloading 100%"
	if got := SanitizeLine(in); got != "Downloading 100%" {
		t.Errorf("got %q, want the last segment", got)
	}
}

func TestSanitizeLineTruncatesAbsurdlyLongLines(t *testing.T) {
	in := strings.Repeat("x", maxLineRunes*3)
	got := SanitizeLine(in)
	if len([]rune(got)) > maxLineRunes+20 {
		t.Errorf("line is %d runes; it should have been truncated", len([]rune(got)))
	}
	if !strings.HasSuffix(got, "[truncated]") {
		t.Error("truncation is not signposted to the reader")
	}
}

func TestSanitizeLineHandlesEmptyAndControlOnlyInput(t *testing.T) {
	if got := SanitizeLine(""); got != "" {
		t.Errorf("empty input produced %q", got)
	}
	if got := SanitizeLine("\x1b[2J\x1b[H"); got != "" {
		t.Errorf("control-only input produced %q", got)
	}
}

// The console frame limit is 64 KiB. A modded boot easily exceeds that across
// 500 lines, and the oversized frame was silently dropped, leaving connecting
// clients with no backlog at all.
func TestHistoryStaysWithinTheFrameBudget(t *testing.T) {
	s := &Supervisor{subs: map[chan string]bool{}}

	// Simulate a noisy modded boot: long lines, more than the line limit.
	long := strings.Repeat("mod loading step ", 40) // ~680 bytes
	for i := 0; i < historyLines*3; i++ {
		s.broadcast(long)
	}

	if len(s.history) > historyLines {
		t.Errorf("history holds %d lines, above the %d line cap", len(s.history), historyLines)
	}
	total := 0
	for _, l := range s.history {
		total += len(l) + 1
	}
	if total > historyBytes {
		t.Errorf("history holds %d bytes, above the %d byte cap", total, historyBytes)
	}
	if total > 64*1024 {
		t.Fatal("history exceeds the console frame limit; replay would be dropped")
	}
	if len(s.history) == 0 {
		t.Fatal("history is empty; nothing would replay at all")
	}
}

func TestHistoryKeepsTheMostRecentLines(t *testing.T) {
	s := &Supervisor{subs: map[chan string]bool{}}
	for i := 0; i < 50; i++ {
		s.broadcast("line")
	}
	s.broadcast("THE NEWEST LINE")
	if s.history[len(s.history)-1] != "THE NEWEST LINE" {
		t.Error("the newest line is not at the end of the history")
	}
}

// A single line larger than the whole budget must not empty the buffer.
func TestHistorySurvivesOneEnormousLine(t *testing.T) {
	s := &Supervisor{subs: map[chan string]bool{}}
	s.broadcast(strings.Repeat("y", historyBytes*2))
	if len(s.history) != 1 {
		t.Errorf("history holds %d lines, want the single oversized one", len(s.history))
	}
}

// Bonghos polls "list" every few seconds for the player count. That reply must
// not fill the operator's console.
func TestInternalCommandRepliesAreSuppressed(t *testing.T) {
	s := &Supervisor{subs: map[chan string]bool{}}

	if err := s.SendInternalCommand("list"); err == nil {
		t.Log("SendCommand failed as expected without a running process")
	}
	// Suppression is armed even though the send itself could not reach a
	// process in this test.
	if !s.suppressed("There are 0 of a max of 10 players online") {
		t.Error("the list reply was not suppressed")
	}
	// Once the reply is seen, suppression stops, so an operator typing the
	// same command sees their own output.
	if s.suppressed("There are 3 of a max of 10 players online: a, b, c") {
		t.Error("suppression persisted past the reply it was armed for")
	}
}

func TestOrdinaryOutputIsNeverSuppressed(t *testing.T) {
	s := &Supervisor{subs: map[chan string]bool{}}
	_ = s.SendInternalCommand("list")
	for _, line := range []string{
		"[12:00:00] [Server thread/INFO]: Done (18.431s)!",
		"[12:00:01] [Server thread/WARN]: Something went wrong",
		"Steve joined the game",
	} {
		if s.suppressed(line) {
			t.Errorf("ordinary output was hidden from the console: %q", line)
		}
	}
}

// Only commands on the closed internal list may be hidden, so nothing can be
// run invisibly through this path.
func TestUnknownCommandsAreNotSuppressible(t *testing.T) {
	s := &Supervisor{subs: map[chan string]bool{}}
	_ = s.SendInternalCommand("op attacker")
	if s.suppressed("op attacker") {
		t.Error("an arbitrary command was hidden from the console")
	}
	if s.suppressed("Made attacker a server operator") {
		t.Error("the reply to an arbitrary command was hidden")
	}
}
