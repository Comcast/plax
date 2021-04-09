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
	"regexp"
	"strings"

	"github.com/Comcast/sheens/match"
)

// Bindings type
type Bindings map[string]interface{}

// NewBindings builds a new set of bindings.
func NewBindings() Bindings {
	bindings := make(map[string]interface{})
	return bindings
}

// Copy the bindings deeply.
func (bs *Bindings) Copy() (*Bindings, error) {
	bytes, err := json.Marshal(bs)
	if err != nil {
		return nil, err
	}

	ret := NewBindings()
	err = json.Unmarshal(bytes, &ret)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

// String returns a string representation (required for parameters).
func (bs *Bindings) String() string {
	return "PARAM=VALUE"
}

// SetKeyValue to set the binding key to the given (native) value.
func (bs *Bindings) SetKeyValue(key string, value interface{}) {
	(*bs)[key] = value
}

// Set the parameter key=value pair assuming the value is either
// JSON-serialized or not.
//
// If we can't deserialize the value, we use the literal string (for
// backwards compatibility).
func (bs *Bindings) Set(kv string) error {
	parts := strings.SplitN(kv, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("bad binding: '%s'", kv)
	}
	k, v := parts[0], parts[1]

	var val interface{}
	if err := json.Unmarshal([]byte(v), &val); err != nil {
		val = v
	}

	bs.SetKeyValue(k, val)

	return nil
}

// Set the parameter key=value pair (without any attempted
// value unmarshalling).
func (bs *Bindings) SetString(value string) error {
	pv := strings.SplitN(value, "=", 2)
	if len(pv) != 2 {
		return fmt.Errorf("bad binding: '%s'", value)
	}

	bs.SetKeyValue(pv[0], pv[1])

	return nil
}

func (bs *Bindings) SubX(ctx *Ctx, src, dst interface{}) error {
	js, err := json.Marshal(&src)
	if err != nil {
		return err
	}
	s, err := bs.Sub(ctx, string(js))
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(s), &dst)
}

// Sub the bindings structurally.
func (bs *Bindings) Sub(ctx *Ctx, src string) (string, error) {
	// Computes the fixed point of SubOnce.

	var (
		// limit is the invocation circuit breaker.
		// ToDo: Expose.
		limit = 10

		// acc remembers all previous values to detect loops.
		acc = make([]string, 0, limit)
	)

	acc = append(acc, src)

	for i := 0; i < limit; i++ {
		var err error
		if src, err = bs.SubOnce(ctx, src); err != nil {
			return "", err
		}
		// Have we encountered this string before?
		for _, previous := range acc {
			if src == previous {
				return src, nil
			}
		}
		// Nope.  Remember it.
		acc = append(acc, src)
	}

	return "", Brokenf("expansion limit (%d) exceeded at '%s' starting from '%s'", limit, src, acc[0])
}

// SubOnce performs a single pass of bindings substitution on src.
func (bs *Bindings) SubOnce(ctx *Ctx, src string) (string, error) {
	var err error
	if src, err = bs.StringSub(ctx, src); err != nil {
		return "", err
	}

	var x interface{}
	if err = json.Unmarshal([]byte(src), &x); err == nil {
		x = bs.Bind(ctx, x)
		js, err := json.Marshal(&x)
		if err != nil {
			return "", err
		}
		src = string(js)
	}

	return src, nil
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

var atAtFilename = regexp.MustCompile(`"?{@@(.+?)}"?`)

func atAtSub(ctx *Ctx, s string) (string, error) {
	var err error
	y := atAtFilename.ReplaceAllStringFunc(s, func(s string) string {
		m := atAtFilename.FindStringSubmatch(s)
		if len(m) != 2 {
			err = Brokenf("internal error: failed to @@ submatch on '%s'", s)
			return fmt.Sprintf("<error: %s>", err)
		}
		filename := m[1]
		var bs []byte
		if bs, err = ioutil.ReadFile(ctx.Dir + "/" + filename); err != nil {
			return fmt.Sprintf("<error: %s>", err)
		}
		return string(bs)
	})
	if err != nil {
		return "", err
	}
	return y, nil
}

var bangBangFilename = regexp.MustCompile(`"?{!!(.+?)!!}"?`)

func bangBangSub(ctx *Ctx, s string) (string, error) {
	var err error
	y := bangBangFilename.ReplaceAllStringFunc(s, func(s string) string {
		m := bangBangFilename.FindStringSubmatch(s)
		if len(m) != 2 {
			err = Brokenf("internal error: failed to !! submatch on '%s'", s)
			return fmt.Sprintf("<error: %s>", err)
		}
		src := m[1]
		ctx.Inddf("    Expansion: Javascript '%s'", short(src))
		var x interface{}
		if x, err = JSExec(ctx, src, nil); err != nil {
			return fmt.Sprintf("<error: %s>", err)
		}
		str, is := x.(string)
		if !is {
			var js []byte
			if js, err = json.Marshal(&x); err != nil {
				return fmt.Sprintf("<error: %s>", err)
			}
			str = string(js)
		}
		return str
	})
	if err != nil {
		return "", err
	}
	return y, nil
}

// StringSubOnce performs the following substitutions in order: @@, !!,
// bindings.
//
// Bindings are substituted textually with added braces: a binding B=V
// will substitute V for {B} in the given string.
//
// This method does not call Bind (structured bindings substitution).
func (bs *Bindings) StringSubOnce(ctx *Ctx, s string) (string, error) {
	s, err := atAtSub(ctx, s)
	if err != nil {
		return "", err
	}

	if s, err = bangBangSub(ctx, s); err != nil {
		return "", err
	}

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

// walk is not used; commented out; for testing purposes only
// func walk(ctx *Ctx, x interface{}, f func(ctx *Ctx, x interface{}) (interface{}, error), limit int) (interface{}, error) {
// 	if limit <= 0 {
// 		return nil, Brokenf("walk() took too many steps")
// 	}

// 	switch vv := x.(type) {
// 	case map[string]interface{}:
// 		acc := make(map[string]interface{}, len(vv))
// 		for k, v := range vv {
// 			k1, err := f(ctx, k)
// 			if err != nil {
// 				return nil, err
// 			}
// 			s, is := k1.(string)
// 			if !is {
// 				return nil, Brokenf("tried to set a %T map key (%#v)", k1, k1)
// 			}
// 			v1, err := walk(ctx, v, f, limit-1)
// 			if err != nil {
// 				return nil, err
// 			}
// 			if v1 != nil {
// 				acc[s] = v1
// 			}
// 		}
// 		return acc, nil
// 	case string:
// 		x, err := f(ctx, vv)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if s, is := x.(string); is && s == vv {
// 			// Nothing changed, so stop now.
// 			return s, nil
// 		}
// 		y, err := walk(ctx, x, f, limit-1)
// 		if err != nil {
// 			return nil, err
// 		}
// 		return y, nil
// 	case []interface{}:
// 		acc := make([]interface{}, len(vv))
// 		for i, x := range vv {
// 			x1, err := walk(ctx, x, f, limit-1)
// 			if err != nil {
// 				return nil, err
// 			}
// 			acc[i] = x1
// 		}
// 		return acc, nil
// 	default:
// 		return x, nil
// 	}

// }
