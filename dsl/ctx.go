/*
 * Copyright 2021 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package dsl

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

var (
	// DefaultLogger is the default logger.
	DefaultLogger Logger = &GoLogger{}

	// DefaultLogLevel is the default log level.
	//
	// ToDo: Use an enum.
	DefaultLogLevel = "info"
)

// Ctx includes a context.Context, logging specifications, and some
// directories for various file inclusions.
type Ctx struct {
	context.Context
	Logger
	IncludeDirs []string
	Dir         string
	LogLevel    string

	// Redact, if true, will redact substrings that match
	// Redactions in logging output.
	Redact bool

	Redactions map[string]*regexp.Regexp
}

// NewCtx build a new dsl.Ctx
func NewCtx(ctx context.Context) *Ctx {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Ctx{
		Context:     ctx,
		Logger:      DefaultLogger,
		LogLevel:    DefaultLogLevel,
		IncludeDirs: make([]string, 0, 1),
		Dir:         ".",
		Redactions:  make(map[string]*regexp.Regexp),
	}
}

// WithCancel builds a new dsl.Ctx with a cancel function.
func (c *Ctx) WithCancel() (*Ctx, func()) {
	ctx, cancel := context.WithCancel(c.Context)
	return &Ctx{
		Context:     ctx,
		Logger:      DefaultLogger,
		LogLevel:    c.LogLevel,
		IncludeDirs: c.IncludeDirs,
		Dir:         c.Dir,
		Redact:      c.Redact,
		Redactions:  c.Redactions, // not copying
	}, cancel
}

// WithTimeout builds a new dsl.Ctx with a timeout function.
func (c *Ctx) WithTimeout(d time.Duration) (*Ctx, func()) {
	ctx, cancel := context.WithTimeout(c.Context, d)
	return &Ctx{
		Context:     ctx,
		Logger:      DefaultLogger,
		LogLevel:    c.LogLevel,
		IncludeDirs: c.IncludeDirs,
		Dir:         c.Dir,
		Redact:      c.Redact,
		Redactions:  c.Redactions, // not copying
	}, cancel
}

// SetLogLevel sets the dsl.Ctx LogLevel.
func (c *Ctx) SetLogLevel(level string) error {
	canonical := strings.ToLower(level)
	// No strings.TrimSpace.

	switch canonical {
	case "info", "debug", "none":
	default:
		return fmt.Errorf("Ctx.LogLevel '%s' isn't 'info', 'debug', or 'none'", canonical)
	}
	c.LogLevel = canonical
	return nil
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

// Indf emits a log line starting with a '|' when ctx.LogLevel isn't 'none'.
func (c *Ctx) Indf(format string, args ...interface{}) {
	switch c.LogLevel {
	case "none", "NONE":
	default:
		c.Redactf("| "+format, args...)
	}
}

// Inddf emits a log line starting with a '|' when ctx.LogLevel is 'debug';
//
// The second 'd' is for "debug".
func (c *Ctx) Inddf(format string, args ...interface{}) {
	switch c.LogLevel {
	case "debug", "DEBUG":
		c.Redactf("| "+format, args...)
	}
}

// Warnf emits a log  with a '!' prefix.
func (c *Ctx) Warnf(format string, args ...interface{}) {
	c.Redactf("! "+format, args...)
}

// Logf emits a log line starting with a '>' when ctx.LogLevel isn't 'none'.
func (c *Ctx) Logf(format string, args ...interface{}) {
	switch c.LogLevel {
	case "none", "NONE":
	default:
		c.Redactf("> "+format, args...)
	}
}

// Logdf emits a log line starting with a '>' when ctx.LogLevel is 'debug';
//
// The second 'd' is for "debug".
func (c *Ctx) Logdf(format string, args ...interface{}) {
	switch c.LogLevel {
	case "debug", "DEBUG":
		c.Redactf("> "+format, args...)
	}
}

// Logger is an interface that allows for pluggable loggers.
//
// Used in the Plax Lambda.
type Logger interface {
	Printf(format string, args ...interface{})
}

// GoLogger is just basic Go logging.
type GoLogger struct {
}

// Printf logs
func (l *GoLogger) Printf(format string, args ...interface{}) {
	log.Printf(format, args...)
}
