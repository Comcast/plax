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
	"io"
	"strings"
	"time"
)

func init() {
	TheChanRegistry.Register(NewCtx(nil), "mock", NewMockChan)
}

type MockChan struct {
	c chan Msg
}

func NewMockChan(ctx *Ctx, _ interface{}) (Chan, error) {
	return &MockChan{
		c: make(chan Msg, 1024),
	}, nil
}

func (c *MockChan) Kind() ChanKind {
	return "mock"
}

func (c *MockChan) Open(ctx *Ctx) error {
	return nil
}

func (c *MockChan) Close(ctx *Ctx) error {
	return nil
}

func (c *MockChan) Sub(ctx *Ctx, topic string) error {
	ctx.Logf("MockChan Sub %s", topic)
	return nil
}

func (c *MockChan) Pub(ctx *Ctx, m Msg) error {
	ctx.Logf("MockChan Pub topic %s", m.Topic)
	ctx.Logdf("             payload %s", JSON(m.Payload))
	return c.To(ctx, m)
}

func (c *MockChan) Recv(ctx *Ctx) chan Msg {
	ctx.Logf("MockChan Recv")
	return c.c
}

func (c *MockChan) Kill(ctx *Ctx) error {
	return Brokenf("Kill is not supported by a %T", c)
}

func (c *MockChan) To(ctx *Ctx, m Msg) error {
	ctx.Logf("MockChan To topic %s", m.Topic)
	ctx.Logdf("            payload %s", JSON(m.Payload))
	m.ReceivedAt = time.Now().UTC()
	select {
	case <-ctx.Done():
	case c.c <- m:
	default:
		panic("Warning: MockChan channel full")
	}
	return nil
}

// Read is a utility function to read input for a MockChan.
//
// Does not close the reader.
func (c *MockChan) Read(ctx *Ctx, in *bufio.Reader) error {
	ctx.Logf("MockChan reading input")
	for {
		line, err := in.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return err
		}
		if len(line) == 0 && err == io.EOF {
			return nil
		}

		parts := strings.SplitN(string(line), " ", 2)
		if len(parts) != 2 {
			ctx.Logf("error: MockChan.Read need topic payload")
			continue
		}
		m := Msg{
			Topic:   parts[0],
			Payload: parts[1],
		}
		if err = c.To(ctx, m); err != nil {
			return err
		}
	}

	return nil
}
