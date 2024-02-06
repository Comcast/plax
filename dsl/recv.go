package dsl

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Comcast/sheens/match"
	jschema "github.com/xeipuuv/gojsonschema"
)

// Recv is a major Step that checks an in-coming message.
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

// Substitute bindings for the Recv.
//
// Returns a new Recv.
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

// validateSchema checks that the payload has schema at the given URI.
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

// attemptMatch performs either pattern- or regex-based matching on
// the incoming message.
func (r *Recv) attemptMatch(ctx *Ctx, t *Test, m Msg) ([]match.Bindings, error) {
	// target will be the target (message) for matching.
	var target interface{}
	if err := json.Unmarshal([]byte(m.Payload), &target); err != nil {
		return nil, err
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
		return nil, Brokenf("bad Recv Target: '%s'", r.Target)
	}

	ctx.Inddf("      match target:  %s", JSON(target))

	if r.Schema != "" {
		if err := validateSchema(ctx, r.Schema, m.Payload); err != nil {
			return nil, err
		}
	}

	target = Canon(target)
	t.Bindings.Clean(ctx, r.ClearBindings)
	pattern, err := t.Bindings.Bind(ctx, r.Pattern)
	if err != nil {
		return nil, err
	}

	ctx.Inddf("      bound pattern: %s", JSON(pattern))
	return match.Match(pattern, target, match.NewBindings())
}

// extendBindings updates the Test bindings based on the
// match-generated bindings.
//
// This code doesn't actually use the Recv, but it's here as a method
// anyway for organization (?).
func (r *Recv) extendBindings(ctx *Ctx, t *Test, bss []match.Bindings) error {
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

	return nil
}

// checkGuard invokes the Guard (if any) and returns whether the Guard
// is satisfied.
func (r *Recv) checkGuard(ctx *Ctx, t *Test, m Msg, bss []match.Bindings) (bool, error) {
	if r.Guard == "" {
		return true, nil
	}

	ctx.Indf("    Recv guard")
	src, err := t.prepareSource(ctx, r.Guard)
	if err != nil {
		return false, err
	}

	env := t.jsEnv(ctx)
	can := Canon(bss)
	env["bindingss"] = can
	env["bss"] = can
	env["msg"] = m

	x, err := JSExec(ctx, src, env)
	if f, is := IsFailure(x); is {
		return false, f
	}
	if f, is := IsFailure(err); is {
		return false, f
	}
	if err != nil {
		return false, err
	}

	switch vv := x.(type) {
	case bool:
		if !vv {
			ctx.Indf("    Recv guard not pleased")
			return false, nil
		}
		ctx.Indf("    Recv guard satisfied")
	default:
		return false, Brokenf("Guard Javascript returned a %T (%v) and not a bool", x, x)
	}

	return true, nil
}

// runRun runs the Run (if any).
func (r *Recv) runRun(ctx *Ctx, t *Test, m Msg, bss []match.Bindings) error {
	if r.Run == "" {
		return nil
	}
	src, err := t.prepareSource(ctx, r.Run)
	if err != nil {
		return err
	}

	env := t.jsEnv(ctx)
	can := Canon(&bss)
	env["bindingss"] = can
	env["bss"] = can
	env["msg"] = m

	_, err = JSExec(ctx, src, env)
	return err
}

// verify is the top-level method for verifying an incoming message.
func (r *Recv) verify(ctx *Ctx, t *Test, m Msg) (bool, error) {
	var (
		err error
		bss []match.Bindings
	)

	if r.Regexp != "" {
		if r.Target != "payload" {
			return false, Brokenf("can only regexp-match against payload (not also topic)")
		}
		ctx.Inddf("      regexp: %s", r.Regexp)
		bss, err = RegexpMatch(r.Regexp, m.Payload)
	} else {
		ctx.Inddf("      pattern:       %s", JSON(r.Pattern))
		bss, err = r.attemptMatch(ctx, t, m)
	}

	if err != nil {
		return false, err
	}
	ctx.Indf("      result: %v", 0 < len(bss))
	ctx.Inddf("      bss: %s", JSON(bss))

	if 0 == len(bss) {
		return false, nil
	}

	if err = r.extendBindings(ctx, t, bss); err != nil {
		return false, err
	}

	happy, err := r.checkGuard(ctx, t, m, bss)
	if err != nil {
		return false, err
	}
	if !happy {
		return false, nil
	}

	ctx.Indf("    Recv satisfied")
	ctx.Inddf("      t.Bindings: %s", JSON(t.Bindings))

	if err = r.runRun(ctx, t, m, bss); err != nil {
		return false, err
	}

	return true, nil
}

// Exec performs that main Recv processing.
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

			// Verify that either no Recv topic was
			// provided or that the Recv topic is equal to
			// the message topic.  Might want to
			// generalize.  See Issue 105.
			if r.Topic == "" || r.Topic == m.Topic {
				// Only increment the number of attempts given a topic match.
				attempts++

				ctx.Indf("    Recv match attempt %d:", attempts)
				happy, err := r.verify(ctx, t, m)
				if err != nil {
					return err
				}
				if happy {
					return nil
				}
			}

			// Give up if the receiver attempts was
			// specified (not 0) and that the actual
			// number of attempts has been reached
			if r.Attempts != 0 && attempts >= r.Attempts {
				ctx.Inddf("      attempts: %d of %d", attempts, r.Attempts)
				ctx.Inddf("      topic: %s", r.Topic)

				// Make a nice error message.
				var match string
				if r.Regexp == "" {
					match = fmt.Sprintf("pattern: %s", r.Pattern)
				} else {
					match = fmt.Sprintf("regexp: %s", r.Regexp)
				}
				var topic string
				if r.Topic == "" {
					topic = "<none>"
				} else {
					topic = r.Topic
				}
				return fmt.Errorf("%d attempt(s) reached; expected maximum of %d attempt(s) to match %s (topic %s)",
					attempts, r.Attempts, match, topic)
			}
		}
	}

	// Should never get here.
	//
	//   Let us become thoroughly sensible of the weakness,
	//   blindness, and narrow limits of human reason.
	//
	//   --David Hume
	//
	return fmt.Errorf("impossible!")
}
