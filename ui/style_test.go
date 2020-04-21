package ui

import (
	"testing"
)

type testStyle struct { // nolint: maligned
	Style      Style
	Code       int
	Reset      int
	Foreground bool
	Background bool
}

// https://en.wikipedia.org/wiki/ANSI_escape_code
var styles = []testStyle{
	// Modifiers
	{Bold, 1, 22, false, false},
	{Dim, 2, 22, false, false}, // Same reset code as bold
	{Italic, 3, 23, false, false},
	{Underline, 4, 24, false, false},
	{Invert, 7, 27, false, false},
	{Hidden, 8, 28, false, false},
	{Strikethrough, 9, 29, false, false},
	// Foreground
	{Black, 30, 39, true, false},
	{Red, 31, 39, true, false},
	{Green, 32, 39, true, false},
	{Yellow, 33, 39, true, false},
	{Blue, 34, 39, true, false},
	{Magenta, 35, 39, true, false},
	{Cyan, 36, 39, true, false},
	{White, 37, 39, true, false},
	{HiBlack, 90, 39, true, false},
	{HiRed, 91, 39, true, false},
	{HiGreen, 92, 39, true, false},
	{HiYellow, 93, 39, true, false},
	{HiBlue, 94, 39, true, false},
	{HiMagenta, 95, 39, true, false},
	{HiCyan, 96, 39, true, false},
	{HiWhite, 97, 39, true, false},
	// Background
	{BGBlack, 40, 49, false, true},
	{BGRed, 41, 49, false, true},
	{BGGreen, 42, 49, false, true},
	{BGYellow, 43, 49, false, true},
	{BGBlue, 44, 49, false, true},
	{BGMagenta, 45, 49, false, true},
	{BGCyan, 46, 49, false, true},
	{BGWhite, 47, 49, false, true},
	{BGHiBlack, 100, 49, false, true},
	{BGHiRed, 101, 49, false, true},
	{BGHiGreen, 102, 49, false, true},
	{BGHiYellow, 103, 49, false, true},
	{BGHiBlue, 104, 49, false, true},
	{BGHiMagenta, 105, 49, false, true},
	{BGHiCyan, 106, 49, false, true},
	{BGHiWhite, 107, 49, false, true},
	// Invalid
	{Style(0), 0, 0, false, false},
}

func TestStyle_Code(t *testing.T) {
	for _, s := range styles {
		t.Run(s.Style.String(), func(t *testing.T) {
			got := int(s.Style)
			if got != s.Code {
				t.Errorf("Got = %d, want = %d", got, s.Code)
			}
		})
	}
}

func TestStyles_ForegroundColor(t *testing.T) {
	for _, s := range styles {
		t.Run(s.Style.String(), func(t *testing.T) {
			got := s.Style.ForegroundColor()
			if got != s.Foreground {
				t.Errorf("Got = %t, want = %t", got, s.Foreground)
			}
		})
	}
}

func TestStyles_BackgroundColor(t *testing.T) {
	for _, s := range styles {
		t.Run(s.Style.String(), func(t *testing.T) {
			got := s.Style.BackgroundColor()
			if got != s.Background {
				t.Errorf("Got = %t, want = %t", got, s.Background)
			}
		})
	}
}

func TestStyle_ResetCode(t *testing.T) {
	for _, s := range styles {
		t.Run(s.Style.String(), func(t *testing.T) {
			got := s.Style.ResetCode()
			if got != s.Reset {
				t.Errorf("Got = %d, want = %d", got, s.Reset)
			}
		})
	}
}

func TestStyles(t *testing.T) {
	tests := []struct {
		name      string
		styles    Styles
		wantOpen  string
		wantClose string
	}{
		{
			name:      "Empty",
			styles:    nil,
			wantOpen:  "",
			wantClose: "",
		},
		{
			name:      "One",
			styles:    Styles{Red},
			wantOpen:  "\x1b[31m",
			wantClose: "\x1b[39m",
		},
		{
			name:      "Multiple",
			styles:    Styles{Bold, Green},
			wantOpen:  "\x1b[1;32m",
			wantClose: "\x1b[22;39m",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotOpen := tc.styles.Open()
			if gotOpen != tc.wantOpen {
				t.Errorf("Open does not match; got %q, want %q", gotOpen, tc.wantOpen)
			}
			gotClose := tc.styles.Close()
			if gotClose != tc.wantClose {
				t.Errorf("Close does not match; got %q, want %q", gotClose, tc.wantClose)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		styles []Style
		want   string
	}{
		{
			name:   "NoStyles",
			input:  "foo",
			styles: nil,
			want:   "foo",
		},
		{
			name:   "Style",
			input:  "foo",
			styles: []Style{Red},
			want:   "\x1b[31mfoo\x1b[39m",
		},
		{
			name:   "Empty",
			input:  "",
			styles: []Style{Red},
			want:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Format(tc.input, tc.styles...)
			if got != tc.want {
				t.Errorf(`Output does not match.
Got  %[1]s (%[1]q)
Want %[2]s (%[2]q)`, got, tc.want)
			}
		})
	}
}
