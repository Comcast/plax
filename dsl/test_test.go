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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestWanted(t *testing.T) {
	ctx := NewCtx(context.Background())

	tst := NewTest(ctx, "a", nil)
	tst.Priority = 42
	t.Run("priority-negative", func(t *testing.T) {
		if !tst.Wanted(ctx, -1, nil) {
			t.Fatal(-1)
		}
	})
	t.Run("priority-high", func(t *testing.T) {
		if !tst.Wanted(ctx, 52, nil) {
			t.Fatal(52)
		}
	})
	t.Run("priority-low", func(t *testing.T) {
		if tst.Wanted(ctx, 1, nil) {
			t.Fatal(1)
		}
	})
}

func TestSubstitute(t *testing.T) {
	var (
		ctx = NewCtx(context.Background())
		tst = NewTest(ctx, "a", nil)
	)

	t.Run("basic", func(t *testing.T) {
		// We bind variables that require recursive
		// subsitution.  Note that these variables (by
		// definition) start with '?'.
		tst.Bindings = map[string]interface{}{
			"?want":  "{?queso}",
			"?queso": "queso",
		}
		s, err := tst.Bindings.StringSub(ctx, `!!"I want " + "{?want}."`)
		if err != nil {
			t.Fatal(err)
		}
		if s != "I want queso." {
			t.Fatal(s)
		}
	})

	t.Run("constantEmbedded", func(t *testing.T) {
		// Same basic test but we using a "binding" for a
		// constant (without the '?' prefix).
		tst.Bindings = map[string]interface{}{
			// Bind 'want' to a string that itself
			// references a binding variable.
			"want": "{?queso}",
			// Bind the variable referenced above.
			"?queso": "queso",
		}
		s, err := tst.Bindings.StringSub(ctx, `!!"I want " + "{want}."`)
		if err != nil {
			t.Fatal(err)
		}
		if s != "I want queso." {
			t.Fatal(s)
		}
	})

	t.Run("constantStructured", func(t *testing.T) {
		// Parameter-like subsitution: Bind a "parameter",
		// which has no "?" but does have "{}".
		tst.Bindings = map[string]interface{}{
			// Bind 'want' to a string that itself
			// references a binding variable.
			"{want}": "{?this}",
			// Bind the variable referenced above.
			"{?this}": "queso",
		}
		x := MaybeParseJSON(`{"need":"{want}"}`)
		var y interface{}
		if err := tst.Bindings.Sub(ctx, x, &y, false); err != nil {
			t.Fatal(err)
		}
		js1, err := json.Marshal(&x)
		if err != nil {
			t.Fatal(err)
		}
		js2, err := json.Marshal(&y)
		if err != nil {
			t.Fatal(err)
		}
		if string(js1) != string(js2) {
			t.Fatal(string(js2))
		}
	})
}

func TestTestIdFromPathname(t *testing.T) {
	var (
		pathname = "here/test-1.yaml"
		want     = "here/test-1"
		got      = TestIdFromPathname(pathname)
	)
	if want != got {
		t.Fatal(got)
	}
}

func TestSubstituteOnce(t *testing.T) {
	var (
		ctx = NewCtx(context.Background())
		tst = NewTest(ctx, "a", nil)
	)

	t.Run("badjs", func(t *testing.T) {
		if _, err := tst.Bindings.StringSubOnce(ctx, "!!nope"); err == nil {
			t.Fatal("should have complained")
		}
	})

	t.Run("jsobj", func(t *testing.T) {
		if s, err := tst.Bindings.StringSubOnce(ctx, `!!({"want":"tacos"})`); err != nil {
			t.Fatal(err)
		} else if s != `{"want":"tacos"}` {
			t.Fatal(s)
		}
	})

	t.Run("jsnots", func(t *testing.T) {
		if _, err := tst.Bindings.StringSubOnce(ctx, `!!function() {}`); err == nil {
			t.Fatal("should have complained")
		}
	})

	t.Run("file", func(t *testing.T) {
		if s, err := tst.Bindings.StringSubOnce(ctx, `@@test_test.go`); err != nil {
			t.Fatal(err)
		} else if len(s) < 1000 {
			t.Fatal(len(s))
		}
	})

	t.Run("filebad", func(t *testing.T) {
		if _, err := tst.Bindings.StringSubOnce(ctx, `@@nope`); err == nil {
			t.Fatal("should have complained")
		}
	})

	tst.Bindings = map[string]interface{}{
		"?need": "chips",
	}

	t.Run("filegood", func(t *testing.T) {
		s, err := tst.Bindings.StringSubOnce(ctx, `@@test_test.go`)
		if err != nil {
			t.Fatal(err)
		}
		// The following comment should be substituted!
		//
		// {?need}
		if strings.Contains(s, "{?need}") {
			t.Fatal("?need")
		}
	})
}

func testFromFile(t *testing.T, filename string) (*Test, *Errors) {
	var (
		ctx0, cancel = context.WithCancel(context.Background())
		ctx          = NewCtx(ctx0)
	)
	defer cancel()

	bs, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	bs, err = IncludeYAML(ctx, bs)
	if err != nil {
		t.Fatal(err)
	}

	tst := NewTest(ctx, filename, nil)
	tst.Dir = "../demos"

	if err := yaml.Unmarshal(bs, &tst); err != nil {
		t.Fatal(err)
	}

	if err := tst.Init(ctx); err != nil {
		t.Fatal(err)
	}

	if errs := tst.Validate(ctx); errs != nil {
		var acc string
		for i, err := range errs {
			acc += fmt.Sprintf("  %02d. %s\n", i, err)
		}
		t.Fatal(errs)
	}

	errs := tst.Run(ctx)

	if err := tst.Close(ctx); err != nil {
		t.Fatal(err)
	}

	return tst, errs

}

func TestFinally(t *testing.T) {
	tst, _ := testFromFile(t, "../demos/finally.yaml")

	if problem, have := tst.State["problem"]; have {
		t.Fatal(problem)
	}

	if x, have := tst.State["n"]; !have {
		t.Fatal("No n")
	} else if n, is := x.(int64); !is {
		t.Fatalf("n is %T", x)
	} else if n != 3 {
		t.Fatal(n)
	}
}
