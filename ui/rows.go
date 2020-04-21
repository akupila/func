package ui

import "strings"

// Rows joins all non-empty rows with new line.
func Rows(rows ...string) string {
	rows = excludeEmpty(rows...)
	return strings.Join(rows, "\n")
}

// Cols joins all non-empty columns with a space.
func Cols(cols ...string) string {
	cols = excludeEmpty(cols...)
	return strings.Join(cols, " ")
}

func excludeEmpty(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if len(v) > 0 {
			out = append(out, v)
		}
	}
	return out
}
