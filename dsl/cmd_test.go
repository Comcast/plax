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
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
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

func TestCmdStderr(t *testing.T) {
	ctx := NewCtx(nil)
	cmd := exec.Cmd{}
	stdin := make(chan string)
	stdout := make(chan string)
	stderr := make(chan string)
	p := Process{
		Name:    "test-echo",
		Command: "bash",
		Args:    []string{"-c", "echo 'more chips, please' >&2"},
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

	var (
		in = c.Recv(ctx)
		to = time.NewTimer(time.Second)
	)

	select {
	case <-ctx.Done():
		t.Fatal("ctx Done")
	case <-to.C:
		t.Fatal("timeout")
	case msg := <-in:
		if msg.Topic != "stderr" {
			t.Fatal(msg.Topic)
		}
		s, _ := msg.Payload.(string)
		if !strings.Contains(s, "chips") {
			t.Fatal(msg.Payload)
		}
	}

	if err = c.Close(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestCmdTermEarly(t *testing.T) {
	var (
		ctx      = NewCtx(nil)
		cmd      = exec.Cmd{}
		filename = "term-proof.tmp"
		script   = fmt.Sprintf(`trap 'rm %s' SIGTERM SIGTERM; while true; do sleep 1; done`, filename)
	)

	p := Process{
		Name:    "test-term",
		Command: "bash",
		Args:    []string{"-c", script},
		cmd:     &cmd,
	}

	if err := ioutil.WriteFile(filename, []byte("process should delete me"), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := NewCmdChan(ctx, p)
	if err != nil {
		t.Fatal("could not create cmd channel: " + err.Error())
	}

	if err = c.Open(ctx); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)

	if err = c.Close(ctx); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		time.Sleep(time.Second)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			return
		}
	}

	t.Fatal("script not terminated?")
}

// TestCmdTermLate is just a cursory check that terminating the
// process after it has existed doesn't blow anything up.
func TestCmdTermLate(t *testing.T) {
	var (
		ctx    = NewCtx(nil)
		cmd    = exec.Cmd{}
		script = `echo bye`
	)

	p := Process{
		Name:    "test-term",
		Command: "bash",
		Args:    []string{"-c", script},
		cmd:     &cmd,
	}

	c, err := NewCmdChan(ctx, p)
	if err != nil {
		t.Fatal("could not create cmd channel: " + err.Error())
	}

	if err = c.Open(ctx); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)

	if err = c.Close(ctx); err != nil {
		t.Fatal(err)
	}
}
