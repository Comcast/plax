package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"time"

	"github.com/Comcast/plax/cmd/plaxrun/plugins/report"
	"github.com/Comcast/plax/junit"

	resty "github.com/go-resty/resty/v2"
	hclog "github.com/hashicorp/go-hclog"
	plugin "github.com/hashicorp/go-plugin"
)

type TestResult struct {
	XMLName      xml.Name     `xml:"test_result"`
	ProductAreas ProductAreas `xml:"product_areas,omitempty"`
	TestRuns     TestRuns     `xml:"test_runs"`
}

type ProductAreas struct {
	ProductArea ID `xml:"product_area_ref"`
}

type ID struct {
	ID int `xml:"id,attr"`
}

type TestFields struct {
	TestFields []TestField `xml:"test_field"`
}

type TestField struct {
	Type  string `xml:"type,attr"`
	Value string `xml:"value,attr"`
}

type TestRuns struct {
	TestRun []TestRun `xml:"test_run"`
}

type TestRun struct {
	Name       string        `xml:"name,attr"`
	Duration   time.Duration `xml:"duration,attr"`
	Status     Status        `xml:"status,attr"`
	Started    int64         `xml:"started,attr"`
	TestFields TestFields    `xml:"test_fields,omitempty"`
	Error      string        `xml:"error,omitempty"`
}

type Status string

// Statuses is enum values for easy access
const (
	Passed  Status = "Passed"
	Failed  Status = "Failed"
	Planned Status = "Planned"
	Skipped Status = "Skipped"
)

type AuthCreds struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// ReportStdoutConfig configures the stdout plugin for either JSON or XML output
type OctaneReportConfig struct {
	HostUrl       string            `yaml:"host_url" json:"host_url"`
	ClientID      string            `yaml:"client_id" json:"client_id"`
	ClientSecret  string            `yaml:"client_secret" json:"client_secret"`
	SharedSPaceID string            `yaml:"shared_space_id" json:"shared_space_id"`
	WorkSpaceID   string            `yaml:"workspace_id" json:"workspace_id"`
	AppModuleID   int               `yaml:"app_module_id" json:"app_module_id"`
	TestFields    map[string]string `yaml:"test_fields,omitempty" json:"test_fields,omitempty"`
}

// OctaneReportPluginImpl dummy structure
type OctaneReportPluginImpl struct {
	// configuration for the report plugin
	http   *resty.Client
	config OctaneReportConfig
}

var (
	// logger for the plugin
	logger = hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})
)

func getStatus(x Status) Status {
	var status Status = x
	switch x {
	case Status(junit.Passed):
		status = Passed
	case Status(junit.Failed):
		status = Failed
	case Status(junit.Error):
		status = Failed
	case Status(junit.Skipped):
		status = Skipped
	default:
		logger.Error("unsupported status", x)
	}
	return status
}

// Config the plugin
func (rpi *OctaneReportPluginImpl) Config(cfgb []byte) error {
	logger.Debug("plaxrun_report_octane: config called")
	err := json.Unmarshal(cfgb, &rpi.config)
	if err != nil {
		return err
	}

	rpi.http = resty.New().
		SetDebug(true).
		SetHostURL(rpi.config.HostUrl).
		OnAfterResponse(func(client *resty.Client, rs *resty.Response) error {
			fmt.Printf("statusCode = %d\n", rs.StatusCode())
			if rs.StatusCode() >= 300 {
				return fmt.Errorf("status code error: %d\n%s", rs.StatusCode(), rs.String())
			}

			return nil
		})

	_, err = rpi.http.R().
		SetHeader("Content-Type", "application/json").
		SetBody(&AuthCreds{ClientID: rpi.config.ClientID, ClientSecret: rpi.config.ClientSecret}).
		Post("/authentication/sign_in")
	if err != nil {
		return err
	}

	return nil

}

// Generate the test report
func (rpi *OctaneReportPluginImpl) Generate(tr *report.TestReport) error {
	logger.Debug("plaxrun_report_octane: generate called")

	testreport := TestResult{
		TestRuns: TestRuns{},
	}
	testfields := TestFields{}
	if len(rpi.config.TestFields) > 0 {
		fmt.Println("test fields", rpi.config.TestFields)
		for k, v := range rpi.config.TestFields {
			testfields.TestFields = append(testfields.TestFields, TestField{Type: k, Value: v})
		}
	}
	for _, testSuite := range tr.TestSuite {
		for _, testCase := range testSuite.TestCase {
			suiteTestItem := TestRun{
				Name:       testCase.Name + " " + testSuite.Name,
				Duration:   *testCase.Time / 1000000,
				Status:     getStatus(Status(testCase.Status)),
				Started:    testCase.Started.Unix() * 1000,
				TestFields: testfields,
			}
			if suiteTestItem.Status == Failed {
				suiteTestItem.Error = testCase.Message
			}
			testreport.TestRuns.TestRun = append(testreport.TestRuns.TestRun, suiteTestItem)

		}
	}
	testreport.ProductAreas.ProductArea.ID = rpi.config.AppModuleID

	file, err := xml.MarshalIndent(testreport, "", " ")
	if err != nil {
		return err
	}
	fmt.Printf("plaxrun_report_octane:  print tr\n%s\n", file)

	_, err = rpi.http.R().
		SetPathParams(map[string]string{
			"shared_space_id": rpi.config.SharedSPaceID,
			"workspace_id":    rpi.config.WorkSpaceID,
		}).
		SetHeader("Content-Type", "application/xml").
		SetBody(testreport).
		Post("/api/shared_spaces/{shared_space_id}/workspaces/{workspace_id}/test-results")

	if err != nil {
		return err
	}
	return nil

}

// main plugin method
func main() {
	logger.Debug("plaxrun_report_octane: start")

	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]plugin.Plugin{
		report.PluginName: &report.ReportPlugin{Impl: &OctaneReportPluginImpl{}},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: report.HandshakeConfig,
		Plugins:         pluginMap,
	})

	logger.Debug("plaxrun_report_octane: stop")
}
