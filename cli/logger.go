package cli

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Logger struct {
	Level LogLevel
	start time.Time

	mu  sync.Mutex
	out io.Writer
}

type LogLevel int

func NewLogger(level int) *Logger {
	return &Logger{
		Level: LogLevel(level),
		start: time.Now(),
		out:   os.Stderr,
	}
}

const (
	Error LogLevel = iota
	Info
	Verbose
	Trace
)

func (l *Logger) printDeltaTime() {
	d := time.Since(l.start)
	sec := int(d.Seconds())
	ms := int(d.Milliseconds()) % 1000
	fmt.Fprintf(l.out, "%d.%03d ", sec, ms)
}

func (l *Logger) Log(level LogLevel, a ...interface{}) {
	if l == nil || l.out == nil || level > l.Level {
		return
	}
	l.mu.Lock()
	l.printDeltaTime()
	fmt.Fprint(l.out, a...)
	l.mu.Unlock()
}

func (l *Logger) Logln(level LogLevel, a ...interface{}) {
	if l == nil || l.out == nil || level > l.Level {
		return
	}
	l.mu.Lock()
	l.printDeltaTime()
	fmt.Fprintln(l.out, a...)
	l.mu.Unlock()
}

func (l *Logger) Logf(level LogLevel, format string, args ...interface{}) {
	if l == nil || l.out == nil || level > l.Level {
		return
	}
	l.mu.Lock()
	l.printDeltaTime()
	fmt.Fprintf(l.out, format, args...)
	l.mu.Unlock()
}

func (r *Logger) Error(a ...interface{})                      { r.Log(Error, a...) }
func (r *Logger) Errorln(a ...interface{})                    { r.Logln(Error, a...) }
func (r *Logger) Errorf(format string, args ...interface{})   { r.Logf(Error, format, args...) }
func (r *Logger) Info(a ...interface{})                       { r.Log(Info, a...) }
func (r *Logger) Infoln(a ...interface{})                     { r.Logln(Info, a...) }
func (r *Logger) Infof(format string, args ...interface{})    { r.Logf(Info, format, args...) }
func (r *Logger) Verbose(a ...interface{})                    { r.Log(Verbose, a...) }
func (r *Logger) Verboseln(a ...interface{})                  { r.Logln(Verbose, a...) }
func (r *Logger) Verbosef(format string, args ...interface{}) { r.Logf(Verbose, format, args...) }
func (r *Logger) Trace(a ...interface{})                      { r.Log(Trace, a...) }
func (r *Logger) Traceln(a ...interface{})                    { r.Logln(Trace, a...) }
func (r *Logger) Tracef(format string, args ...interface{})   { r.Logf(Trace, format, args...) }
