/*
 * Copyright 2023 Comcast Cable Communications Management, LLC
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

package kdspub

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Comcast/plax/dsl"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
)

func init() {
	dsl.TheChanRegistry.Register(dsl.NewCtx(nil), "kdspub", NewKDSPubChan)
}

// KDSOpts is a configuration for a Kinesis consumer for a given
// stream.
type KDSOpts struct {
	// StreamName is of course the name of the KDS.
	StreamName string

	// BufferSize is the size of the underlying channel buffer.
	// Defaults to DefaultChanBufferSize.
	BufferSize int
}

// KDSPubChan is a basic Kinesis stream consumer.
//
// This channel consumes messages from a Kinesis stream.
type KDSPubChan struct {
	c   chan dsl.Msg
	ctl chan bool

	opts *KDSOpts
}

func (c *KDSPubChan) DocSpec() *dsl.DocSpec {
	return &dsl.DocSpec{
		Chan: &KDSPubChan{},
		Opts: &KDSOpts{},
	}
}

func NewKDSPubChan(ctx *dsl.Ctx, o interface{}) (dsl.Chan, error) {
	js, err := json.Marshal(&o)
	if err != nil {
		return nil, dsl.NewBroken(err)
	}

	opts := KDSOpts{
		BufferSize: dsl.DefaultChanBufferSize,
	}

	if err = json.Unmarshal(js, &opts); err != nil {
		return nil, dsl.NewBroken(err)
	}

	return &KDSPubChan{
		c:    make(chan dsl.Msg, opts.BufferSize),
		ctl:  make(chan bool),
		opts: &opts,
	}, nil
}

func (c *KDSPubChan) Kind() dsl.ChanKind {
	return "KDSPUB"
}

func (c *KDSPubChan) Open(ctx *dsl.Ctx) error {

	// Not doing anything here for the monment.  Might eventually
	// want to do establish the session and KDS client here for
	// efficient in case the test wants to publish several
	// messages.
	return nil
}

func (c *KDSPubChan) Close(ctx *dsl.Ctx) error {
	return nil
}

func (c *KDSPubChan) Sub(ctx *dsl.Ctx, topic string) error {
	return dsl.Brokenf("Can't Sub on a KDS (%s)", c.opts.StreamName)
}

func (c *KDSPubChan) Pub(ctx *dsl.Ctx, m dsl.Msg) error {

	ctx.Logf("Publishing to KDS %s", c.opts.StreamName)

	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		ctx.Logf("Error publishing to KDS %s", c.opts.StreamName)

	}

	k := kinesis.NewFromConfig(cfg)

	input := &kinesis.PutRecordInput{
		Data:         []byte(m.Payload),
		StreamName:   aws.String(c.opts.StreamName),
		PartitionKey: aws.String("test"),
	}

	_, err2 := k.PutRecord(context.TODO(), input)

	if err2 != nil {
		ctx.Warnf("warning: KDSPubChan.PUB %s", err)
	}
	return nil
}
func (c *KDSPubChan) Recv(ctx *dsl.Ctx) chan dsl.Msg {
	ctx.Logf("KDSPubChan Recv()")
	return c.c
}

func (c *KDSPubChan) Kill(ctx *dsl.Ctx) error {
	return fmt.Errorf("Kill is not supported by a %T", c)
}

func (c *KDSPubChan) To(ctx *dsl.Ctx, m dsl.Msg) error {
	ctx.Logf("KDSPubChan To %s", m.Topic)
	select {
	case <-ctx.Done():
	case c.c <- m:
	}
	return nil
}
