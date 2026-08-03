package supervisor

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Console output arrives from a pseudo-terminal, so it carries the control
// sequences a terminal would act on: colour codes, cursor movement, progress
// bars redrawing themselves with carriage returns. Rendered verbatim in a
// browser those become noise, and stored verbatim they bloat the history
// buffer with bytes nobody can read.
//
// SanitizeLine strips control sequences while preserving the readable text,
// including the Minecraft section-sign colour codes a server emits, which are
// left intact so a client can render or drop them as it prefers.

// maxLineRunes caps a single console line. Some mod loaders emit enormous
// single-line dumps; keeping one of those in memory per line would blow the
// history budget on its own.
const maxLineRunes = 4000

// SanitizeLine removes ANSI escape sequences and other terminal control
// characters from a console line, returning printable text.
func SanitizeLine(line string) string {
	if line == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(line))

	for i := 0; i < len(line); {
		r, size := utf8.DecodeRuneInString(line[i:])

		switch {
		case r == 0x1b: // ESC: start of an escape sequence
			if n := escapeLength(line[i:]); n > 0 {
				i += n
				continue
			}
			i += size // lone ESC, drop it
			continue

		case r == '\r':
			// A carriage return means the terminal would overwrite the line.
			// Keep only what was written after the last one, which is what a
			// human watching the terminal would have ended up seeing.
			b.Reset()
			i += size
			continue

		case r == '\t':
			b.WriteByte(' ')
			i += size
			continue

		case r == utf8.RuneError && size == 1:
			// Invalid UTF-8 from a binary dump; skip the byte rather than
			// emitting replacement characters across the whole line.
			i++
			continue

		case r < 0x20 || r == 0x7f:
			// Other C0 controls and DEL carry no meaning here.
			i += size
			continue

		case unicode.Is(unicode.Cf, r):
			// Format characters such as zero-width joiners; invisible, and
			// usable to disguise text.
			i += size
			continue
		}

		b.WriteRune(r)
		i += size
	}

	out := strings.TrimRight(b.String(), " ")
	if utf8.RuneCountInString(out) > maxLineRunes {
		runes := []rune(out)
		out = string(runes[:maxLineRunes]) + " …[truncated]"
	}
	return out
}

// escapeLength returns the byte length of the escape sequence starting at the
// beginning of s, or 0 if it is not a recognised sequence.
func escapeLength(s string) int {
	if len(s) < 2 || s[0] != 0x1b {
		return 0
	}
	switch s[1] {
	case '[': // CSI: ESC [ params intermediates final
		for i := 2; i < len(s); i++ {
			c := s[i]
			if c >= 0x40 && c <= 0x7e {
				return i + 1
			}
		}
		return len(s) // unterminated: drop the rest
	case ']': // OSC: ESC ] ... BEL or ESC \
		for i := 2; i < len(s); i++ {
			if s[i] == 0x07 {
				return i + 1
			}
			if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
		}
		return len(s)
	case 'P', '^', '_': // DCS, PM, APC: terminated by ESC \
		for i := 2; i < len(s); i++ {
			if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
		}
		return len(s)
	case '(', ')', '*', '+': // character set selection: one more byte
		if len(s) >= 3 {
			return 3
		}
		return len(s)
	default:
		// Two-byte escapes such as ESC = or ESC >.
		return 2
	}
}
