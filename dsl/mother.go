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
	"encoding/json"
	"fmt"
	"time"
)

// MotherRequest is the structure for a request to Mother.
//
// Every MotherRequest will get exactly one MotherResponse.
type MotherRequest struct {
	Make *MotherMakeRequest `json:"make"`
}

// MotherMakeRequest is the structure for a request to make a new
// channel.
type MotherMakeRequest struct {
	// Name is the requested name for the channel to be created.
	Name string `json:"name"`

	// Type is something like KDSConsumer, MQTT (client), or
	// SQSConsumer: types that are registered with a (or The)
	// ChannelRegistry.
	Type ChanKind `json:"type"`

	// Config is the configuration for the requested channel.
	//
	// This value is usually deserialized from YAML.
	Config interface{} `json:"config,omitempty"`
}

// MotherResponse is the structure of the generic response to a
// request.
type MotherResponse struct {
	// Request is the request the provoked this response.
	Request *MotherRequest `json:"request"`

	// Success reports whether the request succeeded.
	Success bool `json:"success"`

	// Error, if not zero, is an error message for a failed
	// request.
	Error string `json:"error,omitempty"`
}

// Mother is the mother of all (other) channels.
//
// A Mother can make channels, and a Mother is itself a Channel.
type Mother struct {
	t *Test
	c chan Msg
}

func NewMother(ctx *Ctx, _ interface{}) (*Mother, error) {
	return &Mother{
		c: make(chan Msg, 1024),
	}, nil
}

func (c *Mother) Kind() ChanKind {
	return "mother"
}

func (c *Mother) Open(ctx *Ctx) error {
	return nil
}

func (c *Mother) Close(ctx *Ctx) error {
	return nil
}

func (c *Mother) Sub(ctx *Ctx, topic string) error {
	ctx.Logf("Mother.Sub %s", topic)
	return nil
}

// Pub sends a request to Mother.
//
// The message payload should represent a MotherRequest in JSON.
func (c *Mother) Pub(ctx *Ctx, m Msg) error {
	ctx.Logf("Mother.Pub %T %v", m.Payload, m.Payload)

	var (
		req  MotherRequest
		resp MotherResponse
	)

	punt := func(err error) error {
		if err != nil {
			resp.Success = false
			resp.Error = err.Error()
		}
		js, err := json.Marshal(&resp)
		if err != nil {
			return err
		}
		return c.To(ctx, Msg{
			Payload: string(js),
		})
	}

	// Parse the payload as a MotherRequest.
	if err := json.Unmarshal([]byte(m.Payload), &req); err != nil {
		return punt(err)
	}

	resp.Request = &req

	// Handle the request.

	if req.Make == nil {
		return punt(fmt.Errorf("Only 'make' supported"))
	}

	if _, have := c.t.Chans[req.Make.Name]; have {
		return punt(fmt.Errorf("Already have chan '%s'", req.Make.Name))
	}

	// Special cases
	switch req.Make.Type {
	case "cmd":
		if m, is := req.Make.Config.(map[string]interface{}); is {
			m["name"] = req.Make.Name
		}
	}

	ch, err := c.t.makeChan(ctx, req.Make.Type, req.Make.Config)
	if err != nil {
		return punt(err)
	}

	if err := ch.Open(ctx); err != nil {
		return punt(err)
	}

	resp.Success = true
	c.t.Chans[req.Make.Name] = ch

	return punt(nil)
}

func (c *Mother) Recv(ctx *Ctx) chan Msg {
	ctx.Logf("Mother.Recv")
	return c.c
}

func (c *Mother) Kill(ctx *Ctx) error {
	return fmt.Errorf("Kill is not supported by a %T", c)
}

func (c *Mother) To(ctx *Ctx, m Msg) error {
	ctx.Logf("Mother To %s", m.Payload)
	m.ReceivedAt = time.Now().UTC()
	select {
	case <-ctx.Done():
	case c.c <- m:
	default:
		panic("Warning: Mother channel full")
	}
	return nil
}
