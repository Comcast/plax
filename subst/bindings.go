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
)

func Brokenf(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

type Bindings map[string]interface{}

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

func (bs *Bindings) SetValue(k string, v interface{}) {
	(*bs)[k] = v
}

func (bs *Bindings) SetJSON(k, v string) error {
	var x interface{}
	if err := json.Unmarshal([]byte(v), &x); err != nil {
		return err
	}

	bs.SetValue(k, x)

	return nil
}

func (bs *Bindings) String() string {
	return "PARAM=VALUE"
}

func (bs *Bindings) Set(value string) error {
	pv := strings.SplitN(value, "=", 2)
	if len(pv) != 2 {
		return fmt.Errorf("bad binding: '%s'", value)
	}

	return bs.SetJSON(pv[0], pv[1])
}

// replaceBindings replaces all variables in x with their
// corresponding values in bs (if any).
//
// This operation is destructive (and probably shouldn't be).
//
// An array or map should have interface{}-typed elements or values.
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

// UnmarshalBind is a Proc.
func (bs *Bindings) UnmarshalBind(ctx *Ctx, js string) (string, error) {
	var x interface{}
	if err := json.Unmarshal([]byte(js), &x); err != nil {
		return "", fmt.Errorf("Bindings.UnmarshalBind unmarshal %s: %w", js, err)
	}
	x = bs.replaceBindings(ctx, x)
	s, err := json.Marshal(&x)
	if err != nil {
		return "", fmt.Errorf("Bindings.UnmarshalBind marshall %s: %w", js, err)
	}
	return string(s), err

}
