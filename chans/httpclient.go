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
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Comcast/plax/dsl"
)

func init() {
	dsl.TheChanRegistry.Register(dsl.NewCtx(nil), "httpclient", NewHTTPClientChan)
}

// HTTPClient is an HTTPClient client Chan
type HTTPClient struct {
	opts   *HTTPClientOpts
	client *http.Client
	c      chan dsl.Msg

	pollers    map[string]chan bool
	lastPoller string
}

// HTTPClientOpts configures an HTTPClient channel.
type HTTPClientOpts struct {
}

func (c *HTTPClient) Kind() dsl.ChanKind {
	return "httpclient"
}

func (c *HTTPClient) Open(ctx *dsl.Ctx) error {
	c.client = &http.Client{}
	return nil
}

func (c *HTTPClient) Close(ctx *dsl.Ctx) error {
	c.client.CloseIdleConnections()
	return nil
}

func (c *HTTPClient) Sub(ctx *dsl.Ctx, topic string) error {
	return fmt.Errorf("%T doesn't support 'sub'", c)
}

// HTTPRequest represents a complete HTTP request, which is typically
// provided as a message payload in JSON.
//
// We can't just use https://golang.org/pkg/net/http/#Header because
// its URL field is actually a URL and not a string.  (Other reasons,
// too.)
type HTTPRequest struct {
	Method  string
	URL     string
	Headers map[string][]string

	// Body will be the request body.
	//
	// If Body isn't a string, it'll be JSON-serialized.
	Body interface{}

	// Form can contain form values, and you can specify these
	// values instead of providing an explicit Body.
	Form url.Values

	HTTPRequestCtl

	req *http.Request
}

type HTTPRequestCtl struct {

	// Id is used to refer to this request when it has a polling
	// interval.
	Id string

	// PollInterval, when not zero, will cause this channel to
	// repeated the HTTP request at this interval.
	//
	// Value should be a string that time.ParseDuration can parse.
	PollInterval string

	pollInterval time.Duration

	// Terminate, when not zero, should be the Id of a previous polling
	// request, and that polling request will be terminated.
	//
	// No other properties in this struct should be provided.
	Terminate string
}

// extractHTTPRequest attempts to make an http.Request from the
// (payload of the) given message.
//
// The message payload should be a JSON-serialized http.Request.
func extractHTTPRequest(ctx *dsl.Ctx, m dsl.Msg) (*HTTPRequest, error) {
	// m.Body is a JSON serialization of an HTTPRequest.

	// Parse the HTTPRequest.  First get a string representation
	// of the payload.
	js, is := m.Payload.(string)
	if !is {
		bs, err := json.Marshal(&m.Payload)
		if err != nil {
			// ToDo: Better error msg.
			return nil, err
		}
		js = string(bs)
	}

	// Parse the string as JSON representing an HTTPRequest.
	req := HTTPRequest{}
	if err := json.Unmarshal([]byte(js), &req); err != nil {
		return nil, err
	}

	// Parse the URL.
	u, err := url.Parse(req.URL)
	if err != nil {
		return nil, err
	}

	// We allow req.Body to be anything.  If it's not a string,
	// assume it should be JSON-serialized.
	var body string
	if req.Body != nil {
		if body, is = req.Body.(string); !is {
			bs, err := json.Marshal(&req.Body)
			if err != nil {
				// ToDo: Better error msg.
				return nil, err
			}
			body = string(bs)
		}
	}

	// Construct the actual http.Request.
	real := &http.Request{
		URL:    u,
		Method: req.Method,
		Header: req.Headers,
	}

	if req.Form != nil {
		if body != "" {
			return nil, fmt.Errorf("can't specify both Body and Form")
		}
		// real.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		body = req.Form.Encode()
	}

	if body != "" {
		real.Body = ioutil.NopCloser(strings.NewReader(body))
	}

	req.req = real

	return &req, nil
}

func (c *HTTPClient) terminate(ctx *dsl.Ctx, id string) error {
	ctx.Logf("%T terminating poller %s", c, id)

	if id == "last" {
		if c.lastPoller == "" {
			return fmt.Errorf("no last polling request")
		}
		id = c.lastPoller
	}

	ctl, have := c.pollers[id]
	if !have {
		return fmt.Errorf("unknown poller id '%s'", id)
	}
	close(ctl)
	delete(c.pollers, id)
	c.lastPoller = ""

	return nil
}

func (c *HTTPClient) poll(ctx *dsl.Ctx, ctl chan bool, req *HTTPRequest) error {
	go func() {
		d := req.pollInterval
		if d <= 0 {
			ctx.Logf("Warning HTTP request PollInterval %v", d)
			d = time.Second
		}
		ticker := time.NewTicker(d)

	LOOP:
		for {
			select {
			case <-ctx.Done():
			case <-ticker.C:
				ctx.Logf("%T making polling request", c)
				if err := c.do(ctx, req); err != nil {
					r := dsl.Msg{
						Payload: map[string]interface{}{
							"error": err.Error(),
						},
					}

					go c.To(ctx, r)
				}
			case <-ctl:
				break LOOP
			}
		}
	}()

	return nil
}

func (c *HTTPClient) do(ctx *dsl.Ctx, req *HTTPRequest) error {
	ctx.Logf("%T making request", c)
	resp, err := c.client.Do(req.req)
	if err != nil {
		return err
	}
	ctx.Logf("%T received message", c)
	ctx.Logdf("%T received %#v", c, resp)

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	ctx.Logdf("%T received body %s", c, bs)

	var x interface{}
	if 0 < len(bs) {
		if err = json.Unmarshal(bs, &x); err != nil {
			x = string(bs)
		}
	}

	r := dsl.Msg{
		Payload: x,
	}

	return c.To(ctx, r)
}

func (c *HTTPClient) Pub(ctx *dsl.Ctx, m dsl.Msg) error {
	ctx.Logf("%T Pub", c)
	req, err := extractHTTPRequest(ctx, m)
	if err != nil {
		return err
	}

	if req.Terminate != "" {
		return c.terminate(ctx, req.Terminate)
	}

	if req.PollInterval != "" {
		d, err := time.ParseDuration(req.PollInterval)
		if err != nil {
			return err
		}
		req.pollInterval = d
		if req.Id == "" {
			req.Id = "NA"
		}
		ctl := make(chan bool)
		c.pollers[req.Id] = ctl
		c.lastPoller = req.Id
		if err := c.poll(ctx, ctl, req); err != nil {
			return err
		}
		// Start polling but go ahead an do this first one
		// below.
	}

	return c.do(ctx, req)
}

func (c *HTTPClient) Recv(ctx *dsl.Ctx) chan dsl.Msg {
	return c.c
}

func (c *HTTPClient) Kill(ctx *dsl.Ctx) error {
	return fmt.Errorf("%T doesn't support 'Kill'", c)
}

func (c *HTTPClient) To(ctx *dsl.Ctx, m dsl.Msg) error {
	ctx.Logf("%T To", c)
	ctx.Logdf("  %T payload: %s", c, m.Payload)

	m.ReceivedAt = time.Now().UTC()
	select {
	case <-ctx.Done():
	case c.c <- m:
		ctx.Logf("%T queued message", c)
		ctx.Logf("%T queued %s", c, dsl.JSON(m))
	default:
		panic(fmt.Errorf("Warning: %T channel full", c))
	}
	return nil
}

func NewHTTPClientChan(ctx *dsl.Ctx, opts interface{}) (dsl.Chan, error) {
	o := HTTPClientOpts{}

	js, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(js, &o); err != nil {
		return nil, fmt.Errorf("NewHTTPClientChan: %w", err)
	}

	return &HTTPClient{
		opts:    &o,
		c:       make(chan dsl.Msg, DefaultMQTTBufferSize),
		pollers: make(map[string]chan bool),
	}, nil
}
