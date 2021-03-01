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

// TestSuiteRef reference
type TestSuiteRef struct {
	name  string
	tests []string
}

// getTaskFuncs for the TestList
func (ts *TestSuiteRef) getTaskFunc(ctx *plaxDsl.Ctx, tr TestRun) (*async.TaskFunc, error) {
	bs, err := (&tr.trps.Bindings).Copy()
	if err != nil {
		return nil, fmt.Errorf("failed to copy bindings for test suite %s: %w", ts.name, err)
	}

	name := fmt.Sprintf("%s-%s", tr.Name, tr.Version)
	tdr := TestDefRef{
		TestConstraints: TestConstraints{
			Labels: tr.trps.Labels,
		},
		Name:  ts.name,
		tests: ts.tests,
	}

	tf, err := tdr.getTaskFunc(ctx, tr, name, bs)
	if err != nil {
		return nil, err
	}

	return tf, nil
}
