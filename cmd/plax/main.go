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
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"regexp"
	"time"

	_ "github.com/Comcast/plax/chans/std"
	"github.com/Comcast/plax/dsl"
	"github.com/Comcast/plax/invoke"
)

var (
	// Default goreleaser ldflags:
	//
	// -X main.version={{.Version}}
	// -X main.commit={{.Commit}}
	// -X main.date={{.Date}}
	// -X main.builtBy=goreleaser`
	//
	// See https://goreleaser.com/customization/build.

	version = "NA"
	commit  = "NA"
	date    = "NA"
)

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
		specFilename      = flag.String("test", "", "Filename for test specification")
		dir               = flag.String("dir", "", "Directory containing test specs")
		list              = flag.Bool("list", false, "Show report of known tests; don't run anything.  Assumes -dir.")
		labels            = flag.String("labels", "", "Optional list of required test labels")
		priority          = flag.Int("priority", -1, "Optional lowest priority (where larger numbers mean lower priority!); negative means all")
		verbose           = flag.Bool("v", true, "Verbosity")
		vers              = flag.Bool("version", false, "Print version and then exit")
		listChanTypes     = flag.Bool("channel-types", false, "List known channel types and then exit")
		seed              = flag.Int64("seed", 0, "Seed for random number generator")
		nonzeroOnAnyError = flag.Bool("error-exit-code", false, "Return non-zero on any test failure")
		emitJSON          = flag.Bool("json", false, "Emit docs suitable for indexing")
		testSuiteName     = flag.String("test-suite", "", "Name for JUnit test suite")
		logLevel          = flag.String("log", "info", "log level (info, debug, none)")
		retry             = flag.String("retry", "", `Specify retries: number or {"N":N,"Delay":"1s","DelayFactor":1.5}`)
		redact            = flag.Bool("redact", true, "Use redaction gear")

		testRedactPattern = flag.String("check-redact-regexp", "", "regular expression to use for checking redactions (with no test executed)")
		testRedactString  = flag.String("check-redact", "", "input string to use for -check-redact-regexp")
	)

	flag.Var(&bindings, "p", "Parameter values: PARAM=VALUE")
	flag.Var(&includeDirs, "I", "YAML include directories")

	flag.Parse()

	if *vers {
		fmt.Printf("plax %s %s %s\n", version, commit, date)
		return
	}

	if *testRedactPattern != "" {
		fmt.Printf("%s\n", dsl.Redact(regexp.MustCompile(*testRedactPattern), *testRedactString))
		return
	}

	switch *logLevel {
	case "debug", "DEBUG":
		log.Printf("plax version %s %s %s\n", version, commit, date)
	}

	if *listChanTypes {
		for name, _ := range dsl.TheChanRegistry {
			fmt.Printf("%s\n", name)
		}
		return
	}

	if *specFilename == "" && *dir == "" {
		fmt.Fprintf(os.Stderr, "To run a test, use \"-test FILENAME\" or \"-dir DIR\"\n\n")
		flag.Usage()
		os.Exit(1)
	}

	iv := invoke.Invocation{
		SuiteName:          *testSuiteName,
		Bindings:           bindings,
		Seed:               *seed,
		Verbose:            *verbose,
		Filename:           *specFilename,
		Dir:                *dir,
		IncludeDirs:        includeDirs,
		Priority:           *priority,
		Labels:             *labels,
		LogLevel:           *logLevel,
		List:               *list,
		ComplainOnAnyError: *nonzeroOnAnyError,
		Retry:              *retry,
		Redact:             *redact,
	}

	ts, err := iv.Exec(context.Background())
	if err != nil {
		log.Fatalf("Invocation broken: %s", err)
	}

	if *emitJSON {
		// Write the JSON.
		js, err := json.MarshalIndent(ts, "", "  ")
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%s\n", js)
	} else {
		bs, err := xml.MarshalIndent(ts, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", bs)
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
