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

package invoke

import (
	"context"
	"testing"

	"github.com/Comcast/plax/dsl"
)

func TestNilTest(t *testing.T) {
	i := &Invocation{}

	ctx := dsl.NewCtx(nil)
	if err := i.Run(ctx, nil); err == nil {
		t.Fatal("expected protest")
	} else {
		if _, is := dsl.IsBroken(err); !is {
			t.Fatal("expected protest of broken test")
		}
	}
}

func TestInvocationBasic(t *testing.T) {
	i := &Invocation{
		Bindings:  nil,
		SuiteName: "test:mock",
		Filename:  "../demos/mock.yaml",
		Dir:       "",
		Env:       nil,
		Seed:      42,
		Verbose:   true,
	}

	ctx := dsl.NewCtx(context.Background())
	ts, err := i.Exec(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if ts == nil {
		t.Fatal(err)
	}
}
