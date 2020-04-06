package cli

import (
	"bytes"
	"io"
)

// PrefixWriter wraps an io.Writer with a writer that prefixes every line
// with the given prefix.
type PrefixWriter struct {
	Output io.Writer
	Prefix []byte

	buf bytes.Buffer
}

// Write writes the given p to the underlying writer, prefixing every line
// separated by \n with the prefix. The output will never end with only the
// prefix.
func (pw *PrefixWriter) Write(p []byte) (int, error) {
	n := 0
	for {
		i := bytes.IndexByte(p, '\n')
		last := i < 0
		if last {
			i = len(p)
		} else {
			i++
		}
		line := p[:i]
		p = p[i:]
		if len(line) > 0 {
			if _, err := pw.buf.Write(pw.Prefix); err != nil {
				return n, err
			}
		}
		m, err := pw.buf.Write(line)
		n += m
		if err != nil {
			return n, err
		}
		if last {
			break
		}
	}
	if _, err := pw.buf.WriteTo(pw.Output); err != nil {
		return 0, err
	}
	pw.buf.Reset()
	return n, nil
}
