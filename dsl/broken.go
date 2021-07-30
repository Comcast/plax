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

package dsl

import "fmt"

// Broken is an error that represents a test that is broken (and not
// simpling failing).
type Broken struct {
	// ToDo: Consider using an interface.

	Err error
}

// Brokenf makes a nice Broken for you.
func Brokenf(format string, args ...interface{}) *Broken {
	return NewBroken(fmt.Errorf(format, args...))
}

// NewBroken does exactly what you'd expect.
func NewBroken(err error) *Broken {
	return &Broken{
		Err: err,
	}
}

// IsBroken reports whether the given error is a *Broken.  If it is,
// returns it.
//
// This function knows about Errors.
func IsBroken(err error) (*Broken, bool) {
	if err == nil {
		return nil, false
	}

	if errs, is := err.(*Errors); is {
		return errs.IsBroken()
	}

	b, is := err.(*Broken)
	return b, is
}

// Error makes *Broken into an error.
func (b *Broken) Error() string {
	return fmt.Sprintf("Broken: %s", b.Err)
}

func (b *Broken) String() string {
	return b.Error()
}

// Failure is an error that does not represent something that's
// broken.
type Failure struct {
	Err error
}

func (f *Failure) Error() string {
	return string("failure: " + f.Err.Error())
}

func IsFailure(x interface{}) (*Failure, bool) {
	f, is := x.(*Failure)
	return f, is
}

func Failuref(format string, args ...interface{}) *Failure {
	return &Failure{
		Err: fmt.Errorf(format, args...),
	}
}
