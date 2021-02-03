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

import "time"

type Msg struct {
	Topic      string    `json:"topic"`
	Payload    string    `json:"payload"`
	ReceivedAt time.Time `json:"receivedAt"`
}

// ChanOpts represents generic data that is give to a Chan constructor.
type ChanOpts interface{}

// ChanKind is something like 'mqtt', 'kds', etc.
//
// Support for a Chan registers itself in ChanRegistry.
type ChanKind string

// ChanMaker is the signature for a Chan constructor.
type ChanMaker func(ctx *Ctx, def interface{}) (Chan, error)

// ChanRegistry maps a ChanKind to a constructor for that type of
// Chan.
type ChanRegistry map[ChanKind]ChanMaker

func (r ChanRegistry) Register(ctx *Ctx, kind ChanKind, maker ChanMaker) {
	r[kind] = maker
}

// TheChanRegistry is the global, well-known registry of supported
// Chan types.
var TheChanRegistry = make(ChanRegistry)

// Chan can send and receive messages.
type Chan interface {
	// Open starts up the Chan.
	Open(ctx *Ctx) error

	// Chose shuts down this Chan.
	Close(ctx *Ctx) error

	// Kill ungracefully closes an underlying connection (if any).
	//
	// Useful for testing MQTT LWT.
	Kill(ctx *Ctx) error

	// Kind returns this Chan's type.
	Kind() ChanKind

	// Sub, when required, initials a subscription.
	//
	// Use the Recv method to obtain messages that arrive via any
	// subscription.
	Sub(ctx *Ctx, topic string) error

	// Recv returns a channel of messages.
	Recv(ctx *Ctx) chan Msg

	// Pub, when supported, publishes a message on this Chan.
	Pub(ctx *Ctx, m Msg) error

	// To is a utility to send a message to the channel returned
	// by Recv.
	To(ctx *Ctx, m Msg) error
}
