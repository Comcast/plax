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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`

	// Body is the request body.
	Body interface{} `json:"body,omitempty"`

	RequestBodySerialization    dsl.Serialization `json:"requestBodySerialization,omitempty" yaml:"requestbodyserialization,omitempty"`
	ResponseBodyDeserialization dsl.Serialization `json:"responseBodyDeserialization,omitempty" yaml:"responsebodydeserialization,omitempty"`

	// Form can contain form values, and you can specify these
	// values instead of providing an explicit Body.
	Form url.Values `json:"form,omitempty"`

	HTTPRequestCtl `json:"ctl,omitempty" yaml:"ctl"`

	// body will be the serialized Body.
	body []byte

	req *http.Request
}

type HTTPRequestCtl struct {

	// Id is used to refer to this request when it has a polling
	// interval.
	Id string `json:"id,omitempty"`

	// PollInterval, when not zero, will cause this channel to
	// repeated the HTTP request at this interval.
	//
	// Value should be a string that time.ParseDuration can parse.
	PollInterval string `json:"pollInterval"`

	pollInterval time.Duration

	// Terminate, when not zero, should be the Id of a previous polling
	// request, and that polling request will be terminated.
	//
	// No other properties in this struct should be provided.
	Terminate string `json:"terminate,omitempty"`
}

// extractHTTPRequest attempts to make an http.Request from the
// (payload of the) given message.
//
// The message payload should be a JSON-serialized http.Request.
func extractHTTPRequest(ctx *dsl.Ctx, m dsl.Msg) (*HTTPRequest, error) {
	// m.Body is a JSON serialization of an HTTPRequest.

	// Parse the HTTPRequest.
	var (
		js  = m.Payload
		req = &HTTPRequest{
			RequestBodySerialization:    dsl.DefaultSerialization,
			ResponseBodyDeserialization: dsl.DefaultSerialization,
		}
	)
	if err := json.Unmarshal([]byte(js), &req); err != nil {
		return nil, err
	}

	// Parse the URL.
	u, err := url.Parse(req.URL)
	if err != nil {
		return nil, err
	}

	if req.Body != nil {
		s, err := req.RequestBodySerialization.Serialize(req.Body)
		if err != nil {
			return nil, err
		}
		req.body = []byte(s)
	}

	// Construct the actual http.Request.
	real := &http.Request{
		URL:    u,
		Method: req.Method,
		Header: req.Headers,
	}

	if req.Form != nil {
		if req.Body != nil {
			return nil, fmt.Errorf("can't specify both Body and Form")
		}
		// real.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.body = []byte(req.Form.Encode())
	}

	if req.Body != nil {
		real.Body = ioutil.NopCloser(bytes.NewReader(req.body))
	}

	req.req = real

	return req, nil
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
						Payload: dsl.JSON(map[string]interface{}{
							"error": err.Error(),
						}),
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

type HTTPResponse struct {
	StatusCode int                 `json:"statuscode"`
	Body       interface{}         `json:"body"`
	Error      string              `json:"error,omitempty"`
	Headers    map[string][]string `json:"headers"`
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

	r := &HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
	}

	body, err := req.ResponseBodyDeserialization.Deserialize(string(bs))
	if err != nil {
		r.Error = err.Error()
	} else {
		r.Body = body
	}

	js, err := json.Marshal(&r)
	if err != nil {
		m := map[string]interface{}{
			"error": err.Error(),
		}
		js, _ = json.Marshal(&m)
	}

	msg := dsl.Msg{
		Payload: string(js),
	}

	return c.To(ctx, msg)
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

	go func() {
		if err := c.do(ctx, req); err != nil {
			// ToDo: Probably publish this message.
			ctx.Warnf("httpclient request error: %v", err)
		}
	}()

	return nil
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
