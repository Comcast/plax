package report

import (
	"encoding/json"
	"fmt"
	"go/build"
	"net/rpc"
	"os"
	"os/exec"

	plugin "github.com/hashicorp/go-plugin"
)

const (
	pluginPathFormat = "%s/bin/plaxrun_report_%s"
	PluginName       = "report"
)

// Generate interface for the report plugin
type Generator interface {
	Config(interface{}) error
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

func (m *ReportRPCClient) Config(cfg interface{}) error {
	cfgb, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	var resp error

	err = m.rpcClient.Call("Plugin.Config", cfgb, &resp)
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

func NewGenerator(name string, cfg interface{}) (Generator, error) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	path := fmt.Sprintf(pluginPathFormat, gopath, name)

	// We're a host! Start by launching the plugin process.
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         PluginMap,
		Cmd:             exec.Command(path),
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

	err = generator.Config(cfg)
	if err != nil {
		return nil, err
	}

	return generator, nil
}
