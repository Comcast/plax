package dsl

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/Comcast/sheens/match"
)

// RegexpMatch compiles the given pattern (as Go regular expression,
// matches the target against that pattern, and returns the results as
// a list of match.Bindings.
//
// A matched named group becomes a binding. If the name starts with an
// uppercase rune, then the binding variable starts with '?'.
// Otherwise, the variable starts with '?*'
func RegexpMatch(pat string, target interface{}) ([]match.Bindings, error) {
	pat = strings.TrimRight(pat, "\n\r")

	r, err := regexp.Compile(pat)
	if err != nil {
		return nil, err
	}

	s, is := target.(string)
	if !is {
		return nil, fmt.Errorf("RegexpMatch wants a %T target, not a %T", s, target)
	}

	ss := r.FindStringSubmatch(s)

	if ss == nil {
		return nil, nil
	}

	bss := make([]match.Bindings, 1)
	bs := make(match.Bindings)
	bss[0] = bs
	for i, name := range r.SubexpNames() {
		if i == 0 {
			continue
		}
		if name == "" {
			// Unnamed group
			continue
		}

		val := ss[i]

		sym := "?"
		if first, has := FirstRune(name); has {
			if unicode.IsLower(first) {
				sym += "*"
			}
		}
		sym += name

		bs[sym] = val
	}

	return bss, nil
}

// FirstRune returns the first rune or false (if the string has zero
// length).
func FirstRune(s string) (rune, bool) {
	for _, r := range s {
		return r, true
	}
	return ' ', false
}
