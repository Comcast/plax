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
	"bufio"
	"strings"
	"testing"
)

func TestDocsMock(t *testing.T) {
	(&MockChan{}).DocSpec().Write("mock")
}

func TestMock(t *testing.T) {
	ctx := NewCtx(nil)
	c, err := NewMockChan(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if "mock" != c.Kind() {
		t.Fatal(c.Kind())
	}

	if err = c.Open(ctx); err != nil {
		t.Fatal(err)
	}

	go func() {
		m := c.(*MockChan)
		in := bufio.NewReader(strings.NewReader("topic payload"))
		if err = m.Read(ctx, in); err != nil {
			t.Fatal(err)
		}
	}()

	msg := <-c.Recv(ctx)
	if msg.Topic != "topic" {
		t.Fatal(msg.Topic)
	}
	if msg.Payload != "payload" {
		t.Fatal(msg.Payload)
	}

	if err = c.Kill(ctx); err == nil {
		t.Fatal("should have complained")
	}

}
