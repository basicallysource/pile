package server

import (
	"strings"
	"unicode"
)

// Searching as one types (the bins' pieces, sets, parts): the query and
// what is searched are words, and every word of the query must start one.

// words splits text into lowercase words of letters and digits.
func words(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// matches says whether every query word starts one of the words, and whether
// every one is one of them whole.
func matches(query []string, sets ...[]string) (ok, exact bool) {
	exact = true
	for _, q := range query {
		found, whole := false, false
		for _, ws := range sets {
			for _, w := range ws {
				if strings.HasPrefix(w, q) {
					found = true
					whole = whole || w == q
				}
			}
		}
		if !found {
			return false, false
		}
		exact = exact && whole
	}
	return true, exact
}
