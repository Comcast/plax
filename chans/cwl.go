package chans

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"

	"github.com/Comcast/plax/dsl"
)

const (
	streamNameFormat        = "%s-%s"
	timeDateFormat          = "2006-01-02T150405Z0700"
	defaultStartTimePadding = 10 * time.Second
	defaultPollInterval     = 1 * time.Second
)

func init() {
	dsl.TheChanRegistry.Register(dsl.NewCtx(nil), "cwl", NewCWLChan)
}

// CWLOpts is the Cloudwatch Logs Options
type CWLOpts struct {
	_      struct{} `type:"structure"`
	Region *string  `type:"string" json:"region,omitempty" yaml:",omitempty"`
	// GroupName for the Cloudwatch Log Group
	GroupName string `type:"string" json:"groupName,omitempty" yaml:",omitempty"`
	// StreamNamePrefix is the Cloudwatch Log Stream Name prefix
	StreamNamePrefix *string `type:"string" json:",omitempty" yaml:",omitempty"`
	// FilterPattern is based on the Cloudwatch filter pattern syntax
	// Reference: (https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html)
	FilterPattern string `type:"string" json:",omitempty" yaml:",omitempty"`
	// StartTimePadding defines the time in seconds to substract from now
	StartTimePadding *int64 `type:"number" json:",omitempty" yaml:",omitempty"`
	// PollInterval defines the Cloudwatch logs poll time interval in seconds
	PollInterval *int64 `type:"number" json:",omitempty" yaml:",omitempty"`
}

// String returns the string representation of the CWLOpts
func (opts CWLOpts) String() string {
	return awsutil.Prettify(opts)
}

// CWLChan is the Cloudwatch Logs Channel
type CWLChan struct {
	c            chan dsl.Msg
	ctl          chan bool
	client       cloudwatchlogsiface.CloudWatchLogsAPI
	streamName   *string
	startTime    time.Time
	pollInterval time.Duration

	opts *CWLOpts
}

// makeNowTimestamp creates a Unix Epoch timestamp
func makeNowTimestamp() int64 {
	return time.Now().UTC().UnixNano() / int64(time.Millisecond/time.Nanosecond)
}

// NewCWLChan create a new Cloudwatch Log Channel (cwl)
func NewCWLChan(ctx *dsl.Ctx, o interface{}) (dsl.Chan, error) {
	js, err := json.Marshal(&o)
	if err != nil {
		return nil, dsl.NewBroken(err)
	}

	opts := CWLOpts{}

	if err = json.Unmarshal(js, &opts); err != nil {
		return nil, dsl.NewBroken(err)
	}

	var region string
	if opts.Region != nil {
		region = *opts.Region
	} else {
		region = os.Getenv("AWS_DEFAULT_REGION")
		if region == "" {
			err := fmt.Errorf("AWS_DEFAULT_REGION not set")
			ctx.Warnf("NewCWLChan warning: %v", err)
			return nil, err
		}
	}

	var streamName *string = nil

	if opts.StreamNamePrefix != nil {
		streamName = aws.String(fmt.Sprintf(streamNameFormat, *opts.StreamNamePrefix, time.Now().UTC().Format(timeDateFormat)))
	}

	nowTime := time.Now().UTC()
	ctx.Logf("Now Time: %v", nowTime)

	startTimePadding := -defaultStartTimePadding

	if opts.StartTimePadding != nil {
		startTimePadding = -time.Duration(*opts.StartTimePadding) * time.Second
	}

	startTime := nowTime.Add(startTimePadding)
	pollInterval := defaultPollInterval

	if opts.PollInterval != nil {
		pollInterval = time.Duration(*opts.PollInterval) * time.Second
	}

	ctx.Logf("Start Time: %v", startTime)

	mySession := session.Must(session.NewSession())

	// Create a CloudWatchLogs client with additional configuration
	cloudwatchlogs := cloudwatchlogs.New(mySession, aws.NewConfig().WithRegion(region))
	return &CWLChan{
		c:            make(chan dsl.Msg, 1024),
		ctl:          make(chan bool),
		opts:         &opts,
		streamName:   streamName,
		client:       cloudwatchlogs,
		startTime:    startTime,
		pollInterval: pollInterval,
	}, nil
}

// Kind returns the Cloudwatch Log Channel kind
func (c *CWLChan) Kind() dsl.ChanKind {
	return "cwl"
}

// Open the Cloudwatch Log Channel
func (c *CWLChan) Open(ctx *dsl.Ctx) error {
	ctx.Logf("CWLChan.Open(%+v)", *c.opts)

	go c.Consume(ctx)

	return nil
}

// Close the Cloudwatch Log Channel
func (c *CWLChan) Close(ctx *dsl.Ctx) error {
	return nil
}

// Sub on the Cloudwatch Log Channel
func (c *CWLChan) Sub(ctx *dsl.Ctx, topic string) error {
	return dsl.Brokenf("Can't Sub on a CWL (%+v)", *c.opts)
}

// Pub on the Cloudwatch Log Channel
func (c *CWLChan) Pub(ctx *dsl.Ctx, m dsl.Msg) error {
	ctx.Logf("info: CWLChan.Pub(%+v)", *c.opts)

	if c.streamName == nil || c.opts.StreamNamePrefix == nil {
		err := fmt.Errorf("StreamNamePrefix must be provided")
		ctx.Warnf(err.Error())
		return err
	}

	js, err := dsl.MaybeSerialize(m.Payload)
	if err != nil {
		return nil
	}

	var seqToken *string = nil

	err = c.client.DescribeLogStreamsPages(
		&cloudwatchlogs.DescribeLogStreamsInput{
			LogGroupName:        aws.String(c.opts.GroupName),
			LogStreamNamePrefix: c.opts.StreamNamePrefix,
		},
		func(output *cloudwatchlogs.DescribeLogStreamsOutput, lastPage bool) bool {
			for _, stream := range output.LogStreams {
				if *c.streamName == *stream.LogStreamName {
					seqToken = stream.UploadSequenceToken
					return true
				}
			}
			_, err := c.client.CreateLogStream(
				&cloudwatchlogs.CreateLogStreamInput{
					LogGroupName:  aws.String(c.opts.GroupName),
					LogStreamName: c.streamName,
				},
			)
			if err != nil {
				ctx.Logf(err.Error())
			}
			return false
		})
	if err != nil {
		return err
	}

	event := cloudwatchlogs.InputLogEvent{
		Message:   &js,
		Timestamp: aws.Int64(makeNowTimestamp()),
	}
	events := []*cloudwatchlogs.InputLogEvent{
		&event,
	}
	input := cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(c.opts.GroupName),
		LogStreamName: c.streamName,
		LogEvents:     events,
		SequenceToken: seqToken,
	}

	_, err = c.client.PutLogEvents(&input)
	if err != nil {
		return err
	}

	return nil
}

// Recv on the Cloudwatch Log Channel
func (c *CWLChan) Recv(ctx *dsl.Ctx) chan dsl.Msg {
	ctx.Logf("info: CWLChan.Recv(%+v)", *c.opts)
	return c.c
}

// Kill the Cloudwatch Log Channel
func (c *CWLChan) Kill(ctx *dsl.Ctx) error {
	return fmt.Errorf("error: CWLChan.Kill is not supported by a %T", c)
}

// To channel
func (c *CWLChan) To(ctx *dsl.Ctx, m dsl.Msg) error {
	ctx.Logf("info: CWLChan.To(%+v)", *c.opts)
	select {
	case <-ctx.Done():
	case c.c <- m:
	}
	return nil
}

// Consume on the Cloudwatch Log Channel
func (c *CWLChan) Consume(ctx *dsl.Ctx) {
	ctx.Logf("info: CWLChan.Consume(%+v)", *c.opts)

	var (
		nextToken *string = nil
	)

LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case <-c.ctl:
			break LOOP
		default:
		}

		startTimeMilliseconds := c.startTime.UTC().UnixNano() / int64(time.Millisecond/time.Nanosecond)

		input := &cloudwatchlogs.FilterLogEventsInput{
			LogGroupName:  &c.opts.GroupName,
			StartTime:     aws.Int64(startTimeMilliseconds),
			FilterPattern: &c.opts.FilterPattern,
			NextToken:     nextToken,
		}

		ctx.Logdf("debug: FilterLogsEventsInput: %v", input)

		err := c.client.FilterLogEventsPages(
			input,
			func(output *cloudwatchlogs.FilterLogEventsOutput, lastPage bool) bool {
				ctx.Logdf("debug: events: %v", output)
				timestamp := time.Now().UTC()

				for _, event := range output.Events {
					if event.Timestamp != nil {
						timestamp = time.Unix(*event.Timestamp, 0)
					}
					m := dsl.Msg{
						Topic:      c.opts.GroupName,
						Payload:    *event.Message,
						ReceivedAt: timestamp,
					}

					err := c.To(ctx, m)
					if err != nil {
						ctx.Warnf("warn: CWLChan.Consume %s", err)
						return false
					}
				}

				if len(output.Events) > 0 {
					lastSeenTimestamp := output.Events[len(output.Events)-1].Timestamp
					if lastSeenTimestamp != nil {
						lastSeenTime := time.Unix(0, *lastSeenTimestamp*int64(time.Millisecond))
						c.startTime = lastSeenTime.Add(time.Millisecond)
					}
				}

				nextToken = output.NextToken

				return true
			},
		)

		if err != nil {
			ctx.Warnf("warn: CWLChan.Consume %s", err)
			break
		}

		ctx.Logdf("debug: waiting %d second(s)...", c.pollInterval/time.Second)

		time.Sleep(c.pollInterval)
	}
}
