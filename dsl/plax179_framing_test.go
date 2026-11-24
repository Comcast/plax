package dsl

import (
	"bufio"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestReadLinesPreservesScannerCRFraming(t *testing.T) {
	for _, tc := range []struct{ name, input string }{
		{"bare-final-cr", "a\r"},
		{"only-cr", "\r"},
		{"crlf-and-final-cr", "a\r\nb\r"},
		{"double-cr-before-lf", "a\r\r\n"},
		{"interior-cr", "a\rb"},
		{"at-reader-boundary", strings.Repeat("x", 4096) + "\r"},
		{"nul-before-cr", "a\x00b\r"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scanner := bufio.NewScanner(strings.NewReader(tc.input))
			var want []string
			for scanner.Scan() {
				want = append(want, scanner.Text())
			}
			if err := scanner.Err(); err != nil {
				t.Fatal(err)
			}
			if got := collectReadLines(t, strings.NewReader(tc.input)); !reflect.DeepEqual(got, want) {
				t.Errorf("got %d lines; want %d; expected original Scanner framing", len(got), len(want))
			}
		})
	}
}
func collectReadLines(t *testing.T, r io.Reader) []string {
	t.Helper()
	var got []string
	if err := readLines(r, func(line string) bool { got = append(got, line); return true }); err != nil {
		t.Fatal(err)
	}
	return got
}
