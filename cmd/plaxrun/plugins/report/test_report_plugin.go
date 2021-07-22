package report

import (
	"encoding/json"
	"fmt"
	"net/rpc"
	"os/exec"
	"time"

	plugin "github.com/hashicorp/go-plugin"

	"github.com/Comcast/plax/junit"
)

const (
	pluginPathFormat = "%s/%s/plaxrun_report_%s"
	PluginName       = "report"
)

// TestReport is the toplevel object for the plaxrun test report
type TestReport struct {
	Name      string             `yaml:"name" json:"name"`
	Version   string             `yaml:"version" json:"version"`
	TestSuite []*junit.TestSuite `xml:"testsuite" json:"testsuite"`
	Total     int                `xml:"tests,attr" json:"tests"`
	Skipped   int                `xml:"skipped,attr" json:"skipped"`
	Failures  int                `xml:"failures,attr" json:"failures"`
	Errors    int                `xml:"errors,attr" json:"errors"`
	Started   time.Time          `xml:"started,attr" json:"timestamp"`
	Time      time.Duration      `xml:"time,attr" json:"time"`
}

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

// Finish the test report
func (tr *TestReport) Finish(message ...string) {
	now := time.Now().UTC()
	time := now.Sub(tr.Started)
	tr.Time = time
}

// Generate interface for the report plugin
type Generator interface {
	Generate(*TestReport, interface{}) error
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
	return &ReportRPCClient{client: c}, nil
}

func (tr *TestReport) Generate(name string, dir string, config interface{}) error {
	path := fmt.Sprintf(pluginPathFormat, dir, name, name)

	// We're a host! Start by launching the plugin process.
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         PluginMap,
		Cmd:             exec.Command(path),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC,
		},
	})
	defer client.Kill()

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		return err
	}

	// Request the plugin
	raw, err := rpcClient.Dispense(PluginName)
	if err != nil {
		return err
	}

	if plugin, ok := raw.(Generator); ok {
		err := plugin.Generate(tr, config)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("failed to cast TestReportPlugin")
	}

	return nil
}

// handshakeConfigs are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "REPORT_PLUGIN",
	MagicCookieValue: "generate",
}

// pluginMap is the map of plugins we can dispense.
var PluginMap = map[string]plugin.Plugin{
	PluginName: &ReportPlugin{},
}

// Here is an implementation that talks over RPC
type ReportRPCClient struct{ client *rpc.Client }

func (m *ReportRPCClient) Generate(tr *TestReport, cfg interface{}) error {
	trb, err := json.Marshal(tr)
	if err != nil {
		return err
	}

	cfgb, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	args := map[string]interface{}{
		"TestReport": trb,
		"Config":     cfgb,
	}

	var resp error

	err = m.client.Call("Plugin.Generate", args, &resp)
	if err != nil {
		// You usually want your interfaces to return errors. If they don't,
		// there isn't much other choice here.
		panic(err)
	}

	return resp
}

// Here is the RPC server that GenerateRPC talks to, conforming to
// the requirements of net/rpc
type ReportRPCServer struct {
	// This is the real implementation
	Impl Generator
}

func (m *ReportRPCServer) Generate(args map[string]interface{}, resp *interface{}) error {
	tr := &TestReport{}
	trb := args["TestReport"].([]byte)
	err := json.Unmarshal(trb, tr)
	if err != nil {
		return err
	}
	cfgb := args["Config"].([]byte)

	return m.Impl.Generate(tr, cfgb)
}
