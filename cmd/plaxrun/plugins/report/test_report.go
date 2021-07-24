package report

import (
	"time"

	"github.com/Comcast/plax/junit"
)

// TestReport is the toplevel object for the plaxrun test report
type TestReport struct {
	Name      string             `xml:"name,attr,omitempty" json:"name,omitempty"`
	Version   string             `xml:"version,attr,omitempty" json:"version,omitempty"`
	TestSuite []*junit.TestSuite `xml:"testsuite" json:"testsuite"`
	Total     int                `xml:"tests,attr" json:"tests"`
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
func (tr *TestReport) Generate(name string, config interface{}) error {
	generator, err := NewGenerator(name, config)
	if err != nil {
		return err
	}

	err = generator.Generate(tr)
	if err != nil {
		return err
	}

	return nil
}
