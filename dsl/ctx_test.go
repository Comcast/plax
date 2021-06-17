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
	"fmt"
	"log"
	"strings"
	"testing"
	"time"
)

func TestCtxCancel(t *testing.T) {
	ctx, cancel := NewCtx(nil).WithCancel()
	cancel()
	select {
	case <-ctx.Done():
	default:
		t.Fatal("not canceled")
	}
}

func TestCtxTimeout(t *testing.T) {
	ctx, _ := NewCtx(nil).WithTimeout(50 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	select {
	case <-ctx.Done():
	default:
		t.Fatal("didn't time out")
	}
}

type TestLogger struct {
	lines []string
}

func NewTestLogger() *TestLogger {
	return &TestLogger{
		lines: make([]string, 0, 32),
	}
}

func (l *TestLogger) Printf(format string, args ...interface{}) {
	line := fmt.Sprintf(format, args...)
	l.lines = append(l.lines, line)
}

func TestCtxRedactBasic(t *testing.T) {
	ctx := NewCtx(nil)
	if err := ctx.AddRedaction("tacos?"); err != nil {
		t.Fatal(err)
	}
	ctx.Redact = true
	l := NewTestLogger()
	ctx.Logger = l
	ctx.Logf(`Please don't say "tacos".`)

	if len(l.lines) == 0 {
		t.Fatal(0)
	}

	for _, line := range l.lines {
		if strings.Contains(line, "tacos") {
			t.Fatal(line)
		}
	}
}

func TestCtxRedactBindings(t *testing.T) {
	ctx := NewCtx(nil)
	ctx.Redact = true
	l := NewTestLogger()
	ctx.Logger = l

	tst := NewTest(ctx, "test", nil)
	tst.Bindings["?X_NEVERSAY"] = "I love crêpes."
	if err := tst.bindingRedactions(ctx); err != nil {
		t.Fatal(err)
	}
	line := `Never say "{?X_NEVERSAY}"`
	var err error
	line, err = tst.Bindings.StringSub(ctx, line)
	if err != nil {
		t.Fatal(err)
	}

	ctx.Logf("%s", line)

	if len(l.lines) == 0 {
		t.Fatal(0)
	}

	line = l.lines[0]

	log.Println(line)

	if strings.Contains(line, "crêpes") {
		t.Fatal(line)
	}
	if !strings.Contains(line, "redacted") {
		t.Fatal(line)
	}
}
