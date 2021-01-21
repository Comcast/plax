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
	"os/exec"
	"testing"
	"time"
)

func TestNewCmdChan(t *testing.T) {
	ctx := NewCtx(nil)
	cmd := exec.Cmd{}
	stdin := make(chan string)
	stdout := make(chan string)
	stderr := make(chan string)
	p := Process{
		Name:    "test-echo",
		Command: "echo",
		Args:    []string{"howdy, world!"},
		cmd:     &cmd,
		Stderr:  stderr,
		Stdin:   stdin,
		Stdout:  stdout,
	}

	c, err := NewCmdChan(ctx, p)
	if err != nil {
		t.Fatal("could not create cmd channel: " + err.Error())
	}

	if c.Kind() != "cmd" {
		t.Fatal(c.Kind())
	}

	if err = c.Open(ctx); err != nil {
		t.Fatal(err)
	}

	msg := Msg{
		Topic:      "dummy",
		Payload:    "the stars at night are big and bright",
		ReceivedAt: time.Now(),
	}
	if err = c.Pub(ctx, msg); err != nil {
		t.Fatal(err)
	}

	// cmd sub doesn't do anything
	if err = c.Sub(ctx, "dummy"); err != nil {
		t.Fatal(err)
	}

	if err = c.Close(ctx); err != nil {
		t.Fatal(err)
	}
}
