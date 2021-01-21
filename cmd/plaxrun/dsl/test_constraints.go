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

// TestConstraints used to constrain the tests run
type TestConstraints struct {
	Labels   []string     `yaml:"labels"`
	Priority *int         `yaml:"priority,omitempty"`
	Retry    int          `yaml:"retry"`
	Seed     int64        `yaml:"seed"`
	Iterate  *TestIterate `yaml:"iterate,omitempty"`
}
