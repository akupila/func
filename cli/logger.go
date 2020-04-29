package cli

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/func/func/ui"
	"github.com/hashicorp/hcl/v2"
	"github.com/mattn/go-runewidth"
)

type logger struct {
	Verbose bool
	ui.Stack
}

func newLogger(out io.Writer, verbose bool) *logger {
	log := &logger{
		Verbose: verbose,
	}
	ui := ui.New(out, log)

	go func() {
		for {
			ui.Render()
			time.Sleep(time.Second / 60)
		}
	}()

	return log
}

func (l *logger) Infof(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	l.Stack.Push(infoMsg(str))
}

func (l *logger) Verbosef(format string, args ...interface{}) {
	if !l.Verbose {
		return
	}
	str := fmt.Sprintf(format, args...)
	str = ui.Format(str, ui.Dim)
	l.Stack.Push(verboseMsg(str))
}

func (l *logger) Errorf(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	l.Stack.Push(errorMsg(str))
}

func (l *logger) Render(frame ui.Frame) string {
	padLeft := 2
	frame.Width -= padLeft
	return ui.Pad(
		l.Stack.Render(frame),
		ui.Padding{Top: 1, Bottom: 1, Left: padLeft},
	)
}

func (l *logger) Step(name string) *logStep {
	s := newStep(name, l.Verbose)
	s.Icon = true
	l.Stack.Push(s)
	return s
}

type logStep struct { // nolint: maligned
	ui.Stack

	Name    string
	Verbose bool
	Icon    bool
	Prefix  string

	mu      sync.Mutex
	started time.Time
	stopped *time.Time
	err     bool
}

func newStep(name string, verbose bool) *logStep {
	return &logStep{
		Name:    name,
		Verbose: verbose,
		started: time.Now(),
	}
}

var (
	iconWidth = 2
	iconErr   = ui.Format(runewidth.FillRight("❌", iconWidth), ui.Red)
	iconDone  = ui.Format(runewidth.FillRight("✅", iconWidth), ui.Green)
)

func (s *logStep) Render(f ui.Frame) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	cols := make([]string, 0, 3)

	prefix := s.Prefix
	indent := ui.StringWidth(prefix)
	if s.Icon {
		switch {
		case s.err:
			cols = append(cols, iconErr)
		case s.stopped != nil:
			cols = append(cols, iconDone)
		default:
			var spinner string
			if time.Since(s.started) > 250*time.Millisecond {
				spinner = ui.Format(lineIndicator(f), ui.Green, ui.Dim)
			} else {
				spinner = ""
			}
			cols = append(cols, ui.PadRight(spinner, iconWidth))
		}

		indent += iconWidth + 1
		prefix = ui.PadRight(prefix, indent)
	}
	f.Width -= indent

	name := ui.Format(s.Name, ui.Bold)
	cols = append(cols, name)

	if s.stopped != nil {
		d := s.stopped.Sub(s.started)
		cols = append(cols, durStr(d))
	} else if !s.err {
		d := time.Since(s.started).Truncate(time.Second)
		cols = append(cols, durStr(d))
	}

	str := ui.Rows(
		ui.Cols(cols...),
		ui.Prefix(
			s.Stack.Render(f),
			prefix,
		),
	)
	return str
}

func durStr(d time.Duration) string {
	switch {
	case d < 100*time.Millisecond:
		return ""
	case d > 10*time.Second:
		d = d.Round(100 * time.Millisecond)
	case d > time.Second:
		d = d.Round(10 * time.Millisecond)
	default:
		d = d.Round(time.Millisecond)
	}
	return ui.Format("("+d.String()+")", ui.Dim)
}

func (s *logStep) Done() {
	s.mu.Lock()
	now := time.Now()
	s.stopped = &now
	s.mu.Unlock()
}

func (s *logStep) Step(name string) *logStep {
	sub := newStep(name, s.Verbose)
	sub.Prefix = "  "
	s.Stack.Push(sub)
	return sub
}

func (s *logStep) Infof(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	s.Stack.Push(infoMsg(str))
}

func (s *logStep) Verbosef(format string, args ...interface{}) {
	if !s.Verbose {
		return
	}
	str := fmt.Sprintf(format, args...)
	str = ui.Format(str, ui.Dim)
	s.Stack.Push(verboseMsg(str))
}

func (s *logStep) Errorf(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	s.Stack.Push(errorMsg(str))
	s.mu.Lock()
	s.err = true
	s.mu.Unlock()
}

type infoMsg string

func (msg infoMsg) Render(f ui.Frame) string {
	return ui.Wrap(string(msg), f.Width-8)
}

type verboseMsg string

func (msg verboseMsg) Render(f ui.Frame) string {
	m := ui.Wrap(string(msg), f.Width-8)
	return ui.Format(m, ui.Dim)
}

type errorMsg string

func (msg errorMsg) Render(f ui.Frame) string {
	m := ui.Wrap(string(msg), f.Width-8)
	return ui.Format("Error", ui.Red, ui.Bold) + ": " + m
}

func (s *logStep) PrintDiags(diags hcl.Diagnostics, files map[string]*hcl.File) {
	for _, d := range diags {
		s.Stack.Push(diagnostic{
			Diagnostic:  d,
			File:        files[d.Subject.Filename],
			ExtendLines: 3,
		})
	}
	if diags.HasErrors() {
		s.mu.Lock()
		s.err = true
		s.mu.Unlock()
	}
}

type window struct {
	strings.Builder
	MaxLines int
}

func (w *window) Render(f ui.Frame) string {
	raw := strings.TrimSpace(w.String())
	lines := strings.Split(raw, "\n")
	to := len(lines) - 1
	from := to - w.MaxLines
	if from < 0 {
		from = 0
	}
	return ui.Format(strings.Join(lines[from:to], "\n"), ui.Dim)
}
