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
	"fmt"
	"log"
)

func JSON(x interface{}) string {
	js, err := json.Marshal(&x)
	if err != nil {
		js, _ = json.Marshal(map[string]interface{}{
			fmt.Sprintf("%T", x): fmt.Sprintf("%#v", x),
		})
	}
	return string(js)
}

func MaybeParseJSON(x interface{}) interface{} {
	if s, is := x.(string); is {
		var y interface{}
		if err := json.Unmarshal([]byte(s), &y); err == nil {
			return y
		}
	}
	return x
}

func ParseJSON(js string) (interface{}, error) {
	var x interface{}
	if err := json.Unmarshal([]byte(js), &x); err != nil {
		return nil, err
	}
	return x, nil
}

func MustParseJSON(js string) interface{} {
	x, err := ParseJSON(js)
	if err != nil {
		log.Fatalf("MustParseJSON: failed to parse '%s': %s", js, err)
	}
	return x
}
