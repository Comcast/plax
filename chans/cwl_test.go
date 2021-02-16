/*
 * Copyright 2021 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package chans

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Comcast/plax/dsl"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
)

// DescribeLogStreamsPagesResults mocks the return results of cloudwatchlogs.DescribeLogStreamsPages
type DescribeLogStreamsPagesResults struct {
	Output   *cloudwatchlogs.DescribeLogStreamsOutput
	LastPage bool
	Err      error
}

// CreateLogStreamResults mocks the return results of cloudwatchlogs.CreateLogStream
type CreateLogStreamResults struct {
	Output *cloudwatchlogs.CreateLogStreamOutput
	Err    error
}

// PutLogEventsResults mocks the return results of cloudwatchlogs.PutLogEvents
type PutLogEventsResults struct {
	Chan chan string
	Err  error
}

// FilterLogEventsPagesResults mocks the return result of cloudwatchlogs.FilterLogEventsPages
type FilterLogEventsPagesResults struct {
	Chan     chan string
	LastPage bool
	Err      error
}

// CWLAPIResults mocks the AWS cloudwatchlogs API with an err or alternate func results
type CWLAPIResults struct {
	DescribeLogStreamsPages DescribeLogStreamsPagesResults
	CreateLogStream         CreateLogStreamResults
	PutLogEvents            PutLogEventsResults
	FilterLogEventsPages    FilterLogEventsPagesResults
}

// CWLAPI mocks the AWS cloudwatchlogs API interface
type CWLAPI struct {
	cloudwatchlogsiface.CloudWatchLogsAPI
	Results CWLAPIResults
}

// DescribeLogStreamsPages mock method
func (cwlAPI *CWLAPI) DescribeLogStreamsPages(input *cloudwatchlogs.DescribeLogStreamsInput, cb func(*cloudwatchlogs.DescribeLogStreamsOutput, bool) bool) error {
	if cwlAPI.Results.DescribeLogStreamsPages.Err != nil {
		return cwlAPI.Results.DescribeLogStreamsPages.Err
	}

	if cwlAPI.Results.DescribeLogStreamsPages.Output != nil {
		cb(cwlAPI.Results.DescribeLogStreamsPages.Output, cwlAPI.Results.DescribeLogStreamsPages.LastPage)
		return nil
	}

	cb(&cloudwatchlogs.DescribeLogStreamsOutput{}, true)

	return nil
}

// CreateLogStream mock method
func (cwlAPI *CWLAPI) CreateLogStream(input *cloudwatchlogs.CreateLogStreamInput) (*cloudwatchlogs.CreateLogStreamOutput, error) {
	if cwlAPI.Results.CreateLogStream.Err != nil {
		return nil, cwlAPI.Results.CreateLogStream.Err
	}

	if cwlAPI.Results.CreateLogStream.Output != nil {
		return cwlAPI.Results.CreateLogStream.Output, nil
	}

	return &cloudwatchlogs.CreateLogStreamOutput{}, nil
}

// PutLogEvents mock method
func (cwlAPI *CWLAPI) PutLogEvents(input *cloudwatchlogs.PutLogEventsInput) (*cloudwatchlogs.PutLogEventsOutput, error) {
	if cwlAPI.Results.PutLogEvents.Err != nil {
		return nil, cwlAPI.Results.PutLogEvents.Err
	}

	for _, inputLogEvent := range input.LogEvents {
		cwlAPI.Results.PutLogEvents.Chan <- *inputLogEvent.Message
	}

	return &cloudwatchlogs.PutLogEventsOutput{}, nil
}

// FilterLogEventsPages mock method
func (cwlAPI *CWLAPI) FilterLogEventsPages(input *cloudwatchlogs.FilterLogEventsInput, cb func(*cloudwatchlogs.FilterLogEventsOutput, bool) bool) error {
	if cwlAPI.Results.FilterLogEventsPages.Err != nil {
		return cwlAPI.Results.FilterLogEventsPages.Err
	}

	filteredLogEvents := make([]*cloudwatchlogs.FilteredLogEvent, 0)

	for message := range cwlAPI.Results.FilterLogEventsPages.Chan {
		filteredLogEvents = append(
			filteredLogEvents,
			&cloudwatchlogs.FilteredLogEvent{
				Message: aws.String(message),
			},
		)

		break
	}

	cb(
		&cloudwatchlogs.FilterLogEventsOutput{
			Events: filteredLogEvents,
		},
		true,
	)

	return nil
}

// NewCWLAPI mocks the AWS cloudwatchlogs API interface
func NewCWLAPI(results CWLAPIResults) *CWLAPI {
	cwlAPI := CWLAPI{
		Results: results,
	}
	return &cwlAPI
}

// TestCWL Require the AWS_DEFAULT_REGION and AWS_PROFILE environment variables set and non expired AWS credentials
func TestCWL(t *testing.T) {
	opts := CWLOpts{
		Region:           aws.String("us-west-2"),
		GroupName:        "plax",
		StreamNamePrefix: aws.String("test"),
		FilterPattern:    "",
		StartTimePadding: aws.Int64(15),
		PollInterval:     aws.Int64(1),
	}

	ctx := dsl.NewCtx(context.Background())

	c, err := NewCWLChan(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}

	cwlChan, ok := c.(*CWLChan)
	if !ok {
		t.Errorf("Not a CWL Channel")
	}

	mockChan := make(chan string)

	cwlChan.client = NewCWLAPI(CWLAPIResults{
		DescribeLogStreamsPages: DescribeLogStreamsPagesResults{
			Output: &cloudwatchlogs.DescribeLogStreamsOutput{
				LogStreams: []*cloudwatchlogs.LogStream{
					{
						LogStreamName: aws.String("mock"),
					},
				},
			},
		},
		PutLogEvents: PutLogEventsResults{
			Chan: mockChan,
		},
		FilterLogEventsPages: FilterLogEventsPagesResults{
			Chan: mockChan,
		},
	})

	skip := func(err error) {
		t.Skipf("skipping CWL test (%s)", err)
	}

	if err = c.Open(ctx); err != nil {
		skip(err)
	}

	defer c.Close(ctx)

	loglevelWant := "ERROR"
	foodWant := "tacos"

	{
		o := map[string]interface{}{
			"logLevel": loglevelWant,
			"want":     foodWant,
		}

		js, err := json.Marshal(&o)
		if err != nil {
			t.Fatal(err)
		}

		m := dsl.Msg{
			Payload: string(js),
		}

		if err = c.Pub(ctx, m); err != nil {
			skip(err)
		}
	}

	{
		var (
			ch      = c.Recv(ctx)
			msg     = <-ch
			payload = msg.Payload
			parsed  map[string]interface{}
		)

		fmt.Printf("recv\n%s\n", payload)

		if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
			t.Fatal(err)
		}

		loglevel, have := parsed["logLevel"]
		if !have {
			t.Fatal("no 'logLevel' in payload")
		}

		if loglevel != loglevelWant {
			t.Fatalf("%v != %v", loglevel, loglevelWant)
		}

		want, have := parsed["want"]
		if !have {
			t.Fatal("no 'want' in payload")
		}

		if want != foodWant {
			t.Fatalf("%v != %v", want, foodWant)
		}
	}
}
