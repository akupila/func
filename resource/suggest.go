package resource

import "github.com/agext/levenshtein"

func suggest(options []string, want string) (string, bool) {
	for _, suggestion := range options {
		dist := levenshtein.Distance(want, suggestion, nil)
		if dist < 3 { // threshold determined experimentally
			return suggestion, true
		}
	}
	return "", false
}
