## `cwl`

This channel type can produce and consume AWS CloudWatch logs.

### Options


1. `region` (*string) is the region of the AWS Account

1. `groupName` (string) is the Cloudwatch Log Group Name

1. `StreamNamePrefix` (*string) is the Cloudwatch Log Stream Name prefix

1. `FilterPattern` (string) is based on the Cloudwatch Filter Pattern syntax
    Reference: (https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html)

1. `StartTimePadding` (*int64) defines the time in seconds to subtract from now

1. `PollInterval` (*int64) defines the Cloudwatch log poll time interval in seconds

