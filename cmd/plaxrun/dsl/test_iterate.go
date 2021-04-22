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
	"encoding/json"
	"fmt"

	plaxDsl "github.com/Comcast/plax/dsl"
)

// TestIterate is used to iterate over a collection
// and invoke the test suite or test file multiple times
type TestIterate struct {
	DependsOn TestParamDependencyList `yaml:"dependsOn"`
	Param     *string                 `yaml:"param"`
	Params    string                  `yaml:"params"`
	Guard     *TestGuard              `yaml:"guard,omitempty"`
}

// TestIterateBindings is the bindings for an iteration
type TestIterateBindings struct {
	name string
	bs   *plaxDsl.Bindings
}

// TestIterateBindingsList is a list of test iteration bindings
type TestIterateBindingsList []TestIterateBindings

func trimQuotes(s string) string {
	if len(s) >= 2 {
		if s[0] == '"' && s[len(s)-1] == '"' {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func (ti TestIterate) getBindings(ctx *plaxDsl.Ctx, tr TestRun, bs *plaxDsl.Bindings) (TestIterateBindingsList, error) {
	err := ti.DependsOn.process(ctx, tr.Params, bs)
	if err != nil {
		return nil, fmt.Errorf("failed to process dependencies: %w", err)
	}

	ss, err := bs.StringSub(ctx, ti.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to substitute paramList: %w", err)
	}

	ss = trimQuotes(ss)

	var params []interface{}
	err = json.Unmarshal([]byte(ss), &params)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameter list %s: %v", ss, err)
	}

	var tibl TestIterateBindingsList

	for i, param := range params {
		fbs, err := bs.Copy()
		if err != nil {
			return nil, fmt.Errorf("failed to copy bindings: %w", err)
		}

		name := fmt.Sprintf("iteration-%d", i)

		if pm, ok := param.(map[string]interface{}); ok && ti.Param == nil {
			for k, v := range pm {
				fbs.SetKeyValue(k, v)
			}
		} else if ps, ok := param.(string); ok && ti.Param != nil {
			name = fmt.Sprintf("iteration-%s", ps)

			fbs.SetKeyValue(*ti.Param, param.(string))
		} else {
			return nil, fmt.Errorf("invalid parameter value")
		}

		run, err := ti.Guard.Satisfied(ctx, tr, fbs)
		if err != nil {
			return nil, fmt.Errorf("failed to guard %s test: %w", name, err)
		}

		if !run {
			ctx.Logdf("guard stopped %s test iteration from running", name)
			continue
		}

		tibl = append(tibl, TestIterateBindings{
			name: name,
			bs:   fbs,
		})
	}

	return tibl, nil
}
