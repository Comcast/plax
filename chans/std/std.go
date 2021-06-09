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

// Package std exists just to collect _ imports for the standard
// channel types implemented in this repo.
//
// Example use: The plax executable imports this package, so the init
// functions off packages imported here are run.  As a result, the
// plax executable gets all the standard channel types registered.
package std

import (
	_ "github.com/Comcast/plax/chans"
	_ "github.com/Comcast/plax/chans/cwl"
	_ "github.com/Comcast/plax/chans/httpclient"
	_ "github.com/Comcast/plax/chans/httpserver"
	_ "github.com/Comcast/plax/chans/kds"
	_ "github.com/Comcast/plax/chans/mqtt"
	_ "github.com/Comcast/plax/chans/shell"
	_ "github.com/Comcast/plax/chans/sqlc"
	_ "github.com/Comcast/plax/chans/sqs"
)
