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
