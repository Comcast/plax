package dsl

import (
	"fmt"
	"regexp"
	"strings"
)

// WantsRedaction reports whether the parameter's value should be
// redacted.
//
// Currently if a parameter starts with "X_" after ignoring special
// characters, then the parameter's value should be redacted.
func WantsRedaction(p string) bool {
	return strings.HasPrefix(strings.Trim(p, "?!*"), "X_")
}

// AddRedaction compiles the given string as a regular expression and
// installs that regexp as a desired redaction in logging output.
func (c *Ctx) AddRedaction(pat string) error {
	r, err := regexp.Compile(pat)
	if err != nil {
		return err
	}
	c.Redactions[pat] = r
	return nil
}

// Redact might replace part of s with <redacted> depending on the
// given Regexp.
//
// If the Regexp has no groups, all substrings that match the Regexp
// are redacted.
//
// For each named group with a name starting with "redact", that group
// is redacted (for all matches).
//
// If there are groups but none has a name starting with "redact",
// then the first matching (non-captured) group is redacted.
func Redact(r *regexp.Regexp, s string) string {
	replacement := "<redacted>"
	if r.NumSubexp() == 0 {
		return r.ReplaceAllString(s, replacement)
	}

	var acc string
	for {
		match := r.FindStringSubmatchIndex(s)
		if match == nil {
			acc += s
			break
		}
		var (
			redacted   = false
			names      = r.SubexpNames()
			last       = match[1]
			start, end int
		)
		for i, name := range names {
			// First one is anonymous everything group.
			if strings.HasPrefix(name, "redact") {
				redacted = true
				start, end = match[2*i], match[2*i+1]
				break
			}
		}

		if !redacted {
			// The first group will be redacted.
			start, end = match[2], match[3]
		}

		acc += s[0:start] + replacement + s[end:last]

		s = s[last:]
	}

	return acc
}

// Redactf calls c.Printf with any requested redactions with c.Redact
// is true.
func (c *Ctx) Redactf(format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	if c.Redact {
		for _, r := range c.Redactions {
			s = Redact(r, s)
		}
	}
	c.Printf("%s", s)
}
