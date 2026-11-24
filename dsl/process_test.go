/*
 * Copyright 2026 Comcast Cable Communications Management, LLC
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
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestReadLinesLongLine(t *testing.T) {
	// bufio.Scanner rejects tokens larger than 64KiB by default.
	long := strings.Repeat("x", bufio.MaxScanTokenSize+16)
	input := "short\n" + long + "\nlast\n"
	var got []string
	if err := readLines(strings.NewReader(input), func(line string) bool {
		got = append(got, line)
		return true
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d lines: %#v", len(got), got)
	}
	if got[0] != "short" || got[2] != "last" {
		t.Fatalf("unexpected short lines: %#v", got)
	}
	if got[1] != long {
		t.Fatalf("long line length %d, want %d", len(got[1]), len(long))
	}
}

func TestReadLinesCRLFAndEOFWithoutNewline(t *testing.T) {
	var got []string
	if err := readLines(bytes.NewReader([]byte("a\r\nb\r\nc")), func(line string) bool {
		got = append(got, line)
		return true
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("got %#v", got)
	}
}

func TestProcessStdoutLongLine(t *testing.T) {
	ctx := NewCtx(nil)
	n := bufio.MaxScanTokenSize + 16
	p := Process{
		Name:    "long-stdout",
		Command: "bash",
		Args:    []string{"-c", fmt.Sprintf("head -c %d /dev/zero | tr '\\0' 'x'; echo", n)},
	}
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer p.Kill(ctx)

	var (
		to  = time.NewTimer(5 * time.Second)
		got string
	)
	select {
	case <-ctx.Done():
		t.Fatal("ctx done")
	case <-to.C:
		t.Fatal("timeout waiting for long stdout line")
	case got = <-p.Stdout:
	}
	if len(got) != n {
		t.Fatalf("stdout length %d, want %d", len(got), n)
	}
}
