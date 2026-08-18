package supervisor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
func TestInternalCommandEchoAndReplyAreSuppressedInEitherOrder(t *testing.T) {
	orders := map[string][]string{
		"echo then reply": {"list", "There are 0 of a max of 10 players online:"},
		"reply then echo": {"There are 0 of a max of 10 players online:", "list"},
	}
	for name, lines := range orders {
		t.Run(name, func(t *testing.T) {
			s := runningCommandTestSupervisor(t)
			if err := s.SendInternalCommand("list"); err != nil {
				t.Fatalf("SendInternalCommand: %v", err)
			}
			for _, line := range lines {
				if !s.suppressed(line) {
					t.Errorf("internal output was not suppressed: %q", line)
				}
			}
			if s.suppressed("There are 3 of a max of 10 players online: a, b, c") {
				t.Error("suppression persisted after both echo and reply arrived")
			}
		})
	}
}

func TestOrdinaryOutputIsNeverSuppressed(t *testing.T) {
	s := runningCommandTestSupervisor(t)
	if err := s.SendInternalCommand("list"); err != nil {
		t.Fatalf("SendInternalCommand: %v", err)
	}
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

func TestOperatorListCancelsInternalSuppression(t *testing.T) {
	s := runningCommandTestSupervisor(t)
	if err := s.SendInternalCommand("list"); err != nil {
		t.Fatalf("SendInternalCommand: %v", err)
	}
	if err := s.SendCommand("list"); err != nil {
		t.Fatalf("SendCommand: %v", err)
	}
	for _, line := range []string{
		"list",
		"There are 2 of a max of 10 players online: Alex, Steve",
	} {
		if s.suppressed(line) {
			t.Errorf("operator output was hidden: %q", line)
		}
	}
}

func TestInternalPollDoesNotOvertakeOperatorList(t *testing.T) {
	s := runningCommandTestSupervisor(t)
	if err := s.SendCommand("list"); err != nil {
		t.Fatalf("SendCommand: %v", err)
	}
	if err := s.SendInternalCommand("list"); err != nil {
		t.Fatalf("SendInternalCommand: %v", err)
	}
	for _, line := range []string{
		"list",
		"There are 2 of a max of 10 players online: Alex, Steve",
	} {
		if s.suppressed(line) {
			t.Errorf("operator output was hidden by a following poll: %q", line)
		}
	}
}

func TestInternalOutputIsParsedButNotStoredAsConsoleHistory(t *testing.T) {
	serverDir := t.TempDir()
	consoleRead, consoleWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	commandRead, commandWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		consoleRead.Close()
		consoleWrite.Close()
		commandRead.Close()
		commandWrite.Close()
	})

	s := &Supervisor{
		cfg:   Config{ServerDir: serverDir},
		state: StateRunning,
		tty:   commandWrite,
		subs:  map[chan string]bool{},
	}
	parsed := make(chan string, 3)
	s.OnLine(func(line string) { parsed <- line })
	internalLines, cancelInternal := s.SubscribeInternal()
	defer cancelInternal()
	if err := s.SendInternalCommand("list"); err != nil {
		t.Fatalf("SendInternalCommand: %v", err)
	}

	done := make(chan struct{})
	go func() {
		s.readConsole(consoleRead)
		close(done)
	}()
	for _, line := range []string{
		"[16:56:43] [Server thread/INFO]: There are 0 of a max of 10 players online:",
		"list",
		"[16:56:44] [Server thread/INFO]: Saved the game",
	} {
		if _, err := consoleWrite.WriteString(line + "\n"); err != nil {
			t.Fatal(err)
		}
	}
	consoleWrite.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("readConsole did not finish")
	}

	for i := 0; i < 3; i++ {
		select {
		case <-parsed:
		case <-time.After(2 * time.Second):
			t.Fatal("the parser did not receive every console line")
		}
	}
	for _, want := range []string{"players online", "list"} {
		select {
		case line := <-internalLines:
			if !strings.Contains(line, want) {
				t.Fatalf("internal line = %q, want it to contain %q", line, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("bookkeeping subscriber did not receive %q", want)
		}
	}
	history, _, cancel := s.Subscribe()
	cancel()
	if len(history) != 1 || !strings.Contains(history[0], "Saved the game") {
		t.Fatalf("console history = %q, want only the ordinary server line", history)
	}
	logData, err := os.ReadFile(filepath.Join(serverDir, "logs", "bonghos-console.log"))
	if err != nil {
		t.Fatal(err)
	}
	logText := string(logData)
	if strings.Contains(logText, "players online") || strings.Contains(logText, "\nlist\n") {
		t.Fatalf("internal polling leaked into the console log: %q", logText)
	}
	if !strings.Contains(logText, "Saved the game") {
		t.Fatalf("ordinary server output is missing from the console log: %q", logText)
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

func runningCommandTestSupervisor(t *testing.T) *Supervisor {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		r.Close()
		w.Close()
	})
	return &Supervisor{
		state: StateRunning,
		tty:   w,
		subs:  map[chan string]bool{},
	}
}
