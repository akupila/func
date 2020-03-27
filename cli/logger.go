package cli

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Logger writes user-facing log messages.
type Logger struct {
	Level LogLevel
	start time.Time

	mu  sync.Mutex
	out io.Writer
}

// LogLevel defines the log level to use.
type LogLevel int

// Log levels:
const (
	Error LogLevel = iota
	Info
	Verbose
	Trace
)

// NewLogger creates a new logger that writes to stderr.
func NewLogger(level int) *Logger {
	return &Logger{
		Level: LogLevel(level),
		start: time.Now(),
		out:   os.Stderr,
	}
}

func (l *Logger) printDeltaTime() {
	d := time.Since(l.start)
	sec := int(d.Seconds())
	ms := int(d.Milliseconds()) % 1000
	fmt.Fprintf(l.out, "%d.%03d ", sec, ms)
}

func (l *Logger) log(level LogLevel, a ...interface{}) {
	if l == nil || l.out == nil || level > l.Level {
		return
	}
	l.mu.Lock()
	l.printDeltaTime()
	fmt.Fprint(l.out, a...)
	l.mu.Unlock()
}

func (l *Logger) logln(level LogLevel, a ...interface{}) {
	if l == nil || l.out == nil || level > l.Level {
		return
	}
	l.mu.Lock()
	l.printDeltaTime()
	fmt.Fprintln(l.out, a...)
	l.mu.Unlock()
}

func (l *Logger) logf(level LogLevel, format string, args ...interface{}) {
	if l == nil || l.out == nil || level > l.Level {
		return
	}
	l.mu.Lock()
	l.printDeltaTime()
	fmt.Fprintf(l.out, format, args...)
	l.mu.Unlock()
}

func (l *Logger) Error(a ...interface{})                      { l.log(Error, a...) }               // nolint: golint
func (l *Logger) Errorln(a ...interface{})                    { l.logln(Error, a...) }             // nolint: golint
func (l *Logger) Errorf(format string, args ...interface{})   { l.logf(Error, format, args...) }   // nolint: golint
func (l *Logger) Info(a ...interface{})                       { l.log(Info, a...) }                // nolint: golint
func (l *Logger) Infoln(a ...interface{})                     { l.logln(Info, a...) }              // nolint: golint
func (l *Logger) Infof(format string, args ...interface{})    { l.logf(Info, format, args...) }    // nolint: golint
func (l *Logger) Verbose(a ...interface{})                    { l.log(Verbose, a...) }             // nolint: golint
func (l *Logger) Verboseln(a ...interface{})                  { l.logln(Verbose, a...) }           // nolint: golint
func (l *Logger) Verbosef(format string, args ...interface{}) { l.logf(Verbose, format, args...) } // nolint: golint
func (l *Logger) Trace(a ...interface{})                      { l.log(Trace, a...) }               // nolint: golint
func (l *Logger) Traceln(a ...interface{})                    { l.logln(Trace, a...) }             // nolint: golint
func (l *Logger) Tracef(format string, args ...interface{})   { l.logf(Trace, format, args...) }   // nolint: golint
