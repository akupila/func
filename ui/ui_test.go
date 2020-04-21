package ui_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/func/func/ui"
)

func TestRenderer_Render(t *testing.T) {
	tests := []struct {
		name   string
		target ui.Renderer
		want   []string
	}{
		{
			name: "Basic",
			target: RenderFunc(func(f ui.Frame) string {
				return "foo"
			}),
			want: []string{
				"foo\n",
			},
		},
		{
			name: "Identical",
			target: RenderFunc(func(f ui.Frame) string {
				return "foo"
			}),
			want: []string{
				"foo\n", // First render
				"",      // No changes
			},
		},
		{
			name: "Update",
			target: RenderFunc(func(f ui.Frame) string {
				return fmt.Sprintf("%d", f.Number)
			}),
			want: []string{
				"0\n",
				"\x1b[A\r1\n",
			},
		},
		{
			name: "UpdateFirst",
			target: RenderFunc(func(f ui.Frame) string {
				return fmt.Sprintf("%d\nfoo", f.Number)
			}),
			want: []string{
				"0\nfoo\n",
				"\x1b[2A\r1\n\x1b[B",
			},
		},
		{
			name: "UpdateMultiple",
			target: RenderFunc(func(f ui.Frame) string {
				return fmt.Sprintf("foo\n%d\nbar\nbaz\n%d", f.Number, f.Number)
			}),
			want: []string{
				"foo\n0\nbar\nbaz\n0\n",
				"\x1b[4A\r1\n\x1b[2B\r1\n",
			},
		},
		{
			name: "FizzBuzz",
			target: RenderFunc(func(f ui.Frame) string {
				n := f.Number + 1
				if n%3 == 0 {
					return "fizz"
				}
				if n%5 == 0 {
					return "buzz"
				}
				return strconv.Itoa(n)
			}),
			want: []string{
				"1\n",
				"\x1b[A\r2\n",
				"\x1b[A\rfizz\n",
				"\x1b[A\r4\x1b[K\n",
				"\x1b[A\rbuzz\n",
				"\x1b[A\rfizz\n",
				"\x1b[A\r7\x1b[K\n",
				"\x1b[A\r8\n",
			},
		},
		{
			name: "MoreLines",
			target: RenderFunc(func(f ui.Frame) string {
				return strings.Repeat("foo\n", 1+f.Number)
			}),
			want: []string{
				"foo\n\n",
				"\x1b[A\rfoo\n\n",
				"\x1b[A\rfoo\n\n",
			},
		},
		{
			name: "FewerLines",
			target: RenderFunc(func(f ui.Frame) string {
				return strings.Repeat("x\n", 5-f.Number)
			}),
			want: []string{
				"x\nx\nx\nx\nx\n\n",
				"\x1b[2A\r\x1b[K\n\x1b[J",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got bytes.Buffer
			r := ui.New(&got, tc.target)
			r.KeepCursor = true
			for i, want := range tc.want {
				got.Reset()
				changed := r.Render()
				if changed != (want != "") {
					t.Errorf("Render returned %t, want %t", changed, want != "")
				}
				if got.String() != want {
					t.Errorf(`Render %d output does not match.
Got %d bytes:
%s
Want %d bytes:
%s`, i, got.Len(), hex.Dump(got.Bytes()), len(want), hex.Dump([]byte(want)))
				}
			}
		})
	}
}

type RenderFunc func(frame ui.Frame) string

func (fn RenderFunc) Render(frame ui.Frame) string { return fn(frame) }

func ExampleFrame_Indent() {
	frame := ui.Frame{Width: 20}
	sub := frame.Indent(4)
	fmt.Println(sub.Width)
	// Output: 16
}
