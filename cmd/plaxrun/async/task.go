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
	"reflect"
	"strings"
)

// TaskFunc is a function that returns nothing (empty), an error, or a result and error
type TaskFunc struct {
	Func func() error
	Name string
}

// TaskResult has the task Error or Result
type TaskResult struct {
	index  int
	Name   string
	Done   bool
	Error  error
	Result interface{}
}

type TaskResults []TaskResult

func (trs TaskResults) HasError() bool {
	for _, tr := range trs {
		if tr.Error != nil {
			return true
		}
	}

	return false
}

func (trs TaskResults) Error() string {
	var sb strings.Builder

	for _, tr := range trs {
		if tr.Error != nil {
			sb.WriteString(fmt.Sprintf("Task %s at index %d error: %s\n", tr.Name, tr.index, tr.Error.Error()))
		}
	}

	return sb.String()
}

const (
	// TaskReturnEmptyResult indicates an empty function return
	TaskReturnEmptyResult = iota
	// TaskReturnErrorOnly indicates an function that returns an error only
	TaskReturnErrorOnly
	// TaskReturnResultOnly indicatea a function that return a interface{} result only
	TaskReturnResultOnly
	// TaskReturnResultAndError indicates a function that return both an interface{} and error result
	TaskReturnResultAndError
)

// Task structure contains task function reflection information, position in results, and the return type
type Task struct {
	index      int
	name       string
	typeOf     reflect.Type
	valueOf    reflect.Value
	returnType int
}

// NewTask creates a new Task
func NewTask(index int, tf *TaskFunc) (*Task, error) {
	if tf == nil {
		return nil, fmt.Errorf("task function is nil")
	}

	errorType := reflect.TypeOf((*error)(nil)).Elem()
	typeOf := reflect.TypeOf(tf.Func)
	valueOf := reflect.ValueOf(tf.Func)
	returnType := TaskReturnEmptyResult

	switch n := typeOf.NumOut(); {
	case n == 1:
		if typeOf.Out(0).Implements(errorType) {
			returnType = TaskReturnErrorOnly
		} else {
			returnType = TaskReturnResultOnly
		}
	case n == 2:
		if typeOf.Out(1).Implements(errorType) {
			returnType = TaskReturnResultAndError
		} else {
			return nil, fmt.Errorf("second function result must be an error")
		}
	case n > 2:
		return nil, fmt.Errorf("invalid number of function results (%d)", n)
	}

	return &Task{
		index:      index,
		name:       tf.Name,
		typeOf:     typeOf,
		valueOf:    valueOf,
		returnType: returnType,
	}, nil
}

// invoke calls the function of the task and sends its
// result to the res channel.  This work is performed in a new goroutine.
func (t *Task) invoke(ctx context.Context, resch chan TaskResult) {
	result := &TaskResult{
		index: t.index,
		Name:  t.name,
		Done:  true,
	}
	params := []reflect.Value{}

	go func() {
		res := t.valueOf.Call(params)
		switch t.returnType {
		case TaskReturnErrorOnly:
			if v := res[0].Interface(); v != nil {
				err, ok := v.(error)
				if !ok {
					result.Error = fmt.Errorf("value is not an error")
				}
				result.Error = err
			}
		case TaskReturnResultOnly:
			result.Result = res[0].Interface()
		case TaskReturnResultAndError:
			result.Result = res[0].Interface()
			if v := res[1].Interface(); v != nil {
				err, ok := v.(error)
				if !ok {
					result.Error = fmt.Errorf("value is not an error")
				}
				result.Error = err
			}
		default:
			result.Error = fmt.Errorf("invalid function return type (%d)", t.returnType)
		}

		select {
		case <-ctx.Done():
		case resch <- *result:
		}
	}()
}

// call calls the function of the task and sends its
// result to the res channel.  This work is performed syncronously
func (t *Task) call(ctx context.Context) TaskResult {
	result := TaskResult{
		index: t.index,
		Name:  t.name,
		Done:  true,
	}
	params := []reflect.Value{}

	res := t.valueOf.Call(params)
	switch t.returnType {
	case TaskReturnErrorOnly:
		if v := res[0].Interface(); v != nil {
			err, ok := v.(error)
			if !ok {
				result.Error = fmt.Errorf("value is not an error")
			}
			result.Error = err
		}
	case TaskReturnResultOnly:
		result.Result = res[0].Interface()
	case TaskReturnResultAndError:
		result.Result = res[0].Interface()
		if v := res[1].Interface(); v != nil {
			err, ok := v.(error)
			if !ok {
				result.Error = fmt.Errorf("value is not an error")
			}
			result.Error = err
		}
	default:
		result.Error = fmt.Errorf("invalid function return type (%d)", t.returnType)
	}

	return result
}
