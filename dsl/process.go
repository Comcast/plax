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
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"syscall"
)

// Process represents an external process run from a test.
type Process struct {
	// Name is an opaque string used is reports about this
	// Process.
	Name string `json:"name" yaml:"name"`

	// Command is the name of the program.
	//
	// Subject to expansion.
	Command string `json:"command" yaml:"command"`

	// Args is the list of command-line arguments for the program.
	//
	// Subject to expansion.
	Args []string `json:"args" yaml:"args"`

	// ToDo: Environment, dir.

	cmd *exec.Cmd

	Stdout   chan string `json:"-"`
	Stderr   chan string `json:"-"`
	Stdin    chan string `json:"-"`
	ExitCode chan int    `json:"-"`

	// ctl is only used to terminate goroutines when the Process
	// is terminated.
	ctl chan bool
}

// Substitute the bindings into the Process
func (p *Process) Substitute(ctx *Ctx, bs *Bindings) (*Process, error) {
	cmd, err := bs.StringSub(ctx, p.Command)
	if err != nil {
		return nil, err
	}
	args := make([]string, len(p.Args))
	for i, arg := range p.Args {
		s, err := bs.StringSub(ctx, arg)
		if err != nil {
			return nil, err
		}
		args[i] = s
	}
	return &Process{
		Name:    p.Name,
		Command: cmd,
		Args:    args,
	}, nil
}

// TrimEOL is a utility function that removes the last (if any)
// newline character(s).
//
// This function does not trim more than one newline.
func TrimEOL(s string) string {
	s = strings.TrimSuffix(s, "\n")
	return strings.TrimSuffix(s, "\r")
}

// Start starts the program, which runs in the background (until the
// test is complete).
//
// Stderr and stdout are logged via ctx.Logf.
func (p *Process) Start(ctx *Ctx) error {

	p.Stdin = make(chan string)
	p.Stderr = make(chan string)
	p.Stdout = make(chan string)
	p.ctl = make(chan bool)
	p.ExitCode = make(chan int)

	p.cmd = exec.Command(p.Command, p.Args...)

	inPipe, err := p.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("Process %s Run error on StdinPipe: %s", p.Name, err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case line := <-p.Stdin:
				line = TrimEOL(line) + "\n"
				if _, err = io.WriteString(inPipe, line); err != nil {
					log.Printf("Warning: Process Write: %s", err)
				}
			}
		}
	}()

	errPipe, err := p.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("Process %s Run error on StderrPipe: %s", p.Name, err)
	}

	go func() {
		in := bufio.NewScanner(errPipe)
		for in.Scan() {
			line := in.Text()
			ctx.Logf("Process %s stderr line: %s\n", p.Name, line)
			select {
			case <-ctx.Done():
				return
			case <-p.ctl:
				return
			case p.Stderr <- line:
			}
		}
		if err := in.Err(); err != nil {
			ctx.Logf("Process %s stderr error %s", p.Name, err)
		}
	}()

	outPipe, err := p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Process %s Run error on StdoutPipe: %s", p.Name, err)
	}

	go func() {
		in := bufio.NewScanner(outPipe)
		for in.Scan() {
			line := in.Text()
			ctx.Logf("Process %s stdout line: %s\n", p.Name, line)
			select {
			case <-ctx.Done():
				return
			case <-p.ctl:
				return
			case p.Stdout <- line:
			}
		}
		if err := in.Err(); err != nil {
			ctx.Logf("Process %s stdout error %s", p.Name, err)
		}
	}()

	if err := p.cmd.Start(); err != nil {
		ctx.Logf("Process %s error on start: %s", p.Name, err)
		return err
	}

	go func() {
		if err := p.cmd.Wait(); err != nil {
			ctx.Logf("Process %s error on wait: %s", p.Name, err)
		}

		p.ExitCode <- p.cmd.ProcessState.ExitCode()
	}()

	return nil
}

// Term sends a SIGTERM to the process.
func (p *Process) Term(ctx *Ctx) error {
	close(p.ctl)
	p.cmd.Process.Signal(syscall.SIGTERM)
	return nil
}

// Kill kills the Process.
func (p *Process) Kill(ctx *Ctx) error {
	ctx.Logf("Process %s killed", p.Name)
	return p.cmd.Process.Kill()
}
