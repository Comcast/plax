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
	"fmt"
	"io/ioutil"
	"os"
)

// FindInclude searches the include directories for the file
func FindInclude(ctx *Ctx, filename string) ([]byte, error) {
	dirs := ctx.IncludeDirs
	if len(dirs) == 0 {
		// ToDo: To dangerous?
		dirs = []string{"."}
	}

	for _, dir := range dirs {
		path := dir + "/" + filename
		bs, err := ioutil.ReadFile(path)
		if err != nil {
			if err == os.ErrNotExist {
				continue
			}
			if _, is := err.(*os.PathError); is {
				continue
			}
			return nil, err
		}

		ctx.Logf("YAML including %s", path) // ToDo: Logdf
		return bs, nil
	}

	return nil, &os.PathError{
		Op:   "find",
		Path: filename,
		Err:  fmt.Errorf("%s: %v", os.ErrNotExist, dirs),
	}
}
