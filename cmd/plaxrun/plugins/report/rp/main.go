package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"github.com/google/uuid"

	"github.com/Comcast/plax/cmd/plaxrun/plugins/report"
	
	rp "github.com/avarabyeu/goRP/v5/gorp"
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

// Config the plugin
func (rpi *ReportPortalImpl) Config(cfgb []byte) error {
	logger.Debug("plaxrun_report_rp: config called")

	err := json.Unmarshal(cfgb, &rpi.config)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	rpi.http = resty.New().
		// SetDebug(true).
		SetHostURL(rpi.config.Hostname).
		SetAuthToken(rpi.config.Token).
		OnAfterResponse(func(client *resty.Client, rs *resty.Response) error {
			if rs.StatusCode() >= 300 {
				return fmt.Errorf("status code error: %d\n%s", rs.StatusCode(), rs.String())
			}
			
			return nil
		})
		logger.Debug("plaxrun_report_rp: config done")
	return nil
}

// Generate the test report
func (rpi *ReportPortalImpl) Generate(tr *report.TestReport) error {
	logger.Debug("plaxrun_report_rp: generate called")
	client := rp.NewClient(rpi.config.Hostname, rpi.config.Project, rpi.config.Token)

	launchUUID := uuid.New()
	launch, err := client.StartLaunch(&rp.StartLaunchRQ{
		Mode: rp.LaunchModes.Default,
		StartRQ: rp.StartRQ{
			Name:        tr.Name,
			UUID:        &launchUUID,
			StartTime:   rp.Timestamp{Time: time.Now()},
		},
	})

	if err != nil {
		logger.Error(err.Error())
		return err
	}

	for _, testSuite := range tr.TestSuite {
		for _, testCase := range testSuite.TestCase {
				
			testUUID := uuid.New()
			_, err = client.StartTest(&rp.StartTestRQ{
				LaunchID: launch.ID,
				CodeRef:  testSuite.Name,
				UniqueID: "another one unique ID",
				Retry:    false,
				Type:     rp.TestItemTypes.Test,
				StartRQ: rp.StartRQ{
					Name:      testCase.Name,
					StartTime: rp.Timestamp{time.Now()},
					UUID:      &testUUID,
				},
			})
			if err != nil {
				logger.Error(err.Error())
				return err
			}

			_, err = client.SaveLog(&rp.SaveLogRQ{
				LaunchUUID: launchUUID.String(),
				ItemID:     testUUID.String(),
				Level:      rp.LogLevelInfo,
				LogTime:    rp.Timestamp{time.Now()},
				Message:    string(testCase.Message),
			})
			if err != nil {
				logger.Error(err.Error())
				return err
			}

			status := rp.Statuses.Passed
			if testCase.Status != "passed" {
				status = rp.Statuses.Failed
			}

			_, err = client.FinishTest(testUUID.String(), &rp.FinishTestRQ{
				LaunchUUID: launchUUID.String(),
				FinishExecutionRQ: rp.FinishExecutionRQ{
					EndTime: rp.Timestamp{time.Now()},
					Status:  status,
				},
			})
			if err != nil {
				logger.Error(err.Error())
				return err
			}
		}

		if err != nil {
			logger.Error(err.Error())
			continue
		}
	}

	_, err = client.FinishLaunch(launchUUID.String(), &rp.FinishExecutionRQ{
		EndTime: rp.Timestamp{time.Now()},
	})
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	return nil
}

func main() {
	logger.Debug("plaxrun_report_rp: start")

	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]plugin.Plugin{
		report.PluginName: &report.ReportPlugin{Impl: &ReportPortalImpl{}},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: report.HandshakeConfig,
		Plugins:         pluginMap,
	})

	logger.Debug("plaxrun_report_rp: stop")
}
