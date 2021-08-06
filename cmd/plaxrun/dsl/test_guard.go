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
	"path"

	plaxDsl "github.com/Comcast/plax/dsl"
)

// TestGuard for guarding tests, groups, or iterations
type TestGuard struct {
	// Libraries is a list of filenames that should contain
	// Javascript.  This source is loaded into each Javascript
	// environment.
	DependsOn TestParamDependencyList `yaml:"dependsOn"`
	Libraries []string                `yaml:"libraries"`
	Source    string                  `yaml:"src"`
}

func getLibrary(ctx *plaxDsl.Ctx, filename string) (string, error) {
	for _, dir := range ctx.IncludeDirs {
		fn := path.Join(dir, filename)
		js, err := ioutil.ReadFile(fn)
		if err != nil {
			ctx.Logdf("error reading library '%s': %w", fn, err)
			continue
		}
		return string(js), nil
	}

	return "", fmt.Errorf("failed to find library '%s'", filename)
}

func (tg *TestGuard) getLibraries(ctx *plaxDsl.Ctx) (string, error) {
	var src string
	for _, filename := range tg.Libraries {
		js, err := getLibrary(ctx, filename)
		if err != nil {
			return "", err
		}
		src += fmt.Sprintf("// library: %s\n\n", filename) + string(js) + "\n"
	}
	return src, nil
}

func (tg *TestGuard) prepareSource(ctx *plaxDsl.Ctx, bs *plaxDsl.Bindings) (string, error) {
	src, err := bs.StringSub(ctx, tg.Source)
	if err != nil {
		return "", err
	}

	libs, err := tg.getLibraries(ctx)
	if err != nil {
		return "", err
	}
	src = libs + "\n" + fmt.Sprintf("(function()\n{\n%s\n})()", src)

	return src, nil
}

// Satisfied checks if the guard allows the test to be executed
func (tg *TestGuard) Satisfied(ctx *plaxDsl.Ctx, tr TestRun, bs *plaxDsl.Bindings) (bool, error) {
	if tg == nil {
		return true, nil
	}

	if len(tg.Source) == 0 {
		return false, nil
	}

	ebs, err := bs.Copy()
	if err != nil {
		return false, err
	}

	err = tg.DependsOn.process(ctx, tr.Params, ebs)
	if err != nil {
		return false, err
	}

	src, err := tg.prepareSource(ctx, ebs)
	if err != nil {
		return false, err
	}

	env := make(map[string]interface{})
	env["bs"] = ebs

	x, err := plaxDsl.JSExec(ctx, src, env)
	if err != nil {
		return false, err
	}

	switch vv := x.(type) {
	case bool:
		return vv, nil
	default:
		return false, fmt.Errorf("guard Javascript returned a %T (%v) and not a bool: ", x, x)
	}
}
