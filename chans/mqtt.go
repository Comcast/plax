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
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Comcast/plax/dsl"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	// DefaultMQTTBufferSize is the default capacity of the
	// internal Go channel.
	DefaultMQTTBufferSize = 1024
)

func init() {
	dsl.TheChanRegistry.Register(dsl.NewCtx(nil), "mqtt", NewMQTTChan)
}

// MQTT is an MQTT client Chan.
type MQTT struct {
	opts   *MQTTOpts
	mopts  *mqtt.ClientOptions
	client mqtt.Client
	c      chan dsl.Msg
}

func (c *MQTT) DocSpec() *dsl.DocSpec {
	return &dsl.DocSpec{
		Chan: &MQTT{},
		Opts: &MQTTOpts{},
	}
}

// MQTTOpts is partly subset of mqtt.ClientOptions that can be
// deserialized.
type MQTTOpts struct {
	// When this struct or its fields' documentation changes,
	// update doc/manual.md.

	// BrokerURL is the URL for the MQTT broker.
	//
	// This required value has the form "PROTOCOL://HOST:PORT".
	BrokerURL string `json:",omitempty" yaml:",omitempty"`

	// CertFile is the optional filename for the client's certificate.
	CertFile string `json:",omitempty" yaml:",omitempty"`

	// CACertFile is the optional filename for the certificate
	// authority.
	CACertFile string `json:",omitempty" yaml:",omitempty"`

	// KeyFile is the optional filename for the client's private key.
	KeyFile string `json:",omitempty" yaml:",omitempty"`

	// Insecure will given the value for the tls.Config InsecureSkipVerify.
	//
	// InsecureSkipVerify controls whether a client verifies the
	// server's certificate chain and host name. If InsecureSkipVerify
	// is true, crypto/tls accepts any certificate presented by the
	// server and any host name in that certificate. In this mode, TLS
	// is susceptible to machine-in-the-middle attacks unless custom
	// verification is used. This should be used only for testing.
	Insecure bool `json:",omitempty" yaml:",omitempty"`

	// ALPN gives the
	// https://en.wikipedia.org/wiki/Application-Layer_Protocol_Negotiation
	// for the connection.
	//
	// For example, see
	// https://docs.aws.amazon.com/iot/latest/developerguide/protocols.html.
	ALPN string `json:",omitempty" yaml:",omitempty"`

	// Token is the optional value for the header given by
	// TokenHeader.
	//
	// See
	// https://docs.aws.amazon.com/iot/latest/developerguide/custom-authorizer.html.
	//
	// When Token is not empty, then you should probably also
	// provide AuthorizerName and TokenSig.
	Token string `json:",omitempty" yaml:",omitempty"`

	// TokenHeader is the name of the header which will have the
	// value given by Token.
	TokenHeader string `json:",omitempty" yaml:",omitempty"`

	// AuthorizerName is the optional value for the header
	// "x-amz-customauthorizer-name", which is used when making a
	// AWS IoT Core WebSocket connection that will call an AWS IoT
	// custom authorizer.
	//
	// See
	// https://docs.aws.amazon.com/iot/latest/developerguide/custom-authorizer.html.
	AuthorizerName string `json:",omitempty" yaml:",omitempty"`

	// TokenSig is the signature for the token for a WebSocket
	// connection to AWS IoT Core.
	//
	// See
	// https://docs.aws.amazon.com/iot/latest/developerguide/custom-authorizer.html.
	TokenSig string `json:",omitempty" yaml:",omitempty"`

	// BufferSize specifies the capacity of the internal Go
	// channel.
	//
	// The default is DefaultMQTTBufferSize.
	BufferSize int `json:",omitempty yaml:",omitempty"`

	// All durations are given in milliseconds.  Why? Because we
	// shamelessly transform interface{}s to what we want via
	// serialization.

	// PubTimeout is the timeout in milliseconds for MQTT PUBACK.
	PubTimeout int64 `json:",omitempty" yaml:",omitempty"`

	// SubTimeout is the timeout in milliseconds for MQTT SUBACK.
	SubTimeout int64 `json:",omitempty" yaml:",omitempty"`

	// ClientID is MQTT client id.
	ClientID string `json:",omitempty" yaml:",omitempty"`

	// Username is the optional MQTT client username.
	Username string `json:",omitempty" yaml:",omitempty"`

	// Password is the optional MQTT client password.
	Password string `json:",omitempty" yaml:",omitempty"`

	// CleanSession, when true, will not resume a previous MQTT
	// session for this client id.
	CleanSession bool `json:",omitempty" yaml:",omitempty"`

	// WillEnabled, if true, will establish an MQTT Last Will and Testament.
	//
	// See WillTopic, WillPayload, WillQoS, and WillRetained.
	WillEnabled bool `json:",omitempty" yaml:",omitempty"`

	// WillTopic gives the MQTT LW&T topic.
	//
	// See WillEnabled.
	WillTopic string `json:",omitempty" yaml:",omitempty"`

	// WillPayload gives the MQTT LW&T payload.
	//
	// See WillEnabled.
	WillPayload string `json:",omitempty" yaml:",omitempty"`

	// WillQoS specifies the MQTT LW&T QoS.
	//
	// See WillEnabled.
	WillQoS byte `json:",omitempty" yaml:",omitempty"`

	// WillRetained specifies the MQTT LW&T retained flag.
	//
	// See WillEnabled.
	WillRetained bool `json:",omitempty" yaml:",omitempty"`

	// KeepAlive is the duration in seconds that the MQTT client
	// should wait before sending a PING request to the broker.
	KeepAlive int64 `json:",omitempty" yaml:",omitempty"`

	// PingTimeout is the duration in seconds that the client will
	// wait after sending a PING request to the broker before
	// deciding that the connection has been lost.  The default is
	// 10 seconds.
	PingTimeout int64 `json:",omitempty" yaml:",omitempty"`

	// ConnectTimeout is the duration in seconds that the MQTT
	// client will wait after attempting to open a connection to
	// the broker.  A duration of 0 never times out.  The default
	// 30 seconds.
	//
	// This property does not apply to WebSocket connections.
	ConnectTimeout int64 `json:",omitempty" yaml:",omitempty"`

	// MaxReconnectInterval specifies maximum duration in
	// seconds between reconnection attempts.
	MaxReconnectInterval int64 `json:",omitempty" yaml:",omitempty"`

	// AutoReconnect turns on automation reconnection attempts
	// when a connection is lost.
	AutoReconnect bool `json:",omitempty" yaml:",omitempty"`

	// WriteTimeout is the duration to wait for a PUBACK.
	WriteTimeout int64 `json:",omitempty" yaml:",omitempty"`

	// ResumeSubs enables resuming of stored (un)subscribe
	// messages when connecting but not reconnecting if
	// CleanSession is false.
	ResumeSubs bool `json:",omitempty" yaml:",omitempty"`
}

// dur converts a int64 representing milliseconds to a time.Duration.
func dur(ms int64) time.Duration {
	return time.Duration(ms) * time.Millisecond
}

// Opts constructions an mqtt.ClientOptions.
func (o *MQTTOpts) Opts(ctx *dsl.Ctx) (*mqtt.ClientOptions, error) {
	opts := mqtt.ClientOptions{}

	ctx.Logf("MQTT Opts broker: %s", o.BrokerURL)
	opts.AddBroker(o.BrokerURL)
	opts.SetClientID(o.ClientID)
	opts.SetKeepAlive(time.Second * time.Duration(o.KeepAlive))
	opts.SetPingTimeout(dur(o.PingTimeout))
	opts.SetConnectTimeout(dur(o.ConnectTimeout))

	opts.Username = o.Username
	opts.Password = o.Password
	opts.AutoReconnect = o.AutoReconnect
	opts.CleanSession = o.CleanSession

	ctx.Logf("MQTT ClientID: %v", opts.ClientID)
	ctx.Logf("MQTT CleanSession: %v", opts.CleanSession)
	ctx.Logf("MQTT AutoReconnect: %v", opts.AutoReconnect)

	if o.Token != "" {
		var (
			bs     = make([]byte, 16)
			_, err = rand.Read(bs)
			key    = hex.EncodeToString(bs)
		)
		if err != nil {
			return nil, err
		}

		opts.HTTPHeaders = http.Header{
			o.TokenHeader:                      []string{o.Token},
			"x-amz-customauthorizer-name":      []string{o.AuthorizerName},
			"x-amz-customauthorizer-signature": []string{o.TokenSig},
			"sec-WebSocket-Key":                []string{key},
			"sec-websocket-protocol":           []string{"mqtt"},
			"sec-WebSocket-Version":            []string{"13"},
		}
	}

	if o.WillTopic != "" {
		if o.WillPayload == "" {
			return nil, fmt.Errorf("will topic without payload")
		}
		opts.WillEnabled = true
		opts.WillTopic = o.WillTopic
		opts.WillPayload = []byte(o.WillPayload)
		opts.WillRetained = o.WillRetained
		opts.WillQos = byte(o.WillQoS)
	}

	var rootCAs *x509.CertPool
	if rootCAs, _ = x509.SystemCertPool(); rootCAs == nil {
		rootCAs = x509.NewCertPool()
		ctx.Logf("Including system CA certs")
	}
	if o.CACertFile != "" {
		certs, err := ioutil.ReadFile(o.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("couldn't read '%s': %s", o.CACertFile, err)
		}

		if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
			return nil, fmt.Errorf("No certs appended, using system certs only")
		}
	}

	var certs []tls.Certificate
	if o.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(o.CertFile, o.KeyFile)
		if err != nil {
			return nil, dsl.NewBroken(err)
		}
		certs = []tls.Certificate{cert}
	}

	tlsConf := &tls.Config{
		InsecureSkipVerify: o.Insecure,
	}

	if o.ALPN != "" {
		// https://docs.aws.amazon.com/iot/latest/developerguide/protocols.html
		tlsConf.NextProtos = []string{
			o.ALPN,
		}
	}
	if rootCAs != nil {
		tlsConf.RootCAs = rootCAs
	}

	if certs != nil {
		tlsConf.Certificates = certs
	}

	opts.SetTLSConfig(tlsConf)

	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		ctx.Logf("MQTT %s connection lost", o.ClientID)
	}

	return &opts, nil
}

func (c *MQTT) Kind() dsl.ChanKind {
	return "mqtt"
}

func (c *MQTT) Open(ctx *dsl.Ctx) error {
	if c.client != nil {
		c.Close(ctx)
	}

	ctx.Logf("MQTT %s opening", c.mopts.ClientID)

	c.client = mqtt.NewClient(c.mopts)

	// The c.mopts.ConnectTimeout doesn't work when trying AWS IoT
	// Core at 443 with ALPN.  Dangit.  So we roll our own,
	// because we need it.  (Context.WithTimeout() doesn't appear
	// to help much.)

	var (
		err   error
		timer = make(chan struct{})
		con   = make(chan struct{})
	)

	go func() {
		time.Sleep(c.mopts.ConnectTimeout)
		close(timer)
	}()

	go func() {
		if t := c.client.Connect(); t.Wait() && t.Error() != nil {
			err = t.Error()
		}
		close(con)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("interrupted")
	case <-timer:
		return fmt.Errorf("timed out after %s", c.mopts.ConnectTimeout)
	case <-con:
		return err
	}
}

func (c *MQTT) Close(ctx *dsl.Ctx) error {
	ctx.Logf("MQTT %s closing", c.opts.ClientID)
	c.client.Disconnect(1000)
	return nil
}

func (c *MQTT) Sub(ctx *dsl.Ctx, topic string) error {
	t := c.client.Subscribe(topic, 1, nil)
	if ok := t.WaitTimeout(dur(c.opts.SubTimeout)); !ok {
		ctx.Warnf("Warning: MQTT wait timeout on Sub: %s", topic)
	}
	return t.Error()
}

func (c *MQTT) Pub(ctx *dsl.Ctx, m dsl.Msg) error {
	ctx.Logf("MQTT %s Pub %s", c.opts.ClientID, m.Topic)
	js, err := dsl.MaybeSerialize(m.Payload)
	if err != nil {
		return nil
	}
	t := c.client.Publish(m.Topic, 1, false, js)
	t.WaitTimeout(dur(c.opts.PubTimeout))

	return t.Error()
}

func (c *MQTT) Recv(ctx *dsl.Ctx) chan dsl.Msg {
	return c.c
}

// Kill is not currently supported.  (It should be but the paho client does not support ungraceful termination of the connection.)
//
// ToDo: Terminate the subprocess ungracefully.
func (c *MQTT) Kill(ctx *dsl.Ctx) error {
	return fmt.Errorf("MQTT Channel %s: Kill is not yet supported", c.opts.ClientID)
}

func (c *MQTT) To(ctx *dsl.Ctx, m dsl.Msg) error {
	ctx.Logf("MQTT %s To %s", c.opts.ClientID, m.Topic)
	ctx.Logdf("     %s", m.Payload)
	m.ReceivedAt = time.Now().UTC()
	select {
	case <-ctx.Done():
	case c.c <- m:
		ctx.Logf("MQTT %s queued %s", c.opts.ClientID, m.Topic)
	default:
		panic("Warning: MQTT channel full")
	}
	return nil
}

func NewMQTTChan(ctx *dsl.Ctx, opts interface{}) (dsl.Chan, error) {
	o := MQTTOpts{}

	js, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(js, &o); err != nil {
		return nil, fmt.Errorf("NewMQTTChan: %w", err)
	}

	if o.PubTimeout == 0 {
		o.PubTimeout = 1000 // ms
	}

	if o.SubTimeout == 0 {
		o.SubTimeout = 1000 // ms
	}

	if o.ConnectTimeout == 0 {
		o.ConnectTimeout = 1000 // ms
	}

	mopts, err := o.Opts(ctx)
	if err != nil {
		return nil, err
	}

	bufSize := o.BufferSize
	if bufSize == 0 {
		bufSize = DefaultMQTTBufferSize
	}

	c := &MQTT{
		opts:  &o,
		mopts: mopts,
		c:     make(chan dsl.Msg, bufSize),
	}

	// We use the default handler to process all in-coming
	// messages.  This approach enables persistent session
	// subscriptions to get messages into Plax.  (Previously, we
	// only established a handler for each Sub(scribe), so Plax
	// wouldn't see messages that the broker published to a
	// reconnected client with a persistent session.)

	mopts.DefaultPublishHandler = func(_ mqtt.Client, m mqtt.Message) {
		ctx.Logf("MQTT %s receiving %s", o.ClientID, m.Topic())
		ctx.Logdf("     %s", m.Payload())

		msg := dsl.Msg{
			Topic:   m.Topic(),
			Payload: string(m.Payload()),
		}
		go func() {
			if err := c.To(ctx, msg); err != nil {
				ctx.Warnf("warning: %s To for %s from MQTT.Sub handler", err, js)
			}
		}()
	}

	return c, nil

}
