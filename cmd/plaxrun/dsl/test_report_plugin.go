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

	"github.com/Comcast/plax/cmd/plaxrun/plugins/report"
	"github.com/Comcast/plax/dsl"
	plaxDsl "github.com/Comcast/plax/dsl"
)

const (
	stdoutPlugin = "stdout"
)

// TestReportPlugin to execute the report
type TestReportPlugin struct {
	// name of the report plugin
	name string `yaml:"name"`

	// DependsOn the parameters in the map for configuration
	DependsOn TestParamDependencyList `yaml:"dependsOn"`

	// Config is the configuration (if any) for the report plugin.
	Config interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

// TestReportPluginMap dsl for mpa of TestReportPlugins
type TestReportPluginMap map[string]TestReportPlugin

// Generate the test report using the TestReportPlugin for the TestRun
func (trp *TestReportPlugin) Generate(ctx *dsl.Ctx, tpbm TestParamBindingMap, bs plaxDsl.Bindings, tr *report.TestReport) error {
	if trp.name == "" {
		return fmt.Errorf("test report plugin name is nil")
	}

	if tr == nil {
		return fmt.Errorf("test report is nil")
	}

	err := trp.DependsOn.process(ctx, tpbm, &bs)
	if err != nil {
		return err
	}

	var cfgb []byte = nil

	if trp.Config != nil {
		cfgb, err = json.Marshal(trp.Config)
		if err != nil {
			return err
		}

		cfg, err := bs.Sub(ctx, string(cfgb))
		if err != nil {
			return err
		}

		cfgb = []byte(cfg)
	}

	return tr.Generate(trp.name, cfgb)
}

// Generate the test reports from the TestReportPluginMap for the TestRun
func (trpm TestReportPluginMap) Generate(ctx *dsl.Ctx, tpbm TestParamBindingMap, bs plaxDsl.Bindings, tr *report.TestReport, emitJSON bool) error {
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
		err := trp.Generate(ctx, tpbm, bs, tr)
		if err != nil {
			ctx.Logf(err.Error())
		}
	}

	return nil
}
