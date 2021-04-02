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

package subst

import (
	"context"
	"log"
)

// Ctx mostly provides a list of directories a Subber will search to
// find files.
//
// Instead of using a context.Context-like struct, we could have
// IncludeDirs as a field in a Subber.  However, the current approach
// feels slightly more natural to use if still a little embarrassing.
type Ctx struct {
	context.Context
	IncludeDirs []string
	Tracing     bool
}

// NewCtx makes a new Ctx with (a copy of) the given IncludeDirs.
func NewCtx(ctx context.Context, dirs []string) *Ctx {
	acc := make([]string, len(dirs))
	copy(acc, dirs)
	return &Ctx{
		Context:     ctx,
		IncludeDirs: acc,
	}
}

// Copy makes a deep copy of the Ctx.
func (c *Ctx) Copy() *Ctx {
	dirs := make([]string, len(c.IncludeDirs))
	copy(dirs, c.IncludeDirs)
	return &Ctx{
		Context:     c.Context,
		IncludeDirs: dirs,
		Tracing:     c.Tracing,
	}
}

// trf is a log.Printf switched by Ctx.Tracing.
func (c *Ctx) trf(format string, args ...interface{}) {
	if c != nil && !c.Tracing {
		return
	}
	log.Printf(format, args...)
}
