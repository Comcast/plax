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
package report

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/rpc"
	"os"
	"os/exec"
	"time"

	"github.com/Comcast/plax/junit"

	plugin "github.com/hashicorp/go-plugin"
)

// TestReport is the toplevel object for the plaxrun test report
type TestReport struct {
	XMLName   *xml.Name          `xml:"testreport" json:"-,omitempty"`
	Name      string             `xml:"name,attr,omitempty" json:"name,omitempty"`
	Version   string             `xml:"version,attr,omitempty" json:"version,omitempty"`
	TestSuite []*junit.TestSuite `xml:"testsuite" json:"testsuite"`
	Total     int                `xml:"tests,attr" json:"tests"`
	Passed    int                `xml:"passed,attr" json:"passed"`
	Skipped   int                `xml:"skipped,attr" json:"skipped"`
	Failures  int                `xml:"failures,attr" json:"failures"`
	Errors    int                `xml:"errors,attr" json:"errors"`
	Started   time.Time          `xml:"started,attr" json:"timestamp"`
	Time      time.Duration      `xml:"time,attr" json:"time"`
}

// NewTestReport builds the TestReport
func NewTestReport() *TestReport {
	return &TestReport{
		TestSuite: make([]*junit.TestSuite, 0),
		Started:   time.Now().UTC(),
	}
}

// HasError determines if test report has any errors
func (tr *TestReport) HasError() bool {
	return tr.Errors > 0
}

// Finish the TestReport
func (tr *TestReport) Finish(message ...string) {
	now := time.Now().UTC()
	time := now.Sub(tr.Started)
	tr.Time = time
}

// Generate the TestReport
func (tr *TestReport) Generate(name string, cfgb []byte) error {
	generator, err := NewGenerator(name)
	if err != nil {
		return err
	}

	if cfgb != nil {
		err = generator.Config(cfgb)
		if err != nil {
			return err
		}
	}

	err = generator.Generate(tr)
	if err != nil {
		return err
	}

	return nil
}

const (
	pluginNameFormat = "plaxrun_report_%s"
	PluginName       = "report"
)

// Generate interface for the report plugin
type Generator interface {
	Config([]byte) error
	Generate(*TestReport) error
}

// ReportPlugin is the implementation of plugin.Plugin so we can serve/consume this.
type ReportPlugin struct {
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl Generator
}

func (p *ReportPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &ReportRPCServer{Impl: p.Impl}, nil
}

func (*ReportPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &ReportRPCClient{rpcClient: c}, nil
}

// handshakeConfigs are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "REPORT_PLUGIN",
	MagicCookieValue: "generator",
}

// pluginMap is the map of plugins we can dispense.
var PluginMap = map[string]plugin.Plugin{
	PluginName: &ReportPlugin{},
}

// Here is an implementation that talks over RPC
type ReportRPCClient struct {
	client    *plugin.Client
	rpcClient *rpc.Client
}

func (m *ReportRPCClient) Config(cfgb []byte) error {
	var resp error

	err := m.rpcClient.Call("Plugin.Config", cfgb, &resp)
	if err != nil {
		// You usually want your interfaces to return errors. If they don't,
		// there isn't much other choice here.
		return err
	}

	return resp
}

func (m *ReportRPCClient) Generate(tr *TestReport) error {
	trb, err := json.Marshal(tr)
	if err != nil {
		return err
	}

	var resp error

	err = m.rpcClient.Call("Plugin.Generate", trb, &resp)
	if err != nil {
		// You usually want your interfaces to return errors. If they don't,
		// there isn't much other choice here.
		return err
	}

	m.client.Kill()

	return resp
}

// Here is the RPC server that GenerateRPC talks to, conforming to
// the requirements of net/rpc
type ReportRPCServer struct {
	// This is the real implementation
	Impl Generator
}

func (m *ReportRPCServer) Config(cfgb []byte, resp *interface{}) error {
	return m.Impl.Config(cfgb)
}

func (m *ReportRPCServer) Generate(trb []byte, resp *interface{}) error {
	tr := &TestReport{}
	err := json.Unmarshal(trb, tr)
	if err != nil {
		return err
	}

	return m.Impl.Generate(tr)
}

func NewGenerator(name string) (Generator, error) {
	// We're a host! Start by launching the plugin process.
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         PluginMap,
		Cmd:             exec.Command(fmt.Sprintf(pluginNameFormat, name)),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC, plugin.ProtocolGRPC,
		},
		SyncStdout: os.Stdout,
	})

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		return nil, err
	}

	// Request the plugin
	raw, err := rpcClient.Dispense(PluginName)
	if err != nil {
		return nil, err
	}

	reportClient, ok := raw.(*ReportRPCClient)
	if !ok {
		return nil, fmt.Errorf("failed to cast to ReportRPClient")
	}

	reportClient.client = client

	generator, ok := raw.(Generator)
	if !ok {
		return nil, fmt.Errorf("failed to cast plugin to Generator")
	}

	return generator, nil
}
