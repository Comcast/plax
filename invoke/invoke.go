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
	"encoding/xml"
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
	Dir               string
	IncludeDirs       []string
	Env               map[string]string
	Seed              int64
	Priority          int
	Labels            string
	Tests             []string
	LogLevel          string
	Verbose           bool
	List              bool
	EmitJSON          bool
	NonzeroOnAnyError bool
	// Retry will override a test's retry policy (if any).
	Retry   string
	retries *dsl.Retries
}

// Exec the tests
func (inv *Invocation) Exec(ctx context.Context) error {
	dslCtx := dsl.NewCtx(ctx)

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
		ts        = junit.NewTestSuite()
		filenames = make([]string, 0, 8)
		problem   bool
	)

	ts.Name = strings.ReplaceAll(inv.SuiteName,
		"{TS}",
		time.Now().UTC().Format(time.RFC3339Nano))

	// Populate filenames.
	if inv.Dir != "" {
		dir, err := filepath.Abs(inv.Dir)
		if err != nil {
			log.Fatal(err)
		}
		inv.Dir = dir

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

	var res error = nil // final result of test execution success/failure

	// Run tests.
	i := 0
	for _, filename := range filenames {
		t, err := inv.Load(dslCtx, filename)
		if err != nil {
			log.Fatalf("Invocation of %s broken: %s", filename, err)
		}

		if !t.Wanted(dslCtx, inv.Priority, strings.Split(inv.Labels, ","), inv.Tests) {
			// Not marking this TestCase as "skipped".
			continue
		}

		if inv.List {
			fmt.Printf("%s,%d,%s\n", t.Id, t.Priority,
				strings.Join(t.Labels, ","))
			continue
		}

		tc := junit.NewTestCase(filename)
		tc.N = i
		i++
		tc.Suite = ts.Name
		tc.Type = "case"

		log.Printf("Running test %s", filename)

		if err := inv.Run(dslCtx, t); err != nil {
			res = err
			if b, is := dsl.IsBroken(err); is {
				problem = true
				tc.Error = &junit.Error{
					Message: b.Err.Error(),
				}
			} else {
				if !t.Negative {
					problem = true
					log.Printf("Test %s failed: %s", filename, err)
					tc.Failure = &junit.Failure{
						Message: err.Error(),
					}
				} else {
					// clear final result because this is not a problem due to negative :-(
					res = nil
				}
			}
		} else { // err nil
			if t.Negative {
				problem = true
				log.Printf("Test %s (negative) failed (no error)", filename)
				tc.Failure = &junit.Failure{
					Message: "expected error for Negative test",
				}
				res = fmt.Errorf("negative test failure")
			} else {
				log.Printf("Test %s passed", filename)
			}
		}

		if t != nil {
			tc.State = t.State
		}

		tc.Finish("executed")
		ts.Add(*tc)
	}

	if inv.List {
		// We already listed the tests, so nothing left to do.
		return nil
	}

	if inv.EmitJSON {
		// We'll emit some JSON that represents an array of
		// objects suitable of indexing
		acc := make([]interface{}, 0, len(ts.TestCases)+1)

		// Our first "doc" represents the suite of tests we
		// just range.
		jts := JSONTestSuite{
			Time:   ts.Time,
			Tests:  len(ts.TestCases),
			Errors: ts.Errors,
			Failed: ts.Failures,
			Type:   "suite",
		}
		jts.Passed = jts.Tests - jts.Errors - jts.Failed

		acc = append(acc, jts)

		// The remaining "docs" are the test cases themselves.
		for i, tc := range ts.TestCases {
			tc.N = i
			tc.Suite = ts.Name
			tc.Type = "case"
			acc = append(acc, tc)
		}

		// Write the JSON.
		js, err := json.Marshal(&acc)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%s\n", js)
		return res
	}

	// Wire the XML representation of the JUnit test suite.
	bs, err := xml.MarshalIndent(ts, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", bs)

	if inv.NonzeroOnAnyError && problem {
		return fmt.Errorf("Prolem")
	}

	return res
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

// JSONTestSuite test results
type JSONTestSuite struct {
	Type   string
	Time   time.Time
	Tests  int
	Passed int
	Failed int
	Errors int
}
