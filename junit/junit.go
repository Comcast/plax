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

// Package junit provides simplified structures for JUnit test
// reports.
package junit

// https://llg.cubic.org/docs/junit/

import (
	"time"
)

// TestCaseStatus represents the status of the test case
type TestCaseStatus string

const (
	Passed  TestCaseStatus = "passed"
	Failed  TestCaseStatus = "failed"
	Error   TestCaseStatus = "error"
	Skipped TestCaseStatus = "skipped"
)

// TestCase information
type TestCase struct {
	Name    string         `xml:"name,attr" json:"name"`
	File    string         `xml:"file,attr" json:"file"`
	Status  TestCaseStatus `xml:"status,attr" json:"status"`
	Time    *time.Duration `xml:"time,attr,omitempty" json:"time,omitempty"`
	Started *time.Time     `xml:"started,attr,omitempty" json:"started,omitempty"`
	Message string         `xml:"message,omitempty" json:"message,omitempty"`
}

// NewTestCase creates a new TestCase
func NewTestCase(name string) *TestCase {
	now := time.Now().UTC()
	return &TestCase{
		Name:    name,
		Started: &now,
	}
}

// Finish the TestCase
func (tc *TestCase) Finish(status TestCaseStatus, message ...string) {
	tc.Status = status

	if status != Skipped {
		now := time.Now().UTC()
		time := now.Sub(*tc.Started)
		tc.Time = &time
	} else {
		tc.Started = nil
		tc.Time = nil
	}

	if len(message) == 1 {
		tc.Message = message[0]
	}
}

// TestSuite information
type TestSuite struct {
	Name     string        `xml:"name,attr" json:"name"`
	Total    int           `xml:"tests,attr" json:"tests"`
	Passed   int           `xml:"passed,attr" json:"passed"`
	Skipped  int           `xml:"skipped,attr" json:"skipped"`
	Failures int           `xml:"failures,attr" json:"failures"`
	Errors   int           `xml:"errors,attr" json:"errors"`
	TestCase []TestCase    `xml:"testcase" json:"testcase"`
	Started  time.Time     `xml:"started,attr" json:"timestamp"`
	Time     time.Duration `xml:"time,attr" json:"time"`
	Message  string        `xml:"message,omitempty" json:"message,omitempty"`
}

// NewTestSuite creates a new TestSuite
func NewTestSuite(name string) *TestSuite {
	now := time.Now().UTC()
	return &TestSuite{
		Name:     name,
		TestCase: make([]TestCase, 0, 32),
		Started:  now,
	}
}

// Add a TestCase to the TestSuite
func (ts *TestSuite) Add(tc TestCase) {
	ts.TestCase = append(ts.TestCase, tc)
	ts.Total++
	switch tc.Status {
	case Failed:
		ts.Failures++
	case Error:
		ts.Errors++
	case Skipped:
		ts.Skipped++
	case Passed:
		ts.Passed++
	}
}

// Finish the TestSuite
func (ts *TestSuite) Finish(message ...string) {
	now := time.Now().UTC()
	time := now.Sub(ts.Started)
	ts.Time = time
	if len(message) == 1 {
		ts.Message = message[0]
	}
}
