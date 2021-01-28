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
	"sync"
	"time"
)

// Sequential calls the set of tasks in sequential order using the given context
func Sequential(ctx context.Context, tfs ...*TaskFunc) (TaskResults, error) {
	tasks := make([]*Task, len(tfs))
	taskResults := make([]TaskResult, len(tfs))
	count := 0

	for index, tf := range tfs {
		if tf.Func != nil {
			task, err := NewTask(index, tf)
			if err != nil {
				return nil, err
			}

			tasks[index] = task
		}
	}

	for i, task := range tasks {
		if task != nil {
			count++
			taskResults[i] = task.call(ctx)
		}
	}

	return taskResults, nil
}

// SequentialWithTimeout calls the set of tasks in sequential order using the given context with timeout
func SequentialWithTimeout(ctx context.Context, timeout time.Duration, tfs ...*TaskFunc) ([]TaskResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var (
		res []TaskResult
		err error
		wg  sync.WaitGroup
	)

	wg.Add(1)

	go func() {
		res, err = Sequential(ctx, tfs...)

		wg.Done()
	}()

	wg.Wait()

	return res, err
}
