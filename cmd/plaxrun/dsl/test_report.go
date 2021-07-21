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
	"log"
	"plugin"
)

const (
	pluginPathFormat = "%s/%s/report_%s.so"
	stdoutPlugin     = "stdout"
)

// TestReportPlugin name
type TestReportPlugin struct {
	// name of the report plugin
	name string `yaml:"name"`
	// Config is the configuration (if any) for the report plugin.
	Config interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

// TestReportPluginMap
type TestReportPluginMap map[string]TestReportPlugin

type TestReport struct {
	// Plugins, if given, should be the filename of a Go
	// plugin for a report plugin.
	//
	// See https://golang.org/pkg/plugin/.
	//
	// The subdirectory 'chans/sqlc/mysql' has an example for
	// loading a MySQL driver at runtime.
	Plugins TestReportPluginMap
}

// open the report plugin
func (trp *TestReportPlugin) open(ctx *Ctx) (*plugin.Plugin, error) {
	if trp.name != "" {
		name := fmt.Sprintf(pluginPathFormat, ctx.ReportPluginDir, trp.name, trp.name)
		log.Printf("DEBUG loading report plugin %s", name)
		p, err := plugin.Open(name)
		if err != nil {
			return nil, fmt.Errorf("failed to open report plugin %s: %s", name, err)
		}

		return p, nil
	}

	return nil, fmt.Errorf("test report plugin name was empty")
}

// Generate the test report using the TestReportPlugin for the TestRun
func (trp *TestReportPlugin) Generate(ctx *Ctx, tr *TestRun, config interface{}) error {
	p, err := trp.open(ctx)
	if err != nil {
		return err
	}

	f, err := p.Lookup("Generate")
	if err != nil {
		return err
	}

	err = f.(func(*Ctx, *TestRun, interface{}) error)(ctx, tr, config)
	if err != nil {
		return err
	}

	return nil
}

// Generate the test reports from the TestReportPluginMap for the TestRun
func (trpm TestReportPluginMap) Generate(ctx *Ctx, tr *TestRun) error {
	if _, ok := trpm[stdoutPlugin]; !ok {
		if trpm == nil {
			trpm = make(TestReportPluginMap)
		}

		if *tr.trps.EmitJSON {
			trpm[stdoutPlugin] = TestReportPlugin{
				Config: map[string]string{
					"Type": "JSON",
				},
			}
		} else {
			trpm[stdoutPlugin] = TestReportPlugin{
				Config: map[string]string{
					"Type": "XML",
				},
			}
		}
	}

	for key, trp := range trpm {
		trp.name = key
		err := trp.Generate(ctx, tr, trp.Config)
		if err != nil {
			ctx.Logf(err.Error())
		}
	}

	return nil
}
