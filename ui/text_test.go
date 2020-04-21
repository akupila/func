package ui_test

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/func/func/ui"
	"github.com/mattn/go-runewidth"
)

func TestStringWidth(t *testing.T) {
	tests := []struct {
		name string
		str  string
		want int
	}{
		{
			name: "Empty",
			str:  "",
			want: 0,
		},
		{
			name: "Simple",
			str:  "foo",
			want: 3,
		},
		{
			name: "Wide",
			str:  "✅",
			want: 2, // len() returns 3
		},
		{
			name: "IgnoreANSI",
			str:  "\x1b[1;31m❌\x1b[0m",
			want: 2, // len() returns 14
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ui.StringWidth(tc.str)
			if got != tc.want {
				t.Errorf("Got = %d, want = %d", got, tc.want)
			}
		})
	}
}

func ExampleStringWidth() {
	// str contains a string with a double width character, wrapped in ANSI
	// styling. When printed in a terminal, the visible width is 2 columns.
	str := "\x1b[1;31m❌\x1b[0m"
	fmt.Println("len:", len(str))
	fmt.Println("runewidth.StringWidth:", runewidth.StringWidth(str))
	fmt.Println("StringWidth:", ui.StringWidth(str))
	// Output:
	// len: 14
	// runewidth.StringWidth: 11
	// StringWidth: 2
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		str  string
		cols int
		tail string
		want string
	}{
		{
			name: "Empty",
			str:  "",
			cols: 80,
			want: "",
		},
		{
			name: "Cut",
			str:  "long text",
			cols: 6,
			want: "long t",
		},
		{
			name: "Wide",
			str:  "✅✅✅",
			cols: 4,
			want: "✅✅",
		},
		{
			name: "WideMid",
			str:  "✅✅✅",
			cols: 5,
			want: "✅✅",
		},
		{
			name: "Tail",
			str:  "long text",
			cols: 8,
			tail: ">",
			want: "long te>",
		},
		{
			name: "StyledTruncMid",
			str:  "long \x1b[1mtext\x1b[0m here",
			cols: 7,
			want: "long \x1b[1mte\x1b[0m",
		},
		{
			name: "StyledTruncAfter",
			str:  "long \x1b[1mtext\x1b[0m here",
			cols: 12,
			want: "long \x1b[1mtext\x1b[0m he\x1b[0m", // Duplicate reset is necessary
		},
		{
			name: "ColorTail",
			str:  "foo bar baz",
			cols: 7,
			tail: "\x1b[2m>\x1b[0m",
			want: "foo ba\x1b[2m>\x1b[0m",
		},
		{
			name: "LongTail",
			str:  "foo bar baz",
			cols: 8,
			tail: ">>",
			want: "foo ba>>",
		},
		{
			name: "WideTail",
			str:  "foo bar baz",
			cols: 10,
			tail: "更多",
			want: "foo ba更多",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ui.Truncate(tc.str, tc.cols, tc.tail)
			if got != tc.want {
				t.Errorf(`Output does not match.
Got:
%s
Want:
%s`, hex.Dump([]byte(got)), hex.Dump([]byte(tc.want)))
			}
		})
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name string
		str  string
		cols int
		want string
	}{
		{
			name: "Empty",
			str:  "",
			cols: 80,
			want: "",
		},
		{
			name: "NoBreak",
			str:  "foo bar baz",
			cols: 80,
			want: "foo bar baz",
		},
		{
			name: "Wrap",
			str:  "foo bar baz",
			cols: 5,
			want: "foo b\nar ba\nz",
		},
		{
			name: "KeepLineBreaks",
			str:  "foo\nbar baz qux",
			cols: 6,
			want: "foo\nbar ba\nz qux",
		},
		{
			name: "WrapANSI",
			str:  "foo \x1b[1mbar\x1b[0m baz",
			cols: 4,
			want: "foo \x1b[1m\nbar\x1b[0m \nbaz",
		},
		{
			name: "Wide",
			str:  "❌❌❌❌❌",
			cols: 4,
			want: "❌❌\n❌❌\n❌",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ui.Wrap(tc.str, tc.cols)
			if got != tc.want {
				t.Errorf(`Output does not match.
Got:
%s
Want:
%s`, hex.Dump([]byte(got)), hex.Dump([]byte(tc.want)))
			}
		})
	}
}

func TestPadX(t *testing.T) {
	tests := []struct {
		name      string
		str       string
		cols      int
		wantRight string
		wantLeft  string
	}{
		{
			name:      "Empty",
			str:       "",
			cols:      3,
			wantLeft:  "   ",
			wantRight: "   ",
		},
		{
			name:      "Simple",
			str:       "foo",
			cols:      5,
			wantLeft:  "  foo",
			wantRight: "foo  ",
		},
		{
			name:      "ANSI",
			str:       "\x1b[31mError\x1b[0m",
			cols:      8,
			wantLeft:  "   \x1b[31mError\x1b[0m",
			wantRight: "\x1b[31mError\x1b[0m   ",
		},
		{
			name:      "Wide",
			str:       "❌❌❌",
			cols:      10,
			wantLeft:  "    ❌❌❌",
			wantRight: "❌❌❌    ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotLeft := ui.PadLeft(tc.str, tc.cols)
			if gotLeft != tc.wantLeft {
				t.Errorf("PadLeft():  got %q, want %q", gotLeft, tc.wantLeft)
			}
			gotRight := ui.PadRight(tc.str, tc.cols)
			if gotRight != tc.wantRight {
				t.Errorf("PadRight(): got %q, want %q", gotRight, tc.wantRight)
			}
		})
	}
}

func ExamplePrefix() {
	multiline := "hello\n\nworld"
	prefixed := ui.Prefix(multiline, ">>")
	fmt.Println(prefixed)
	// Output:
	// >>hello
	// >>
	// >>world
}

func TestPrefix(t *testing.T) {
	tests := []struct {
		name   string
		str    string
		prefix string
		want   string
	}{
		{
			name:   "Empty",
			str:    "",
			prefix: ">",
			want:   "",
		},
		{
			name:   "SingleLine",
			str:    "foo",
			prefix: ">",
			want:   ">foo",
		},
		{
			name:   "MultiLine",
			str:    "foo\nbar\nbaz",
			prefix: ">",
			want:   ">foo\n>bar\n>baz",
		},
		{
			name:   "Spaces",
			str:    "foo\n\nbar",
			prefix: ">",
			want:   ">foo\n>\n>bar",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ui.Prefix(tc.str, tc.prefix)
			if got != tc.want {
				t.Errorf(`Output does not match.
Got:
%s
Want:
%s`, hex.Dump([]byte(got)), hex.Dump([]byte(tc.want)))
			}
		})
	}
}

func ExamplePad() {
	padding := ui.Padding{
		Left: 2,
	}
	in := "hello\nworld"
	out := ui.Pad(in, padding)
	fmt.Println(out)
	// Output:
	//   hello
	//   world
}

func TestPad(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		padding ui.Padding
		want    string
	}{
		{
			name:    "Pad",
			input:   "foo",
			padding: ui.Padding{Top: 1, Bottom: 2, Left: 3, Right: 4},
			want:    "\n   foo    \n\n",
		},
		{
			name:    "Multiline",
			input:   "foo\nbar\nbaz",
			padding: ui.Padding{Top: 1, Bottom: 1, Left: 1, Right: 1},
			want:    "\n foo \n bar \n baz \n",
		},
		{
			name:    "Empty",
			input:   "",
			padding: ui.Padding{Top: 1, Bottom: 1, Left: 1, Right: 1},
			want:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ui.Pad(tc.input, tc.padding)
			if got != tc.want {
				t.Errorf(`Output does not match.
Got:
%s
Want:
%s`, showWS(got), showWS(tc.want))
			}
		})
	}
}

// showWS replaces whitespace characters with a visible characters
func showWS(s string) string {
	var out strings.Builder
	for _, c := range s {
		if c == '\n' {
			out.WriteString("↵\n")
			continue
		}
		if c == ' ' {
			out.WriteString("·")
			continue
		}
		out.WriteRune(c)
	}
	return out.String()
}
