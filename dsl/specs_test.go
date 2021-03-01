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
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func cd(t *testing.T, dir string) {
	cwd, _ := os.Getwd()
	t.Cleanup(func() {
		os.Chdir(cwd)
	})
	os.Chdir(dir)
}

func TestSpecs(t *testing.T) {
	var (
		ctx = NewCtx(context.Background())
		dir = "../demos"
	)

	ctx.IncludeDirs = []string{dir}

	fs, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		filename := f.Name()

		if !strings.HasSuffix(filename, ".yaml") {
			continue
		}

		t.Run(filename, func(t *testing.T) {
			path := dir + "/" + f.Name()

			bs, err := ioutil.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			bs, err = IncludeYAML(ctx, bs)
			if err != nil {
				t.Fatal(err)
			}

			tst := NewTest(ctx, filename, nil)
			tst.Dir = dir

			if err := yaml.Unmarshal(bs, &tst); err != nil {
				t.Fatal(err)
			}

			if !tst.Wanted(ctx, -1, []string{"selftest"}, []string{}) {
				return
			}

			testTest(t, tst)
		})
	}
}

func testTest(t *testing.T, tst *Test) {
	var (
		ctx0, cancel = context.WithCancel(context.Background())
		ctx          = NewCtx(ctx0)
	)

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
	if err := tst.Run(ctx); err != nil {
		if _, is := IsBroken(err); is {
			t.Fatal(err)
		}
		if !tst.Negative {
			t.Fatal(err)
		}
	}

	if err := tst.Close(ctx); err != nil {
		t.Fatal(err)
	}

	cancel()
}
