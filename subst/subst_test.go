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
	"context"
	"testing"
)

func TestSub(t *testing.T) {
	var (
		ctx = NewCtx(context.Background(), []string{"."})

		check = func(err error, want, got string, t *testing.T) {
			if err != nil {
				t.Fatal(err)
			}
			if want != got {
				t.Fatalf("want %s != %s got", want, got)
			}
		}
	)

	b, err := NewSubber("")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("", func(t *testing.T) {
		bs := NewBindings()
		bs.Set("need=" + JSON("tacos"))
		got, err := b.Sub(ctx, bs, "I would like some {need|string}.")
		check(err, "I would like some tacos.", got, t)
	})

	t.Run("", func(t *testing.T) {
		bs := NewBindings()
		bs.Set(`need={"several":"tacos"}`)
		got, err := b.Sub(ctx, bs, "I would like some {need|jq .several|string}.")
		check(err, "I would like some tacos.", got, t)
	})

	t.Run("", func(t *testing.T) {
		bs := NewBindings()
		bs.Set(`enjoy=["chips","queso"]`)
		got, err := b.Sub(ctx, bs, "Enjoy {enjoy|text$}.")
		check(err, "Enjoy chips,queso.", got, t)
	})

	t.Run("", func(t *testing.T) {
		bs := NewBindings()
		bs.Set(`enjoy=["chips","queso"]`)
		got, err := b.Sub(ctx, bs, "Enjoy {enjoy|text}.")
		check(err, `Enjoy ["chips","queso"].`, got, t)
	})

	t.Run("", func(t *testing.T) {
		bs := NewBindings()
		got, err := b.Sub(ctx, bs, "I would like a {@foo.txt | trim}.")
		check(err, "I would like a quesadilla.", got, t)
	})

	t.Run("", func(t *testing.T) {
		bs := NewBindings()
		bs.SetValue("?ENJOY", "tacos")
		got, err := b.Sub(ctx, bs, "I would like {?ENJOY}.")
		check(err, "I would like tacos.", got, t)
	})

	t.Run("", func(t *testing.T) {
		bs := NewBindings()
		bs.SetJSON("?ENJOY", `["tacos"]`)
		got, err := b.Sub(ctx, bs, "I would like {?ENJOY | jq .[0]}.")
		check(err, "I would like tacos.", got, t)
	})

	t.Run("", func(t *testing.T) {
		bs := NewBindings()
		got, err := b.Sub(ctx, bs, "I would like {@foo.yaml | jq .enjoys}.")
		check(err, "I would like tacos.", got, t)
	})

	t.Run("", func(t *testing.T) {
		bs := NewBindings()
		got, err := b.Sub(ctx, bs, "I would like {@foo.json | jq .enjoys | string}.")
		check(err, "I would like tacos.", got, t)
	})

	t.Run("", func(t *testing.T) {
		bs := NewBindings()
		got, err := b.Sub(ctx, bs, `{"he":"{@foo.json | json}"}`)
		check(err, `{"he":{"enjoys":"tacos"}}`, got, t)
	})

	t.Run("", func(t *testing.T) {
		bs := NewBindings()
		got, err := b.Sub(ctx, bs, `{"he":"{@foo.json}"}`)
		check(err, `{"he":{"enjoys":"tacos"}}`, got, t)
	})

	t.Run("", func(t *testing.T) {
		bs := NewBindings()
		got, err := b.Sub(ctx, bs, `{"send":"{@foo.json | jq .enjoys | json}"}`)
		check(err, `{"send":"tacos"}`, got, t)
	})

	t.Run("", func(t *testing.T) {
		bs := NewBindings()
		bs.Set(`need={"many":"{things}"}`)
		bs.Set(`things="{@foo.txt | js $.length | json}"`)
		got, err := b.Sub(ctx, bs, "I would like {need | jq .many | json} things.")
		check(err, "I would like 12 things.", got, t)
	})

	t.Run("", func(t *testing.T) {
		bs := NewBindings()
		bs.Set(`enjoy=["queso","chips"]`)
		s := `{"enjoy":["tacos","{enjoy|json$}","{enjoy|jq .[1]|json}"],"n":"{enjoy|js $.length|json}"}`
		MustParseJSON(s)
		got, err := b.Sub(ctx, bs, s)
		check(err, `{"enjoy":["tacos","queso","chips","chips"],"n":2}`, got, t)
	})

	t.Run("", func(t *testing.T) {
		bs := NewBindings()
		bs.Set(`enjoy=[{"some":"queso"},{"some":"chips"}]`)
		// Only the first thing emitted should be used.  We'll
		// also do a pipe within jq to see if the parsing
		// works.
		s := `{enjoy| jq .[]|.some | string}`
		got, err := b.Sub(ctx, bs, s)
		check(err, `queso`, got, t)
	})

	t.Run("", func(t *testing.T) {
		bs := NewBindings()
		bs.Set(`enjoy={"want":["queso","chips"]}`)
		s := `{"deliver":"{enjoy | jq .[\"want\"][0] | json}"}`
		MustParseJSON(s)
		got, err := b.Sub(ctx, bs, s)
		check(err, `{"deliver":"queso"}`, got, t)
	})

	t.Run("", func(t *testing.T) {
		bs := NewBindings()
		bs.Set(`order={"tacos":3,"queso":2}`)
		s := `{"deliver":{"chips":1,"":"{order | json@}"}}`
		MustParseJSON(s)
		got, err := b.Sub(ctx, bs, s)
		MustParseJSON(got)
		// Canonicalized map key order!
		check(err, `{"deliver":{"chips":1,"queso":2,"tacos":3}}`, got, t)
	})

}

func TestSubberCopy(t *testing.T) {
	var (
		bs    = NewBindings()
		procs = []Proc{bs.UnmarshalBind}
	)
	s, err := NewSubber("")
	if err != nil {
		t.Fatal(err)
	}
	s.Procs = procs

	s1 := s.Copy()
	s1.Procs[0] = nil

	if s.Procs[0] == nil {
		t.Fatal(s.Procs)
	}
}

func TestSubberWithProcs(t *testing.T) {
	var (
		bs    = NewBindings()
		proc  = bs.UnmarshalBind
		procs = []Proc{proc}
	)
	s, err := NewSubber("")
	if err != nil {
		t.Fatal(err)
	}
	s.Procs = procs

	s1 := s.WithProcs(proc)

	if len(s.Procs) != 1 {
		t.Fatal(s.Procs)
	}
	if len(s1.Procs) != 2 {
		t.Fatal(s1.Procs)
	}
	if s1.Procs[1] == nil {
		t.Fatal(s1.Procs)
	}

}

func TestLimit(t *testing.T) {
	var (
		ctx    = NewCtx(context.Background(), nil)
		bs     = NewBindings()
		b, err = NewSubber("")
	)
	if err != nil {
		t.Fatal(err)
	}
	bs.SetValue("?GOTO", "and {?GOTO}")

	// ctx.Tracing = true

	s, err := b.Sub(ctx, bs, `{?GOTO}`)
	if err == nil {
		t.Fatal(s)
	}
}

func TestProcs(t *testing.T) {
	var (
		ctx    = NewCtx(context.Background(), nil)
		bs     = NewBindings()
		proc   = bs.UnmarshalBind
		b, err = NewSubber("")
	)
	if err != nil {
		t.Fatal(err)
	}

	b = b.WithProcs(proc)
	bs.SetValue("?ENJOY", "queso")

	s, err := b.Sub(ctx, bs, `"?ENJOY"`)
	if err != nil {
		t.Fatal(err)
	}
	if s != `"queso"` {
		t.Fatal(s)
	}
}
