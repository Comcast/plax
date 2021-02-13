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
	"time"

	"github.com/Comcast/plax/dsl"
)

func init() {
	dsl.TheChanRegistry.Register(dsl.NewCtx(nil), "httpserver", NewHTTPServerChan)
}

// HTTPServer is an HTTP server Chan.
//
// An HTTPServer will emit the requests it receives from HTTP clients,
// and the server should receive the responses to forward to those
// clients.
//
// To use this channel, you first 'recv' a client request, and then
// you 'pub' the response, which the Chan will forward to the HTTP
// client.
//
// Note that you have to do 'pub' each specific response for each
// client request.
type HTTPServer struct {
	opts  *HTTPServerOpts
	reqs  chan dsl.Msg
	resps chan dsl.Msg

	server *http.Server
}

// HTTPServerOpts configures an HTTPServer channel.
type HTTPServerOpts struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	ParseJSON bool   `json:"parsejson" yaml:"parsejson"`
}

func (c *HTTPServer) Kind() dsl.ChanKind {
	return "httpserver"
}

func (c *HTTPServer) Open(ctx *dsl.Ctx) error {
	addr := fmt.Sprintf("%s:%d", c.opts.Host, c.opts.Port)

	type Payload struct {
		Path    string              `json:"path"`
		Headers map[string][]string `json:"headers,omitempty"`
		Method  string              `json:"method"`
		Body    interface{}         `json:"body,omitempty"`
		Error   string              `json:"error,omitempty"`
	}

	type Response struct {
		Headers       map[string][]string `json:"headers,omitempty"`
		Body          interface{}         `json:"body,omitempty"`
		StatusCode    int                 `json:"statuscode,omitempty"`
		Serialization *dsl.Serialization  `json:"serialization,omitempty"`
	}

	punt := func(w http.ResponseWriter, err error) {
		w.WriteHeader(501)
		m := map[string]interface{}{
			"error": err.Error(),
		}
		js, err := json.Marshal(&m)
		if err != nil {
			js = []byte(fmt.Sprintf(`{"error":"%s"}`, err.Error())) // Too optimistic
		}

		w.Write(js)
	}

	f := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := &Payload{
			Path:    r.URL.Path,
			Headers: r.Header,
			Method:  r.Method,
		}

		if r.Method == http.MethodPost {
			bs, err := ioutil.ReadAll(r.Body)
			if err != nil {
				punt(w, err)
				return
			}

			if 0 < len(bs) && c.opts.ParseJSON {
				var body interface{}
				if err := json.Unmarshal(bs, &body); err != nil {
					punt(w, err)
					return
				}
				payload.Body = body
			} else {
				payload.Body = string(bs)
			}
		}

		js, err := json.Marshal(payload)
		if err != nil {
			punt(w, err)
			return
		}

		req := dsl.Msg{
			Topic:   r.URL.Path,
			Payload: string(js),
		}

		select {
		case <-ctx.Done():
		case c.reqs <- req:
			select {
			case <-ctx.Done():
			case resp := <-c.resps:
				r := &Response{
					StatusCode:    200, // ToDo: opt
					Serialization: &dsl.DefaultSerialization,
				}
				if err := json.Unmarshal([]byte(resp.Payload), &r); err != nil {
					w.WriteHeader(501)
					w.Write([]byte(err.Error() + " on payload"))
					return
				}
				body, err := r.Serialization.Serialize(r.Body)
				if err != nil {
					w.WriteHeader(501)
					w.Write([]byte(err.Error() + " on response"))
					return
				}
				w.WriteHeader(r.StatusCode)
				// ToDo: Check err, bytes written.
				w.Write([]byte(body))
			}
		}
	})

	c.server = &http.Server{
		Addr:           addr,
		Handler:        f,
		ReadTimeout:    10 * time.Second, // ToDo: opt
		WriteTimeout:   10 * time.Second, // ToDo: opt
		MaxHeaderBytes: 1 << 16,          // ToDo: opt
	}

	go func() {
		// ToDo: Report failure to listen better.
		if err := c.server.ListenAndServe(); err != nil {
			ctx.Logf("httpserver ListenAndServe error: %v", err)
		}
	}()

	return nil
}

func (c *HTTPServer) Close(ctx *dsl.Ctx) error {
	return c.server.Close()
}

func (c *HTTPServer) Sub(ctx *dsl.Ctx, topic string) error {
	return dsl.Brokenf("%T doesn't support 'sub'", c)
}

func (c *HTTPServer) Pub(ctx *dsl.Ctx, m dsl.Msg) error {
	ctx.Logf("%T Pub", c)
	return c.To(ctx, m)
}

func (c *HTTPServer) Recv(ctx *dsl.Ctx) chan dsl.Msg {
	return c.reqs
}

func (c *HTTPServer) Kill(ctx *dsl.Ctx) error {
	return dsl.Brokenf("%T doesn't support 'Kill'", c)
}

func (c *HTTPServer) To(ctx *dsl.Ctx, m dsl.Msg) error {
	ctx.Logf("%T To", c)
	ctx.Logdf("  %T payload: %s", c, m.Payload)

	m.ReceivedAt = time.Now().UTC()
	select {
	case <-ctx.Done():
	case c.resps <- m:
		ctx.Logf("%T queued message", c)
		ctx.Logf("%T queued %s", c, dsl.JSON(m))
	default:
		panic(fmt.Errorf("Warning: %T channel full", c))
	}
	return nil
}

func NewHTTPServerChan(ctx *dsl.Ctx, opts interface{}) (dsl.Chan, error) {
	o := HTTPServerOpts{}
	if err := dsl.As(opts, &o); err != nil {
		return nil, dsl.Brokenf("failed to create HTTP server Chan: %v", err)
	}

	return &HTTPServer{
		opts:  &o,
		reqs:  make(chan dsl.Msg, DefaultMQTTBufferSize),
		resps: make(chan dsl.Msg, DefaultMQTTBufferSize),
	}, nil
}
