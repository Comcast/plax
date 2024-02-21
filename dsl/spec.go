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

	"github.com/Comcast/plax/subst"
	"github.com/Comcast/sheens/match"
	jschema "github.com/xeipuuv/gojsonschema"
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

type Recv struct {
	Chan  string
	Topic string

	// Pattern is a Sheens pattern
	// https://github.com/Comcast/sheens/blob/main/README.md#pattern-matching
	// for matching incoming messages.
	//
	// Use a pattern for matching JSON-serialized messages.
	//
	// Also see Regexp.
	Pattern interface{}

	// Regexp, which is an alternative to Pattern, gives a (Go)
	// regular expression used to match incoming messages.
	//
	// A named group match becomes a bound variable.
	Regexp string

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

	// Schema is an optional URI for a JSON Schema that's used to
	// validate incoming messages before other processing.
	Schema string `json:",omitempty" yaml:",omitempty"`

	// Max attempts to receive a message; optionally for a specific topic
	Attempts int `json:",omitempty" yaml:",omitempty`

	ch Chan
}

// Substitute bindings for the receiver
func (r *Recv) Substitute(ctx *Ctx, t *Test) (*Recv, error) {

	// Canonicalize r.Target.
	switch r.Target {
	case "payload", "Payload", "":
		r.Target = "payload"
	case "msg", "message", "Message":
		r.Target = "msg"
	default:
		return nil, NewBroken(fmt.Errorf("bad Recv Target: '%s'", r.Target))
	}

	t.Bindings.Clean(ctx, r.ClearBindings)

	topic, err := t.Bindings.StringSub(ctx, r.Topic)
	if err != nil {
		return nil, err
	}
	ctx.Inddf("    Effective topic: %s", topic)

	var pat = r.Pattern
	var reg = r.Regexp
	if r.Regexp == "" {
		// ToDo: Probably go with an explicit
		// 'PatternSerialization' property.  Might also need a
		// 'MessageSerialization' property, too.  Alternately,
		// rely on regex matching for non-text messages and
		// patterns.
		js, err := t.Bindings.SerialSub(ctx, "", r.Pattern)
		if err != nil {
			return nil, err
		}
		var x interface{}
		if err = json.Unmarshal([]byte(js), &x); err != nil {
			// See the ToDo above.  If we can't
			// deserialize, we'll just go with the string
			// literal.
			pat = js
		} else {
			pat = x
		}

		ctx.Inddf("    Effective pattern: %s", JSON(pat))

	} else {
		if r.Pattern != nil {
			return nil, Brokenf("can't have both Pattern and Regexp")
		}
		if reg, err = t.Bindings.StringSub(ctx, reg); err != nil {
			return nil, err
		}
		ctx.Inddf("    Effective regexp: %s", reg)
	}

	guard, err := t.Bindings.StringSub(ctx, r.Guard)
	if err != nil {
		return nil, err
	}

	run, err := t.Bindings.StringSub(ctx, r.Run)
	if err != nil {
		return nil, err
	}

	return &Recv{
		Chan:     r.Chan,
		Topic:    topic,
		Pattern:  pat,
		Regexp:   reg,
		Timeout:  r.Timeout,
		Target:   r.Target,
		Guard:    guard,
		Run:      run,
		Schema:   r.Schema,
		Attempts: r.Attempts,
		ch:       r.ch,
	}, nil
}

func validateSchema(ctx *Ctx, schemaURI string, payload string) error {
	ctx.Indf("      schema: %s", schemaURI)
	var (
		doc    = jschema.NewStringLoader(payload)
		schema = jschema.NewReferenceLoader(schemaURI)
	)

	v, err := jschema.Validate(schema, doc)
	if err != nil {
		return Brokenf("schema validation error: %v", err)
	}
	if !v.Valid() {
		var (
			errs       = v.Errors()
			complaints = make([]string, len(errs))
		)
		for i, err := range errs {
			complaints[i] = err.String()
			ctx.Indf("      schema invalidation: %s", err)
		}
		return fmt.Errorf("schema (%s) validation errors: %s",
			schemaURI, strings.Join(complaints, "; "))
	}
	ctx.Indf("      schema validated")
	return nil
}

// Exec the receiver
func (r *Recv) Exec(ctx *Ctx, t *Test) error {
	var (
		timeout  = r.Timeout
		in       = r.ch.Recv(ctx)
		attempts = 0
	)

	if timeout == 0 {
		timeout = time.Second * 60 * 20 * 24
	}

	tm := time.NewTimer(timeout)

	if r.Regexp != "" {
		ctx.Inddf("    Recv regexp %s", r.Regexp)
	} else {
		ctx.Inddf("    Recv pattern (%T) %v", r.Pattern, r.Pattern)
	}

	ctx.Inddf("    Recv target %s", r.Target)
	for {
		select {
		case <-ctx.Done():
			ctx.Indf("    Recv canceled")
			return nil
		case <-tm.C:
			ctx.Indf("    Recv timeout (%v)", timeout)
			return fmt.Errorf("timeout after %s waiting for %s", timeout, r.Pattern)
		case m := <-in:
			ctx.Indf("    Recv dequeuing topic '%s' (vs '%s')", m.Topic, r.Topic)
			ctx.Inddf("                   %s", m.Payload)

			var (
				err error
				bss []match.Bindings
			)

			// Verify that either no Recv topic was
			// provided or that the receiver topic is
			// equal to the message topic
			if r.Topic == "" || r.Topic == m.Topic {
				ctx.Indf("    Recv match:")

				if r.Regexp != "" {
					ctx.Inddf("      regexp: %s", r.Regexp)
					if r.Target != "payload" {
						return Brokenf("can only regexp-match against payload (not also topic)")
					}
					bss, err = RegexpMatch(r.Regexp, m.Payload)
				} else {
					ctx.Inddf("      pattern:       %s", JSON(r.Pattern))

					// target will be the target (message) for matching.
					var target interface{}
					if err = json.Unmarshal([]byte(m.Payload), &target); err != nil {
						return err
					}

					switch r.Target {
					case "payload":
						// Match against only the (deserialized) payload.
					case "msg":
						// Match against the full message
						// (with topic and deserialized
						// payload).
						target = map[string]interface{}{
							"Topic":   m.Topic,
							"Payload": target,
						}
					default:
						return Brokenf("bad Recv Target: '%s'", r.Target)
					}

					ctx.Inddf("      match target:  %s", JSON(target))

					if r.Schema != "" {
						if err := validateSchema(ctx, r.Schema, m.Payload); err != nil {
							return err
						}
					}

					target = Canon(target)
					t.Bindings.Clean(ctx, r.ClearBindings)
					pattern, err := t.Bindings.Bind(ctx, r.Pattern)
					if err != nil {
						return err
					}

					ctx.Inddf("      bound pattern: %s", JSON(pattern))
					bss, err = match.Match(pattern, target, match.NewBindings())
				}

				if err != nil {
					return err
				}
				ctx.Indf("      result: %v", 0 < len(bss))

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
						js, _ = subst.JSONMarshal(&bss[0])
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

					ctx.BindingsRedactions(t.Bindings)

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

				// Only increment the number of attempts given a topic match.
				attempts++
			}

			// Verify the receiver attempts was specified (not 0) and that
			// the actual number of attempts has been reached
			if r.Attempts != 0 && attempts >= r.Attempts {
				ctx.Inddf("      attempts: %d of %d", attempts, r.Attempts)
				ctx.Inddf("      topic: %s", r.Topic)
				match := fmt.Sprintf("pattern: %s", r.Pattern)
				if r.Regexp != "" {
					match = fmt.Sprintf("regexp: %s", r.Regexp)
				}
				if r.Topic != "" {
					return fmt.Errorf("%d attempt(s) reached; expected maximum of %d attempt(s) to match %s on topic %s", attempts, r.Attempts, match, r.Topic)
				}
				return fmt.Errorf("%d attempt(s) reached; expected maximum of %d attempt(s) to match %s", attempts, r.Attempts, match)
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
		"bindings": bs,
		"bs":       bs,
		"test":     t,
		"elapsed":  float64(t.elapsed) / 1000 / 1000, // Milliseconds
	}
}
