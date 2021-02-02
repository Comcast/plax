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
	"testing"
	"time"
)

var dejson = MustParseJSON

func addMock(t *testing.T, ctx *Ctx, p *Phase) {
	p.AddStep(ctx, &Step{
		Pub: &Pub{
			// Chan defaults to mother.
			Payload: dejson(`{"make":{"name":"mock1","type":"mock"}}`),
		},
	})

	p.AddStep(ctx, &Step{
		Recv: &Recv{
			// Default chan is now mock1, which is
			// not what we want here.
			Chan:    "mother",
			Pattern: dejson(`{"success":true}`),
		},
	})

}

func newTest(t *testing.T) (*Ctx, *Spec, *Test) {
	var (
		ctx = NewCtx(nil)
		s   = NewSpec()
		tst = NewTest(ctx, "", s)
	)

	return ctx, s, tst
}

func run(t *testing.T, ctx *Ctx, tst *Test) {
	if err := tst.Init(ctx); err != nil {
		t.Fatal(err)
	}

	if err := tst.Run(ctx); err != nil {
		// We are ignoring errors from final phases.  ToDo:
		// Not that.
		t.Fatal(err)
	}
}

func TestBasic(t *testing.T) {

	ctx, s, tst := newTest(t)
	ctx.LogLevel = "debug"

	{
		p := &Phase{
			Doc: "Create a mock channel",
		}
		s.Phases["phase1"] = p

		addMock(t, ctx, p)

		p.AddStep(ctx, &Step{
			Goto: "mock-test",
		})
	}

	{
		p := &Phase{
			Doc: "Test our mock channel",
		}
		s.Phases["mock-test"] = p

		p.AddStep(ctx, &Step{
			Pub: &Pub{
				Payload: `{"want":"tacos"}`,
			},
		})

		p.AddStep(ctx, &Step{
			Recv: &Recv{
				Pattern: `{"want":"?*x"}`,
				Timeout: time.Second,
			},
		})

		p.AddStep(ctx, &Step{
			Pub: &Pub{
				Payload: `{"want":"chips"}`,
			},
		})

		p.AddStep(ctx, &Step{
			Recv: &Recv{
				Pattern: `{"want":"?*x"}`,
				Timeout: time.Second,
			},
		})

	}

	run(t, ctx, tst)

}

func runTest(t *testing.T, ctx *Ctx, tst *Test) error {
	if err := tst.Init(ctx); err != nil {
		t.Fatal(err)
	}

	return tst.Run(ctx)
}

func MustParseJSON(js string) interface{} {
	var x interface{}
	if err := json.Unmarshal([]byte(js), &x); err != nil {
		panic(fmt.Errorf("failed to parse %s: %s", js, err))
	}
	return x
}

func TestSerialization(t *testing.T) {
	for name, ser := range Serializations {
		t.Run(name, func(t *testing.T) {
			payload := `{"want":"tacos"}`
			x, err := ser.Deserialize(payload)
			if err != nil {
				t.Fatal(err)
			}
			y, err := ser.Serialize(x)
			if err != nil {
				t.Fatal(err)
			}
			if payload != y {
				t.Fatal(y)
			}
		})
	}

	t.Run("illegal", func(t *testing.T) {
		if _, err := NewSerialization("graffiti"); err == nil {
			t.Fatal("expected a complaint")
		}
	})
}
