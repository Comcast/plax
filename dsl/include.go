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

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
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

// ReadIncluded is a utility function that's convenient for Include().
func ReadIncluded(ctx *Ctx, filename string) (interface{}, error) {
	// ToDo: Reconsider the following line.
	bs, err := FindInclude(ctx, filename)
	if err != nil {
		return nil, err
	}
	var x interface{}
	if err := yaml.Unmarshal(bs, &x); err != nil {
		return nil, err
	}
	return x, nil
}

// IncludeMap includes the filename (v) at (at)
func IncludeMap(ctx *Ctx, k string, v interface{}, at []string) (map[string]interface{}, error) {
	filename, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("%#v is a %T and not a %T",
			v, v, filename)
	}

	ctx.Logf("including map %s at %v", filename, at)

	y, err := ReadIncluded(ctx, filename)
	if err != nil {
		return nil, err
	}

	z, err := Include(ctx, y, append(at, k))
	if err != nil {
		return nil, err
	}

	m0, ok := z.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("value should be a map, not a %T", z)
	}

	return m0, nil
}

// Include looks for '#include' or 'include:' to find YAML files to
// include in the input data at the given location.
//
// 'include: FILENAME' will read FILENAME, which should be YAML
// representation of a map.  That map is added to the map that
// contained the 'include: FILENAME' property.  The FILENAME is
// relative to the given directory 'dir'.
//
// '#include<FILENAME>' will replace that value with the thing
// represented by FILENAME in YAML.  Unlike cpp, the FILENAME is
// relative to the given directory 'dir'.
func Include(ctx *Ctx, x interface{}, at []string) (interface{}, error) {
	switch vv := x.(type) {
	case string:
		s, ok := x.(string)
		if ok && strings.HasPrefix(s, "#include") {
			filename := strings.Trim(s[8:], "<>")
			ctx.Logf("including value %s at %v", filename, at)
			y, err := ReadIncluded(ctx, filename)
			if err != nil {
				return nil, err
			}
			return Include(ctx, y, at)
		}
		return x, nil
	case map[string]interface{}:
		m := make(map[string]interface{}, len(vv))
		for k, v := range vv {
			switch k {
			case "include":
				m0, err := IncludeMap(ctx, k, v, at)
				if err != nil {
					return nil, fmt.Errorf("failed to include map: %w", err)
				}
				for k0, v0 := range m0 {
					m[k0] = v0
				}
			case "includes":
				vl, ok := v.([]interface{})
				if !ok {
					return nil, fmt.Errorf("includes expects a list")
				}
				for _, v := range vl {
					m0, err := IncludeMap(ctx, k, v, at)
					if err != nil {
						return nil, fmt.Errorf("failed to include as map: %w", err)
					}
					for k0, v0 := range m0 {
						m[k0] = v0
					}
				}
			default:
				v, err := Include(ctx, v, append(at, k))
				if err != nil {
					return nil, err
				}
				m[k] = v
			}
		}
		return m, nil
	case []interface{}:
		a := make([]interface{}, 0, len(vv))
		for _, y := range vv {
			splicing := false
			s, ok := y.(string)
			if ok && strings.HasPrefix(s, "$include") {
				splicing = true
				y = "#" + s[1:] // Sorry!
			}
			z, err := Include(ctx, y, at)
			if err != nil {
				return nil, err
			}
			if !splicing {
				a = append(a, z)
			} else {
				if a0, is := z.([]interface{}); is {
					a = append(a, a0...)
				} else {
					return nil, fmt.Errorf("%#v is a %T, which can't be spliced into an array", z, z)
				}
			}
		}
		return a, nil
	default:
		return x, nil
	}

	return x, nil
}

// IncludeYAML surrounds Include() with YAML (un)marshaling.
//
// Intended to be used right after reading bytes that represent YAML.
func IncludeYAML(ctx *Ctx, bs []byte) ([]byte, error) {
	var x interface{}
	if err := yaml.Unmarshal(bs, &x); err != nil {
		return nil, err
	}
	y, err := Include(ctx, x, []string{})
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(&y)
}
