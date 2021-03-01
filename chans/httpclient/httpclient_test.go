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
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Comcast/plax/dsl"
)

func TestDocs(t *testing.T) {
	(&HTTPClient{}).DocSpec().Write("httpclient")
}

// TestHTTPRequestPolling check that a HTTPRequest channel actually
// makes multiple requests when a PollInterval is given.
func TestHTTPRequestPolling(t *testing.T) {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	var (
		ctx      = dsl.NewCtx(context.Background())
		interval = 50 * time.Millisecond // PollInterval
		want     = 3                     // The number of messages we want to receive.

		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, `{"I have fixed your doorbell from the ringing":"There is no charge"}`)
		}))
	)

	defer ts.Close()

	c, err := NewHTTPClientChan(ctx, &HTTPClientOpts{})
	if err != nil {
		t.Fatal(err)
	}

	if err = c.Open(ctx); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := c.Close(ctx); err != nil {
			t.Fatal(err)
		}
	}()

	payload, err := json.Marshal(&HTTPRequest{
		Method: "GET",
		URL:    ts.URL,
		HTTPRequestCtl: HTTPRequestCtl{
			PollInterval: interval.String(),
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	err = c.Pub(ctx, dsl.Msg{
		Payload: string(payload),
	})

	if err != nil {
		t.Fatal(err)
	}

	var (
		ch = c.Recv(ctx)

		// min is the quickest poll that we'll allow.
		min = interval - interval/10
	)

	// Check that we get this many messages from the channel.
	for i := 0; i < want; i++ {
		var (
			to   = time.NewTimer(2 * interval)
			then = time.Now()
		)
		select {
		case <-ctx.Done():
			t.Fatal("ctx done")
		case <-to.C:
			t.Fatal("timeout")
		case msg := <-ch:
			// We got a message.
			if 0 < i {
				// All messages after the first one.
				if elapsed := time.Now().Sub(then); elapsed < min {
					t.Fatalf("too fast: %v, i=%d msg=%s", elapsed, i, dsl.JSON(msg))
				}
				// The timer in the 'select' will
				// complain about a slow poll.
			}
		}
	}

	// Check termination. We have a little race in our test, but
	// hopefully it won't cause trouble.

	payload, err = json.Marshal(&HTTPRequest{
		Method: "GET",
		URL:    ts.URL,
		HTTPRequestCtl: HTTPRequestCtl{
			Terminate: "last",
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	err = c.Pub(ctx, dsl.Msg{
		Payload: string(payload),
	})

	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-ctx.Done():
		t.Fatal("ctx done")
	case <-time.NewTimer(2 * interval).C:
		// Timeout before we received a message: good.
	case <-ch:
		t.Fatal("received another request")
	}
}
