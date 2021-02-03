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

	"github.com/Comcast/plax/dsl"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

func init() {
	dsl.TheChanRegistry.Register(dsl.NewCtx(nil), "sqs", NewSQSChan)
}

var (
	// DefaultChanBufferSize is the default buffer size for
	// underlying Go channels used by some Chans.
	DefaultChanBufferSize = 1024
)

// SQSOpts is a configuration for an SQS consumer/producer.
//
// For now, the target queue URL is provided when the channel is
// created.  Eventually perhaps the queue URL could be the
// message/subscription topic.
type SQSOpts struct {
	// Endpoint is optional AWS service endpoint, which can be
	// provided to point to a non-standard endpoint (like a local
	// implementation).
	Endpoint string

	// QueueURL is the target SQS queue URL.
	QueueURL string

	// DelaySeconds is the publishing delay in seconds.
	//
	// Defaults to zero.
	DelaySeconds int64

	// VisibilityTimeout is the default timeout for a message reappearing after a receive operation and before a delete operation.  Defaults to 10 seconds.
	VisibilityTimeout int64

	// MaxMessages is the maximum number of message to request.
	//
	// Defaults to 1.
	MaxMessages int

	// DoNotDelete turns off automatic message deletion upon receipt.
	DoNotDelete bool

	// BufferSize is the size of the underlying channel buffer.
	// Defaults to DefaultChanBufferSize.
	BufferSize int

	// MsgDelaySeconds enables extraction of property DelaySeconds
	// from published message's payload, which should be a JSON of
	// an map.
	//
	// This hack means that a test cannot specify DelaySeconds for
	// a payload that is not a JSON representation of a map.
	// ToDo: Reconsider.
	MsgDelaySeconds bool

	// WaitTimeSeconds is the SQS receive wait time.
	//
	// Defaults to one second.
	WaitTimeSeconds int64
}

// SQSOpts is an SQS consumer/producer.
//
// In this implementation, message and subscription topics are
// ignored.
type SQSChan struct {
	c   chan dsl.Msg
	ctl chan bool
	svc *sqs.SQS

	opts *SQSOpts
}

func NewSQSChan(ctx *dsl.Ctx, o interface{}) (dsl.Chan, error) {
	js, err := json.Marshal(&o)
	if err != nil {
		return nil, dsl.NewBroken(err)
	}

	opts := SQSOpts{
		VisibilityTimeout: 10,
		MaxMessages:       1,
		WaitTimeSeconds:   1,
		BufferSize:        DefaultChanBufferSize,
	}

	if err = json.Unmarshal(js, &opts); err != nil {
		return nil, dsl.NewBroken(err)
	}

	return &SQSChan{
		c:    make(chan dsl.Msg, opts.BufferSize),
		ctl:  make(chan bool),
		opts: &opts,
	}, nil
}

func (c *SQSChan) Kind() dsl.ChanKind {
	return "SQS"
}

func (c *SQSChan) Open(ctx *dsl.Ctx) error {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	if c.opts.Endpoint != "" {
		sess.Config.Endpoint = &c.opts.Endpoint
	}

	c.svc = sqs.New(sess)

	go c.Consume(ctx)

	return nil
}

func (c *SQSChan) Close(ctx *dsl.Ctx) error {
	// ToDo: Terminate the consumer via c.ctl.
	return nil
}

func (c *SQSChan) Sub(ctx *dsl.Ctx, topic string) error {
	return dsl.Brokenf("Can't Sub on an SQS queue (%s)", c.opts.QueueURL)
}

func (c *SQSChan) Pub(ctx *dsl.Ctx, m dsl.Msg) error {
	ctx.Logf("SQSChan Pub()")

	delay := c.opts.DelaySeconds
	payload := m.Payload

	if c.opts.MsgDelaySeconds {

		// Extract and remove DelaySeconds from the message
		// Payload.
		var o map[string]interface{}
		err := json.Unmarshal([]byte(m.Payload), &o)
		if err != nil {
			return dsl.Brokenf("when using MsgDelaySeconds, SQS message must be a JSON map")
		}
		if x, have := o["DelaySeconds"]; have {
			switch n := x.(type) {
			case int:
				delay = int64(n)
			case int64:
				delay = n
			case float64:
				delay = int64(n)
			default:
				return dsl.Brokenf("when using MsgDelaySeconds, DelaySeconds in SQS payload a number (not a %T)", n)
			}
			delete(o, "DelaySeconds")
			js, err := json.Marshal(&o)
			if err != nil {
				return dsl.Brokenf("failed to re-JSON-serialize SQS message: %v", err)
			}
			payload = string(js)
		}
	}

	_, err := c.svc.SendMessage(&sqs.SendMessageInput{
		DelaySeconds: &delay,
		MessageBody:  aws.String(payload),
		QueueUrl:     aws.String(c.opts.QueueURL),
	})

	return err
}

func (c *SQSChan) Recv(ctx *dsl.Ctx) chan dsl.Msg {
	ctx.Logf("SQSChan Recv()")
	return c.c
}

func (c *SQSChan) Kill(ctx *dsl.Ctx) error {
	return fmt.Errorf("Kill is not supported by a %T", c)
}

func (c *SQSChan) To(ctx *dsl.Ctx, m dsl.Msg) error {
	ctx.Logf("SQSChan To %s", m.Topic)
	select {
	case <-ctx.Done():
	case c.c <- m:
	}
	return nil
}

func (c *SQSChan) Consume(ctx *dsl.Ctx) {
	ctx.Logf("Consuming SQS %s", c.opts.QueueURL)

LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case <-c.ctl:
			break LOOP
		default:
		}

		result, err := c.svc.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(c.opts.QueueURL),
			MaxNumberOfMessages: aws.Int64(1),
			VisibilityTimeout:   &c.opts.VisibilityTimeout,
			WaitTimeSeconds:     aws.Int64(c.opts.WaitTimeSeconds),
		})

		if err != nil {
			ctx.Warnf("warning: SQSChan.Consume %s: %s", err, c.opts.QueueURL)
			break
		}

		for _, msg := range result.Messages {
			m := dsl.Msg{
				Topic:   c.opts.QueueURL,
				Payload: *msg.Body,
			}

			// ToDo: Consider channel depth, etc.
			// ToDo: Respect ctl?

			if err = c.To(ctx, m); err != nil {
				ctx.Warnf("warning: SQSChan.Consume %s: %s", err, c.opts.QueueURL)
			}

			if !c.opts.DoNotDelete {
				_, err := c.svc.DeleteMessage(&sqs.DeleteMessageInput{
					QueueUrl:      aws.String(c.opts.QueueURL),
					ReceiptHandle: msg.ReceiptHandle,
				})
				if err != nil {
					ctx.Warnf("warning: SQSChan.Consume %s: %s", err, c.opts.QueueURL)
				}
			}
		}
	}
}
