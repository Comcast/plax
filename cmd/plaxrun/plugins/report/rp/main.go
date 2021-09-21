package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Comcast/plax/cmd/plaxrun/plugins/report"

	resty "github.com/go-resty/resty/v2"
	hclog "github.com/hashicorp/go-hclog"
	plugin "github.com/hashicorp/go-plugin"
)

type ReportPortalConfig struct {
	Hostname string `yaml:"hostname"`
	Token    string `yaml:"token"`
	Project  string `yaml:"project"`
}

type ReportPortalImpl struct {
	http   *resty.Client
	config ReportPortalConfig
}

var (
	logger = hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})
)

type Attribute struct {
	Key    string `json:"key"`
	System bool   `json:"system"`
	Value  string `json:"value"`
}

type Attributes []Attribute

// LaunchMode - DEFAULT/DEBUG
type LaunchMode string

// launchModeValuesType contains enum values for launch mode
type launchModeValuesType struct {
	Default LaunchMode
	Debug   LaunchMode
}

// LaunchModes is enum values for easy access
var LaunchModes = launchModeValuesType{
	Default: "DEFAULT",
	Debug:   "DEBUG",
}

type StartLaunchRQ struct {
	Attributes  Attributes `json:"attributes"`
	Description string     `json:"description"`
	Name        string     `json:"name"`
	Mode        string     `json:"mode"`
	Rerun       bool       `json:"rerun"`
	RerunOf     string     `json:"rerunOf"`
	StartTime   time.Time  `json:"startTime"`
}

type Status string

// Statuses is enum values for easy access
const (
	Passed      Status = "PASSED"
	Failed      Status = "FAILED"
	Stopped     Status = "STOPPED"
	Skipped     Status = "SKIPPED"
	Interrupted Status = "INTERRUPTED"
	Canceled    Status = "CANCELLED"
	Info        Status = "INFO"
	Warn        Status = "WARN"
)

type AttributeKey string

const (
	TotalKey   AttributeKey = "TOTAL"
	PassedKey  AttributeKey = "PASSED"
	FailedKey  AttributeKey = "FAILED"
	SkippedKey AttributeKey = "SKIPPED"
)

type Created struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
}

type FinishLaunchRQ struct {
	Attributes  Attributes `json:"attributes"`
	Description string     `json:"description"`
	EndTime     time.Time  `json:"endTime"`
	Status      Status     `json:"status"`
}

type FinishLaunchRS struct {
	Created
	Number int `json:"number"`
}

type StopLaunchRS struct {
	Message string `json:"message"`
}

type Parameter struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Parameters []Parameter

type TestType string

const (
	Suite        = "SUITE"
	Story        = "STORY"
	Test         = "TEST"
	Scenario     = "SCENARIO"
	Step         = "STEP"
	BeforeClass  = "BEFORE_CLASS"
	BeforeGroups = "BEFORE_GROUPS"
	BeforeMethod = "BEFORE_METHOD"
	BeforeSuite  = "BEFORE_SUITE"
	BeforeTest   = "BEFORE_TEST"
	AfterClass   = "AFTER_CLASS"
	AfterGroups  = "AFTER_GROUPS"
	AfterMethod  = "AFTER_METHOD"
	AfterSuite   = "AFTER_SUITE"
	AfterTest    = "AFTER_TEST"
)

type StartTestItemRQ struct {
	Attributes  Attributes `json:"attributes"`
	CodeRef     string     `json:"codeRef"`
	Description string     `json:"description"`
	HasStats    bool       `json:"hasStats"`
	LaunchUUID  string     `json:"launchUuid"`
	Name        string     `json:"name"`
	Parameters  Parameters `json:"parameters"`
	Retry       bool       `json:"retry"`
	RetryOf     string     `json:"retryOf"`
	StartTime   time.Time  `json:"startTime"`
	TestCaseID  string     `json:"testCaseId"`
	Type        TestType   `json:"type"`
	UniqueID    string     `json"uniqueId"`
}

type FinishExecutionRQ struct {
	Attributes  Attributes `json:"attributes"`
	Description string     `json:"description"`
	EndTime     time.Time  `json:"endTime"`
	LaunchUUID  string     `json:"launchUuid"`
	Retry       bool       `json:"retry"`
	RetryOf     string     `json:"retryOf"`
	Status      Status     `json:"status"`
	TestCaseID  string     `json:"testCaseId"`
}

type FinishExecutionRS struct {
	Message string `json:"message"`
}

func (rpi *ReportPortalImpl) startLaunch(rq *StartLaunchRQ) (*Created, error) {
	logger.Debug("plaxrun_report_portal: startLaunch called")

	if rpi.http == nil {
		return nil, fmt.Errorf("http client is nil")
	}

	var rs Created

	_, err := rpi.http.R().
		SetPathParams(map[string]string{"project": rpi.config.Project}).
		SetBody(rq).
		SetResult(&rs).
		Post("/api/v1/{project}/launch")
	if err != nil {
		return nil, err
	}

	logger.Debug("plaxrun_report_portal: startLaunch done")

	return &rs, nil
}

func (rpi *ReportPortalImpl) finishLaunch(launchId string, rq *FinishLaunchRQ) (*FinishLaunchRS, error) {
	logger.Debug("plaxrun_report_portal: finishLaunch called")

	if rpi.http == nil {
		return nil, fmt.Errorf("http client is nil")
	}

	var rs FinishLaunchRS

	_, err := rpi.http.R().
		SetPathParams(map[string]string{
			"project":  rpi.config.Project,
			"launchId": launchId,
		}).
		SetBody(rq).
		SetResult(&rs).
		Put("/api/v1/{project}/launch/{launchId}/finish")
	if err != nil {
		return nil, err
	}

	logger.Debug("plaxrun_report_portal: finishLaunch done")

	return &rs, nil
}

func (rpi *ReportPortalImpl) startTestItem(rq *StartTestItemRQ) (*Created, error) {
	logger.Debug("plaxrun_report_portal: startTestItem called")

	if rpi.http == nil {
		return nil, fmt.Errorf("http client is nil")
	}

	var rs Created

	_, err := rpi.http.R().
		SetPathParams(map[string]string{"project": rpi.config.Project}).
		SetBody(rq).
		SetResult(&rs).
		Post("/api/v1/{project}/item")
	if err != nil {
		return nil, err
	}

	logger.Debug("plaxrun_report_portal: startTestItem done")

	return &rs, nil
}

func (rpi *ReportPortalImpl) finishTestItem(testItemId string, rq *FinishExecutionRQ) (*FinishExecutionRS, error) {
	logger.Debug("plaxrun_report_portal: finishTestItem called")

	if rpi.http == nil {
		return nil, fmt.Errorf("http client is nil")
	}

	var rs FinishExecutionRS

	_, err := rpi.http.R().
		SetPathParams(map[string]string{
			"project":    rpi.config.Project,
			"testItemId": testItemId,
		}).
		SetBody(rq).
		SetResult(&rs).
		Put("/api/v1/{project}/item/{testItemId}")
	if err != nil {
		return nil, err
	}

	logger.Debug("plaxrun_report_portal: finishTestItem done")

	return &rs, nil
}

// Config the plugin
func (rpi *ReportPortalImpl) Config(cfgb []byte) error {
	logger.Debug("plaxrun_report_portal: config called")

	err := json.Unmarshal(cfgb, &rpi.config)
	if err != nil {
		return err
	}

	rpi.http = resty.New().
		// SetDebug(true).
		SetHostURL(rpi.config.Hostname).
		SetAuthToken(rpi.config.Token).
		OnAfterResponse(func(client *resty.Client, rs *resty.Response) error {
			fmt.Printf("statusCode = %d\n", rs.StatusCode())
			if rs.StatusCode() >= 300 {
				return fmt.Errorf("status code error: %d\n%s", rs.StatusCode(), rs.String())
			}

			return nil
		})

	return nil
}

// Generate the test report
func (rpi *ReportPortalImpl) Generate(tr *report.TestReport) error {
	logger.Debug("plaxrun_report_portal: generate called")

	launch, err := rpi.startLaunch(&StartLaunchRQ{
		StartTime: tr.Started,
		Mode:      string(LaunchModes.Default),
		Name:      tr.Name,
	})
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	for _, testSuite := range tr.TestSuite {
		suiteTestItem, err := rpi.startTestItem(&StartTestItemRQ{
			Name:       testSuite.Name,
			LaunchUUID: launch.ID,
			StartTime:  testSuite.Started,
			Type:       Suite,
		})
		if err != nil {
			logger.Error(err.Error())
			return err
		}

		for _, testCase := range testSuite.TestCase {
			suiteTestItem, err := rpi.startTestItem(&StartTestItemRQ{
				Name:       testCase.Name,
				LaunchUUID: launch.ID,
				StartTime:  *testCase.Started,
				Type:       Test,
			})
			if err != nil {
				logger.Error(err.Error())
				return err
			}

			_, err = rpi.finishTestItem(suiteTestItem.ID, &FinishExecutionRQ{
				LaunchUUID:  launch.ID,
				TestCaseID:  suiteTestItem.ID,
				Status:      Status(testCase.Status),
				EndTime:     testCase.Started.Add(*testCase.Time),
				Description: testCase.Message,
			})
			if err != nil {
				logger.Error(err.Error())
				return err
			}
		}

		status := Passed
		if testSuite.Errors > 0 || testSuite.Failures > 0 {
			status = Failed
		}

		_, err = rpi.finishTestItem(suiteTestItem.ID, &FinishExecutionRQ{
			LaunchUUID:  launch.ID,
			TestCaseID:  suiteTestItem.ID,
			Status:      status,
			EndTime:     testSuite.Started.Add(testSuite.Time),
			Description: testSuite.Message,
		})
		if err != nil {
			logger.Error(err.Error())
			return err
		}
	}

	status := Passed

	if tr.Errors > 0 || tr.Failures > 0 {
		status = Failed
	}

	_, err = rpi.finishLaunch(launch.ID, &FinishLaunchRQ{
		Attributes: Attributes{
			{
				Key:   string(TotalKey),
				Value: strconv.Itoa(tr.Total),
			},
		},
		Status:      status,
		EndTime:     tr.Started.Add(tr.Time),
		Description: "TestReport complete",
	})
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	return nil
}

func main() {
	logger.Debug("plaxrun_report_portal: start")

	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]plugin.Plugin{
		report.PluginName: &report.ReportPlugin{Impl: &ReportPortalImpl{}},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: report.HandshakeConfig,
		Plugins:         pluginMap,
	})

	logger.Debug("plaxrun_report_portal: stop")
}
