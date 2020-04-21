package ui

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// StringWidth returns the number of visible cells the given string occupies.
//
// ANSI escape codes are not included in the computation.
func StringWidth(str string) int {
	w := 0
	esc := false
	for _, r := range str {
		if isEsc(r) {
			esc = true
			continue
		}
		if esc {
			if isEscDone(r) {
				esc = false
			}
			continue
		}
		w += runewidth.RuneWidth(r)
	}
	return w
}

// Truncate truncates a string if it exceeds the given number of visible
// columns. ANSI escape codes are not included in width computation. If the
// string needs to be truncated, the tail is written at the end of the, while
// still fitting within the allowed columns.
//
// In case the output is truncated and the text contains ANSI styling escape
// codes, all styles are reset to ensure styles don't leak out.
func Truncate(str string, cols int, tail string) string {
	runes := []rune(str)
	tailWidth := StringWidth(tail)
	width := 0
	esc := false
	hasEsc := false
	truncate := -1
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if isEsc(r) {
			hasEsc = true
			esc = true
			continue
		}
		if esc {
			if isEscDone(r) {
				esc = false
			}
			continue
		}
		cw := runewidth.RuneWidth(r)
		if width+cw > cols {
			truncate = i - tailWidth
			break
		}
		width += cw
	}
	if truncate > 0 {
		out := string(runes[0:truncate])
		if hasEsc {
			out += "\x1b[0m"
		}
		out += tail
		return out
	}
	return str
}

// Wrap wraps the given string so that it fits within the given number of
// columns. ANSI escape characters are ignored in width calculation.
//
// The wrapping algorithm is very simple; the text is wrapped exactly at the
// character, rather than trying to ensure more grammatically correct wrapping,
// such as at word boundaries.
func Wrap(str string, cols int) string {
	var out strings.Builder
	var width int
	var esc bool
	for _, r := range str {
		if isEsc(r) {
			esc = true
		}
		if esc {
			out.WriteRune(r)
			if isEscDone(r) {
				esc = false
			}
			continue
		}
		cw := runewidth.RuneWidth(r)
		if r == '\n' {
			out.WriteRune(r)
			width = 0
			continue
		}
		if width+cw > cols {
			out.WriteByte('\n')
			width = 0
		}
		out.WriteRune(r)
		width += cw
	}
	return out.String()
}

// PadLeft prepends the given string with spaces so it is n columns wide, not
// including ANSI escape codes.
func PadLeft(str string, cols int) string {
	w := StringWidth(str)
	remain := cols - w
	return strings.Repeat(" ", remain) + str
}

// PadRight adds spaces to the given string with so it is n columns wide, not
// including ANSI escape codes.
func PadRight(str string, cols int) string {
	w := StringWidth(str)
	remain := cols - w
	return str + strings.Repeat(" ", remain)
}

// Prefix prefixes the given string with a prefix. If the string contains
// multiple lines, all lines are prefixed. If the string is empty, an empty
// string is returned.
func Prefix(str string, prefix string) string {
	if str == "" {
		return ""
	}
	var out strings.Builder
	lines := strings.Split(str, "\n")
	for i, l := range lines {
		if i > 0 {
			out.WriteByte('\n')
		}
		out.WriteString(prefix)
		out.WriteString(l)
	}
	return out.String()
}

// Padding describes the padding in spaces to apply in Pad().
type Padding struct {
	Top    int
	Bottom int
	Right  int
	Left   int
}

// Pad pads the string  with the given padding. If the string contains line
// breaks, the entire text is padded as a block.
//
// If the string is empty, no padding is applied.
func Pad(s string, p Padding) string {
	if len(s) == 0 {
		return ""
	}
	l := strings.Repeat(" ", p.Left)
	r := strings.Repeat(" ", p.Right)
	var out strings.Builder
	out.WriteString(strings.Repeat("\n", p.Top))
	if len(s) > 0 {
		lines := strings.Split(s, "\n")
		for i, line := range lines {
			if i > 0 {
				out.WriteByte('\n')
			}
			out.WriteString(l)
			out.WriteString(line)
			out.WriteString(r)
		}
	}
	out.WriteString(strings.Repeat("\n", p.Bottom))
	return out.String()
}

func isEsc(r rune) bool {
	return r == '\x1b'
}

func isEscDone(r rune) bool {
	if r >= 'A' && r <= 'Z' {
		return true
	}
	if r >= 'a' && r <= 'z' {
		return true
	}
	return false
}
