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
	"time"
)

func init() {
	TheChanRegistry.Register(NewCtx(nil), "cmd", NewCmdChan)
}

// CmdChan is a channel that's backed by a subprocess.
type CmdChan struct {
	p *Process
	c chan Msg
}

// NewCmdChan obviously makes a new CmdChan.
//
// The cfg should represent a Process.
func NewCmdChan(ctx *Ctx, cfg interface{}) (Chan, error) {
	var p Process
	if err := As(cfg, &p); err != nil {
		return nil, err
	}
	return &CmdChan{
		p: &p,
		c: make(chan Msg, 1024),
	}, nil
}

func (c *CmdChan) Kind() ChanKind {
	return "cmd"
}

// Open starts the subprocess and the associated pipes.
func (c *CmdChan) Open(ctx *Ctx) error {

	if err := c.p.Start(ctx); err != nil {
		return err
	}

	go func() {
		pub := func(topic, payload string) {
			msg := Msg{
				Topic:   topic,
				Payload: payload,
			}
			select {
			case <-ctx.Done():
				return
			case c.c <- msg:
			}
		}

		for {
			select {
			case <-ctx.Done():
				return
			case line := <-c.p.Stdout:
				pub("stdout", line)
			case line := <-c.p.Stderr:
				pub("stderr", line)
			}
		}
	}()

	return nil
}

// Close currently just closes the subprocess's stdin.
func (c *CmdChan) Close(ctx *Ctx) error {
	ctx.Logf("CmdChan %s Close", c.p.Name)
	close(c.p.Stdin)
	return nil
}

// Sub doesn't currently do anything.
//
// ToDo: Have stdout and stderr "topics".
func (c *CmdChan) Sub(ctx *Ctx, topic string) error {
	ctx.Logf("CmdChan %s Sub", c.p.Name)
	return nil
}

// Pub sends the given message payload to the subprocess's stdin.
//
// The topic is ignored.
func (c *CmdChan) Pub(ctx *Ctx, m Msg) error {
	ctx.Logf("CmdChan %s Pub", c.p.Name)
	return c.To(ctx, m)
}

func (c *CmdChan) Recv(ctx *Ctx) chan Msg {
	ctx.Logf("CmdChan %s Recv", c.p.Name)
	return c.c
}

// Kill is not currently supported.  (It should be.)
//
// ToDo: Terminate the subprocess ungracefully.
func (c *CmdChan) Kill(ctx *Ctx) error {
	return fmt.Errorf("CmdChan %s: Kill is not yet supported", c.p.Name)
}

// To sends the given message payload to the subprocess's stdin.
//
// The topic is ignored.
func (c *CmdChan) To(ctx *Ctx, m Msg) error {
	ctx.Logf("CmdChan %s To", c.p.Name)
	m.ReceivedAt = time.Now().UTC()
	select {
	case <-ctx.Done():
	case c.c <- m:
	default:
		panic("Warning: CmdChan channel full")
	}
	return nil
}
