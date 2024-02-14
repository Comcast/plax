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

// Package main is the 'plaxrun' command-line program for running many
// tests with various configurations.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	plaxDsl "github.com/Comcast/plax/dsl"

	_ "github.com/Comcast/plax/chans/std"

	"github.com/Comcast/plax/cmd/plaxrun/dsl"
	_ "github.com/Comcast/plax/cmd/plaxrun/plugins"
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
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get current working directory: %v", err)
	}

	var (
		trps = &dsl.TestRunParams{
			Bindings:        make(plaxDsl.Bindings),
			IncludeDirs:     dsl.IncludeDirList{wd},
			Filename:        flag.String("run", "spec.yaml", "Filename for test run specification"),
			Dir:             flag.String("dir", ".", "Directory containing test files"),
			ReportPluginDir: flag.String("reportPluginDir", "plugins/report", "Directory containing the report plugins"),
			EmitJSON:        flag.Bool("json", false, "Emit JSON test output; instead of JUnit XML"),
			Groups:          dsl.TestGroupList{},
			Verbose:         flag.Bool("v", true, "Verbosity"),
			LogLevel:        flag.String("log", "info", "Log level (info, debug, none)"),
			Labels:          flag.String("labels", "", "Labels for tests to run"),
			SuiteName:       flag.String("s", "", "Suite name to execute; -t options represent the tests in the suite to execute"),
			Priority:        flag.Int("priority", -1, "Test priority"),
			Redact:          flag.Bool("redact", true, "enable redactions when -log debug"),
		}
		vers = flag.Bool("version", false, "Print version and then exit")
	)

	flag.Var(&trps.Bindings, "p", fmt.Sprintf("Parameter Bindings: %s", trps.Bindings.String()))
	flag.Var(&trps.IncludeDirs, "I", "YAML include directories")
	flag.Var(&trps.Groups, "g", fmt.Sprintf("Groups to execute: %s", trps.Groups.String()))
	flag.Var(&trps.Tests, "t", fmt.Sprintf("Tests to execute: %s", trps.Tests.String()))

	flag.Parse()

	if *vers {
		fmt.Printf("plaxrun %s %s %s\n", version, commit, date)
		return
	}

	switch *trps.LogLevel {
	case "debug", "DEBUG":
		log.Printf("plaxrun version %s %s %s\n", version, commit, date)
	}

	if len(trps.Groups) == 0 && len(trps.Tests) == 0 && trps.SuiteName == nil {
		log.Fatal(fmt.Errorf("at least 1 test or test group or test suite must be specified"))
	}

	ctx := dsl.NewCtx(context.Background())

	testRun, err := dsl.NewTestRun(ctx, trps)
	if err != nil {
		log.Fatal(err)
	}

	err = testRun.Exec(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
