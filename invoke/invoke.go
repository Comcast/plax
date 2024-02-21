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

// Package invoke provides gear supporting to facilitate running Plax
// tests in different environments (and configurations).
//
// Also see cmd/plaxrun.
package invoke

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Comcast/plax/dsl"
	"github.com/Comcast/plax/junit"

	"gopkg.in/yaml.v3"
)

// Invocation struct for execution of a suite of tests
type Invocation struct {
	Bindings  map[string]interface{}
	SuiteName string
	Filename  string
	// Dir will be added to ctx.IncludeDirs to resolve YAML (and
	// perhaps other) includes.
	Dir         string
	IncludeDirs []string
	Env         map[string]string
	Seed        int64
	Priority    int
	Labels      string
	Tests       []string
	LogLevel    string
	Verbose     bool
	List        bool

	// ComplainOnAnyError will cause Exec() to return an error if
	// any test case fails or is broken.
	//
	// The default (false) can make sense when the caller's
	// perspective is that test failures are actually normal.
	ComplainOnAnyError bool

	// Retry will override a test's retry policy (if any).
	Retry string

	// React will set dsl.Ctx.Redact to enable log redactions.
	Redact bool

	retries *dsl.Retries
}

const (
	negativeTestWarning = "negative test warning: %s"
)

// Exec executes the Invocation.
//
// When ComplainOnAnyError is true, then the last test problem (if
// any) is returned by this method.  Otherwise, only a non-test error
// (if any) is returned.
//
// This method calls Run(t) for each test t in the Invocation.
func (inv *Invocation) Exec(ctx context.Context) (*junit.TestSuite, error) {
	dslCtx := dsl.NewCtx(ctx)
	dslCtx.Redact = inv.Redact
	if inv.Redact {
		//Add redactions for the X_ variables
		dslCtx.Redactions.Add(".*{X_.*}=(.*)")
		dslCtx.Redactions.Add(".*\"?X_.*\":(.*)}]")
	}

	if len(inv.LogLevel) > 0 {
		if err := dslCtx.SetLogLevel(inv.LogLevel); err != nil {
			log.Fatal(err)
		}
	}

	inv.retries = dsl.NewRetries()

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// Add invocation includeDirs to the dslCtx
	if inv.IncludeDirs != nil {
		dslCtx.IncludeDirs = inv.IncludeDirs
	}

	// Add current working directory to includeDirs
	dslCtx.IncludeDirs = append(dslCtx.IncludeDirs, wd)

	if inv.Retry != "" {
		if n, err := strconv.Atoi(inv.Retry); err == nil {
			inv.retries.N = n
		} else {
			// JSON representation of an invoke.Retries?
			if err := json.Unmarshal([]byte(inv.Retry), &inv.retries); err != nil {
				log.Fatalf("error parsing retry: %s", err)
			}
		}
	}

	var (
		suiteName = strings.ReplaceAll(inv.SuiteName, "{TS}", time.Now().UTC().Format(time.RFC3339Nano))
		ts        = junit.NewTestSuite(suiteName)
		filenames = make([]string, 0, 8)
	)

	// Populate filenames.
	if inv.Dir != "" {
		dir, err := filepath.Abs(inv.Dir)
		if err != nil {
			log.Fatal(err)
		}
		inv.Dir = dir

		if suiteName == "" {
			ts.Name = inv.Dir
		}

		// Set the context directory
		dslCtx.Dir = dir

		// Add the test spec's directory to the end of the includeDirs.
		dslCtx.IncludeDirs = append(dslCtx.IncludeDirs, inv.Dir)

		fs, err := ioutil.ReadDir(inv.Dir)
		if err != nil {
			log.Fatal(err)
		}
		for _, f := range fs {
			if !strings.HasSuffix(f.Name(), ".yaml") {
				continue
			}
			pathname := inv.Dir + "/" + f.Name()
			filenames = append(filenames, pathname)
		}
	} else {
		dir, err := filepath.Abs(filepath.Dir(inv.Filename))
		if err != nil {
			log.Fatal(err)
		}
		inv.Dir = dir

		if suiteName == "" {
			ts.Name = inv.Dir
		}

		// Set the context directory
		dslCtx.Dir = dir

		// Add the test spec's directory to the end of the includeDirs.
		dslCtx.IncludeDirs = append(dslCtx.IncludeDirs, inv.Dir)

		filename, err := filepath.Abs(inv.Filename)
		if err != nil {
			log.Fatal(err)
		}
		filenames = append(filenames, filename)
	}

	var (
		// problem will remember the last test failure (if any).
		problem error = nil

		// problemFilename will be the filename of the
		// last test that failed (if any).
		problemFilename string
	)

	// Run tests.
	for _, filename := range filenames {
		t, err := inv.Load(dslCtx, filename)
		if err != nil {
			log.Fatalf("Invocation of %s broken: %s", filename, err)
		}

		if inv.List {
			fmt.Printf("%s,%d,%s\n", t.Id, t.Priority,
				strings.Join(t.Labels, ","))
			continue
		}

		tc := junit.NewTestCase(t.Name, filename)

		if !t.Wanted(dslCtx, inv.Priority, strings.Split(inv.Labels, ","), inv.Tests) {
			// marking this TestCase as "skipped".
			tc.Finish(junit.Skipped, fmt.Sprintf("priority: test=%d, invoke=%d, labels: test=%v, invoke=%v", t.Priority, inv.Priority, t.Labels, inv.Labels))
			ts.Add(*tc)
			continue
		}

		log.Printf("Running test %s", filename)

		if err := inv.Run(dslCtx, t); err != nil {
			if b, is := dsl.IsBroken(err); is {
				// Any broken test is a failure (even
				// for a 'negative' test).
				problem = err
				problemFilename = filename
				log.Printf("Test %s broken: %s", filename, b.Err)
				tc.Finish(junit.Error, b.Error())
			} else {
				if t.Negative {
					log.Printf("Test %s (negative) passed", filename)
					tc.Finish(junit.Passed, fmt.Sprintf(negativeTestWarning, err.Error()))
				} else {
					problem = err
					problemFilename = filename
					log.Printf("Test %s failed: %s", filename, err)
					tc.Finish(junit.Failed, err.Error())
				}
			}
		} else {
			if t.Negative {
				problem = fmt.Errorf("negative test failure")
				problemFilename = filename
				log.Printf("Test %s (negative) failed (no error)", filename)
				tc.Finish(junit.Failed, fmt.Sprintf(negativeTestWarning, "expected error due to negative test"))
			} else {
				log.Printf("Test %s passed", filename)
				tc.Finish(junit.Passed)
			}
		}

		ts.Add(*tc)
	}

	if inv.List {
		// We already listed the tests, so nothing left to do.
		return nil, nil
	}

	ts.Finish()

	if problem != nil && inv.ComplainOnAnyError {
		return ts, fmt.Errorf("at least one test failed (%s: %s)", problemFilename, problem)
	}

	// Unless we are asked to complain via inv.ComplainOnAnyError,
	// we report that we are happy.
	//
	// We have some log.Fatal(err) calls above that could probably
	// be replaced by 'return err', which should correctly report
	// an error that is not a test error.
	return ts, nil
}

// Load a test
func (inv *Invocation) Load(ctx *dsl.Ctx, filename string) (*dsl.Test, error) {
	bs, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	if ctx.IncludeDirs == nil {
		ctx.IncludeDirs = make([]string, 0, 4)
	}
	if inv.Dir == "" {
		ctx.IncludeDirs = append(ctx.IncludeDirs, ".")
	} else {
		ctx.IncludeDirs = append(ctx.IncludeDirs, inv.Dir)
	}

	t := dsl.NewTest(ctx, filename, nil)
	t.Dir = inv.Dir

	if bs, err = dsl.IncludeYAML(ctx, bs); err != nil {
		return nil, dsl.NewBroken(fmt.Errorf("spec parse: %w", err))
	}

	if err := yaml.Unmarshal(bs, &t); err != nil {
		return nil, dsl.NewBroken(fmt.Errorf("spec parse: %w", err))
	}

	// We are seeing reports of t.Name (below) causing a panic due
	// to an invalid memory address or nil pointer deference.  The
	// following check attempts to gather more information when
	// that situation occurs.
	//
	// Warning: The logging output will include data that could be
	// sensitive.
	//
	// This check is executed only if the environment variable
	// PLAX_DEBUG is not empty.
	if 0 < len(os.Getenv("PLAX_DEBUG")) {
		if t == nil {
			log.Printf("internal error: nil test in Invoke.Load\n%s\n", bs)
			log.Printf("internal error test: %#v", t)
			return nil, dsl.NewBroken(fmt.Errorf("internal error: nil test in Invoke.Load"))
		}
	}

	if t.Name == "" {
		basename := filepath.Base(filename)
		t.Name = strings.TrimSuffix(basename, filepath.Ext(basename))
	}

	return t, nil
}

// Run executes the test with possible retries.
func (inv *Invocation) Run(ctx *dsl.Ctx, t *dsl.Test) error {
	if t == nil {
		return dsl.NewBroken(fmt.Errorf("given test is nil"))
	}

	retries := t.Retries
	if inv.retries != nil {
		retries = inv.retries
	}
	if retries == nil {
		retries = dsl.NewRetries()
	}

	var err error
	delay := retries.Delay

	for j := 0; j <= retries.N; j++ {
		if 0 < j {
			delay = retries.NextDelay(delay)
			log.Printf("Retry %d (%v delay) on error '%v'\n", j, delay, err)
			time.Sleep(delay)
		}
		err = inv.RunOnce(ctx, t)
		if err == nil {
			return nil
		}
		if _, is := dsl.IsBroken(err); is {
			return err
		}
		// Non-broken error
	}

	return err
}

// RunOnce executes the test at most one time.
func (inv *Invocation) RunOnce(ctx *dsl.Ctx, t *dsl.Test) error {

	ctx, cancel := ctx.WithCancel()
	defer cancel()

	if t == nil {
		return dsl.Brokenf("test is nil")
	}

	if t.Seed != 0 {
		log.Printf("Setting pseudo-random number generator seed: %v", t.Seed)
		rand.Seed(t.Seed)
	}

	for p, v := range inv.Bindings {
		if _, have := inv.Bindings[p]; have {
			log.Printf("Updating initial binding of '%s'", p)
		}
		t.Bindings[p] = v
	}

	if err := t.Init(ctx); err != nil {
		return err
	}

	if errs := t.Validate(ctx); errs != nil {
		var acc string
		for i, err := range errs {
			acc += fmt.Sprintf("  %02d. %s\n", i, err)
		}
		return dsl.Brokenf("Validation failed:\n\n%s\n", acc)
	}
	if err := t.Run(ctx); err != nil {
		return err
	}

	if err := t.Close(ctx); err != nil {
		return err
	}

	return nil
}
