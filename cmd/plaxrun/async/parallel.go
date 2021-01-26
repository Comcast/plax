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
	"sync"
	"time"
)

// Parallel calls the set of tasks in parallel using the given context
func Parallel(ctx context.Context, tfs ...*TaskFunc) ([]TaskResult, error) {
	tasks := make([]*Task, len(tfs))
	taskResults := make([]TaskResult, len(tfs))
	resch := make(chan TaskResult)
	count := 0

	for index, tf := range tfs {
		if tf != nil {
			task, err := NewTask(index, tf)
			if err != nil {
				return nil, err
			}

			tasks[index] = task
		}
	}

	for _, task := range tasks {
		if task != nil {
			count++
			task.invoke(ctx, resch)
		}
	}

	for c := 0; c < count; c++ {
		select {
		case <-ctx.Done():
			err := fmt.Errorf("context canceled")
			// Set error on incomplete tasks
			//taskResults[res.index] = err
			return taskResults, err
		case res := <-resch:
			taskResults[res.index] = res
		}
	}

	return taskResults, nil
}

// ParallelWithTimeout calls the set of tasks in parallel using the given context with timeout
func ParallelWithTimeout(ctx context.Context, timeout time.Duration, tfs ...*TaskFunc) (TaskResults, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var (
		res []TaskResult
		err error
		wg  sync.WaitGroup
	)

	wg.Add(1)

	go func() {
		res, err = Parallel(ctx, tfs...)

		wg.Done()
	}()

	wg.Wait()

	return res, err
}
