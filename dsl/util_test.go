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
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestYAMLKeys attempts to check that YAML deserialization of a map
// with string keys actually gives a map[string]interface{}.
//
// https://github.com/go-yaml/yaml/issues/384
func TestYAMLKeys(t *testing.T) {
	var (
		s = `
want:
  queso: 1
  tacos: 3
  chips: 42
  '100': 100
`
		x interface{}
	)

	if err := yaml.Unmarshal([]byte(s), &x); err != nil {
		t.Fatal(err)
	}
	if m, is := x.(map[string]interface{}); is {
		x, have := m["want"]
		if !have {
			t.Fatal("lost 'want'")
		}
		if _, is := x.(map[string]interface{}); !is {
			t.Fatalf("%T", x)
		}

	} else {
		t.Fatalf("%T", x)
	}
}

func TestMaybeSerialize(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		var x interface{} = "tacos"
		want := `"tacos"`
		got, err := MaybeSerialize(&x)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatal(got)
		}
	})

	t.Run("sad", func(t *testing.T) {
		var x interface{} = func() {}
		got, err := MaybeSerialize(&x)
		if err == nil {
			t.Fatal("expected an error")
		}
		if !strings.HasPrefix(got, `{"error":`) {
			t.Fatal(got)
		}
	})
}
