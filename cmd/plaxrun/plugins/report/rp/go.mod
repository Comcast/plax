module github.com/Comcast/plax/cmd/plaxrun/plugins/report/plaxrun_report_rp

go 1.17

//replace github.com/Comcast/plax => ../work/comcast/plax

require (
	github.com/Comcast/plax v0.8.5
	github.com/avarabyeu/goRP/v5 v5.0.1
	github.com/go-resty/resty/v2 v2.7.0
	github.com/google/uuid v1.1.2
	github.com/hashicorp/go-hclog v0.16.2
	github.com/hashicorp/go-plugin v1.4.2
)

exclude github.com/manifoldco/promptui v0.8.0
