package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"

	"github.com/Comcast/plax/cmd/plaxrun/plugins/report"

	hclog "github.com/hashicorp/go-hclog"
	plugin "github.com/hashicorp/go-plugin"
)

// ReportStdoutType definition
type ReportStdoutType string

const (
	// JSON output
	JSON ReportStdoutType = "JSON"
	// XML output
	XML ReportStdoutType = "XML"
)

// ReportStdoutConfig configures the stdout plugin for either JSON or XML output
type ReportStdoutConfig struct {
	Type ReportStdoutType
}

// ReportPluginImpl dummy structure
type ReportPluginImpl struct{}

var (
	// logger for the plugin
	logger = hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})
	// config the plugin
	config ReportStdoutConfig
)

// Config the plugin
func (ReportPluginImpl) Config(cfg interface{}) error {
	logger.Debug("plaxrun_report_stdout: config called")

	cfgb, ok := cfg.([]byte)
	if !ok {
		return fmt.Errorf("failed to cast config to []byte")
	}
	err := json.Unmarshal(cfgb, &config)
	if err != nil {
		return err
	}

	return nil
}

// Generate the test report
func (ReportPluginImpl) Generate(tr *report.TestReport) error {
	logger.Debug("plaxrun_report_stdout: generate called")

	var (
		bs  = make([]byte, 0)
		err error
	)

	switch config.Type {
	case JSON:
		// Write the JSON.
		bs, err = json.MarshalIndent(tr, "", "  ")
		if err != nil {
			return err
		}
	case XML:
		// Write the XML
		bs, err = xml.MarshalIndent(tr, "", "  ")
		if err != nil {
			return err
		}
	}

	if len(bs) > 0 {
		fmt.Printf("%s\n", bs)
	}

	return nil
}

// main plugin method
func main() {
	logger.Debug("plaxrun_report_stdout: start")

	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]plugin.Plugin{
		report.PluginName: &report.ReportPlugin{Impl: ReportPluginImpl{}},
	}

	// Serve the plugin
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: report.HandshakeConfig,
		Plugins:         pluginMap,
	})

	logger.Debug("plaxrun_report_stdout: stop")
}
