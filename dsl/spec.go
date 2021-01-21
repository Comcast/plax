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
	"strings"
	"time"

	"github.com/Comcast/sheens/match"
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
	Run       string     `yaml:",omitempty"`

	// Wait is wait time in milliseconds as a string.
	Wait string `yaml:",omitempty"`

	Goto string `yaml:",omitempty"`

	Branch string `yaml:",omitempty"`

	Ingest *Ingest `yaml:",omitempty"`
}

func (s *Step) exec(ctx *Ctx, t *Test) (string, error) {
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
	Chan    string
	Topic   string
	Payload interface{}
	Run     string `json:",omitempty" yaml:",omitempty"`

	ch Chan
}

func (p *Pub) Substitute(ctx *Ctx, t *Test) (*Pub, error) {
	topic, err := t.Bindings.StringSub(ctx, p.Topic)
	if err != nil {
		return nil, err
	}
	ctx.Inddf("    Effective topic: %s", topic)

	var pay interface{}
	if err := t.Bindings.Sub(ctx, p.Payload, &pay, true); err != nil {
		return nil, err
	}

	payjs, err := json.Marshal(&pay)
	if err != nil {
		return nil, err
	}
	ctx.Inddf("    Effective payload: %s", payjs)

	run, err := t.Bindings.StringSub(ctx, p.Run)
	if err != nil {
		return nil, err
	}
	if run != "" {
		ctx.Inddf("    Effective code (run): %s", run)
	}

	return &Pub{
		Chan:    p.Chan,
		Topic:   topic,
		Payload: string(payjs),
		Run:     run,
		ch:      p.ch,
	}, nil

}

func (p *Pub) Exec(ctx *Ctx, t *Test) error {
	ctx.Indf("    Pub topic '%s'", p.Topic)
	ctx.Inddf("        payload %s", p.Payload)

	err := p.ch.Pub(ctx, Msg{
		Topic:   p.Topic,
		Payload: p.Payload,
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

type Recv struct {
	Chan    string
	Topic   string
	Pattern interface{}
	Timeout time.Duration

	// Target is an optional switch to specify what part of the
	// incoming message is considered for matching.
	//
	// By default, only the payload is matched.  If Target is
	// "message", then matching is performed against
	//
	//   {"Topic":TOPIC,"Payload":PAYLOAD}
	//
	// which allows matching based on the topic of in-bound
	// messages.
	Target string

	// ClearBindings will remove all bindings for variables that
	// do not start with '?!' before executing this step.
	ClearBindings bool

	// Guard is optional Javascript (!) that should return a
	// boolean to indicate whether this Recv has been satisfied.
	//
	// The code is executed in a function body, and the code
	// should 'return' a boolean.
	//
	// The following variables are bound in the global
	// environment:
	//
	//   bindingss: the set (array) of bindings returned by match()
	//
	//   elapsed: the elapsed time in milliseconds since the last step
	//
	//   msg: the receved message ({"topic":TOPIC,"payload":PAYLOAD})
	//
	//   print: a function that prints its arguments to stdout.
	//
	Guard string `json:",omitempty" yaml:",omitempty"`

	Run string `json:",omitempty" yaml:",omitempty"`

	ch Chan
}

func (r *Recv) Substitute(ctx *Ctx, t *Test) (*Recv, error) {

	// Always remove "temporary" bindings.
	for p, _ := range t.Bindings {
		if strings.HasPrefix(p, "?*") {
			delete(t.Bindings, p)
		}
	}

	if r.ClearBindings {
		ctx.Indf("    Clearing bindings (%d) by request", len(t.Bindings))
		for p, _ := range t.Bindings {
			if !strings.HasPrefix(p, "?!") {
				delete(t.Bindings, p)
			}
		}
	}

	topic, err := t.Bindings.StringSub(ctx, r.Topic)
	if err != nil {
		return nil, err
	}
	if topic != r.Topic {
		ctx.Indf("    Topic expansion: %s", topic)
	}

	// Pattern must always be structured.  If we are given a
	// string, it's interpreted as a JSON string.

	ctx.Inddf("    Given pattern: %s", JSON(r.Pattern))
	var pat interface{}
	if err := t.Bindings.Sub(ctx, r.Pattern, &pat, true); err != nil {
		return nil, err
	}
	ctx.Inddf("    Effective pattern: %s", JSON(pat))

	guard, err := t.Bindings.StringSub(ctx, r.Guard)
	if err != nil {
		return nil, err
	}

	run, err := t.Bindings.StringSub(ctx, r.Run)
	if err != nil {
		return nil, err
	}

	return &Recv{
		Chan:    r.Chan,
		Topic:   topic,
		Pattern: pat,
		Timeout: r.Timeout,
		Target:  r.Target,
		Guard:   guard,
		Run:     run,
		ch:      r.ch,
	}, nil
}

func (r *Recv) Exec(ctx *Ctx, t *Test) error {
	var (
		timeout = r.Timeout
		in      = r.ch.Recv(ctx)
		pat     = r.Pattern
	)

	if timeout == 0 {
		timeout = time.Second * 60 * 20 * 24
	}

	tm := time.NewTimer(timeout)

	switch r.Target {
	case "payload", "Payload", "":
		r.Target = "payload"
	case "msg", "message", "Message":
		r.Target = "msg"
	default:
		return NewBroken(fmt.Errorf("Bad Recv Target: '%s'", r.Target))
	}

	ctx.Inddf("    Recv pattern %s", JSON(pat))
	ctx.Inddf("    Recv target %s", r.Target)
	for {
		select {
		case <-ctx.Done():
			ctx.Indf("    Recv canceled")
			return nil
		case <-tm.C:
			ctx.Indf("    Recv timeout (%v)", timeout)
			return fmt.Errorf("timeout after %s waiting for %s", timeout, JSON(pat))
		case m := <-in:
			ctx.Indf("    Recv dequeuing '%s'", m.Topic)
			ctx.Inddf("                   %s", JSON(m.Payload))

			m.Payload = MaybeParseJSON(m.Payload)
			var target interface{} = map[string]interface{}{
				"Topic":   m.Topic,
				"Payload": m.Payload,
			}

			switch r.Target {
			case "payload":
				target = m.Payload
			case "msg":
			default:
				return NewBroken(fmt.Errorf("Bad Recv Target: '%s'", r.Target))
			}

			ctx.Inddf("    Recv considering %s", JSON(m))
			if pat != nil {

				// We are giving empty bindings to
				// 'Match' because we have already
				// substituted bindings in pat as part of
				// our recursive, fancy substitution
				// logic (that includes '!!' and '@@'
				// substitutions along with bindings
				// substitions, which can occur in
				// string contexts in additional to
				// structural contexts.
				//
				// If we waited to structural bindings
				// substitution until now, then
				// string-context bindings substitution
				// would be inconsistent with that
				// late use of bindings here.
				//
				// ToDo: Reconsider.

				bss, err := match.Match(pat, Canon(target), match.NewBindings())
				if err != nil {
					return err
				}
				ctx.Indf("    Recv match:")
				ctx.Inddf("      pattern: %s", JSON(pat))
				ctx.Inddf("      msg:     %s", JSON(m))
				ctx.Indf("      result: %v", 0 < len(bss))
				ctx.Inddf("      bss: %s", JSON(bss))
				if 0 < len(bss) {

					if 1 < len(bss) {
						// Let's protest if we get
						// multiple sets of bindings.
						//
						// Better safe than sorry?  If
						// we start running into this
						// situation, let's figure out
						// the best way to proceed.
						// Otherwise we might not notice
						// unintended behavior.
						return fmt.Errorf("multiple bindings sets: %s", JSON(bss))
					}

					// Extend rather than replace
					// t.Bindings.  Note that we have to
					// extend t.Bindings rather than replace
					// it due to the bindings substitution
					// logic.  See the comments above
					// 'Match' above.
					//
					// ToDo: Contemplate possibility for
					// inconsistencies.
					//
					// Thanks, Carlos, for this fix!
					if t.Bindings == nil {
						// Some unit tests might not
						// have initialized t.Bindings.
						t.Bindings = make(map[string]interface{})
					}
					for p, v := range bss[0] {
						if x, have := t.Bindings[p]; have {
							// Let's see if we are
							// changing an existing
							// binding.  If so, note
							// that.
							js0 := JSON(v)
							js1 := JSON(x)
							if js0 != js1 {
								ctx.Indf("    Updating binding for %s", p)
							}
						}
						t.Bindings[p] = v
					}

					if r.Guard != "" {
						ctx.Indf("    Recv guard")
						src, err := t.prepareSource(ctx, r.Guard)
						if err != nil {
							return err
						}

						// Convert bss to a stripped representation ...
						js, _ := json.Marshal(&bss)
						var bindingss interface{}
						json.Unmarshal(js, &bindingss)
						// And again ...
						var bs interface{}
						js, _ = json.Marshal(&bss[0])
						json.Unmarshal(js, &bs)

						env := t.jsEnv(ctx)
						env["bindingss"] = bindingss
						env["msg"] = m

						x, err := JSExec(ctx, src, env)
						if f, is := IsFailure(x); is {
							return f
						}
						if f, is := IsFailure(err); is {
							return f
						}
						if err != nil {
							return err
						}

						switch vv := x.(type) {
						case bool:
							if !vv {
								ctx.Indf("    Recv guard not pleased")
								continue
							}
							ctx.Indf("    Recv guard satisfied")
						default:
							return Brokenf("Guard Javascript returned a %T (%v) and not a bool", x, x)
						}
					}

					ctx.Indf("    Recv satisfied")
					ctx.Inddf("      t.Bindings: %s", JSON(t.Bindings))

					if r.Run != "" {
						src, err := t.prepareSource(ctx, r.Run)
						if err != nil {
							return err
						}

						// Convert bss to a stripped representation ...
						env := t.jsEnv(ctx)
						can := Canon(&bss)
						env["bindingss"] = can
						env["bss"] = can
						env["msg"] = m

						if _, err = JSExec(ctx, src, env); err != nil {
							return err
						}
					}

					return nil
				}
			}
		}
	}

	return fmt.Errorf("impossible!")
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

	var pay interface{}
	if err = t.Bindings.Sub(ctx, i.Payload, &pay, true); err != nil {
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
	m := Msg{
		Topic:   i.Topic,
		Payload: i.Payload,
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
		"bindings": bs,
		"bs":       bs,
		"test":     t,
		"elapsed":  float64(t.elapsed) / 1000 / 1000, // Milliseconds
	}
}
