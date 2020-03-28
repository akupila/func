package cli

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// LogLevel defines the log level to use.
type LogLevel int

// Log levels:
const (
	Error LogLevel = iota
	Info
	Verbose
	Trace
)

// logger writes user-facing log messages.
type logger struct {
	Level LogLevel

	mu     sync.Mutex
	Output io.Writer
}

var startTime = time.Now()

func (l *logger) printDeltaTime() {
	d := time.Since(startTime)
	sec := int(d.Seconds())
	ms := int(d.Milliseconds()) % 1000
	fmt.Fprintf(l.Output, "%d.%03d ", sec, ms)
}

func (l *logger) log(level LogLevel, a ...interface{}) {
	if l == nil || l.Output == nil || level > l.Level {
		return
	}
	l.mu.Lock()
	l.printDeltaTime()
	fmt.Fprint(l.Output, a...)
	l.mu.Unlock()
}

func (l *logger) logln(level LogLevel, a ...interface{}) {
	if l == nil || l.Output == nil || level > l.Level {
		return
	}
	l.mu.Lock()
	l.printDeltaTime()
	fmt.Fprintln(l.Output, a...)
	l.mu.Unlock()
}

func (l *logger) logf(level LogLevel, format string, args ...interface{}) {
	if l == nil || l.Output == nil || level > l.Level {
		return
	}
	l.mu.Lock()
	l.printDeltaTime()
	fmt.Fprintf(l.Output, format, args...)
	l.mu.Unlock()
}

func (l *logger) Error(a ...interface{})                      { l.log(Error, a...) }
func (l *logger) Errorln(a ...interface{})                    { l.logln(Error, a...) }
func (l *logger) Errorf(format string, args ...interface{})   { l.logf(Error, format, args...) }
func (l *logger) Info(a ...interface{})                       { l.log(Info, a...) }
func (l *logger) Infoln(a ...interface{})                     { l.logln(Info, a...) }
func (l *logger) Infof(format string, args ...interface{})    { l.logf(Info, format, args...) }
func (l *logger) Verbose(a ...interface{})                    { l.log(Verbose, a...) }
func (l *logger) Verboseln(a ...interface{})                  { l.logln(Verbose, a...) }
func (l *logger) Verbosef(format string, args ...interface{}) { l.logf(Verbose, format, args...) }
func (l *logger) Trace(a ...interface{})                      { l.log(Trace, a...) }
func (l *logger) Traceln(a ...interface{})                    { l.logln(Trace, a...) }
func (l *logger) Tracef(format string, args ...interface{})   { l.logf(Trace, format, args...) }
