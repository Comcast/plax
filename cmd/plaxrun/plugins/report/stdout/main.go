package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"

	plaxRunDsl "github.com/Comcast/plax/cmd/plaxrun/dsl"
	plaxDsl "github.com/Comcast/plax/dsl"
)

type ReportStdoutType string

const (
	JSON ReportStdoutType = "JSON"
	XML  ReportStdoutType = "XML"
)

type ReportStdoutConfig struct {
	Type ReportStdoutType
}

// Generate the stdout test report
func Generate(ctx *plaxRunDsl.Ctx, tr *plaxRunDsl.TestRun, cfg interface{}) error {
	var config ReportStdoutConfig

	if err := plaxDsl.As(cfg, &config); err != nil {
		return err
	}

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
