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
	"fmt"
)

// JSON attempts to serialize its input with a fallback to Go '%#v'
// serialization.
func JSON(x interface{}) string {
	js, err := json.Marshal(&x)
	if err != nil {
		js, _ = json.Marshal(map[string]interface{}{
			fmt.Sprintf("%T", x): fmt.Sprintf("%#v", x),
		})
	}
	return string(js)
}

// JSONMarshal exists to SetEscapeHTML(false) to avoid messing with <,
// >, and &.
//
// Strangely (to me), SetEscapeHTML(false) also seems to change
// newline treatment.  See TestJSONMarshal's 'newline' test.
func JSONMarshal(x interface{}) ([]byte, error) {
	var (
		buf = &bytes.Buffer{}
		enc = json.NewEncoder(buf)
	)
	enc.SetEscapeHTML(false)
	err := enc.Encode(x)

	if err != nil {
		return nil, err
	}
	bs := buf.Bytes()

	if 0 < len(bs) {
		if bs[len(bs)-1] == 0x0a {
			bs = bs[0 : len(bs)-1]
		}
	}

	return bs, nil
}
