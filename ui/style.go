package ui

import (
	"strconv"
	"strings"
)

// A Style is an ANSI style attribute to set when formatting text.
type Style uint8

//go:generate go run golang.org/x/tools/cmd/stringer -type Style

// Modifiers:
const (
	Bold          Style = 1
	Dim           Style = 2 // Faint
	Italic        Style = 3
	Underline     Style = 4
	Invert        Style = 7 // Reverse
	Hidden        Style = 8 // Conceal
	Strikethrough Style = 9 // Crossed-out
)

// Foreground colors:
const (
	Black Style = 30 + iota
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

// Bright foreground colors:
const (
	HiBlack Style = 90 + iota
	HiRed
	HiGreen
	HiYellow
	HiBlue
	HiMagenta
	HiCyan
	HiWhite
)

// Background colors:
const (
	BGBlack Style = 40 + iota
	BGRed
	BGGreen
	BGYellow
	BGBlue
	BGMagenta
	BGCyan
	BGWhite
)

// Bright background colors:
const (
	BGHiBlack Style = 100 + iota
	BGHiRed
	BGHiGreen
	BGHiYellow
	BGHiBlue
	BGHiMagenta
	BGHiCyan
	BGHiWhite
)

// ForegroundColor returns true if the style describes a foreground color.
func (s Style) ForegroundColor() bool {
	if s >= Black && s <= White {
		return true
	}
	if s >= HiBlack && s <= HiWhite {
		return true
	}
	return false
}

// BackgroundColor returns true if the style describes a background color.
func (s Style) BackgroundColor() bool {
	if s >= BGBlack && s <= BGWhite {
		return true
	}
	if s >= BGHiBlack && s <= BGHiWhite {
		return true
	}
	return false
}

// ResetCode returns the code to reset the given style.
//
// Because Bold and Dim have the same reset code (22), any previous bold/dim
// text is also reset.
func (s Style) ResetCode() int {
	switch s {
	case Bold, Dim:
		return 22
	case Italic:
		return 23
	case Underline:
		return 24
	case Invert:
		return 27
	case Hidden:
		return 28
	case Strikethrough:
		return 29
	}
	if s.ForegroundColor() {
		return 39
	}
	if s.BackgroundColor() {
		return 49
	}
	return 0
}

// Styles contains a list of styles.
type Styles []Style

// Open returns the ANSI escape sequence that enables the given style.
func (ss Styles) Open() string {
	if len(ss) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("\x1b[")
	for i, s := range ss {
		out.WriteString(strconv.Itoa(int(s)))
		if i < len(ss)-1 {
			out.WriteRune(';')
		}
	}
	out.WriteRune('m')
	return out.String()
}

// Close returns the ANSI escape sequence to reset any styles set by Open().
func (ss Styles) Close() string {
	if len(ss) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("\x1b[")
	for i, s := range ss {
		out.WriteString(strconv.Itoa(s.ResetCode()))
		if i < len(ss)-1 {
			out.WriteRune(';')
		}
	}
	out.WriteRune('m')
	return out.String()
}

// Format returns the ANSI escape sequence to enable the styles, followed by
// the given text and en ANSI escape sequence to reset back styles. Because
// there is no context on what styles may have been set before, it is not
// possible to reset the color back. Instead, the style is set back to the
// default style. Only attributes that were modified are reset; for example
// setting a text bold within red text is supported.
//
// If no styles are set or s is empty, the original text is returned.
func (ss Styles) Format(s string) string {
	if len(ss) == 0 {
		return s
	}
	return ss.Open() + s + ss.Close()
}

// Format wraps the given string with ANSI styles. If the input string is
// empty, an empty string is returned.
func Format(str string, styles ...Style) string {
	if str == "" {
		return ""
	}
	ss := Styles(styles)
	return ss.Format(str)
}
