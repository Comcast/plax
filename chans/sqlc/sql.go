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

// Package sqlc provides a Plax channel type for talking to a SQL
// database.
package sqlc

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"plugin"
	"reflect"

	"github.com/Comcast/plax/dsl"

	_ "modernc.org/sqlite"
)

func init() {
	dsl.TheChanRegistry.Register(dsl.NewCtx(nil), "sql", NewChan)
}

var (
	// DefaultChanBufferSize is the default buffer size for
	// underlying Go channels used by some Chans.

	DefaultChanBufferSize = 1024
)

// Opts configures an SQL channel.
//
// DriverName and DatasourceName are per
// https://golang.org/pkg/database/sql/#Open.
type Opts struct {
	// DriverName names the (Go) SQL database driver, which must
	// be loaded previously (usually at compile-time or via a Go
	// plug-in).
	//
	// This package has a cgo-free SQLite driver
	// (modernc.org/sqlite) already loaded, so "sqlite" works
	// here.  Also see the 'mysql' subdirectory, which contains a
	// package that can be compiled as a Go plug-in that provides
	// a driver for MySQL.
	DriverName string

	// DatasourceName is the URI for the connection.  See your
	// driver documentation for details.
	//
	// You can use ":memory:" with DriverName "sqlite" to
	// experiment with in in-memory, SQLite-compatible database.
	DatasourceName string

	// BufferSize is the size of the internal channel that queues
	// results from the database.  Default is
	// DefaultChanBufferSize.
	BufferSize int

	// DriverPlugin, if given, should be the filename of a Go
	// plugin for a Go SQL driver.
	//
	// See https://golang.org/pkg/plugin/.
	//
	// The subdirectory 'chans/sqlc/mysql' has an example for
	// loading a MySQL driver at runtime.
	DriverPlugin string
}

// Chan is channel type that talks to a SQL database.
//
// When the input is a Query, the output consists of zero or more maps
// of strings to values, where each string is a column name, followed
// by a map from "hone" to the input query.
//
// When the input is an Exec statement, the output consists of a
// single map with 'rowsAffected' and 'lastInsertId' keys.
type Chan struct {
	c    chan dsl.Msg
	ctl  chan bool
	db   *sql.DB
	opts *Opts
}

// Input is input to a Pub operation.
type Input struct {

	// Query, if provided, should be a SQL statement (like SELECT)
	// that returns rows.
	Query string `json:"query,omitempty"`

	// Exec, if provided, should be a SQL statement that doesn't
	// return rows.  Examples: CREATE TABLE, INSERT INTO.
	Exec string `json:"exec,omitempty"`

	// Args is the array of parameters for the statement.
	Args []interface{} `json:"args"`
}

// asInput attemtps to parse a Input from the given string (JSON).
func asInput(js string) (*Input, error) {
	var msg Input
	if err := json.Unmarshal([]byte(js), &msg); err != nil {
		return nil, err
	}
	if msg.Query == "" && msg.Exec == "" {
		return nil, fmt.Errorf("need either query or exec statement")
	}
	if msg.Query != "" && msg.Exec != "" {
		return nil, fmt.Errorf("can't have both a query and an exec statement")
	}
	return &msg, nil
}

func NewChan(ctx *dsl.Ctx, o interface{}) (dsl.Chan, error) {
	js, err := json.Marshal(&o)
	if err != nil {
		return nil, dsl.NewBroken(err)
	}

	opts := Opts{
		BufferSize: DefaultChanBufferSize,
	}

	if err = json.Unmarshal(js, &opts); err != nil {
		return nil, dsl.NewBroken(err)
	}

	if opts.DriverPlugin != "" {
		log.Printf("DEBUG loading SQL driver plugin %s", opts.DriverPlugin)
		if _, err := plugin.Open(opts.DriverPlugin); err != nil {
			return nil, fmt.Errorf("failed to open SQL driver plugin %s: %s",
				opts.DriverPlugin, err)
		}
	}

	return &Chan{
		c:    make(chan dsl.Msg, opts.BufferSize),
		ctl:  make(chan bool),
		opts: &opts,
	}, nil
}

func (c *Chan) DocSpec() *dsl.DocSpec {
	return &dsl.DocSpec{
		Chan:  &Chan{},
		Opts:  &Opts{},
		Input: &Input{},
	}
}

func (c *Chan) Kind() dsl.ChanKind {
	return "SQL"
}

func (c *Chan) Open(ctx *dsl.Ctx) error {
	db, err := sql.Open(c.opts.DriverName, c.opts.DatasourceName)
	if err != nil {
		return err
	}
	c.db = db
	return nil
}

func (c *Chan) Close(ctx *dsl.Ctx) error {
	return c.db.Close()
}

func (c *Chan) Sub(ctx *dsl.Ctx, topic string) error {
	return dsl.Brokenf("Can't Sub on an SQL channel")
}

// say is a utility for emitting some basic messages from the channel.
func (c *Chan) say(ctx *dsl.Ctx, key, format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	acc := map[string]interface{}{
		key: s,
	}
	js, err := json.Marshal(&acc)
	if err != nil {
		js, _ = json.Marshal(&s)
	}
	msg := dsl.Msg{
		Payload: string(js),
	}
	select {
	case <-ctx.Done():
	case c.c <- msg:
	}
}

// complain formats and emits a complaint.
func (c *Chan) complain(ctx *dsl.Ctx, format string, args ...interface{}) {
	c.say(ctx, "error", format, args...)
}

func (c *Chan) Pub(ctx *dsl.Ctx, m dsl.Msg) error {
	msg, err := asInput(m.Payload)
	if err != nil {
		return err
	}
	if msg.Query != "" {
		go func() {
			rs, err := c.db.QueryContext(ctx, msg.Query, msg.Args...)
			if err != nil {
				c.complain(ctx, "bad SQL query %s: %s", msg.Query, err)
				return
			}

			defer rs.Close()

			cols, err := rs.ColumnTypes()
			if err != nil {
				c.complain(ctx, "error getting result columns: %s", err)
				return
			}
			names := make([]string, len(cols))
			vals := make([]interface{}, len(cols))
			for i, col := range cols {
				names[i] = col.Name()
				vals[i] = reflect.New(col.ScanType()).Interface()
			}
			defer func() {
				c.say(ctx, "done", "%s", msg.Query)
			}()
		LOOP:
			for rs.Next() {
				if err := rs.Scan(vals...); err != nil {
					c.complain(ctx, "error scanning row: %s", err)
					return
				}
				acc := make(map[string]interface{}, len(cols))
				for i, v := range vals {
					acc[names[i]] = v
				}
				js, err := json.Marshal(&acc)
				if err != nil {
					c.complain(ctx, "error marshaling result: %s", err)
					return
				}
				m := dsl.Msg{
					Payload: string(js),
				}
				select {
				case <-ctx.Done():
					break LOOP
				case c.c <- m:
				}
			}

		}()
	} else {
		go func() {
			r, err := c.db.ExecContext(ctx, msg.Exec, msg.Args...)
			if err != nil {
				c.complain(ctx, "SQL statement error %s: %s", msg.Exec, err)
				return
			}

			lastInsertId, err := r.LastInsertId()
			if err != nil {
				c.complain(ctx, "SQL statement exec error %s: %s", msg.Exec, err)
				return
			}

			rowsAffected, err := r.RowsAffected()
			if err != nil {
				c.complain(ctx, "SQL statement exec error %s: %s", msg.Exec, err)
				return
			}

			acc := struct {
				LastInsertId int `json:"lastInsertId"`
				RowsAffected int `json:"rowsAffected"`
			}{
				LastInsertId: int(lastInsertId),
				RowsAffected: int(rowsAffected),
			}

			js, err := json.Marshal(&acc)
			if err != nil {
				c.complain(ctx, "error marshaling SQL result: %s", err)
				return
			}
			m := dsl.Msg{
				Payload: string(js),
			}
			select {
			case <-ctx.Done():
			case c.c <- m:
			}
		}()
	}

	return nil
}

func (c *Chan) Recv(ctx *dsl.Ctx) chan dsl.Msg {
	return c.c
}

func (c *Chan) Kill(ctx *dsl.Ctx) error {
	return c.db.Close()
}

func (c *Chan) To(ctx *dsl.Ctx, m dsl.Msg) error {
	select {
	case <-ctx.Done():
	case c.c <- m:
	}
	return nil
}
