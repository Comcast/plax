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
	"encoding/json"
	"fmt"
	"log"
	"testing"

	"github.com/alecthomas/jsonschema"
)

func TestDocsMother(t *testing.T) {
	(&Mother{}).DocSpec().Write("mother")

	if false {

		s := jsonschema.Reflect(&MotherRequest{})

		{

			x := s.Definitions["MotherMakeRequest"]
			x.Description = "Hello"

			ps := x.Properties
			y, have := ps.Get("name")
			y.(*jsonschema.Type).Description = "World"
			log.Printf("hack %T %#v %#v", y, JSON(y), have)

		}

		bs, err := json.MarshalIndent(&s, "", "  ")
		if err != nil {
			t.Fatal(err)
		}

		fmt.Printf("debug %s\n", bs)
	}

}
