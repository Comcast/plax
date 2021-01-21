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
package async

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func sleepAndSay(msg string) string {
	time.Sleep(2 * time.Second)
	fmt.Printf("sleepAndSay: %s\n", msg)
	return msg
}

func sleepAndError(err error) error {
	time.Sleep(1 * time.Second)
	fmt.Printf("sleepAndError: %v\n", err)
	return err
}

func sleepAndSayWithError(msg string, err error) (string, error) {
	time.Sleep(1 * time.Second)
	fmt.Printf("sleepAndSayWithError: %s, %v\n", msg, err)
	return msg, err
}

func sleep() {
	fmt.Printf("sleeping...\n")
	time.Sleep(5 * time.Second)
	fmt.Printf("sleep over\n")
}

func badFunc() (string, string, error) {
	fmt.Printf("badFunc:\n")
	return "", "", nil
}

var (
	testFuncs = []TaskFunc{
		{
			Name: "sleepAndSay", 
			Func: func() string { 
				return sleepAndSay("I like tacos!")
			}
		},
		{
			Name: "sleepAndError", 
			Func: func() error { 
				return sleepAndError(fmt.Errorf("tacos are bad"))
			},
		},
		{
			Name: "sleepAndSayWithError",
			Func: func() (string, error) {
				return sleepAndSayWithError("I don't like tacos!", fmt.Errorf("tacos are bad"))
			},
		},
		{
			Name: "sleepAndSayWithProvidedError",
			Func: func() (string, error) { 
				return sleepAndSayWithError("I like tacos!", nil)
			},
		},
		{
			Name: "sleep",
			Func: func() { 
				sleep()
			}
		},
	},
	}
)

func TestParallel(t *testing.T) {
	// Output will be array of results or an error
	// results ,err := Parallel(context.Background(),
	// 	func() string { return sleepAndSay("I like tacos!") },
	// 	func() error { return sleepAndError(fmt.Errorf("tacos are bad")) },
	// 	func() (string, error) {
	// 		return sleepAndSayWithError("I don't like tacos!", fmt.Errorf("tacos are bad"))
	// 	},
	// 	func() (string, error) { return sleepAndSayWithError("I like tacos!", nil) },
	// 	func() { sleep() })
	results, err := Parallel(context.Background(), testFuncs...)

	if err != nil {
		t.Error(err)
	}

	fmt.Printf("Results: %v", results)
}

func TestParallelWithBadFunc(t *testing.T) {
	testFuncs := append(testFuncs, badFunc)

	// Output will be array of results or an error
	_, err := Parallel(context.Background(), testFuncs...)

	if err == nil {
		t.Errorf("expected bad function")
	}
}

func TestParallelWithTimeout(t *testing.T) {
	results, err := ParallelWithTimeout(context.Background(), 3*time.Second, testFuncs...)

	if err != nil {
		t.Logf("Error: %v\n", err)
	}

	if len(results) != 5 {
		t.Error(fmt.Errorf("Error in results: %v", results))
	}
}
