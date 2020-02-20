package cloudformation

import (
	"bytes"
	"unicode"
)

func isAllowed(r rune) bool {
	if r >= 'a' && r <= 'z' {
		return true
	}
	if r >= 'A' && r <= 'Z' {
		return true
	}
	if r >= '0' && r <= '9' {
		return true
	}
	return false
}

func resourceName(input string) string {
	var buf bytes.Buffer
	upper := true
	for _, r := range input {
		if !isAllowed(r) {
			upper = true
			continue
		}
		if upper {
			r = unicode.ToUpper(r)
		}
		buf.WriteRune(r)
		upper = false
	}
	return buf.String()
}
