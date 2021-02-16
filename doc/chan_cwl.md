## `cwl`

CWLChan is the Cloudwatch Logs Channel

### Options

CWLOpts is the Cloudwatch Logs Options

1. `region` is the region of the AWS Account

1. `groupName` (string) is the Cloudwatch Log Group Name

1. `StreamNamePrefix` is the Cloudwatch Log Stream Name prefix

1. `FilterPattern` (string) is based on the Cloudwatch Filter Pattern syntax
    Reference: (https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html)

1. `StartTimePadding` defines the time in seconds to subtract from now

1. `PollInterval` defines the Cloudwatch log poll time interval in seconds

