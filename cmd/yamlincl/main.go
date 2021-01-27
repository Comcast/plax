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

// Package main is the command-line 'yamlincl' utilitiy.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/Comcast/plax/dsl"

	"gopkg.in/yaml.v3"
)

func main() {

	dirs := make(IncludeDirs, 0, 4)
	flag.Var(&dirs, "I", "directories to search")

	flag.Parse()

	ctx := dsl.NewCtx(nil)
	ctx.IncludeDirs = dirs

	bs, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	bs, err = dsl.IncludeYAML(ctx, bs)
	if err != nil {
		log.Fatal(err)
	}

	var x interface{}
	if err := yaml.Unmarshal(bs, &x); err != nil {
		log.Fatal(err)
	}

	if bs, err = yaml.Marshal(&x); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s\n", bs)
}

// IncludeDir are directories to search when YAML-including.
//
// We make an explicit type to enable flag.Var to parse multiple
// parameters.
type IncludeDirs []string

func (ps *IncludeDirs) String() string {
	return "DIR"
}

func (ps *IncludeDirs) Set(value string) error {
	*ps = append(*ps, value)
	return nil
}
