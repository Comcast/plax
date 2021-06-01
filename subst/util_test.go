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
	"bytes"
	"encoding/json"
	"log"
	"strings"
	"testing"
)

func MustParseJSON(js string) interface{} {
	var x interface{}
	if err := json.Unmarshal([]byte(js), &x); err != nil {
		log.Fatalf("MustParseJSON: failed to parse '%s': %s", js, err)
	}
	return x
}

func TestJSONSad(t *testing.T) {
	var (
		x = func() {} // Can't JSON-serialize.
		s = JSON(&x)  // Try anyway.
	)
	if !strings.Contains(s, "func") {
		t.Fatal(s)
	}
}

func TestJSONMarshal(t *testing.T) {
	t.Run("ampersand", func(t *testing.T) {
		x := "chips & salsa"

		js1, err := json.Marshal(&x)
		if err != nil {
			t.Fatal(err)
		}

		ampersandByte := "&"[0] // To mimick ...

		// Check that the stock json.Marshal encoded the ampersand.
		if bytes.Contains(js1, []byte{ampersandByte}) {
			t.Fatal(string(js1))
		}

		js2, err := JSONMarshal(&x)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(js2, []byte{ampersandByte}) {
			t.Fatal(string(js2))
		}
	})

	t.Run("newline", func(t *testing.T) {
		x := "chips and salsa"

		js1, err := json.Marshal(&x)
		if err != nil {
			t.Fatal(err)
		}
		js2, err := JSONMarshal(&x)
		if err != nil {
			t.Fatal(err)
		}
		if string(js1) != string(js2) {
			t.Fatalf("% X != % X", js1, js2)
		}
	})
}
