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

package subst

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/itchyny/gojq"
	"gopkg.in/yaml.v1"
)

var (
	// DefaultDelimiters are the default opening and closing
	// deliminers for a pipe expression.
	DefaultDelimiters = "{}"

	// DefaultSerialization is the default serialization for a
	// pipe expression when the Subber doesn't think it knows
	// better.
	DefaultSerialization = "json"

	// DefaultLimit is the default limit for the number of
	// recursive substitution calls a Subber will make.
	//
	// If you hit this limit intentionally, then that's pretty
	// impressive.  However, probably not something to brag about.
	DefaultLimit = 10
)

// Proc is a "processor" that a Subber can call.
//
// A Proc computes an entire replacement for the given string.
//
// Classic example is deserialization a JSON string input, doing
// structural replacement of bindings, and then reserializing.  See
// Bindings.UnmarshalBind, which does exactly that.
type Proc func(*Ctx, string) (string, error)

// Subber performs string-oriented substitutions based on a syntax
// like {VAR | PROC | SERIALIZATION}.
type Subber struct {
	// Procs is a list of processors that are called during
	// (recursive) substitution processing.
	Procs []Proc

	// Limit is the maximum number of recursive Sub calls.
	//
	// Default is DefaultLimit.
	Limit int

	// DefaultSerialization is the serialization when an explicit
	// serialization isn't provided and the Subber doesn't think
	// it knows better (via scruffy heuristics).
	DefaultSerialization string

	// pipePattern is the (compiled) Regexp that included the
	// delimiters provided to NewSubber.
	pipePattern *regexp.Regexp

	// ToDo: We might want a switch that controls whether the
	// Subber returns an error when it encounters a string-based
	// VAR that is not bound.  There are some (exotic?) situations
	// where an unbound VAR is a string-based substitution
	// expression should be left as is and without and error.
	// Note that a structured unbound VAR is usually fine in a
	// 'recv' context because the point can be to bind that var
	// via pattern matching.  A switch could be helpful to get the
	// right behavior in different contexts.
}

// NewSubber makes a new Subber with the pipe expression delimiters
// given by the first and second runes of the given string.
//
// Uses DefaultDelimiters by default.
//
// Uses DefaultSerialization and DefaultLimit.
func NewSubber(delimeters string) (*Subber, error) {
	if delimeters == "" {
		delimeters = DefaultDelimiters
	}

	var left, right, zero rune
	var i int
	for _, r := range delimeters {
		switch i {
		case 0:
			left = r
			i++
		case 1:
			right = r
			i++
		default:
			return nil, fmt.Errorf("need exactly two runes for delimiters")
		}
	}

	if left == zero || right == zero {
		return nil, fmt.Errorf("need exactly two runes for delimiters")
	}

	exp := fmt.Sprintf(
		// Phantom key for splicing value.  Can be either "{}" or "".
		`("(?:%s *%s)?" *: *)?`+
			`("?)`+ // Optional opening quote
			`%s`+

			// Source (variable) name.  Leading ? or @ not
			// required.  Allowed characters as follows.
			// Note that the var must have at least one of
			// those legal characters.  The set of legal
			// characters is probably too large.  ToDo:
			// Reconsider, but note that any reduction
			// would be a breaking change.
			//
			// Note that this syntax is only for
			// brace/pipe/string-based substitutions and
			// not structural bindings subsitutions.
			`([?@]?[-.a-zA-Z0-9!_?]+)`+

			`( *\| *(.*?))?`+ // optional processor (e.g., jq)
			`( *\| *([a-z]*[@$]?))?`+ // optional serialization and interpolation
			`%s`+
			`("?)`, // Optional closing quote
		string(left), string(right),
		string(left), string(right))

	re, err := regexp.Compile(exp)
	if err != nil {
		return nil, err
	}

	s := &Subber{
		pipePattern:          re,
		Procs:                make([]Proc, 0, 4),
		Limit:                DefaultLimit,
		DefaultSerialization: DefaultSerialization,
	}

	return s, nil
}

// Copy makes a deep copy of a Subber.
func (b *Subber) Copy() *Subber {
	ps := make([]Proc, len(b.Procs))
	copy(ps, b.Procs)
	return &Subber{
		pipePattern:          b.pipePattern,
		Procs:                ps,
		Limit:                b.Limit,
		DefaultSerialization: b.DefaultSerialization,
	}
}

// WithProcs returns a copied Subber with the given Procs added.
func (b *Subber) WithProcs(ps ...Proc) *Subber {
	b = b.Copy()
	for _, p := range ps {
		b.Procs = append(b.Procs, p)
	}
	return b
}

// readFile searches ctx.IncludeDirs to find the file with the give
// name.
func readFile(ctx *Ctx, name string) ([]byte, error) {
	for _, dir := range ctx.IncludeDirs {
		path := filepath.Join(dir, name)
		bs, err := ioutil.ReadFile(path)
		if err == nil {
			return bs, nil
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("file '%s' not found in include paths %v", name, ctx.IncludeDirs)
}

func parseSerialization(s string) (serial string, spliceMode string, err error) {
	// ToDo: Make these serializations pluggable (sort of like Procs).

	s = strings.TrimSpace(s)
	spliceMode = "inplace"
	switch s {
	case "":
		serial = "default"
	case "string", "text":
		serial = "string"
	case "string$", "text$":
		serial = "string"
		spliceMode = "array"
	case "trim":
		serial = "trim"
	case "json":
		serial = "json"
	case "json@":
		serial = "json"
		spliceMode = "map"
	case "json$":
		serial = "json"
		spliceMode = "array"
	default:
		err = fmt.Errorf("unknown serialization: %#v", s)
	}
	return
}

func serialJSON(x interface{}, spliceMode string) (string, error) {
	var acc string
	switch spliceMode {
	case "inplace", "":
		js, err := json.Marshal(&x)
		if err != nil {
			return "", err
		}
		acc = string(js)
	case "array":
		xs, is := x.([]interface{})
		if !is {
			return "", fmt.Errorf("%#v isn't an %T for splice mode %s", x, xs, spliceMode)
		}
		js, err := json.Marshal(&xs)
		if err != nil {
			return "", err
		}
		acc = string(js[1 : len(js)-1])
	case "map":
		m, is := x.(map[string]interface{})
		if !is {
			return "", fmt.Errorf("%#v isn't an %T for splice mode %s", x, m, spliceMode)
		}
		js, err := json.Marshal(&m)
		if err != nil {
			return "", err
		}
		acc = string(js[1 : len(js)-1])
	default:
		return "", fmt.Errorf("unknown splice mode %#v", spliceMode)
	}

	return acc, nil
}

func serialString(x interface{}, spliceMode string) (string, error) {
	var acc string
	switch spliceMode {
	case "inplace", "":
		s, is := x.(string)
		if !is {
			// With shame, we'll try to JSON-serialize.
			//
			// ToDo: Reconsider.
			js, err := json.Marshal(&x)
			if err != nil {
				return "", err
			}
			return string(js), nil
		}
		acc = s
	case "array": // Doubtful
		xs, is := x.([]interface{})
		if !is {
			return "", fmt.Errorf("%#v isn't an %T for string splice mode %s", x, xs, spliceMode)
		}
		ss := make([]string, len(xs))
		for i, x := range xs {
			s, is := x.(string)
			if !is {
				return "", fmt.Errorf("%#v isn't an %T within string splice mode %s", x, xs, spliceMode)
			}
			ss[i] = s
		}
		acc = strings.Join(ss, ",")
	default:
		return "", fmt.Errorf("unknown string splice mode %#v", spliceMode)
	}

	return acc, nil
}

func serial(x interface{}, serialization, spliceMode string) (string, error) {
	switch serialization {
	case "json":
		return serialJSON(x, spliceMode)
	case "text", "string":
		return serialString(x, spliceMode)
	case "trim":
		s, err := serialString(x, spliceMode)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(s), nil
	default:
		return "", fmt.Errorf("unknown serialization %s", serialization)
	}
}

type pipe struct {

	// submatch is the result of the raw regexp match.
	submatch []string

	// source is the "VAR", which is either a key in Bindings or
	// "@FILENAME".
	source string

	// proc is the parsed processor (if any).
	proc func(interface{}) (interface{}, error)

	// leftQuote and rightQuote are the double-quotes surrounding
	// a pipe expression.  These quotes are sometimes dropped
	// during substitution.
	leftQuote, rightQuote string

	// serial, which should probably be an enum (ToDo), indicates
	// how to serialize a value before putting it into a string.
	serial string

	// spliceMode, which should be an enum (ToDo), indicates
	// if/how to splice a value into a larger structure.  Legal
	// values: "map", "array", or "".
	spliceMode string

	// pairKeyColon is part of the match that has the leading
	// colon with the pipe is a map value.
	pairKeyColon string
}

func unescapeQuotes(s string) string {
	return strings.ReplaceAll(s, `\"`, `"`)
}

func (b *Subber) parsePipe(ctx *Ctx, s string) (*pipe, error) {
	ss := b.pipePattern.FindStringSubmatch(s)
	if ss == nil {
		return nil, nil
	}
	pairKey := ss[1]

	// Get the delimiters if provided.
	copy(ss[1:], ss[2:])
	leftQuote := ss[1]
	copy(ss[1:], ss[2:])
	rightQuote := ss[len(ss)-1]

	// Find the (trailing) serialization (if any).  We should
	// probably not use '|' to prefix a serialization since that
	// symbol would be ambigious.

	var serial, spliceMode string
	for i := len(ss) - 1; 0 < i; i-- {
		maybe := ss[i]
		if maybe == "" {
			continue
		}
		var err error
		if serial, spliceMode, err = parseSerialization(maybe); err == nil {
			// We found the explicit serialization.
			ss[i] = ""
			break
		}
	}

	// The processor (if any) is here.
	proc := ss[3]

	p := &pipe{
		submatch:     ss,
		source:       ss[1], // The "VAR"
		pairKeyColon: pairKey,
		leftQuote:    leftQuote,
		rightQuote:   rightQuote,
	}

	if proc != "" {
		parts := strings.SplitN(proc, " ", 2)
		switch parts[0] {
		case "jq":
			if !strings.HasPrefix(proc, "jq ") {
				return nil, fmt.Errorf("bad jq query in '%s'", proc)
			}
			q, err := gojq.Parse(unescapeQuotes(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("jq parse error: %s on %s", err, parts[1])
			}
			p.proc = func(x interface{}) (interface{}, error) {
				i := q.Run(x)
				// Only consider the first thing returned.
				// ToDo: Elaborate.
				y, _ := i.Next()
				if err, is := y.(error); is {
					return "", err
				}
				return y, nil
			}
		case "js":
			p.proc = func(x interface{}) (interface{}, error) {
				src := parts[1]
				env := map[string]interface{}{
					"$": x,
				}
				return JSExec(ctx, src, env)
			}
		default:
			return nil, fmt.Errorf("unknown processor '%s'", parts[0])
		}
	}

	if serial == "default" || serial == "" {
		// ToDo: consider more/better heuristics here.
		if !p.quoted() {
			serial = "text"
		} else {
			serial = b.DefaultSerialization
		}
	}

	p.serial = serial
	p.spliceMode = spliceMode

	return p, nil
}

var filenamePattern = regexp.MustCompile(`^@(.*\.([a-zA-Z]+))$`)

func parseFilename(s string) (name string, ext string, ok bool) {
	ss := filenamePattern.FindStringSubmatch(s)
	if ss == nil {
		return
	}
	name = ss[1]
	ext = ss[2]
	ok = true
	return
}

func deserialize(ctx *Ctx, bs []byte, syntax string) (interface{}, error) {
	var x interface{}
	var err error
	switch syntax {
	case "json":
		err = json.Unmarshal(bs, &x)
	case "yaml":
		err = yaml.Unmarshal(bs, &x)

		// The map[interface{}]interface{} YAML
		// deserialization rears its ugly head again.
		//
		// subst_test.go:32: %!v(PANIC=Error method: invalid
		// value: map[enjoys:tacos])
		//
		// From map[interface {}]interface {} map[interface
		// {}]interface {}{"enjoys":"tacos"}

		if err == nil {
			x, err = StringKeys(x)
		}

	case "string", "txt", "text", "":
		x = string(bs)
	default:
		return nil, fmt.Errorf("unknown serialization syntax %s", syntax)
	}
	return x, err
}

func (p *pipe) process(ctx *Ctx, bs Bindings) (string, error) {
	v, have := bs[p.source]
	if !have {
		ctx.trf("pipe.Process %s", p.source)
		if name, syntax, ok := parseFilename(p.source); ok {
			bs, err := readFile(ctx, name)
			if err != nil {
				return "", err
			}
			x, err := deserialize(ctx, bs, syntax)
			if err != nil {
				return "", err
			}
			v = x
		} else {
			return "", fmt.Errorf("source '%s' variable not bound in '%s'", p.source, p.submatch[0])
		}
	}

	if p.proc != nil {
		x, err := p.proc(v)
		if err != nil {
			return "", err
		}
		v = x
	}

	s, err := serial(v, p.serial, p.spliceMode)

	if err != nil {
		return "", err
	}

	if p.pairKeyColon != "" {
		if p.spliceMode == "map" {
		} else {
			s = p.pairKeyColon + s
		}
	}

	// We drop a pair of double quotes, but we restore other
	// configurations.
	if !p.quoted() {
		s = p.leftQuote + s + p.rightQuote
	}

	return s, err
}

func (p *pipe) quoted() bool {
	return p.leftQuote == `"` && p.rightQuote == `"`
}

func (b *Subber) pipeSub(ctx *Ctx, bs Bindings, s string) (string, error) {
	var e error
	y := b.pipePattern.ReplaceAllStringFunc(s, func(s string) string {
		p, err := b.parsePipe(ctx, s)
		if err != nil {
			e = err
			return fmt.Sprintf(`<error %s>`, err)
		}
		if p == nil {
			e = fmt.Errorf("unable to parse '%s'", s)
			return fmt.Sprintf(`<error %s>`, err)
		}
		got, err := p.process(ctx, bs)
		if err != nil {
			e = err
			return fmt.Sprintf(`<error %s>`, err)
		}

		return got
	})
	if e != nil {
		return "", e
	}
	return y, nil
}

// Sub performs recursive, string-based Bindings substitutions on the
// given input string.
func (b *Subber) Sub(ctx *Ctx, bs Bindings, s string) (string, error) {
	var (
		// s0 is just for an error message (if required).
		s0 = s

		// acc remembers all previous values to detect loops.
		acc = make([]string, 0, b.Limit)
	)

	for i := 0; i < b.Limit; i++ {
		ctx.trf("Subber.Sub at %s", s)
		var err error
		s, err = b.pipeSub(ctx, bs, s)
		if err != nil {
			ctx.trf("Subber.Sub error at %s", s)
			return "", err
		}

		for j, f := range b.Procs {
			if s, err = f(ctx, s); err != nil {
				ctx.trf("Subber.Sub proc error at %s", s)
				return "", fmt.Errorf("subst proc %d: %w", j, err)
			}
		}

		// Have we encountered this string before?
		for _, s1 := range acc {
			if s == s1 {
				ctx.trf("Subber.Sub output %s", s)
				return s, nil
			}
		}
		// Nope.  Remember it.
		acc = append(acc, s)
	}

	ctx.trf("Subber.Sub limited at %s", s)
	return "", fmt.Errorf("recursive subst limit (%d) exceeded on at '%s' starting from '%s'", b.Limit, s, s0)
}
