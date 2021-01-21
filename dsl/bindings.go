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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/Comcast/sheens/match"
)

// Bindings type
type Bindings map[string]interface{}

// NewBindings builds a new set of bindings
func NewBindings() Bindings {
	bindings := make(map[string]interface{})
	return bindings
}

// Copy the bindings deeply
func (bs *Bindings) Copy() (*Bindings, error) {
	bytes, err := json.Marshal(bs)
	if err != nil {
		return nil, err
	}

	ret := NewBindings()
	err = json.Unmarshal(bytes, &ret)

	return &ret, nil
}

// String representation required for parameters
func (bs *Bindings) String() string {
	return "PARAM=VALUE"
}

// SetKeyValue to set the binding key to the given JSON value
func (bs *Bindings) SetKeyValue(key string, value string) {
	var v interface{}
	if err := json.Unmarshal([]byte(value), &v); err != nil {
		v = value
	}

	(*bs)[key] = v
}

// Set the parameter key=value pair
func (bs *Bindings) Set(value string) error {
	pv := strings.SplitN(value, "=", 2)
	if len(pv) != 2 {
		return fmt.Errorf("bad binding: '%s'", value)
	}

	bs.SetKeyValue(pv[0], pv[1])

	return nil
}

// Sub the bindings
func (bs *Bindings) Sub(ctx *Ctx, src, target interface{}, maybeJSON bool) error {
	// Computes the fixed point of SubOnce.

	// We use a canonical (we hope) string representation to
	// determine termination.

	canonical := func(x interface{}) (string, error) {
		js, err := json.Marshal(&x)
		if err != nil {
			return "", err
		}
		return string(js), nil
	}

	// src0 is just for an error message (if required).
	src0, err := canonical(src)
	if err != nil {
		return nil
	}
	var (
		limit = 10

		// acc remembers all previous values to detect loops.
		acc = make([]string, 0, limit)
	)

	var s string
	for i := 0; i < limit; i++ {
		var err error
		var x interface{}
		if err = bs.SubOnce(ctx, src, &x, maybeJSON); err != nil {
			return err
		}
		if s, err = canonical(x); err != nil {
			return err
		}
		// Have we enountered this string before?
		for _, s0 := range acc {
			if s == s0 {
				// Need to deserialize into target.
				// Then we are done.
				return json.Unmarshal([]byte(s), &target)
			}
		}
		// Nope.  Remember it.
		acc = append(acc, s)
	}

	return fmt.Errorf("expansion limit (%d) exceeded at '%s' starting from '%s'", limit, s, src0)
}

// SubOnce the bindings
func (bs *Bindings) SubOnce(ctx *Ctx, src, target interface{}, maybeJSON bool) error {
	// If we are given a string, perform string-based expansion on
	// that string.
	if s, is := src.(string); is {
		var err error
		if src, err = bs.StringSub(ctx, s); err != nil {
			return err
		}
	}

	if maybeJSON {
		// Src might be a string of JSON.  If we can parse it, assume
		// that it is!  Then we can do structured bindings.
		if s, is := src.(string); is {
			var x interface{}
			if err := json.Unmarshal([]byte(s), &x); err == nil {
				// ctx.Indf("    Interpreting as JSON: %s", short(s))
				src = x // Assuming it was meant to be JSON.
			} else {
				ctx.Indf("    Note: string representation isn't JSON: %s", short(s))
			}
		}
	}
	// Perform structured bindings substitution.
	src = bs.Bind(ctx, src)

	// Attempt to deserialize the result into the target.

	js, err := json.Marshal(&src)
	if err != nil {
		return err
	}

	return json.Unmarshal(js, &target)
}

// StringSub computes the fixed point of StringSubOnce.
func (bs *Bindings) StringSub(ctx *Ctx, s string) (string, error) {
	// Computes the fixed point.

	var (
		// s0 is just for an error message (if required).
		s0 = s

		limit = 10

		// acc remembers all previous values to detect loops.
		acc = make([]string, 0, limit)
	)

	for i := 0; i < limit; i++ {
		var err error
		s, err = bs.StringSubOnce(ctx, s)
		if err != nil {
			return "", err
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

	return "", fmt.Errorf("expansion limit (%d) exceeded on at '%s' starting from '%s'", limit, s, s0)
}

// StringSubOnce performs the following subsitutions in order: @@, !!,
// bindings.
//
// Bindings are substituted textually with added braces: a binding B=V
// will substitute V for {B} in the given string.
//
// This method does not call Bind (structured bindings substitution).
func (bs *Bindings) StringSubOnce(ctx *Ctx, s string) (string, error) {
	b := *bs
	// Maybe read a file.
	if strings.HasPrefix(s, "@@") {
		ctx.Inddf("    Expansion: file '%s'", short(s[2:]))
		bs, err := ioutil.ReadFile(ctx.Dir + "/" + s[2:])
		if err != nil {
			return "", err
		}
		s = string(bs)
	}

	// Maybe execute Javascript.
	if strings.HasPrefix(s, "!!") {
		ctx.Inddf("    Expansion: Javascript '%s'", short(s[2:]))
		x, err := JSExec(ctx, s[2:], nil)
		if err != nil {
			return "", err
		}
		str, is := x.(string)
		if !is {
			js, err := json.Marshal(&x)
			if err != nil {
				return "", err
			}
			str = string(js)
		}
		s = str
	}

	// Bindings are substituted textually with added braces: a
	// binding B=V will substitute V for {B} in the given string.
	for k, v := range b {
		str, is := v.(string)
		if !is {
			js, err := json.Marshal(&v)
			if err != nil {
				return "", err
			}
			str = string(js)
		}
		s0 := s
		s = strings.ReplaceAll(s, "{"+k+"}", str)
		if s != s0 {
			ctx.Inddf("    Expansion: replacing '%s' with '%s'", k, short(str))
		}
	}

	return s, nil
}

// replaceBindings replaces all variables in x with their
// corresponding values in bs (if any).
//
// This operation is destructive (and probably shouldn't be).
func (bs *Bindings) replaceBindings(ctx *Ctx, x interface{}) interface{} {
	b := *bs
	switch vv := x.(type) {
	case string:
		if match.DefaultMatcher.IsVariable(vv) {
			if binding, have := b[vv]; have {
				return binding
			}
		}
		return x
	case map[string]interface{}:
		acc := make(map[string]interface{}, len(vv))
		for k, v := range vv {
			acc[k] = bs.replaceBindings(ctx, v)
		}
		return acc
	case []interface{}:
		acc := make([]interface{}, len(vv))
		for i, y := range vv {
			acc[i] = bs.replaceBindings(ctx, y)
		}
		return acc
	default:
		return x
	}
}

// Bind replaces all bindings in the given (structured) thing.
func (bs *Bindings) Bind(ctx *Ctx, x interface{}) interface{} {
	return bs.replaceBindings(ctx, x)
}
