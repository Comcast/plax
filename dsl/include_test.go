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
	"io/ioutil"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func chdir(t *testing.T, dir string) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err = os.Chdir(cwd); err != nil {
			t.Fatal(err)
		}
	})
}

func TestInclude(t *testing.T) {
	ctx := NewCtx(nil)

	bs, err := ioutil.ReadFile("../demos/include.yaml")
	if err != nil {
		t.Fatal(err)
	}

	ctx.IncludeDirs = []string{"../demos"}

	bs, err = IncludeYAML(ctx, bs)
	if err != nil {
		t.Fatal(err)
	}

	var tst Test
	if err = yaml.Unmarshal(bs, &tst); err != nil {
		t.Fatal(err)
	}

	if tst.Doc == "" {
		t.Fatal("Preamble not included")
	}

	p, have := tst.Spec.Phases["receive"]
	if !have {
		t.Fatal("receive not included")
	}

	if 0 == len(p.Steps) {
		t.Fatal("receive empty")
	}
}
