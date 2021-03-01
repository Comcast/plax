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
	"io/ioutil"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestWanted(t *testing.T) {
	ctx := NewCtx(context.Background())

	tst := NewTest(ctx, "a", nil)
	tst.Priority = 42
	t.Run("priority-negative", func(t *testing.T) {
		if !tst.Wanted(ctx, -1, nil, nil) {
			t.Fatal(-1)
		}
	})
	t.Run("priority-high", func(t *testing.T) {
		if !tst.Wanted(ctx, 52, nil, nil) {
			t.Fatal(52)
		}
	})
	t.Run("priority-low", func(t *testing.T) {
		if tst.Wanted(ctx, 1, nil, nil) {
			t.Fatal(1)
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
