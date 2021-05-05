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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Comcast/plax/subst"
)

func JSON(x interface{}) string {
	js, err := subst.JSONMarshal(&x)
	if err != nil {
		js, _ = subst.JSONMarshal(map[string]interface{}{
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

func MaybeSerialize(x interface{}) (string, error) {
	if s, is := x.(string); is {
		return s, nil
	}
	js, err := json.Marshal(&x)
	if err != nil {
		// We still return something useful, but we also
		// return the error.  Probably dumb.
		return JSON(map[string]interface{}{
			"error": err.Error(),
			"on":    fmt.Sprintf("%#v", x),
		}), err
	}
	return string(js), nil
}

// Canon constructs a canonical (via JSON) representation.
func Canon(x interface{}) interface{} {
	js, err := json.Marshal(&x)
	if err != nil {
		panic(err)
	}
	var y interface{}
	if err = json.Unmarshal(js, &y); err != nil {
		panic(err)
	}
	return y
}

// short returns a short version of the given string.
func short(s string) string {
	limit := 30
	s = strings.ReplaceAll(s, "\n", " ")
	if limit < len(s)-3 {
		return s[0:limit] + "..."
	}
	return s
}

// As is shameful.
func As(src interface{}, dst interface{}) error {
	js, err := json.Marshal(&src)
	if err != nil {
		return err
	}
	return json.Unmarshal(js, &dst)
}
