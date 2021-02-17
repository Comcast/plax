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

// Package main is the standard Plax command-line program.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	_ "github.com/Comcast/plax/chans/std"
	"github.com/Comcast/plax/dsl"
	"github.com/Comcast/plax/invoke"
)

// Version of plax
const Version = "1.0.3"

func main() {

	// This function executes either a single test or a tests in a
	// directory.

	rand.Seed(time.Now().UnixNano())

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	var (
		// params are command-line provide test parameters.
		//
		// We make then their own type to enable flag.Var to parse multiple values.
		bindings          = make(dsl.Bindings)
		includeDirs       = IncludeDirs{"."}
		specFilename      = flag.String("test", "test.yaml", "Filename for test specification")
		dir               = flag.String("dir", "", "Directory containing test specs")
		list              = flag.Bool("list", false, "Show report of known tests; don't run anything.  Assumes -dir.")
		labels            = flag.String("labels", "", "Optional list of required test labels")
		priority          = flag.Int("priority", -1, "Optional lowest priority (where larger numbers mean lower priority!); negative means all")
		verbose           = flag.Bool("v", true, "Verbosity")
		version           = flag.Bool("version", false, "Print version and then exit")
		listChanTypes     = flag.Bool("channel-types", false, "List known channel types and then exit")
		seed              = flag.Int64("seed", 0, "Seed for random number generator")
		nonzeroOnAnyError = flag.Bool("error-exit-code", false, "Return non-zero on any test failure")
		emitJSON          = flag.Bool("json", false, "Emit docs suitable for indexing")
		testSuiteName     = flag.String("test-suite", "NA", "Name for JUnit test suite")
		logLevel          = flag.String("log", "info", "log level (info, debug, none)")
		retry             = flag.String("retry", "", `Specify retries: number or {"N":N,"Delay":"1s","DelayFactor":1.5}`)
	)

	flag.Var(&bindings, "p", "Parameter values: PARAM=VALUE")
	flag.Var(&includeDirs, "I", "YAML include directories")

	flag.Parse()

	log.Printf("plax version %s", Version)

	if *version {
		return
	}

	if *listChanTypes {
		for name, _ := range dsl.TheChanRegistry {
			fmt.Printf("%s\n", name)
		}
		return
	}

	iv := invoke.Invocation{
		SuiteName:         *testSuiteName,
		Bindings:          bindings,
		Seed:              *seed,
		Verbose:           *verbose,
		Filename:          *specFilename,
		Dir:               *dir,
		IncludeDirs:       includeDirs,
		Priority:          *priority,
		Labels:            *labels,
		LogLevel:          *logLevel,
		List:              *list,
		EmitJSON:          *emitJSON,
		NonzeroOnAnyError: *nonzeroOnAnyError,
		Retry:             *retry,
	}

	err := iv.Exec(context.Background())
	if err != nil {
		log.Fatalf("Invocation broken: %s", err)
	}
}

// IncludeDir are directories to search when YAML-including.
//
// We make an explicit type to enable flag.Var to parse multiple
// parameters.
type IncludeDirs []string

func (ps *IncludeDirs) String() string {
	return "DIR"
}

func (ps *IncludeDirs) Set(value string) error {
	*ps = append(*ps, value)
	return nil
}

type JSONTestSuite struct {
	Type   string
	Time   time.Time
	Tests  int
	Passed int
	Failed int
	Errors int
}
