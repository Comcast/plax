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
	"strings"

	"github.com/Comcast/sheens/match"
	"github.com/itchyny/gojq"
)

// Bindings maps variables to their values, which should probably all
// be native Go values.
type Bindings map[string]interface{}

func NewBindings() Bindings {
	bindings := make(map[string]interface{})
	return bindings
}

// Copy bindings deeply.
//
// This method shamelessly uses JSON serialization, which can break on
// certains types of values.
func (bs *Bindings) Copy() (*Bindings, error) {
	bytes, err := JSONMarshal(bs)
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

// SetValue sets a binding explicitly (without any deserialization).
func (bs *Bindings) SetValue(k string, v interface{}) {
	(*bs)[k] = v
}

// SetJSON parses the value as JSON and stores the result at the given
// key.
func (bs *Bindings) SetJSON(key, value string) error {
	var x interface{}
	if err := json.Unmarshal([]byte(value), &x); err != nil {
		return err
	}

	bs.SetValue(key, x)

	return nil
}

// String returns a JSON repreentation of the Bindings.
func (bs *Bindings) String() string {
	return JSON(*bs)
}

// Set (for flag.Var) parses KEY=VALUE, where VALUE is a JSON
// representation of a value.  Then key set to that value.
func (bs *Bindings) Set(value string) error {
	pv := strings.SplitN(value, "=", 2)
	if len(pv) != 2 {
		return fmt.Errorf("bad binding: '%s'", value)
	}

	return bs.SetJSON(pv[0], pv[1])
}

// replaceBindings replaces all structural variables in x with their
// corresponding values in bs (if any).  See docs for Bindings.Bind().
func (bs *Bindings) replaceBindings(ctx *Ctx, x interface{}) (interface{}, error) {
	b := *bs
	switch vv := x.(type) {
	case string:
		if match.DefaultMatcher.IsVariable(vv) {
			parts := strings.SplitN(vv, "|", 2)
			if len(parts) == 2 {
				if 1 < len(parts[1]) {
					v := strings.TrimRight(parts[0], " ")
					expr := strings.TrimSpace(parts[1][1:])
					if !strings.HasPrefix(expr, "jq ") {
						return nil, fmt.Errorf("bad pipe expr in '%s'", expr)
					}
					src := strings.TrimSpace(expr[2:])
					q, err := gojq.Parse(unescapeQuotes(src))
					if err != nil {
						return nil, fmt.Errorf("jq parse error: %s on %s", err, expr[3:])
					}
					binding, have := b[v]
					if have {
						i := q.Run(binding)
						// Only consider the first thing returned.
						// ToDo: Elaborate.
						y, _ := i.Next()
						if err, is := y.(error); is {
							return nil, err
						}
						return y, nil
					}
				}
			}
			if binding, have := b[vv]; have {
				return binding, nil
			}
		}
		return x, nil
	case map[string]interface{}:
		acc := make(map[string]interface{}, len(vv))
		for k, v := range vv {
			bk, err := bs.replaceBindings(ctx, k)
			if err != nil {
				return nil, err
			}
			s, is := bk.(string)
			if !is {
				return nil, fmt.Errorf("%s isn't a %T (for a map key)",
					bk, s)
			}
			bv, err := bs.replaceBindings(ctx, v)
			if err != nil {
				return nil, err
			}
			acc[s] = bv
		}
		return acc, nil
	case []interface{}:
		acc := make([]interface{}, len(vv))
		for i, y := range vv {
			y, err := bs.replaceBindings(ctx, y)
			if err != nil {
				return nil, err
			}
			acc[i] = y
		}
		return acc, nil
	default:
		return x, nil
	}
}

// Bind replaces all structural variables in x with their
// corresponding values in bs (if any).
//
// This operation is destructive (and probably shouldn't be).
//
// An array or map should have interface{}-typed elements or values.
//
// An unbound variable does not result in an error.  See some comments
// in the Subber type.
func (bs *Bindings) Bind(ctx *Ctx, x interface{}) (interface{}, error) {
	return bs.replaceBindings(ctx, x)
}

// UnmarshalBind is a Proc for a Subber.
//
// The given string is parsed as JSON, and bs.Bind is called on that
// value.  The result is JSON-serialized and returned.
func (bs *Bindings) UnmarshalBind(ctx *Ctx, js string) (string, error) {
	var x interface{}
	if err := json.Unmarshal([]byte(js), &x); err != nil {
		return "", fmt.Errorf("Bindings.UnmarshalBind unmarshal %s: %w", js, err)
	}
	x, err := bs.replaceBindings(ctx, x)
	if err != nil {
		return "", err
	}
	s, err := JSONMarshal(&x)
	if err != nil {
		return "", fmt.Errorf("Bindings.UnmarshalBind marshall %s: %w", js, err)
	}
	return string(s), err

}
