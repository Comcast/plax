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

import "fmt"

// StringKeys will replace map[interface{}]interface{} with
// map[string]interface{} when that's possible and will return an
// error if not.
func StringKeys(x interface{}) (interface{}, error) {
	switch vv := x.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{}, len(vv))
		for p, v := range vv {
			s, is := p.(string)
			if !is {
				return nil, fmt.Errorf("map key %#v is a %T and not a %T",
					p, p, s)
			}
			y, err := StringKeys(v)
			if err != nil {
				return nil, err
			}
			m[s] = y
		}
		return m, nil
	case []interface{}:
		a := make([]interface{}, len(vv))
		for i, x := range vv {
			y, err := StringKeys(x)
			if err != nil {
				return nil, err
			}
			a[i] = y
		}
		return a, nil
	default:
		return x, nil
	}
}
