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

func (ctx *Ctx) subst() *subst.Ctx {
	c := subst.NewCtx(ctx.Context, ctx.IncludeDirs)
	// ToDo: LogLevel.
	return c
}

func (bs *Bindings) StringSub(ctx *Ctx, s string) (string, error) {
	c := ctx.subst()
	b := subber.WithProcs(proc(ctx, bangBangSub), proc(ctx, atAtSub))
	b.DefaultSerialization = "text"
	s, err := b.Sub(c, *(*subst.Bindings)(bs), s)
	if err != nil {
		return "", err
	}
	return s, nil
}

func proc(ctx *Ctx, f func(*Ctx, string) (string, error)) subst.Proc {
	return func(_ *subst.Ctx, s string) (string, error) {
		return bangBangSub(ctx, s)
	}
}

func (bs *Bindings) Sub(ctx *Ctx, s string) (string, error) {
	c := ctx.subst()
	b := subber.WithProcs(proc(ctx, bangBangSub), proc(ctx, atAtSub))
	b.DefaultSerialization = "json"
	m := (*subst.Bindings)(bs)
	b.Procs = append(b.Procs, m.UnmarshalBind)
	s, err := b.Sub(c, *m, s)
	if err != nil {
		return "", err
	}
	return s, nil
}

func guessSerialization(s string) string {
	var x interface{}
	if err := json.Unmarshal([]byte(s), &x); err == nil {
		return "json"
	}
	return "text"
}

func (bs *Bindings) SerialSub(ctx *Ctx, serialization string, payload interface{}) (string, error) {

	// We have a payload that could be a non-string, a string of
	// JSON, or a string of not-JSON.  p.Serialization allows us
	// to distinguish the last two cases; however, to support some
	// backwards compatibility in an era of casual typing, we also
	// have guessSerialization(), which can offer a serialization
	// if p.Serialization is zero.

	var s string
	var structured bool
	if str, is := payload.(string); is {
		switch serialization {
		case "":
			serialization = guessSerialization(str)
			ctx.Inddf("    Guessing serialization: %s", serialization)
			structured = serialization == "json"
		case "json", "string":
			structured = serialization == "json"
		}
		s = str
	} else {
		structured = true
		js, err := json.Marshal(&payload)
		if err != nil {
			return "", err
		}
		s = string(js)
	}

	if structured {
		return bs.Sub(ctx, s)
	}

	return bs.StringSub(ctx, s)

}

func (b *Bindings) SubX(ctx *Ctx, src interface{}, dst *interface{}) error {
	js, err := json.Marshal(&src)
	if err != nil {
		return err
	}

	s, err := b.SerialSub(ctx, "json", string(js))
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(s), &dst)
}

func (b *Bindings) Bind(ctx *Ctx, x interface{}) (interface{}, error) {
	bs := (*subst.Bindings)(b)
	return bs.Bind(nil, x)
}

func (b *Bindings) Copy() (*Bindings, error) {
	acc := make(Bindings)
	for p, v := range *b {
		acc[p] = v
	}
	return &acc, nil
}

// SetKeyValue to set the binding key to the given string (litera) value.
func (bs *Bindings) SetKeyValue(key string, value string) {
	(*bs)[key] = value
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

func (bs *Bindings) Clean(ctx *Ctx, clear bool) {
	// Always remove temporary bindings.
	for p := range *bs {
		if strings.HasPrefix(p, "?*") {
			delete(*bs, p)
		}
	}

	if clear {
		ctx.Indf("    Clearing bindings (%d) by request", len(*bs))
		for p := range *bs {
			if !strings.HasPrefix(p, "?!") {
				delete(*bs, p)
			}
		}
	}
}
