module github.com/Comcast/plax/cmd/plaxrun/plugins/report/plaxrun_report_rp

go 1.16

//replace github.com/Comcast/plax => ../work/comcast/plax

require (
	github.com/Comcast/plax v0.8.5
	github.com/go-resty/resty/v2 v2.3.0
	github.com/hashicorp/go-hclog v0.16.2
	github.com/hashicorp/go-plugin v1.4.2
	github.com/stretchr/testify v1.4.0 // indirect
)
