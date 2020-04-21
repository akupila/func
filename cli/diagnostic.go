package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/func/func/ui"
	"github.com/hashicorp/hcl/v2"
	"github.com/mitchellh/go-wordwrap"
)

type diagnostic struct {
	*hcl.Diagnostic
	File *hcl.File

	ExtendLines int
}

func (d diagnostic) Render(f ui.Frame) string {
	var out strings.Builder

	out.WriteByte('\n')

	switch d.Severity {
	case hcl.DiagError:
		out.WriteString(ui.Format("Error", ui.Red) + ": ")
	case hcl.DiagWarning:
		out.WriteString(ui.Format("Warning", ui.Yellow) + ": ")
	}
	out.WriteString(d.Summary)
	out.WriteString("\n")

	if d.Detail != "" {
		padRight := 8 // So text doesn't go all the way to the edge; easier to read
		out.WriteString(wordwrap.WrapString(d.Detail, uint(f.Width-padRight)))
		out.WriteString("\n")
	}
	out.WriteString("\n")

	if d.Subject != nil {
		subjectRange := *d.Subject
		contextRange := subjectRange
		if d.Context != nil {
			contextRange = hcl.RangeOver(contextRange, *d.Context)
		}

		if d.File != nil && d.File.Bytes != nil {
			out.WriteString(ui.Format(d.Subject.Filename, ui.Dim, ui.Italic))
			out.WriteString("\n\n")

			src := d.File.Bytes
			sc := hcl.NewRangeScanner(src, d.Subject.Filename, bufio.ScanLines)

			for sc.Scan() {
				lineRange := sc.Range()
				if lineRange.Start.Line < d.Subject.Start.Line-d.ExtendLines {
					// Too early, skip
					continue
				}
				if lineRange.Start.Line > d.Subject.Start.Line+d.ExtendLines {
					// Too late, skip
					continue
				}

				linePrefix := fmt.Sprintf("%4d │ ", lineRange.Start.Line)
				codeWidth := f.Width - ui.StringWidth(linePrefix)

				if !lineRange.Overlaps(contextRange) {
					// Not in context, print dim
					out.WriteString(ui.Format(linePrefix, ui.Dim))
					out.WriteString(wrapCode(string(sc.Bytes()), codeWidth))
					out.WriteString("\n")
					continue
				}

				beforeRange, highlightRange, afterRange := lineRange.PartitionAround(subjectRange)
				out.WriteString(ui.Format(fmt.Sprintf("%4d ", lineRange.Start.Line), ui.Bold))
				out.WriteString(ui.Format("│ ", ui.Dim))
				if highlightRange.Empty() {
					out.WriteString(wrapCode(string(sc.Bytes()), codeWidth))
				} else {
					before := beforeRange.SliceBytes(src)
					highlighted := highlightRange.SliceBytes(src)
					after := afterRange.SliceBytes(src)
					out.WriteString(wrapCode(string(before), codeWidth))
					out.WriteString(ui.Format(wrapCode(string(highlighted), codeWidth), ui.Bold, ui.Underline))
					out.WriteString(wrapCode(string(after), codeWidth))
				}
				out.WriteByte('\n')
			}
		}
	}

	return out.String()
}

func wrapCode(code string, w int) string {
	wrapped := ui.Wrap(code, w)
	var out strings.Builder
	for i, l := range strings.Split(wrapped, "\n") {
		if i > 0 {
			out.WriteString(ui.Format("\n     ┊ ", ui.Dim))
		}
		out.WriteString(l)
	}
	return out.String()
}
