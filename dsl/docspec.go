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
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"unicode"
)

type DocSpec struct {
	SrcDirs []string

	Chan    Chan
	Doc     string
	OptsDoc string
	Opts    interface{}

	InputDoc string
	Input    interface{}

	OutputDoc string
	Output    interface{}
}

type docAndName struct {
	Doc  string
	Name string
}

type comments map[string]map[string]*docAndName

var DebugDocScan = false

func (ds *DocSpec) parseComments() (comments, error) {

	logf := func(format string, args ...interface{}) {
		if DebugDocScan {
			log.Printf(format, args...)
		}
	}

	acc := make(comments)
	for _, srcDir := range ds.SrcDirs {
		logf("DocSpec dir '%s'", srcDir)

		if srcDir == "" {
			// Maybe warn?
			return make(comments), nil
		}

		fset := token.NewFileSet()

		d, err := parser.ParseDir(fset, srcDir, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}

		var (
			typeDoc   string
			typeName  string
			fieldDocs map[string]*docAndName
		)

		for pkg, f := range d {
			logf("DocSpec pkg %v", pkg)
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.GenDecl:
					typeDoc = x.Doc.Text()
				case *ast.TypeSpec:
					// https://github.com/golang/go/issues/27477
					typeName = x.Name.String()
					logf("DocSpec type %s", typeName)
					fieldDocs = make(map[string]*docAndName)
					acc[typeName] = fieldDocs
					logf("DocSpec type %s doc %s", typeName, short(typeDoc))
					dnn := &docAndName{
						Doc: typeDoc,
					}
					fieldDocs["type"] = dnn
				case *ast.Field:
					// For an embedded struct x.Names is
					// nil and "the field name is the type
					// name".  See reflect.Field.
					var name string
					if x.Names != nil {
						name = x.Names[0].String()
					} else {
						name = "Embedded"
						logf("embedded?")
					}
					if !exported(name) {
						return false
					}
					if fieldDocs == nil {
						logf("DocSpec skipping %s.%s", typeName, name)
						return true
					}
					doc := strings.TrimSpace(x.Doc.Text())
					if 0 < len(doc) && name == "Embedded" {
						// Hope that the first word is the name of the thing.
						name = strings.Split(doc, " ")[0]
					}
					logf("DocSpec %s field %s doc %s", typeName, name, short(doc))
					dnn := &docAndName{
						Doc:  doc,
						Name: name,
					}
					fieldDocs[name] = dnn

					if x.Tag != nil {
						tag := x.Tag.Value
						tag = tag[1 : len(tag)-1]
						parts := regexp.MustCompile(" +").Split(tag, -1)
						for _, part := range parts {
							ss := strings.SplitN(part, ":", 2)
							if ss[0] != "json" {
								continue
							}
							ss = strings.Split(strings.Trim(ss[1], `"`), ",")
							jsonname := ss[0]
							if jsonname == "-" {
								break
							}
							logf("DocSpec field json %s", jsonname)
							fieldDocs[name].Name = jsonname
							break
						}
					}
				case *ast.FuncType:
					return false
				default:
					// log.Printf("DocSpec default %T %v", n, n)
				}
				return true
			})
		}
	}

	return acc, nil
}

func SrcDir(level int) string {
	if _, file, _, ok := runtime.Caller(level); ok {
		// runtime.FuncForPC(pc).Name()
		return filepath.Dir(file)
	}
	return "."
}

func (ds *DocSpec) Write(name string) error {

	ds.SrcDirs = []string{
		// Don't judge
		SrcDir(1), // This (dsl) directory, we hope.
		SrcDir(2), // The channel's source directory, we hope.
	}

	out, err := os.Create("chan_" + name + ".md")
	if err != nil {
		return err
	}
	gerr := ds.Gen(out, name)
	oerr := out.Close()

	if gerr != nil {
		if oerr != nil {
			err = fmt.Errorf("errors: %s; %s", gerr, oerr)
		} else {
			err = gerr
		}
	} else {
		if oerr != nil {
			err = oerr
		}
	}

	return err
}

func (ds *DocSpec) Gen(out io.Writer, name string) error {

	cs, err := ds.parseComments()
	if err != nil {
		return err
	}

	get := func(name, field string) *docAndName {
		def := &docAndName{
			Name: field,
		}
		if name == "" || field == "" {
			return def
		}
		m, have := cs[name]
		if !have {
			log.Printf("No doc for %s", name)
			return def
		}
		dnn, have := m[field]
		if !have {
			log.Printf("No doc for %s.%s", name, field)
			return def
		}
		return dnn
	}

	dropFirstSentencePat := regexp.MustCompile(`(?s)[-.]*[.]\s+(.*)`)
	dropFirstSentence := func(s string) string {
		if m := dropFirstSentencePat.FindStringSubmatch(s); m != nil {
			return m[1]
		}
		return s
	}

	dropFirstWordPat := regexp.MustCompile(`(?s)\w+\s+(.*)`)
	dropFirstWord := func(s string) string {
		if m := dropFirstWordPat.FindStringSubmatch(s); m != nil {
			return m[1]
		}
		return s
	}

	// indent applies a hanging indentation.
	indent := func(doc string, padding string) string {
		lines := strings.Split(doc, "\n")
		return strings.Join(lines, "\n"+padding)
	}

	ind := "    "

	{
		typ := reflect.ValueOf(ds.Chan).Elem().Type()
		fmt.Fprintf(out, "## `%s`\n\n", name)
		if doc := get(typ.Name(), "type").Doc; doc != "" {
			doc = dropFirstSentence(doc)
			fmt.Fprintf(out, "%s\n", doc)
		}

	}

	getJSONName := func(f reflect.StructField) string {
		def := f.Name
		s, have := f.Tag.Lookup("json")
		if !have {
			return def
		}
		parts := strings.Split(s, ",")
		if name := parts[0]; name != "" {
			return name
		}
		return def
	}

	ptrType := func(typ reflect.Type) reflect.Type {
		for typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		return typ
	}

	var genTypeDoc func(typ reflect.Type, padding string)

	genFieldsDocs := func(typ reflect.Type, padding string) {
		typ = ptrType(typ)
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if f.PkgPath != "" {
				continue
			}
			name := getJSONName(f)
			if name == "-" {
				continue
			}
			ftyp := f.Type.Name()
			if ftyp != "" && !exported(ftyp) {
				ftyp = fmt.Sprintf(" (%s) ", ftyp) // Sorry
			} else {
				ftyp = " "
			}
			if doc := get(typ.Name(), f.Name).Doc; doc != "" {
				doc = dropFirstWord(doc)
				fmt.Fprintf(out, "%s1. `%s`%s%s\n\n", padding, name, ftyp, indent(doc, padding+ind))
			} else {
				fmt.Fprintf(out, "%s1. `%s`%s\n\n", padding, name, ftyp)
			}
			switch ptrType(f.Type).Kind() {
			case reflect.Struct:
				genTypeDoc(f.Type, padding+ind)
			}
		}
	}

	genTypeDoc = func(typ reflect.Type, padding string) {
		if doc := get(typ.Name(), "type").Doc; doc != "" {
			doc = dropFirstSentence(doc)
			fmt.Fprintf(out, "%s%s\n", padding, doc)
		}
		genFieldsDocs(typ, padding)
	}

	if ds.Opts != nil {
		fmt.Fprintf(out, "### Options\n\n")
		typ := reflect.ValueOf(ds.Opts).Elem().Type()
		genTypeDoc(typ, "")
	}

	if ds.Input != nil {
		fmt.Fprintf(out, "### Input\n\n")
		typ := reflect.ValueOf(ds.Input).Elem().Type()
		genTypeDoc(typ, "")
	}

	if ds.Output != nil {
		fmt.Fprintf(out, "### Output\n\n")
		typ := reflect.ValueOf(ds.Output).Elem().Type()
		genTypeDoc(typ, "")
	}

	return nil
}

func exported(name string) bool {
	for _, r := range name {
		return unicode.IsUpper(r)
	}
	return false
}
