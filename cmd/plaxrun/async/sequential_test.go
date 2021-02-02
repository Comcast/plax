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

func TestSequential(t *testing.T) {
	// Output will be array of results or an error
	results, err := Sequential(context.Background(), testFuncTasks...)

	if err != nil {
		t.Error(err)
	}

	fmt.Printf("Results: %v", results)
}

func TestSequentialWithBadFuncNumberReturnValues(t *testing.T) {
	var testFuncTasks = []*TaskFunc{}
	testFuncs := append(testFuncTasks, &badTaskFunc1)

	// Output will be array of results or an error
	_, err := Sequential(context.Background(), testFuncs...)

	if err == nil {
		t.Errorf("expected bad function")
	}
}

func TestSequentialWithBadFuncLastReturnValueNotError(t *testing.T) {
	var testFuncTasks = []*TaskFunc{}
	testFuncs := append(testFuncTasks, &badTaskFunc2)

	// Output will be array of results or an error
	_, err := Sequential(context.Background(), testFuncs...)

	if err == nil {
		t.Errorf("expected bad function")
	}
}

func TestSequentialWithNilTaskFunc(t *testing.T) {
	var testFuncTasks = []*TaskFunc{}
	testFuncs := append(testFuncTasks, nil)

	// Output will be array of results or an error
	_, err := Sequential(context.Background(), testFuncs...)

	if err == nil {
		t.Errorf("expected bad function")
	}
}

func TestSequentialWithNilFunc(t *testing.T) {
	var testFuncTasks = []*TaskFunc{}
	testFuncs := append(testFuncTasks, &badTaskFunc3)

	// Output will be array of results or an error
	_, err := Sequential(context.Background(), testFuncs...)

	if err == nil {
		t.Errorf("expected bad function")
	}
}
func TestSequentialWithTimeout(t *testing.T) {
	results, err := SequentialWithTimeout(context.Background(), 3*time.Second, testFuncTasks...)

	if err != nil {
		t.Logf("Error: %v\n", err)
	}

	if len(results) != 5 {
		t.Error(fmt.Errorf("Error in results: %v", results))
	}
}
