package qrcode

import (
	"strings"
	"testing"
)

const sampleURI = "otpauth://totp/Bonghos:klaude?secret=2LF2JDU44UF5EWSKRKC2XVAYNM53DESD&issuer=Bonghos&digits=6&period=30"

func TestTerminalRendersAKnownURI(t *testing.T) {
	art, err := Terminal(sampleURI, 120)
	if err != nil {
		t.Fatalf("Terminal: %v", err)
	}
	if art == "" {
		t.Fatal("Terminal returned empty output")
	}

	lines := strings.Split(strings.TrimRight(art, "\n"), "\n")
	if len(lines) < 12 {
		t.Errorf("only %d rows; that is too small to be a real QR code", len(lines))
	}

	// Every row must be the same width, or the modules are not square and
	// scanners will struggle.
	width := len([]rune(lines[0]))
	for i, l := range lines {
		if got := len([]rune(l)); got != width {
			t.Fatalf("row %d is %d columns wide, want %d", i, got, width)
		}
	}

	// The quiet zone means the first and last rows are entirely background.
	for _, idx := range []int{0, len(lines) - 1} {
		if strings.Trim(lines[idx], "\u2588") != "" {
			t.Errorf("row %d is not a clear quiet zone: %q", idx, lines[idx])
		}
	}

	// A real code contains a mix of module states.
	if !strings.ContainsAny(art, " \u2580\u2584") {
		t.Error("output contains no foreground modules")
	}
}

func TestTerminalRefusesToOverflowANarrowTerminal(t *testing.T) {
	if _, err := Terminal(sampleURI, 20); err != ErrTooWide {
		t.Errorf("narrow terminal returned %v, want ErrTooWide", err)
	}
	// Zero width means "do not check", for non-interactive callers.
	if _, err := Terminal(sampleURI, 0); err != nil {
		t.Errorf("unchecked width returned %v", err)
	}
}

func TestTerminalRejectsEmptyInput(t *testing.T) {
	for _, in := range []string{"", "   ", "\n\t"} {
		if _, err := Terminal(in, 120); err == nil {
			t.Errorf("Terminal(%q) accepted empty input", in)
		}
	}
}

// A payload too large for any QR version must fail rather than panic, since
// callers treat an error as "print the manual fallback".
func TestTerminalFailsCleanlyOnOversizedInput(t *testing.T) {
	huge := strings.Repeat("A", 8000)
	if _, err := Terminal(huge, 500); err == nil {
		t.Error("oversized payload was accepted")
	}
}

func TestSVGRendersAKnownURI(t *testing.T) {
	svg, err := SVG(sampleURI, 4)
	if err != nil {
		t.Fatalf("SVG: %v", err)
	}
	if !strings.HasPrefix(svg, "<svg ") || !strings.HasSuffix(svg, "</svg>") {
		t.Fatalf("output is not a standalone SVG document: %.80s…", svg)
	}
	if !strings.Contains(svg, `xmlns="http://www.w3.org/2000/svg"`) {
		t.Error("missing SVG namespace; browsers will not render it")
	}
	if strings.Count(svg, "<rect") < 50 {
		t.Errorf("only %d rects; that is too few modules", strings.Count(svg, "<rect"))
	}
	// The frontend refuses anything script-like, so the generator must never
	// emit it.
	for _, bad := range []string{"<script", "onload=", "xlink:href", "javascript:"} {
		if strings.Contains(strings.ToLower(svg), bad) {
			t.Errorf("generated SVG contains %q", bad)
		}
	}
	// A white background is required or the code will not scan on a dark page.
	if !strings.Contains(svg, `fill="#ffffff"`) {
		t.Error("missing the light background needed for the quiet zone")
	}
}

func TestSVGDefaultsAnInvalidModuleSize(t *testing.T) {
	for _, size := range []int{0, -5} {
		svg, err := SVG(sampleURI, size)
		if err != nil {
			t.Fatalf("SVG(%d): %v", size, err)
		}
		if svg == "" {
			t.Errorf("SVG(%d) returned empty output", size)
		}
	}
}

func TestSVGRejectsEmptyInput(t *testing.T) {
	if _, err := SVG("  ", 4); err == nil {
		t.Error("SVG accepted empty input")
	}
}

// Non-interactive output (a pipe, a log file, CI) must report an error so the
// caller prints the manual secret instead of writing block characters into it.
func TestTerminalForRejectsANonTerminal(t *testing.T) {
	if _, err := TerminalFor(sampleURI, -1); err == nil {
		t.Error("TerminalFor accepted a non-terminal file descriptor")
	}
	if _, err := Stdout(sampleURI); err == nil {
		t.Log("stdout happens to be a terminal in this environment")
	}
}
