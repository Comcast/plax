module github.com/Comcast/plax/cmd/plaxrun/plugins/report/plaxrun_report_stdout

go 1.16

replace github.com/Comcast/plax => ../../../../..

require (
	github.com/Comcast/plax v0.0.0-00010101000000-000000000000
	github.com/hashicorp/go-hclog v0.16.2
	github.com/hashicorp/go-plugin v1.4.2
)
