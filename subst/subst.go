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
	DefaultDelimiters    = "{}"
	DefaultSerialization = "json"
	DefaultLimit         = 10
)

type Proc func(*Ctx, string) (string, error)

type Subber struct {
	pipePattern          *regexp.Regexp
	Procs                []Proc
	Limit                int
	DefaultSerialization string
}

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
			`([?@]?[.a-zA-Z0-9!]+)`+ // Source (variable) name
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

func (b *Subber) Copy() *Subber {
	ps := make([]Proc, 0, len(b.Procs))
	for _, p := range b.Procs {
		ps = append(ps, p)
	}
	return &Subber{
		pipePattern:          b.pipePattern,
		Procs:                ps,
		Limit:                b.Limit,
		DefaultSerialization: b.DefaultSerialization,
	}
}

func (b *Subber) WithProcs(ps ...Proc) *Subber {
	acc := b.Procs
	for _, p := range ps {
		acc = append(acc, p)
	}
	b = b.Copy()
	b.Procs = acc
	return b
}

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
	source                string
	procsrc               string
	proc                  func(interface{}) (interface{}, error)
	leftQuote, rightQuote string
	serial                string
	spliceMode            string
	pairKeyColon          string

	submatch []string
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
	copy(ss[1:], ss[2:])
	leftQuote := ss[1]
	copy(ss[1:], ss[2:])
	rightQuote := ss[len(ss)-1]

	if ss[5] == "" {
		ss[4], ss[5] = ss[2], ss[3]
		ss[2] = ""
		ss[3] = ""
	}

	proc := ss[3]

	p := &pipe{
		submatch:     ss,
		source:       ss[1],
		procsrc:      proc,
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

	serial, mode, err := parseSerialization(ss[5])
	if err != nil {
		return nil, err
	}
	if serial == "default" {
		// ToDo: consider more/better heuristics here.
		if !p.quoted() {
			serial = "text"
		} else {
			serial = b.DefaultSerialization
		}
	}

	p.serial = serial
	p.spliceMode = mode

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

func (b *Subber) Sub(ctx *Ctx, bs Bindings, s string) (string, error) {
	var (
		// s0 is just for an error message (if required).
		s0 = s

		// acc remembers all previous values to detect loops.
		acc = make([]string, 0, b.Limit)
	)

	for i := 0; i < b.Limit; i++ {
		var err error
		s, err = b.pipeSub(ctx, bs, s)
		if err != nil {
			return "", err
		}

		for j, f := range b.Procs {
			if s, err = f(ctx, s); err != nil {
				return "", fmt.Errorf("subst proc %d: %w", j, err)
			}
		}

		// Have we encountered this string before?
		for _, s1 := range acc {
			if s == s1 {
				return s, nil
			}
		}
		// Nope.  Remember it.
		acc = append(acc, s)
	}

	return "", fmt.Errorf("recursive subst limit (%d) exceeded on at '%s' starting from '%s'", b.Limit, s, s0)
}
