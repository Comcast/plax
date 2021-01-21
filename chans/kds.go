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
package chans

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Comcast/plax/dsl"

	kds "github.com/harlow/kinesis-consumer"
)

func init() {
	dsl.TheChanRegistry.Register(dsl.NewCtx(nil), "kds", NewKDSChan)
}

// KDSOpts is a configuration for a Kinesis consumer for a given
// stream.
type KDSOpts struct {
	StreamName string

	// BufferSize is the size of the underlying channel buffer.
	// Defaults to DefaultChanBufferSize.
	BufferSize int
}

type KDSChan struct {
	c   chan dsl.Msg
	ctl chan bool

	opts *KDSOpts
}

func NewKDSChan(ctx *dsl.Ctx, o interface{}) (dsl.Chan, error) {
	js, err := json.Marshal(&o)
	if err != nil {
		return nil, dsl.NewBroken(err)
	}

	opts := KDSOpts{
		BufferSize: DefaultChanBufferSize,
	}

	if err = json.Unmarshal(js, &opts); err != nil {
		return nil, dsl.NewBroken(err)
	}

	return &KDSChan{
		c:    make(chan dsl.Msg, opts.BufferSize),
		ctl:  make(chan bool),
		opts: &opts,
	}, nil
}

func (c *KDSChan) Kind() dsl.ChanKind {
	return "KDS"
}

func (c *KDSChan) Open(ctx *dsl.Ctx) error {
	go c.Consume(ctx)
	return nil
}

func (c *KDSChan) Close(ctx *dsl.Ctx) error {
	return nil
}

func (c *KDSChan) Sub(ctx *dsl.Ctx, topic string) error {
	return dsl.Brokenf("Can't Sub on a KDS (%s)", c.opts.StreamName)
}

func (c *KDSChan) Pub(ctx *dsl.Ctx, m dsl.Msg) error {
	return dsl.Brokenf("Can't (yet) Pub on a KDS (%s)", c.opts.StreamName)
}

func (c *KDSChan) Recv(ctx *dsl.Ctx) chan dsl.Msg {
	ctx.Logf("KDSChan Recv()")
	return c.c
}

func (c *KDSChan) Kill(ctx *dsl.Ctx) error {
	return fmt.Errorf("Kill is not supported by a %T", c)
}

func (c *KDSChan) To(ctx *dsl.Ctx, m dsl.Msg) error {
	ctx.Logf("KDSChan To %s", m.Topic)
	select {
	case <-ctx.Done():
	case c.c <- m:
	}
	return nil
}

func (c *KDSChan) Consume(ctx *dsl.Ctx) {
	ctx.Logf("Consuming KDS %s", c.opts.StreamName)

	/*
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))

		client := kinesis.New(sess)

		c, err := kds.New(stream, kds.WithClient(client))
	*/

	k, err := kds.New(c.opts.StreamName)

	if err != nil {
		log.Fatalf("consumer error: %v", err)
	}

LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case <-c.ctl:
			break LOOP
		default:
		}

		err = k.Scan(ctx, func(r *kds.Record) error {
			var (
				js = r.Data
				m  = dsl.Msg{
					Topic: c.opts.StreamName,
				}
				x interface{}
			)

			if err := json.Unmarshal(js, &x); err != nil {
				x = map[string]interface{}{
					"plain": string(js),
				}
			}

			// ToDo: Consider channel depth, etc.
			// ToDo: Respect ctl?
			return c.To(ctx, m)
		})

		if err != nil {
			ctx.Warnf("warning: KDSChan.Consume %s", err)
			break
		}
	}
}
