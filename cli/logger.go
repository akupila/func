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

func (l *Logger) Error(a ...interface{})                      { l.Log(Error, a...) }
func (l *Logger) Errorln(a ...interface{})                    { l.Logln(Error, a...) }
func (l *Logger) Errorf(format string, args ...interface{})   { l.Logf(Error, format, args...) }
func (l *Logger) Info(a ...interface{})                       { l.Log(Info, a...) }
func (l *Logger) Infoln(a ...interface{})                     { l.Logln(Info, a...) }
func (l *Logger) Infof(format string, args ...interface{})    { l.Logf(Info, format, args...) }
func (l *Logger) Verbose(a ...interface{})                    { l.Log(Verbose, a...) }
func (l *Logger) Verboseln(a ...interface{})                  { l.Logln(Verbose, a...) }
func (l *Logger) Verbosef(format string, args ...interface{}) { l.Logf(Verbose, format, args...) }
func (l *Logger) Trace(a ...interface{})                      { l.Log(Trace, a...) }
func (l *Logger) Traceln(a ...interface{})                    { l.Logln(Trace, a...) }
func (l *Logger) Tracef(format string, args ...interface{})   { l.Logf(Trace, format, args...) }
