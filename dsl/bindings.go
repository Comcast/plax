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

type Bindings subst.Bindings

var subber *subst.Subber

func init() {
	b, err := subst.NewSubber("")
	if err != nil {
		panic(err)
	}
	subber = b
}

func (b *Bindings) StringSub(ctx *Ctx, s string) (string, error) {
	return subber.Sub(nil, *(*subst.Bindings)(b), s)
}

func (b *Bindings) Sub(ctx *Ctx, s string) (string, error) {
	return subber.Sub(nil, *(*subst.Bindings)(b), s)
}

func (b *Bindings) SubX(ctx *Ctx, src interface{}, dst *interface{}) error {
	js, err := json.Marshal(&src)
	if err != nil {
		return err
	}

	s, err := subber.Sub(nil, *(*subst.Bindings)(b), string(js))
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(s), &dst)
}

func (b *Bindings) Bind(ctx *Ctx, x interface{}) interface{} {
	bs := (*subst.Bindings)(b)
	return bs.Bind(nil, x)
}

func (bs *Bindings) Set(value string) error {
	pv := strings.SplitN(value, "=", 2)
	if len(pv) != 2 {
		return fmt.Errorf("bad binding: '%s'", value)
	}

	var v string
	if err := json.Unmarshal([]byte(pv[1]), &v); err != nil {
		v = value
	}

	(*bs)[pv[0]] = v

	return nil
}

func (bs *Bindings) String() string {
	acc := make([]string, 0, len(*bs))
	for k, v := range *bs {
		js, err := json.Marshal(&v)
		if err != nil {
			js = []byte(fmt.Sprintf("%#v", v))
		}
		acc = append(acc, fmt.Sprintf("%s=%s", k, js))
	}
	return strings.Join(acc, ",")
}
