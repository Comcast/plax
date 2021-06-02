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
	"os"
	"strconv"
	"strings"

	"github.com/Comcast/plax/cmd/plaxrun/async"
	plaxDsl "github.com/Comcast/plax/dsl"
)

// TestDef is a test file or suite (directory)
type TestDef struct {
	Path   string                  `yaml:"path"`
	Module PluginModule            `yaml:"version"`
	Params TestParamDependencyList `yaml:"params"`
}

// TestDefMap is a map of TestDefs
type TestDefMap map[string]TestDef

// TestDefRef references a TestDef with its constraints
type TestDefRef struct {
	TestConstraints `yaml:",inline"`
	Name            string       `yaml:"name"`
	Params          TestParamMap `yaml:"params"`
	Guard           *TestGuard   `yaml:"guard,omitempty"`
	tests           []string     `yaml:"-"`
}

// TestDefRefList is a list of TestDefRefs
type TestDefRefList []TestDefRef

func (tdrl TestDefRefList) getTaskFuncs(ctx *plaxDsl.Ctx, tr TestRun, name string, bs *plaxDsl.Bindings) ([]*async.TaskFunc, error) {
	tl := make([]*async.TaskFunc, 0)

	for _, tdr := range tdrl {

		n := fmt.Sprintf("%s:%s", name, tdr.Name)

		// Each test needs to have its own copy of the bindings
		cbs, err := bs.Copy()
		if err != nil {
			return nil, err
		}

		err = tdr.Params.bind(ctx, cbs)
		if err != nil {
			return nil, fmt.Errorf("failed to substitute test ref parameters: %w", err)
		}

		run, err := tdr.Guard.Satisfied(ctx, tr, cbs)
		if err != nil {
			return nil, fmt.Errorf("failed to guard %s test: %w", name, err)
		}

		if !run {
			ctx.Logdf("guard stopped %s test from running", name)
			return tl, nil
		}

		tf, err := tdr.getTaskFunc(ctx, tr, n, cbs)
		if err != nil {
			return nil, err
		}

		tl = append(tl, tf)
	}

	return tl, nil
}

func (tdr TestDefRef) getTaskFunc(ctx *plaxDsl.Ctx, tr TestRun, name string, bs *plaxDsl.Bindings) (*async.TaskFunc, error) {
	td, ok := tr.Tests[tdr.Name]
	if !ok {
		return nil, fmt.Errorf("failed to find test def %s", tdr.Name)
	}

	fmt.Fprintf(os.Stderr, "\nProcessing parameters for %s\n\n", name)

	for _, tpd := range td.Params {
		err := tpd.process(ctx, tr.Params, bs)
		if err != nil {
			return nil, fmt.Errorf("failed to process test params: %w", err)
		}
	}

	priority := -1
	if tdr.Priority != nil {
		priority = *tdr.Priority
	}

	// Empty labels
	labelArr := make([]string, 0)

	if tr.trps.Labels != nil {
		// Labels from command line
		labelArr = strings.Split(*tr.trps.Labels, ",")
	}

	if tdr.Labels != nil {
		// Labels from test definition
		arr := strings.Split(*tdr.Labels, ",")
		labelArr = append(labelArr, arr...)
	}

	// All labels from command line and test definition
	labels := strings.Join(labelArr, ",")

	def := PluginDef{
		PluginDefNameKey:        name,
		PluginDefParamsKey:      bs,
		PluginDefSeedKey:        tdr.Seed,
		PluginDefPriorityKey:    priority,
		PluginDefLabelsKey:      labels,
		PluginDefTestsKey:       tdr.tests,
		PluginDefRetryKey:       strconv.Itoa(tdr.Retry),
		PluginDefVerboseKey:     tr.trps.Verbose,
		PluginDefLogLevelKey:    tr.trps.LogLevel,
		PluginDefEmitJSONKey:    tr.trps.EmitJSON,
		PluginDefIncludeDirsKey: tr.trps.IncludeDirs,
	}

	path := td.Path
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():
		def[PluginDefDirKey] = path
	case mode.IsRegular():
		def[PluginDefFilenameKey] = path
	}

	module := td.Module
	if len(module) == 0 {
		module = DefaultPluginModule
	}

	plugin, err := MakePlugin(module, def)
	if err != nil {
		return nil, err
	}

	return &async.TaskFunc{
		Name: name,
		Func: func() error {
			return plugin.Invoke(ctx)
		},
	}, nil
}

// TestList are the individual tests to execute
//
// We make an explicit type to enable flag.Var to parse multiple
// parameters.
type TestList []string

// String representation
func (tl *TestList) String() string {
	return "Test Name"
}

// Set the test in the testList
func (tl *TestList) Set(value string) error {
	*tl = append(*tl, value)
	return nil
}

// getTaskFuncs for the TestList
func (tl *TestList) getTaskFuncs(ctx *plaxDsl.Ctx, tr TestRun) ([]*async.TaskFunc, error) {
	tfs := make([]*async.TaskFunc, 0)

	for _, n := range *tl {
		bs, err := (&tr.trps.Bindings).Copy()
		if err != nil {
			return nil, fmt.Errorf("failed to copy bindings for test %s: %w", n, err)
		}

		name := fmt.Sprintf("%s-%s", tr.Name, tr.Version)

		tdr := TestDefRef{
			TestConstraints: TestConstraints{
				Labels:   tr.trps.Labels,
				Priority: tr.trps.Priority,
			},
			Name: n,
		}

		tf, err := tdr.getTaskFunc(ctx, tr, name, bs)
		if err != nil {
			return nil, fmt.Errorf("failed to get task for test %s: %w", n, err)
		}

		tfs = append(tfs, tf)
	}

	return tfs, nil
}
