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

	"github.com/Comcast/plax/cmd/plaxrun/async"
	plaxDsl "github.com/Comcast/plax/dsl"
)

// TestGroupRef references a group by name
type TestGroupRef struct {
	Name   string       `yaml:"name"`
	Params TestParamMap `yaml:"params"`
	Guard  *TestGuard   `yaml:"guard,omitempty"`
}

// getTaskFuncs for the TestGroupRef
func (tgr TestGroupRef) getTaskFuncs(ctx *plaxDsl.Ctx, tr TestRun, name string, bs *plaxDsl.Bindings) ([]*async.TaskFunc, error) {
	tl := make([]*async.TaskFunc, 0)

	tg, ok := tr.Groups[tgr.Name]
	if !ok {
		return nil, fmt.Errorf("failed to get test group %s", tgr.Name)
	}

	name = fmt.Sprintf("%s:%s", name, tgr.Name)

	err := tgr.Params.bind(ctx, bs)
	if err != nil {
		return nil, fmt.Errorf("failed to substitute %s group ref parameters: %w", name, err)
	}

	run, err := tgr.Guard.Satisfied(ctx, tr, bs)
	if err != nil {
		return nil, fmt.Errorf("failed to guard %s group: %w", name, err)
	}

	if !run {
		ctx.Logdf("guard stopped %s test group from running", name)
		return tl, nil
	}

	tl, err = tg.getTaskFuncs(ctx, tr, name, bs)
	if err != nil {
		return nil, fmt.Errorf("failed to get task funcs for test group %s: %w", tgr.Name, err)
	}

	return tl, nil
}

// TestGroupRefList is a list of TestGroupRefs
type TestGroupRefList []TestGroupRef

func (tgrl TestGroupRefList) getTaskFuncs(ctx *plaxDsl.Ctx, tr TestRun, name string, bs *plaxDsl.Bindings) ([]*async.TaskFunc, error) {
	tl := make([]*async.TaskFunc, 0)

	for _, tgr := range tgrl {
		bs, err := bs.Copy()
		if err != nil {
			return nil, fmt.Errorf("failed to copy bindings for test group %s", tgr.Name)
		}

		tfs, err := tgr.getTaskFuncs(ctx, tr, name, bs)
		if err != nil {
			return nil, err
		}

		tl = append(tl, tfs...)
	}

	return tl, nil
}

// TestGroup is a set of grouped tests or nested groups
type TestGroup struct {
	Iterate *TestIterate     `yaml:"iterate,omitempty"`
	Params  TestParamMap     `yaml:"params"`
	Tests   TestDefRefList   `yaml:"tests"`
	Groups  TestGroupRefList `yaml:"groups"`
}

func (tg TestGroup) getTaskFuncs(ctx *plaxDsl.Ctx, tr TestRun, name string, bs *plaxDsl.Bindings) ([]*async.TaskFunc, error) {
	var (
		tibsl TestIterateBindingsList
		err   error
	)

	tg.Params.bind(ctx, bs)

	if tg.Iterate != nil {
		tibsl, err = tg.Iterate.getBindings(ctx, tr, bs)
		if err != nil {
			return nil, fmt.Errorf("failed to get test iteration bindings: %w", err)
		}
	} else {
		tibsl = append(tibsl, TestIterateBindings{
			bs: bs,
		})
	}

	tl := make([]*async.TaskFunc, 0)

	for _, tibs := range tibsl {
		n := name
		if len(tibs.name) > 0 {
			n = fmt.Sprintf("%s:%s", name, tibs.name)
		}

		tfs, err := tg.Tests.getTaskFuncs(ctx, tr, n, tibs.bs)
		if err != nil {
			return nil, fmt.Errorf("failed to get test def tasks: %w", err)
		}

		tl = append(tl, tfs...)

		tfs, err = tg.Groups.getTaskFuncs(ctx, tr, n, tibs.bs)
		if err != nil {
			return nil, fmt.Errorf("failed to get test group tasks: %w", err)
		}

		tl = append(tl, tfs...)
	}

	return tl, nil
}

// TestGroupMap is a map of TestGroups
type TestGroupMap map[string]TestGroup

// TestGroupList are the test groups to execute
//
// We make an explicit type to enable flag.Var to parse multiple
// parameters.
type TestGroupList []string

// String representation
func (tgl *TestGroupList) String() string {
	return "Test Group Name"
}

// Set the groups
func (tgl *TestGroupList) Set(value string) error {
	*tgl = append(*tgl, value)
	return nil
}

// hasGroup checks if group is the the list of groups
func (tgl *TestGroupList) hasGroup(name string) bool {
	for _, n := range *tgl {
		if n == name {
			return true
		}
	}

	return false
}

// getTaskFuncs for the TestGroupList
func (tgl *TestGroupList) getTaskFuncs(ctx *plaxDsl.Ctx, tr TestRun) ([]*async.TaskFunc, error) {
	tfs := make([]*async.TaskFunc, 0)

	for _, n := range *tgl {
		tg, ok := tr.Groups[n]
		if !ok {
			return nil, fmt.Errorf("failed to find test group %s", n)
		}

		bs, err := (&tr.trps.Bindings).Copy()
		if err != nil {
			return nil, fmt.Errorf("failed to copy bindings for test group %s: %w", n, err)
		}

		name := fmt.Sprintf("%s-%s", tr.Name, tr.Version)
		tgr := TestGroupRef{
			Name:   n,
			Params: tg.Params,
		}

		gtfs, err := tgr.getTaskFuncs(ctx, tr, name, bs)
		if err != nil {
			return nil, fmt.Errorf("failed to get tasks for test group %s: %w", n, err)
		}

		tfs = append(tfs, gtfs...)
	}

	return tfs, nil
}
