package dsl

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// WantsRedaction reports whether the parameter's value should be
// redacted.
//
// Currently if a parameter starts with "X_" after ignoring special
// characters, then the parameter's value should be redacted.
func WantsRedaction(p string) bool {
	return strings.HasPrefix(strings.Trim(p, "?!*"), "X_")
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

// Redactions is set of patterns that can be redacted by the Redactf
// method.
type Redactions struct {
	// Redact enables or disables redactions.
	//
	// The sketchy field name is for backwards compatibility.
	Redact bool

	// Pattens maps strings representing regular expressions to
	// Repexps.
	Patterns map[string]*regexp.Regexp

	// RWMutex makes this gear safe for concurrent use.
	sync.RWMutex
}

// NewRedactions makes a disabled Redactions.
func NewRedactions() *Redactions {
	return &Redactions{
		Patterns: make(map[string]*regexp.Regexp),
	}
}

// Add compiles the given string as a regular expression and installs
// that regexp as a desired redaction.
func (r *Redactions) Add(pat string) error {
	p, err := regexp.Compile(pat)
	if err == nil {
		r.Lock()
		r.Patterns[pat] = p
		r.Unlock()
	}
	return err
}

// Redactf calls fmt.Sprintf and then redacts the result.
func (r *Redactions) Redactf(format string, args ...interface{}) string {
	s := fmt.Sprintf(format, args...)
	if !r.Redact {
		return s
	}
	r.RLock()
	for _, p := range r.Patterns {
		s = Redact(p, s)
	}
	r.RUnlock()
	return s
}

// AddRedaction compiles the given string as a regular expression and
// installs that regexp as a desired redaction in logging output.
func (c *Ctx) AddRedaction(pat string) error {
	return c.Redactions.Add(pat)
}

// Redactf calls c.Printf with any requested redactions.
func (c *Ctx) Redactf(format string, args ...interface{}) {
	c.Printf("%s", c.Redactions.Redactf(format, args...))
}
