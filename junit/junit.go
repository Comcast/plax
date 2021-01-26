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

type Skipped struct {
	Message string `xml:"message,attr"`
}

type Error struct {
	Message     string `xml:"message,attr"`
	Type        string `xml:"type,attr"`
	Description string `xml:"description,omitempty"`
}

type Failure struct {
	Message     string `xml:"message,attr"`
	Type        string `xml:"type,attr"`
	Description string `xml:"description,omitempty"`
}

type TestCase struct {
	Name    string   `xml:"name,attr"`
	Status  string   `xml:"status,attr"`
	Time    int64    `xml:"time,attr" json:"-"`
	Skipped *Skipped `xml:"skipped,omitempty"`
	Error   *Error   `xml:"error,omitempty"`
	Failure *Failure `xml:"failure,omitempty"`

	Timestamp time.Time `xml:"-"`
	Suite     string    `xml:"-"`
	N         int       `xml:"-"`
	Type      string    `xml:"-"`

	// State is often test.State, which can be used to emit
	// computed values like elapsed time.
	//
	// This value isn't XML-serialized.
	State interface{} `xml:"-"`

	started time.Time
}

func NewTestCase(name string) *TestCase {
	now := time.Now().UTC()
	return &TestCase{
		Name:      name,
		Timestamp: now,
		started:   now,
	}
}

func (tc *TestCase) Finish(status string) {
	elapsed := time.Now().Sub(tc.started)
	tc.Time = int64(elapsed) / 1000 / 1000 / 1000
	tc.Status = status
}

type TestSuite struct {
	Name      string     `xml:"testsuite,attr"`
	Tests     int        `xml:"tests,attr"`
	Failures  int        `xml:"failures,attr"`
	Errors    int        `xml:"errors,attr"`
	TestCases []TestCase `xml:"testcase"`

	Time time.Time `xml:"-"`
}

func NewTestSuite() *TestSuite {
	return &TestSuite{
		TestCases: make([]TestCase, 0, 32),
		Time:      time.Now().UTC(),
	}
}

func (ts *TestSuite) Add(tc TestCase) {
	ts.TestCases = append(ts.TestCases, tc)
	ts.Tests++
	if tc.Failure != nil {
		ts.Failures++
	}
	if tc.Error != nil {
		ts.Errors++
	}
}
