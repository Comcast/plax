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
	"testing"
)

func TestJSExec(t *testing.T) {
	ctx := NewCtx(nil)

	t.Run("bindings", func(t *testing.T) {
		x, err := JSExec(ctx, "1+x", map[string]interface{}{
			"x": 2,
		})
		if err != nil {
			t.Fatal(err)
		}

		switch vv := x.(type) {
		case int64:
			if vv != 3 {
				t.Fatal(x)
			}
		default:
			t.Fatal(x)
		}
	})

	t.Run("now", func(t *testing.T) {
		_, err := JSExec(ctx, "now()", nil)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("tsMs", func(t *testing.T) {
		_, err := JSExec(ctx, "tsMs(now())", nil)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("matchhappy", func(t *testing.T) {
		src := `
var pat = {"want":"?x"};
var msg = {"want":"queso"};
var bs = {"?y":"guacamole"};
var bss = match(pat, msg, bs);
bss[0]["?x"] == "queso";
`
		x, err := JSExec(ctx, src, nil)
		if err != nil {
			t.Fatal(err)
		}
		b, is := x.(bool)
		if !is {
			t.Fatal(x)
		}
		if !b {
			t.Fatal(b)
		}
	})

	t.Run("matchsad", func(t *testing.T) {
		src := `
var pat = {"?danger":"?x","?bad":"?z"};
var msg = {"want":"queso"};
var bs = {"?y":"guacamole"};
var bss = match(pat, msg, bs);
bss[0]["?x"] == "queso";
`
		_, err := JSExec(ctx, src, nil)
		if err == nil {
			t.Fatal("should have complained (politely)")
		}
	})

}
