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

require (
	github.com/fatih/color v1.7.0 // indirect
	github.com/golang/protobuf v1.3.4 // indirect
	github.com/hashicorp/yamux v0.0.0-20180604194846-3520598351bb // indirect
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.13 // indirect
	github.com/mitchellh/go-testing-interface v0.0.0-20171004221916-a61a99592b77 // indirect
	github.com/oklog/run v1.0.0 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	google.golang.org/genproto v0.0.0-20190819201941-24fa4b261c55 // indirect
	google.golang.org/grpc v1.27.1 // indirect
)

exclude github.com/manifoldco/promptui v0.8.0
