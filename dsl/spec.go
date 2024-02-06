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
	"fmt"
	"time"

	"github.com/Comcast/plax/subst"
)

var DefaultInitialPhase = "phase1"

// Spec represents a set of named test Phases.
type Spec struct {
	// InitialPhase is the starting phase, which defaults to
	// DefaultInitialPhase.
	InitialPhase string

	// FinalPhases is an option list of phases to execute after
	// the execution starting at InitialPhase terminates.
	FinalPhases []string

	// Phases maps phase names to Phases.
	//
	// Each Phase is subject to bindings substitution.
	Phases map[string]*Phase
}

func NewSpec() *Spec {
	return &Spec{
		InitialPhase: DefaultInitialPhase,
		Phases:       make(map[string]*Phase),
	}
}

// Phase is a list of Steps.
type Phase struct {
	// Doc is an optional documentation string.
	Doc string `yaml:",omitempty"`

	// Steps is a sequence of Steps, which are attempted in order.
	//
	// Each Step is subject to bindings substitution.
	Steps []*Step
}

func (p *Phase) AddStep(ctx *Ctx, s *Step) {
	steps := p.Steps
	if steps == nil {
		steps = make([]*Step, 0, 8)
	}
	p.Steps = append(steps, s)
}

func (p *Phase) Exec(ctx *Ctx, t *Test) (string, error) {
	var (
		next string
		err  error
		last = len(p.Steps) - 1
	)
	for i, s := range p.Steps {
		ctx.Indf("  Step %d", i)
		ctx.Inddf("    Bindings: %s", JSON(t.Bindings))

		if next, err = s.exec(ctx, t); err != nil {
			_, broke := IsBroken(err)
			err := fmt.Errorf("step %d: %w", i, err)
			if broke {
				return "", NewBroken(err)
			} else {
				return "", err
			}
		}
		if i < last && next != "" {
			return "", Brokenf("Goto or Branch not last in %s", JSON(p))
		}
		if i == last {
			ctx.Indf("    Next phase: '%s'", next)
		}
	}
	return next, err
}

// Step represents a single action.
type Step struct {
	// Doc is an optional documentation string.
	Doc string `yaml:",omitempty"`

	// Fails indicates that this Step is expected to fail, which
	// currently means returning an error from exec.
	Fails bool `yaml:",omitempty"`

	// Skip will make the test execution skip this step.
	Skip bool `yaml:",omitempty"`

	Pub       *Pub       `yaml:",omitempty"`
	Sub       *Sub       `yaml:",omitempty"`
	Recv      *Recv      `yaml:",omitempty"`
	Kill      *Kill      `yaml:",omitempty"`
	Reconnect *Reconnect `yaml:",omitempty"`
	Close     *Close     `yaml:",omitempty"`
	Run       string     `yaml:",omitempty"`

	// Wait is wait time in milliseconds as a string.
	Wait string `yaml:",omitempty"`

	Goto string `yaml:",omitempty"`

	Branch string `yaml:",omitempty"`

	Ingest *Ingest `yaml:",omitempty"`
}

// exec calls exe() and then handles Fails (if any).
func (s *Step) exec(ctx *Ctx, t *Test) (string, error) {
	next, err := s.exe(ctx, t)
	if err != nil {
		if _, is := IsBroken(err); is {
			return "", err
		}
		if s.Fails {
			return s.Goto, nil
		}
		return "", err
	}

	return next, err
}

// exe executes the step.
//
// Called by exec().
func (s *Step) exe(ctx *Ctx, t *Test) (string, error) {
	// ToDo: Warn if multiple Pub, Sub, Recv, Wait, Goto specified?

	t.Tick(ctx)

	if s.Skip {
		ctx.Indf("    Skip")
		return "", nil
	}

	if s.Pub != nil {
		ctx.Indf("    Pub to %s", s.Pub.Chan)

		e, err := s.Pub.Substitute(ctx, t)
		if err != nil {
			return "", err
		}

		if err := t.ensureChan(ctx, e.Chan, &e.ch); err != nil {
			return "", err
		}

		if err := e.Exec(ctx, t); err != nil {
			return "", err
		}
	}
	if s.Sub != nil {
		ctx.Indf("    Sub %s", s.Sub.Chan)

		e, err := s.Sub.Substitute(ctx, t)
		if err != nil {
			return "", err
		}

		if err := t.ensureChan(ctx, e.Chan, &e.ch); err != nil {
			return "", err
		}

		if err := e.Exec(ctx, t); err != nil {
			return "", err
		}
	}
	if s.Recv != nil {
		ctx.Indf("    Recv %s", s.Recv.Chan)

		e, err := s.Recv.Substitute(ctx, t)
		if err != nil {
			return "", err
		}

		if err := t.ensureChan(ctx, e.Chan, &e.ch); err != nil {
			return "", err
		}

		if err := e.Exec(ctx, t); err != nil {
			return "", err
		}
	}
	if s.Reconnect != nil {
		ctx.Indf("    Reconnect %s", s.Reconnect.Chan)

		e, err := s.Reconnect.Substitute(ctx, t)
		if err != nil {
			return "", err
		}

		if err := t.ensureChan(ctx, e.Chan, &e.ch); err != nil {
			return "", err
		}

		if err := e.Exec(ctx, t); err != nil {
			return "", err
		}
	}
	if s.Close != nil {
		ctx.Indf("    Close %s", s.Close.Chan)

		e, err := s.Close.Substitute(ctx, t)
		if err != nil {
			return "", err
		}

		if err := t.ensureChan(ctx, e.Chan, &e.ch); err != nil {
			return "", err
		}

		if err := e.Exec(ctx, t); err != nil {
			return "", err
		}
	}
	if s.Ingest != nil {
		ctx.Indf("    Ingest %s", s.Ingest.Chan)

		e, err := s.Ingest.Substitute(ctx, t)
		if err != nil {
			return "", err
		}

		if err := t.ensureChan(ctx, e.Chan, &e.ch); err != nil {
			return "", err
		}

		if err := e.Exec(ctx, t); err != nil {
			return "", err
		}
	}

	if s.Kill != nil {
		ctx.Indf("    Kill %s", s.Kill.Chan)

		e, err := s.Kill.Substitute(ctx, t)
		if err != nil {
			return "", err
		}

		if err := t.ensureChan(ctx, e.Chan, &e.ch); err != nil {
			return "", err
		}

		if err := e.Exec(ctx, t); err != nil {
			return "", err
		}
	}

	if s.Branch != "" {
		ctx.Indf("    Branch %s", short(s.Branch))

		src, err := t.Bindings.StringSub(ctx, s.Branch)
		if err != nil {
			return "", err
		}

		if src, err = t.prepareSource(ctx, src); err != nil {
			return "", err
		}

		x, err := JSExec(ctx, src, t.jsEnv(ctx))
		if err != nil {
			return "", err
		}

		target, is := x.(string)
		if !is {
			return "", Brokenf("Branch Javascript returned a %T (%#v) and not a %T", x, x, target)
		}

		ctx.Indf("    Branch returned '%s'", target)

		return target, nil
	}

	if s.Run != "" {
		ctx.Indf("    Run %s", short(s.Run))

		src, err := t.Bindings.StringSub(ctx, s.Run)
		if err != nil {
			return "", err
		}

		if src, err = t.prepareSource(ctx, src); err != nil {
			return "", err
		}

		_, err = JSExec(ctx, src, t.jsEnv(ctx))

		ctx.Inddf("    Bindings: %s", JSON(t.Bindings))

		return "", err
	}

	if s.Wait != "" {
		ctx.Indf("    Wait %s", s.Wait)

		duration, err := t.Bindings.StringSub(ctx, s.Wait)
		if err != nil {
			return "", err
		}

		if err := Wait(ctx, duration); err != nil {
			return "", err
		}

		return "", nil
	}

	return s.Goto, nil
}

// Wait will attempt to parse the duration and then sleep accordingly.
func Wait(ctx *Ctx, durationString string) error {
	d, err := time.ParseDuration(durationString)
	if err != nil {
		return Brokenf("error parsing Wait '%s'", durationString)
	}

	time.Sleep(d)

	return nil
}

type Pub struct {
	Chan  string
	Topic string

	// Schema is an optional URI for a JSON Schema that's used to
	// validate outgoing messages.
	Schema string `json:",omitempty" yaml:",omitempty"`

	Payload interface{}

	payload string

	// Serialization specifies how a string Payload should be
	// deserialized (if at all).
	//
	// Legal values: 'json', 'text'.  Default is 'json'.
	//
	// If given a non-string, that value is always used as is.
	//
	// If given a string, if serialization is 'json' or not
	// specified, then the string is parsed as JSON.  If the
	// serialization is 'text', then the string is used as is.
	Serialization string `json:",omitempty" yaml:",omitempty"`

	Run string `json:",omitempty" yaml:",omitempty"`

	ch Chan
}

func (p *Pub) Substitute(ctx *Ctx, t *Test) (*Pub, error) {

	topic, err := t.Bindings.StringSub(ctx, p.Topic)
	if err != nil {
		return nil, err
	}
	ctx.Inddf("    Effective topic: %s", topic)

	payload, err := t.Bindings.SerialSub(ctx, p.Serialization, p.Payload)
	if err != nil {
		return nil, err
	}

	ctx.Inddf("    Effective payload: %s", payload)

	run, err := t.Bindings.StringSub(ctx, p.Run)
	if err != nil {
		return nil, err
	}
	if run != "" {
		ctx.Inddf("    Effective code (run): %s", run)
	}

	return &Pub{
		Chan:          p.Chan,
		Topic:         topic,
		Payload:       p.Payload,
		Serialization: p.Serialization,
		payload:       payload,
		Run:           run,
		ch:            p.ch,
	}, nil

}

func (p *Pub) Exec(ctx *Ctx, t *Test) error {
	ctx.Indf("    Pub topic '%s'", p.Topic)
	ctx.Inddf("        payload %s", p.payload)

	if p.Schema != "" {
		if err := validateSchema(ctx, p.Schema, p.payload); err != nil {
			return err
		}
	}

	err := p.ch.Pub(ctx, Msg{
		Topic:   p.Topic,
		Payload: p.payload,
	})

	if err != nil {
		return err
	}

	if p.Run != "" {
		src, err := t.prepareSource(ctx, p.Run)
		if err != nil {
			return err
		}

		env := map[string]interface{}{
			"test":    t,
			"elapsed": float64(t.elapsed) / 1000 / 1000, // Milliseconds
		}
		if _, err = JSExec(ctx, src, env); err != nil {
			return err
		}
	}

	return nil

}

type Sub struct {
	Chan  string
	Topic string

	// Pattern, which is deprecated, is really 'Topic'.
	Pattern string

	ch Chan
}

func (s *Sub) Substitute(ctx *Ctx, t *Test) (*Sub, error) {

	// Backwards compatibility.
	if s.Pattern != "" {
		ctx.Indf("warning: Sub.Pattern is deprecated. Use Sub.Topic instead.")
		if s.Topic != "" {
			return nil, fmt.Errorf("just specify Topic (and not Pattern, which is deprecated)")
		}
		s.Topic = s.Pattern // We'll use s.Topic from here on.
		s.Pattern = ""
	}
	pat, err := t.Bindings.StringSub(ctx, s.Topic)
	if err != nil {
		return nil, err
	}
	return &Sub{
		Chan:  s.Chan,
		Topic: pat,
		ch:    s.ch,
	}, nil
}

func (s *Sub) Exec(ctx *Ctx, t *Test) error {
	ctx.Indf("    Sub %s", s.Topic)
	return s.ch.Sub(ctx, s.Topic)
}

type Kill struct {
	Chan string

	ch Chan
}

func (p *Kill) Substitute(ctx *Ctx, t *Test) (*Kill, error) {
	return p, nil
}

func (p *Kill) Exec(ctx *Ctx, t *Test) error {
	ctx.Indf("    Kill %s", JSON(p))

	return p.ch.Kill(ctx)
}

type Reconnect struct {
	Chan string

	ch Chan
}

func (p *Reconnect) Substitute(ctx *Ctx, t *Test) (*Reconnect, error) {
	return p, nil
}

func (p *Reconnect) Exec(ctx *Ctx, t *Test) error {
	ctx.Indf("    Reconnect %s", JSON(p))

	return p.ch.Open(ctx)
}

type Close struct {
	Chan string

	ch Chan
}

func (p *Close) Substitute(ctx *Ctx, t *Test) (*Close, error) {
	return p, nil
}

func (p *Close) Exec(ctx *Ctx, t *Test) error {
	ctx.Indf("    Close %s", JSON(p))

	err := p.ch.Close(ctx)
	if err == nil {
		ctx.Indf("    Removing %s", p.Chan)
		delete(t.Chans, p.Chan)
	}

	return err
}

type Ingest struct {
	Chan    string
	Topic   string
	Payload interface{}
	// Timeout time.Duration

	ch Chan
}

func (i *Ingest) Substitute(ctx *Ctx, t *Test) (*Ingest, error) {
	topic, err := t.Bindings.StringSub(ctx, i.Topic)
	if err != nil {
		return nil, err
	}

	var pay string
	if s, is := i.Payload.(string); is {
		pay = s
	} else {
		js, err := subst.JSONMarshal(&i.Payload)
		if err != nil {
			return nil, err
		}
		pay = string(js)
	}

	if pay, err = t.Bindings.Sub(ctx, pay); err != nil {
		return nil, err
	}

	return &Ingest{
		Chan:    i.Chan,
		Topic:   topic,
		Payload: pay,
		ch:      i.ch,
	}, nil

}

func (i *Ingest) Exec(ctx *Ctx, t *Test) error {
	payload, is := i.Payload.(string)
	if !is {
		js, err := subst.JSONMarshal(&i.Payload)
		if err != nil {
			return err
		}
		payload = string(js)
	}
	m := Msg{
		Topic:   i.Topic,
		Payload: payload,
	}

	return i.ch.To(ctx, m)
}

type Exec struct {
	Process
	Pattern interface{}
}

func (e *Exec) Exec(ctx *Ctx, t *Test) error {
	panic("todo")
}

func CopyBindings(bs map[string]interface{}) map[string]interface{} {
	if bs == nil {
		return make(map[string]interface{})
	}
	acc := make(map[string]interface{}, len(bs))
	for p, v := range bs {
		acc[p] = v
	}
	return acc
}

func (t *Test) jsEnv(ctx *Ctx) map[string]interface{} {
	bs := CopyBindings(t.Bindings)
	return map[string]interface{}{
		"bindings": Canon(bs),
		"bs":       Canon(bs),
		"test":     t,
		"elapsed":  float64(t.elapsed) / 1000 / 1000, // Milliseconds
	}
}
