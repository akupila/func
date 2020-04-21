package ui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

// UI provides declarative rendering for a console user interface.
type UI struct {
	// KeepCursor disables hiding the cursor in the middle of a render. By
	// default it is enabled as it may cause the cursor to flash in the output,
	// but can be disabled for tests.
	KeepCursor bool

	target Renderer

	mu  sync.Mutex
	out io.Writer

	input     string        // Rendered input (raw from target)
	prevLines []string      // Previous output, aligned to fit on screen
	framebuf  *bytes.Buffer // Reused buffer during render

	cols, rows int

	frame int
}

// A Renderer generates the user interface as a string to print to the user's
// terminal. The returned string may contain ANSI escape codes for styling.
//
// The Renderer must always render the entire UI.
type Renderer interface {
	Render(frame Frame) string
}

// New creates a new UI renderer.
func New(output io.Writer, target Renderer) *UI {
	r := &UI{
		out:      output,
		target:   target,
		framebuf: &bytes.Buffer{},
	}
	r.resize()
	r.handleResize()
	return r
}

func (r *UI) handleResize() {
	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)
	go func() {
		for range sigwinch {
			r.resize()
		}
	}()
}

func (r *UI) resize() {
	cols, rows, err := terminal.GetSize(0)
	if err != nil {
		r.cols = 80
		r.rows = 25
		return
	}
	r.mu.Lock()
	if r.cols > 0 {
		// Update -> clear screen and re-render
		r.framebuf.Reset()
		w := r.framebuf
		w.WriteString(clearScreen)
		w.WriteString(moveTopLeft)
		r.flush()
		r.prevLines = nil
	}
	r.cols = cols
	r.rows = rows
	r.mu.Unlock()
	r.Render()
}

// A Frame is a single frame passed to a Render().
type Frame struct {
	Number int

	// Number of columns the rendered content should attempt to fit within.
	Width int

	// Rows contains the number of rows in the terminal.
	Cols int
	Rows int
}

// Indent returns a copy of the frame with a smaller width to be passed to a
// sub renderer.
func (f Frame) Indent(cols int) Frame {
	return Frame{
		Number: f.Number,
		Width:  f.Width - cols,
		Cols:   f.Cols,
		Rows:   f.Rows,
	}
}

// Render calls Render on the target and generates a diff to flush to the
// output, attempting to minimize the amount of data to print to the terminal.
//
// In case a line exceeds the allocated space, it is truncated.
func (r *UI) Render() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	frame := Frame{
		Number: r.frame,
		Rows:   r.rows,
		Width:  r.cols,
	}

	r.input = r.target.Render(frame)
	nextLines := strings.Split(r.input, "\n")
	for i, l := range nextLines {
		nextLines[i] = Truncate(l, r.cols, ">")
	}

	r.framebuf.Reset()
	w := r.framebuf

	if !r.KeepCursor {
		w.WriteString(hideCursor)
	}

	hasChanges := false
	cursor := len(r.prevLines)
	for i, next := range nextLines {
		if i >= len(r.prevLines) {
			// No previous line to overwrite
			w.WriteString(next)
			w.WriteByte('\n')
			hasChanges = true
			cursor = i + 1
			continue
		}
		prev := r.prevLines[i]
		if prev == next {
			continue
		}
		hasChanges = true
		d := moveCursor(w, i, cursor)
		if d != 0 {
			w.WriteByte('\r')
		}
		w.WriteString(next)
		if StringWidth(prev) > StringWidth(next) {
			w.WriteString(clearRight)
		}
		w.WriteByte('\n')
		cursor = i + 1
	}
	if !hasChanges {
		return false
	}

	moveCursor(w, len(nextLines), cursor)
	if len(nextLines) < len(r.prevLines) {
		w.WriteString(clearDown)
	}
	if !r.KeepCursor {
		w.WriteString(showCursor)
	}

	r.flush()

	r.prevLines = nextLines
	r.frame++

	return true
}

// flush writes the frame buffer to the output and resets the internal buffer.
func (r *UI) flush() {
	_, err := io.Copy(r.out, r.framebuf)
	if err != nil {
		panic(err)
	}
	r.framebuf.Reset()
}

// moveCursor returns the escape code for moving the cursor to the target line.
// Returns the number of lines changed.
func moveCursor(w io.Writer, target int, cursor int) int {
	if target == cursor {
		return 0
	}
	d := target - cursor
	if d < 0 {
		if d == -1 {
			fmt.Fprint(w, "\x1b[A") // Up 1
			return d
		}
		fmt.Fprintf(w, "\x1b[%dA", -d) // Up n
		return d
	}
	if d == 1 {
		fmt.Fprint(w, "\x1b[B") // Down 1
		return d
	}
	fmt.Fprintf(w, "\x1b[%dB", d) // Down n
	return d
}

const (
	clearRight  = "\x1b[K"
	clearDown   = "\x1b[J"
	clearScreen = "\x1b[2J"
	moveTopLeft = "\x1b[H"
	showCursor  = "\x1b[?25h"
	hideCursor  = "\x1b[?25l"
)
