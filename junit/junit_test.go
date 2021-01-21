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
package junit

import (
	"encoding/xml"
	"fmt"
	"testing"
	"time"
)

func TestJUnit(t *testing.T) {
	tc := TestCase{
		Name: "queso",
		Skipped: &Skipped{
			Message: "too much trouble",
		},
		Time: time.Now().UTC().UnixNano(),
	}

	ts := NewTestSuite()

	ts.Add(tc)
	tc.Failure = &Failure{
		Message: "Didn't even try.",
	}
	ts.Add(tc)

	bs, err := xml.MarshalIndent(&ts, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s\n", bs)
}
