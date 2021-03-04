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

// Package shell provides a 'cmd' channel type.
//
// This package almost should be called 'cmd', but tends to have a
// special meaning in Go.
package shell

import (
	"fmt"
	"strconv"
	"time"

	"github.com/Comcast/plax/dsl"
)

func init() {
	dsl.TheChanRegistry.Register(dsl.NewCtx(nil), "cmd", NewCmdChan)
}

// CmdChan is a channel that's backed by a subprocess.
//
// This channel forwards messages to a shell's stdin, and messages
// written to the shell's stdout and stderr are emitted.
type CmdChan struct {
	p *dsl.Process

	// to receives messages that are forwarded to the Process's stdin.
	to chan dsl.Msg

	// from receives messages from the Process's stdin and stdout,
	// and these messages are emitted from this channel for recv consideration.
	from chan dsl.Msg
}

// NewCmdChan obviously makes a new CmdChan.
//
// The cfg should represent a Process.
func NewCmdChan(ctx *dsl.Ctx, cfg interface{}) (dsl.Chan, error) {
	var p dsl.Process
	if err := dsl.As(cfg, &p); err != nil {
		return nil, err
	}
	return &CmdChan{
		p:    &p,
		to:   make(chan dsl.Msg, 1024),
		from: make(chan dsl.Msg, 1024),
	}, nil
}

func (c *CmdChan) DocSpec() *dsl.DocSpec {
	return &dsl.DocSpec{
		Chan: &CmdChan{},
		Opts: &dsl.Process{},
	}
}

func (c *CmdChan) Kind() dsl.ChanKind {
	return "cmd"
}

// Open starts the subprocess and the associated pipes.
func (c *CmdChan) Open(ctx *dsl.Ctx) error {

	if err := c.p.Start(ctx); err != nil {
		return err
	}

	go func() {
		out := func(topic, payload string) {
			msg := dsl.Msg{
				Topic:   topic,
				Payload: payload,
			}
			select {
			case <-ctx.Done():
				return
			case c.from <- msg:
			}
		}

		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-c.to:
				select {
				case <-ctx.Done():
				case c.p.Stdin <- msg.Payload:
				}
			case line := <-c.p.Stdout:
				out("stdout", line)
			case line := <-c.p.Stderr:
				out("stderr", line)
			case exitCode := <-c.p.ExitCode:
				out("exit", strconv.Itoa(exitCode))
				return
			}
		}
	}()

	return nil
}

// Close attempts to terminate the underlying process.
func (c *CmdChan) Close(ctx *dsl.Ctx) error {
	ctx.Logf("CmdChan %s Close", c.p.Name)
	return c.p.Term(ctx)
}

// Sub doesn't currently do anything.
//
// ToDo: Have stdout and stderr "topics".
func (c *CmdChan) Sub(ctx *dsl.Ctx, topic string) error {
	ctx.Logf("CmdChan %s Sub", c.p.Name)
	return nil
}

// Pub sends the given message payload to the subprocess's stdin.
//
// The topic is ignored.
func (c *CmdChan) Pub(ctx *dsl.Ctx, m dsl.Msg) error {
	ctx.Logf("CmdChan %s Pub", c.p.Name)
	return c.To(ctx, m)
}

func (c *CmdChan) Recv(ctx *dsl.Ctx) chan dsl.Msg {
	ctx.Logf("CmdChan %s Recv", c.p.Name)
	return c.from
}

// Kill is not currently supported.  (It should be.)
//
// ToDo: Terminate the subprocess ungracefully.
func (c *CmdChan) Kill(ctx *dsl.Ctx) error {
	return fmt.Errorf("CmdChan %s: Kill is not yet supported", c.p.Name)
}

// To sends the given message payload to the subprocess's stdin.
//
// The topic is ignored.
func (c *CmdChan) To(ctx *dsl.Ctx, m dsl.Msg) error {
	ctx.Logf("CmdChan %s To", c.p.Name)
	m.ReceivedAt = time.Now().UTC()
	select {
	case <-ctx.Done():
	case c.to <- m:
	default:
		return fmt.Errorf("CmdChan %s input buffer is full", c.p.Name)
	}
	return nil
}
