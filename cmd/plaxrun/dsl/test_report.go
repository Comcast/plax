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

	"github.com/Comcast/plax/cmd/plaxrun/plugins/report"
)

const (
	stdoutPlugin = "stdout"
)

// TestReportPlugin to execute the report
type TestReportPlugin struct {
	// name of the report plugin
	name string `yaml:"name"`

	// Config is the configuration (if any) for the report plugin.
	Config interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

// TestReportPluginMap dsl for mpa of TestReportPlugins
type TestReportPluginMap map[string]TestReportPlugin

// Generate the test report using the TestReportPlugin for the TestRun
func (trp *TestReportPlugin) Generate(ctx *Ctx, tr *report.TestReport) error {
	if trp.name == "" {
		return fmt.Errorf("test report plugin name is nil")
	}

	if tr == nil {
		return fmt.Errorf("test report is nil")
	}

	return tr.Generate(trp.name, ctx.ReportPluginDir, trp.Config)
}

// Generate the test reports from the TestReportPluginMap for the TestRun
func (trpm TestReportPluginMap) Generate(ctx *Ctx, tr *report.TestReport, emitJSON bool) error {
	if _, ok := trpm[stdoutPlugin]; !ok {
		if trpm == nil {
			trpm = make(TestReportPluginMap)
		}

		if emitJSON {
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
		err := trp.Generate(ctx, tr)
		if err != nil {
			ctx.Logf(err.Error())
		}
	}

	return nil
}
