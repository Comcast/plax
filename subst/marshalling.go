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
	"reflect"
)

var DefaultMarshaller = &JSONMarshaller{}

type Marshaller interface {
	Marshal(interface{}) (string, error)
	Unmarshal(string, interface{}) error
}

type JSONMarshaller struct {
}

func (m *JSONMarshaller) Marshal(x interface{}) (string, error) {
	bs, err := json.Marshal(x)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

func (m *JSONMarshaller) Unmarshal(s string, x interface{}) error {
	return json.Unmarshal([]byte(s), x)
}

type StringMarshaller struct {
}

func (m *StringMarshaller) Marshal(x interface{}) (string, error) {
	s, is := x.(string)
	if !is {
		return "", fmt.Errorf("%#v (%T) isn't a %T", x, x, s)
	}
	return s, nil
}

func (m *StringMarshaller) Unmarshal(s string, x interface{}) error {
	rv := reflect.ValueOf(x)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &json.InvalidUnmarshalError{reflect.TypeOf(x)}
	}
	rv.Set(reflect.ValueOf(s))

	return nil
}
