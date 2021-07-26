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
	"io/ioutil"
	"strings"
	"time"
)

var (
	DefaultMaxSteps = 100
)

// Test is the top-level type for a complete test.
type Test struct {
	// Id usually comes from the filename that defines the test.
	Id string `json:",omitempty" yaml:",omitempty"`

	// Name is the test specification name
	Name string `json:",omitempty" yaml:",omitempty"`

	// Doc is an optional documentation string.
	Doc string `json:",omitempty" yaml:",omitempty"`

	// Labels is an optional set of labels (e.g., "cpe", "app").
	Labels []string `json:",omitempty" yaml:",omitempty"`

	// Priority 0 is the highest priority.
	Priority int

	// Spec is the test specification.
	//
	// Parts of the Spec are subject to bindings substitution.
	Spec *Spec

	// State is arbitrary state the Javascript code can use.
	State map[string]interface{}

	// Bindings is the first set of bindings returned by the last
	// pattern match (if any).
	Bindings Bindings

	// Chans is the map of Chan names to Chans.
	Chans map[string]Chan

	// T is the time the last Step executed.
	T time.Time

	// Optional seed for random number generator.
	//
	// Effectively defaults to the current time in UNIX
	// nanoseconds
	Seed int64

	// MaxSteps, when not zero, is the maximum number of steps to
	// execute.
	//
	// Can act as a circuit breaker for infinite loops due to
	// branches.
	MaxSteps int

	// Libraries is a list of filenames that should contain
	// Javascript.  This source is loaded into each Javascript
	// environment.
	//
	// Warning: These files are loaded for each Javascript
	// invocation (because re-using the Javascript environment is
	// not a safe thing to do -- and I don't think I can "seal"
	// one environment and extend it per-invocation).
	Libraries []string

	// Negative indicates that a reported failure (but not error)
	// should be interpreted as a success.
	Negative bool

	// elapsed is duration between the most recent steps.
	elapsed time.Duration

	// Dir is the base directory for reading relative pathnames
	// (for libraries, includes, and ##FILENAMEs).
	Dir string

	// Retries is an optional retry specification.
	//
	// This data isn't actually used in the code here.  Instead,
	// this data is here to make it easy for a test to specify its
	// own retry policy (if any).  Actual implementation provided
	// by invoke.Run().
	Retries *Retries

	// Registry is the channel (type) registry for this test.
	//
	// Defaults to TheChanRegistry.
	Registry ChanRegistry
}

// NewTest create a initialized NewTest from the id and Spec
func NewTest(ctx *Ctx, id string, s *Spec) *Test {
	return &Test{
		Id:       id,
		Spec:     s,
		Chans:    make(map[string]Chan),
		State:    make(map[string]interface{}),
		Bindings: make(map[string]interface{}),
		MaxSteps: DefaultMaxSteps,
		T:        time.Now().UTC(),
	}
}

// Wanted reports whether a test meets the given requirements.
func (t *Test) Wanted(ctx *Ctx, lowestPriority int, labels []string, tests []string) bool {
	if 0 <= lowestPriority && t.Priority > lowestPriority {
		return false
	}
LABELS:
	for _, label := range labels {
		if label == "" {
			continue
		}
		for _, have := range t.Labels {
			if label == have {
				continue LABELS
			}
		}
		return false
	}

	// Iterate over specified suite tests to see if they are wanted
	for _, name := range tests {
		if t.Name == name {
			// Suite tests is wanted
			return true
		}
	}
	// Reaching here means the suite test was not wanted
	if len(tests) > 0 {
		return false
	}

	return true
}

// Tick returns the duration since the last Tick.
func (t *Test) Tick(ctx *Ctx) time.Duration {
	now := time.Now().UTC()
	t.elapsed = now.Sub(t.T)
	t.T = now
	return t.elapsed
}

// HappyTerminalPhases is the set of phase names that indicate that
// the test has completed successfully.
var HappyTerminalPhases = []string{"", "happy", "done"}

// HappyTerminalPhase reports whether the given phase name represents
// a happy terminal phase.
func HappyTerminalPhase(name string) bool {
	for _, s := range HappyTerminalPhases {
		if s == name {
			return true
		}
	}
	return false
}

// Errors collects errors from the main test as well as from final
// phase executions.
type Errors struct {
	InitErr     error
	Err         error
	FinalErrors map[string]error
}

// NewErrors does what you expect.
func NewErrors() *Errors {
	return &Errors{
		FinalErrors: make(map[string]error),
	}
}

// IsFine reports whether all errors are actually nil.
func (es *Errors) IsFine() bool {
	if es == nil {
		return true
	}

	if es.InitErr != nil || es.Err != nil {
		return false
	}

	for _, err := range es.FinalErrors {
		if err != nil {
			return false
		}
	}

	return true
}

// IsBroken is a possibly useless method that reports the first Broken
// error (if any).
//
// Also see the function IsBroken.
func (es *Errors) IsBroken() (*Broken, bool) {
	b := &Broken{
		Err: es,
	}

	if _, is := IsBroken(es.InitErr); is {
		return b, true
	}

	if _, is := IsBroken(es.Err); is {
		return b, true
	}

	for _, err := range es.FinalErrors {
		if _, is := IsBroken(err); is {
			return b, true
		}
	}

	return nil, false
}

// Errors instances are errors.
func (es *Errors) Error() string {
	var acc string
	if es.InitErr != nil {
		acc = "InitErr: " + es.Err.Error()
	}

	if es.Err != nil {
		if 0 < len(acc) {
			acc += "; "
		}
		acc = "Err: " + es.Err.Error()
	}

	for phase, err := range es.FinalErrors {
		if err == nil {
			continue
		}
		if 0 < len(acc) {
			acc += "; "
		}
		acc += "final " + phase + ": " + err.Error()
	}

	return acc
}

// Run initializes the Mother channel, runs the test, and runs final
// phases (if any).
func (t *Test) Run(ctx *Ctx) *Errors {

	errs := NewErrors()

	if err := t.InitChans(ctx); err != nil {
		errs.InitErr = err
		return errs
	}

	// Run the main sequence.

	from := t.Spec.InitialPhase
	if from == "" {
		from = DefaultInitialPhase
	}

	errs.Err = t.RunFrom(ctx, from)

	// Run the final phases.

	for _, phase := range t.Spec.FinalPhases {
		if e := t.RunFrom(ctx, phase); e != nil {
			errs.FinalErrors[phase] = e
		}
	}

	if !errs.IsFine() {
		return errs
	}

	return nil
}

// bindingRedactions adds redaction patterns for values of binding
// variables that start with X_.
func (t *Test) bindingRedactions(ctx *Ctx) error {
	return ctx.BindingsRedactions(t.Bindings)
}

// RunFrom begins test execution starting at the given phase.
func (t *Test) RunFrom(ctx *Ctx, from string) error {
	stepsTaken := 0
	for {
		if err := t.bindingRedactions(ctx); err != nil {
			return err
		}
		p, have := t.Spec.Phases[from]
		if !have {
			return fmt.Errorf("No phase '%s'", from)
		}
		ctx.Indf("Phase %s", from)

		next, err := p.Exec(ctx, t)
		if err != nil {
			_, broke := IsBroken(err)
			err := fmt.Errorf("phase %s: %w", from, err)
			if broke {
				return NewBroken(err)
			} else {
				return err
			}
		}

		stepsTaken++
		if 0 < t.MaxSteps && t.MaxSteps <= stepsTaken {
			return fmt.Errorf("MaxSteps (%d) reached", t.MaxSteps)
		}

		if HappyTerminalPhase(next) {
			return nil
		}

		from = next
	}
}

func (t *Test) makeChan(ctx *Ctx, kind ChanKind, opts interface{}) (Chan, error) {
	if t.Registry == nil {
		t.Registry = TheChanRegistry
	}

	maker, have := t.Registry[kind]
	if !have {
		return nil, fmt.Errorf("unknown Chan kind: '%s'", kind)
	}

	var x interface{}
	if err := t.Bindings.SubX(ctx, opts, &x); err != nil {
		return nil, err
	}

	return maker(ctx, x)
}

// Validate executes a few sanity checks.
//
// Should probably be called after Init().
func (t *Test) Validate(ctx *Ctx) []error {
	errs := make([]error, 0, 8)

	// Check that each step has exactly one operation.
	for name, p := range t.Spec.Phases {
		for i, s := range p.Steps {
			ops := 0
			if s.Pub != nil {
				ops++
			}
			if s.Sub != nil {
				ops++
			}
			if s.Recv != nil {
				ops++
			}
			if s.Goto != "" {
				ops++
			}
			if s.Ingest != nil {
				ops++
			}
			if s.Kill != nil {
				ops++
			}
			if s.Run != "" {
				ops++
			}
			if s.Reconnect != nil {
				ops++
			}
			if s.Wait != "" {
				ops++
			}
			if s.Branch != "" {
				ops++
			}
			if s.Doc != "" {
				ops++
			}
			if ops != 1 {
				errs = append(errs,
					fmt.Errorf("Step %d of phase %s does not have exactly one ops (%d)",
						i, name, ops))
			}
		}
	}

	// Check that any Goto Step is the last step in a Phase.
	//
	// ToDo: Maybe require all Phases to have Goto.
	for name, p := range t.Spec.Phases {
		for i := 0; i < len(p.Steps)-1; i++ {
			s := p.Steps[i]
			if s.Goto != "" {
				errs = append(errs,
					fmt.Errorf("Goto step %d in phase '%s' is not the last step",
						i, name))
			}
		}
	}

	// Check that each Goto Step has a defined Phase.
	for phaseName, p := range t.Spec.Phases {
		for i, s := range p.Steps {
			if s.Goto == "" {
				continue
			}
			if HappyTerminalPhase(s.Goto) {
				continue
			}
			if _, have := t.Spec.Phases[s.Goto]; !have {
				errs = append(errs,
					fmt.Errorf("No phase '%s', which is targeted by step %d in phase '%s'",
						s.Goto, i, phaseName))
			}
		}
	}
	if len(errs) == 0 {
		return nil
	}

	return errs
}

func (t *Test) Init(ctx *Ctx) error {
	// Previously we parsed Wait strings here, but that approach
	// was wrong because is wouldn't work for late bindings
	// subsitution.  So we delay parsing until Wait execution
	// time.

	return nil
}

func (t *Test) InitChans(ctx *Ctx) error {
	ctx.Indf("InitChans")

	m, err := NewMother(ctx, nil)
	if err != nil {
		return err
	}
	m.t = t
	if t.Chans == nil {
		t.Chans = make(map[string]Chan)
	}
	t.Chans["mother"] = m

	return nil
}

func (t *Test) ensureChan(ctx *Ctx, name string, dst *Chan) error {

	if name == "" {

		switch len(t.Chans) {
		case 0:
			// Internal error: Should always have mother.
			return fmt.Errorf("channel name is empty and can't find a default")
		case 1:
			for s, _ := range t.Chans {
				name = s
				break
			}
		case 2:
			for s, _ := range t.Chans {
				if s != "mother" {
					name = s
					break
				}
			}
			if name == "" {
				return fmt.Errorf("channel name is empty and can't find a default")
			}
		default:
			return fmt.Errorf("channel name is empty and can't choose a default")
		}
		ctx.Indf("    chan: %s", name)

	}

	c, have := t.Chans[name]
	if !have {
		return fmt.Errorf("no channel named '%s'", name)
	}

	if c == nil {
		// Internal error?
		return fmt.Errorf("channel named '%s' is nil", name)
	}

	*dst = c

	return nil
}

func (t *Test) Close(ctx *Ctx) error {
	for _, c := range t.Chans {
		if err := c.Close(ctx); err != nil {
			return err
		}
	}
	return nil
}

func TestIdFromPathname(s string) string {
	for _, suffix := range []string{"yaml", "json"} {
		if strings.HasSuffix(s, "."+suffix) {
			i := len(s) - len(suffix) - 1
			return s[0:i]
		}
	}
	return s
}

func (t *Test) getLibraries(ctx *Ctx) (string, error) {
	var src string
	for _, filename := range t.Libraries {
		filename = t.Dir + "/" + filename
		js, err := ioutil.ReadFile(filename)
		if err != nil {
			return "", fmt.Errorf("error reading library '%s': %w", filename, err)
		}
		src += fmt.Sprintf("// library: %s\n\n", filename) + string(js) + "\n"
	}
	return src, nil
}

func (t *Test) prepareSource(ctx *Ctx, code string) (string, error) {
	libs, err := t.getLibraries(ctx)
	if err != nil {
		return "", err
	}
	src := libs + "\n" + fmt.Sprintf("(function()\n{\n%s\n})()", code)
	return src, nil
}

// Bind replaces all bindings in the given (structured) thing.
func (t *Test) Bind(ctx *Ctx, x interface{}) (interface{}, error) {
	return t.Bindings.Bind(ctx, x)
}

// Retries represents a specification for how to retry a failed test.
type Retries struct {
	// N is the maximum number of retries.
	N int

	// Delay is the initial delay before the first retry.
	Delay time.Duration

	// DelayFactor is multiplied by the last delay to return the
	// next delay.
	DelayFactor float64
}

// NewRetries returns the default Retries specification.  N is 0.
func NewRetries() *Retries {
	return &Retries{
		N:           0,           // By default, no retries.
		Delay:       time.Second, // I guess.
		DelayFactor: 2,           // 1, 2, 4, 8, 16, 32, ...
	}
}

// NextDelay multiplies the given delay by r.DelayFactor.
func (r *Retries) NextDelay(d time.Duration) time.Duration {
	return time.Duration(float64(d) * r.DelayFactor)
}
