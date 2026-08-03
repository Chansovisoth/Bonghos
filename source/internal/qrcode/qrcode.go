// Package qrcode renders QR codes for terminals.
//
// It exists so TOTP enrolment can offer a scannable code without pulling in a
// frontend toolchain or an image viewer. Rendering is always best-effort: the
// secret and otpauth URI remain the authoritative fallback, and no failure here
// may interrupt account creation.
package qrcode

import (
	"errors"
	"os"
	"strings"

	"golang.org/x/term"
	"rsc.io/qr"
)

// quietZone is the four-module margin the QR specification requires. Without
// it many scanners fail, particularly against a terminal's background colour.
const quietZone = 4

// ErrTooWide reports that the rendered code would not fit the terminal.
var ErrTooWide = errors.New("terminal is too narrow to display the QR code")

// Terminal renders text as a QR code using half-block characters, which give
// square-ish modules because terminal cells are roughly twice as tall as they
// are wide. The result is light-on-dark safe: modules are drawn with an
// explicit white background so it scans on both light and dark themes.
//
// width is the usable terminal width in columns; pass 0 to skip the fit check.
func Terminal(text string, width int) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", errors.New("nothing to encode")
	}
	// Medium correction tolerates a little terminal rendering noise without
	// inflating the code beyond a typical 80-column window.
	code, err := qr.Encode(text, qr.M)
	if err != nil {
		return "", err
	}

	size := code.Size + quietZone*2
	if width > 0 && size > width {
		return "", ErrTooWide
	}

	// dark reports whether the module at these coordinates is black, treating
	// everything in the quiet zone as light.
	dark := func(x, y int) bool {
		x -= quietZone
		y -= quietZone
		if x < 0 || y < 0 || x >= code.Size || y >= code.Size {
			return false
		}
		return code.Black(x, y)
	}

	// Each output line encodes two module rows: the upper half-block is the
	// foreground colour and the lower half is the background.
	var b strings.Builder
	for y := 0; y < size; y += 2 {
		for x := 0; x < size; x++ {
			top := dark(x, y)
			bottom := dark(x, y+1) // out of range reads as light

			switch {
			case top && bottom:
				b.WriteString(" ")
			case top && !bottom:
				b.WriteString("\u2584") // lower half block
			case !top && bottom:
				b.WriteString("\u2580") // upper half block
			default:
				b.WriteString("\u2588") // full block
			}
		}
		b.WriteString("\n")
	}
	return b.String(), nil
}

// TerminalFor renders text sized for the given file descriptor, returning an
// error when the destination is not an interactive terminal or is too narrow.
// Callers treat any error as "print the manual fallback instead".
func TerminalFor(text string, fd int) (string, error) {
	if !term.IsTerminal(fd) {
		return "", errors.New("output is not an interactive terminal")
	}
	width, _, err := term.GetSize(fd)
	if err != nil || width <= 0 {
		width = 80 // a reasonable assumption rather than a hard failure
	}
	return Terminal(text, width)
}

// Stdout renders text sized for the current standard output.
func Stdout(text string) (string, error) {
	return TerminalFor(text, int(os.Stdout.Fd()))
}

// SVG renders text as a standalone SVG document, used by the Web UI so the
// browser needs no QR library. moduleSize is the pixel size of one module.
func SVG(text string, moduleSize int) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", errors.New("nothing to encode")
	}
	if moduleSize <= 0 {
		moduleSize = 4
	}
	code, err := qr.Encode(text, qr.M)
	if err != nil {
		return "", err
	}
	size := (code.Size + quietZone*2) * moduleSize

	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="`)
	b.WriteString(itoa(size))
	b.WriteString(`" height="`)
	b.WriteString(itoa(size))
	b.WriteString(`" viewBox="0 0 `)
	b.WriteString(itoa(size))
	b.WriteString(" ")
	b.WriteString(itoa(size))
	b.WriteString(`" shape-rendering="crispEdges" role="img" aria-label="TOTP enrolment QR code">`)
	// The white background is the quiet zone; without it scanners struggle
	// against a dark page.
	b.WriteString(`<rect width="100%" height="100%" fill="#ffffff"/>`)
	for y := 0; y < code.Size; y++ {
		for x := 0; x < code.Size; x++ {
			if !code.Black(x, y) {
				continue
			}
			b.WriteString(`<rect x="`)
			b.WriteString(itoa((x + quietZone) * moduleSize))
			b.WriteString(`" y="`)
			b.WriteString(itoa((y + quietZone) * moduleSize))
			b.WriteString(`" width="`)
			b.WriteString(itoa(moduleSize))
			b.WriteString(`" height="`)
			b.WriteString(itoa(moduleSize))
			b.WriteString(`" fill="#000000"/>`)
		}
	}
	b.WriteString(`</svg>`)
	return b.String(), nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}
