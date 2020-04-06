package cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

// LogLevel defines the log level to use.
type LogLevel int

// Log levels:
const (
	Info LogLevel = iota
	Debug
	Trace
)

var allLevels LogLevel = -1

// StdLogger is a logger that writes to the given output without ANSI movement
// escape codes.
type StdLogger struct {
	Output io.Writer
	Level  LogLevel
}

func (l *StdLogger) output(level LogLevel, format string, args []interface{}) {
	if l.Level < level {
		return
	}
	msg := fmt.Sprintf(format, args...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	if _, err := fmt.Fprint(l.Output, msg); err != nil {
		panic(err)
	}
}

// Errorf writes an error level log message.
func (l *StdLogger) Errorf(format string, args ...interface{}) { l.output(allLevels, format, args) }

// Warningf writes a warning level log message.
func (l *StdLogger) Warningf(format string, args ...interface{}) { l.output(allLevels, format, args) }

// Infof writes an info level log message.
func (l *StdLogger) Infof(format string, args ...interface{}) { l.output(Info, format, args) }

// Debugf writes a debug level log message.
func (l *StdLogger) Debugf(format string, args ...interface{}) { l.output(Debug, format, args) }

// Tracef writes a trace level log message.
func (l *StdLogger) Tracef(format string, args ...interface{}) { l.output(Trace, format, args) }

// WithPrefix creates a new logger that prefixes every log line with the given
// prefix. In case the output is already prefixed, the parent prefix appears
// first in the output.
func (l *StdLogger) WithPrefix(prefix string) PrefixLogger {
	out := l.Output
	for {
		pw, ok := out.(*PrefixWriter)
		if !ok {
			break
		}
		out = pw.Output
		prefix = string(pw.Prefix) + prefix
	}
	return &StdLogger{
		Level: l.Level,
		Output: &PrefixWriter{
			Output: out,
			Prefix: []byte(prefix),
		},
	}
}

// Writer returns an io.Writer for the given log level. If the log level is not
// exceeded, any data written is discarded.
func (l *StdLogger) Writer(level LogLevel) io.Writer {
	if l.Level < level {
		return ioutil.Discard
	}
	return l.Output
}
