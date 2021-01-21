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
	"fmt"
	"time"

	"github.com/Comcast/sheens/match"
	"github.com/dop251/goja"
)

// JSExec executes the javascript source with the given context and environment mappings
func JSExec(ctx *Ctx, src string, env map[string]interface{}) (interface{}, error) {
	x, err := jsExec(ctx, src, env)
	if err != nil {
		if _, is := IsFailure(err); is {
			return x, err
		}
		return nil, Brokenf("Javascript problem: %s", err)
	}
	return x, nil
}

func jsExec(ctx *Ctx, src string, env map[string]interface{}) (interface{}, error) {

	js := goja.New()

	for k, v := range env {
		js.Set(k, v)
	}

	js.Set("print", func(args ...interface{}) {
		var acc string
		for i, x := range args {
			if 0 < i {
				acc += " "
			}
			acc += fmt.Sprintf("%s", JSON(x))
		}
		ctx.Inddf("    JS | %s\n", acc)
	})

	js.Set("now", func() interface{} {
		return time.Now().UTC().Format(time.RFC3339Nano)
	})

	js.Set("match", func(pat, msg interface{}, bs map[string]interface{}) []map[string]interface{} {
		if bs == nil {
			bs = match.NewBindings()
		}
		bss, err := match.Match(pat, msg, bs)
		if err != nil {
			panic(js.ToValue(err.Error()))
		}
		// Strip type (match.Bindings) to enable standard
		// Javascript access to the maps.
		acc := make([]map[string]interface{}, len(bss))
		for i, bs := range bss {
			acc[i] = map[string]interface{}(bs)
		}
		return acc
	})

	js.Set("Failure", func(msg string) Failure {
		return Failure(msg)
	})

	js.Set("tsMs", func(s string) int64 {
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			ctx.Indf("    warning: '%s' didn't parse as a time.RFC3339Nano", s)
			return 0
		}
		return t.UnixNano() / 1000 / 1000
	})

	v, err := js.RunString(src)
	if v != nil {
		x := v.Export()
		if f, is := IsFailure(x); is {
			return nil, f
		}
	}
	if err != nil {
		if f, is := IsFailure(err); is {
			return nil, f
		}
		return nil, err
	}

	return v.Export(), nil
}
