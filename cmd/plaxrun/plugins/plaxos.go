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
package plugins

import (
	"context"
	"fmt"

	// Import required to dynamically register channels
	_ "github.com/Comcast/plax/chans"
	plaxInvoke "github.com/Comcast/plax/invoke"

	"github.com/Comcast/plax/cmd/plaxrun/dsl"
)

// PlaxOSPlugin used to invoke plax
type PlaxOSPlugin struct {
	invocation *plaxInvoke.Invocation
}

func init() {
	dsl.ThePluginRegistry.Register(
		"github.com/Comcast/plax",
		func(def dsl.PluginDef) (dsl.Plugin, error) {
			name, err := def.GetPluginDefName()
			if err != nil {
				return nil, err
			}

			params, err := def.GetPluginDefParams()
			if err != nil {
				return nil, err
			}

			bps := make(map[string]interface{}, len(params))
			for k, v := range params {
				key := fmt.Sprintf("?%s", k)
				value := v
				bps[key] = value
			}

			seed, err := def.GetPluginDefSeed()
			if err != nil {
				return nil, err
			}

			verbose, err := def.GetPluginDefVerbose()
			if err != nil {
				return nil, err
			}

			priority, err := def.GetPluginDefPriorty()
			if err != nil {
				return nil, err
			}

			labels, err := def.GetPluginDefLabels()
			if err != nil {
				return nil, err
			}

			logLevel, err := def.GetPluginDefLogLevel()
			if err != nil {
				return nil, err
			}

			list, err := def.GetPluginDefList()
			if err != nil {
				return nil, err
			}

			emitJSON, err := def.GetPluginDefEmitJSON()
			if err != nil {
				return nil, err
			}

			nonZeroOnAnyError, err := def.GetPluginDefNonzeroOnAnyErrorKey()
			if err != nil {
				return nil, err
			}

			retry, err := def.GetPluginDefRetry()

			i := plaxInvoke.Invocation{
				SuiteName:         name,
				Bindings:          bps,
				Seed:              seed,
				Verbose:           verbose,
				Priority:          priority,
				Labels:            labels,
				LogLevel:          logLevel,
				List:              list,
				EmitJSON:          emitJSON,
				NonzeroOnAnyError: nonZeroOnAnyError,
				Retry:             retry,
			}

			i.Dir, err = def.GetPluginDefDir()
			if err != nil {
				i.Filename, err = def.GetPluginDefFilename()
				if err != nil {
					return nil, fmt.Errorf("Neither Dir or Filename was provided in the plugin definition")
				}
			}

			p := &PlaxOSPlugin{
				invocation: &i,
			}

			return p, nil
		},
	)
}

// Invoke invokes the plugin
func (p *PlaxOSPlugin) Invoke(ctx context.Context) error {
	return p.invocation.Exec(ctx)
}
