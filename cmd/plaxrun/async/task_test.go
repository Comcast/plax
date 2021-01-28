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
	"fmt"
	"testing"
)

func TestTaskResults_HasError(t *testing.T) {
	tests := []struct {
		name string
		trs  TaskResults
		want bool
	}{
		{
			name: "No errors",
			trs: []TaskResult{
				{
					index:  0,
					Name:   "First",
					Done:   true,
					Error:  nil,
					Result: nil,
				},
			},
			want: false,
		},
		{
			name: "With error",
			trs: []TaskResult{
				{
					index:  0,
					Name:   "First",
					Done:   true,
					Error:  nil,
					Result: nil,
				},
				{
					index:  1,
					Name:   "Second",
					Done:   true,
					Error:  fmt.Errorf("Error"),
					Result: nil,
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.trs.HasError(); got != tt.want {
				t.Errorf("TaskResults.HasError() = %v, want %v", got, tt.want)
			}
		})
	}
}
