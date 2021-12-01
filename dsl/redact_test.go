package dsl

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"testing"
)

func TestRedactionWanted(t *testing.T) {
	secret := "?X_SECRET"

	if !WantsRedaction(secret) {
		t.Fatal(secret)
	}

	secret = "?*X_SECRET"
	if !WantsRedaction(secret) {
		t.Fatal(secret)
	}

	public := "?PUBLIC"
	if WantsRedaction(public) {
		t.Fatal(public)
	}
}

func TestRedactionsAdd(t *testing.T) {
	var (
		r = NewRedactions()
	)
	r.Redact = true

	if err := r.Add("secret[0-9]+"); err != nil {
		t.Fatal(err)
	}

	line := r.Redactf("don't say this secret42")
	if strings.Contains(line, "secret") {
		t.Fatal(line)
	}

}

func TestRedactionsConcurrent(t *testing.T) {
	var (
		r  = NewRedactions()
		wg = &sync.WaitGroup{}
	)
	r.Redact = true

	wg.Add(1)
	go func() {
		for i := 0; i < 100; i++ {
			if err := r.Add(fmt.Sprintf("secret-%02d", i)); err != nil {
				t.Fatal(err)
			}
		}
		wg.Done()
	}()

	for i := 0; i < 100; i++ {
		secret := fmt.Sprintf("secret-%02d", i)
		r.Redactf("don't say %s", secret)
		// Might or might not have the redaction in
		// place, so don't check.
	}

	wg.Wait()

}

func TestRedact(t *testing.T) {
	type Pair struct {
		Pattern, String, Expected string
	}
	for _, p := range []Pair{
		{
			Pattern:  "make some (tacos)",
			String:   "Please make some tacos.",
			Expected: "Please make some <redacted>.",
		},
		{
			// Replace the last component of the token.
			// The first group isn't captured.  Just here
			// for the test.
			Pattern:  `"token":"[^.]+\.(?:[^.]+)\.([^"]+)`,
			String:   `"token":"bydiiuee.sdhyerhbxgygs.shdhgvfed"`,
			Expected: `"token":"bydiiuee.sdhyerhbxgygs.<redacted>"`,
		},
		{
			// Multiple groups but only one marked as redacting.
			Pattern:  `"token":"([^.]+)\.([^.]+)\.(?P<redact>[^"]+)`,
			String:   `"token":"bydiiuee.sdhyerhbxgygs.shdhgvfed"`,
			Expected: `"token":"bydiiuee.sdhyerhbxgygs.<redacted>"`,
		},
		{
			// Multiple groups; just redact first captured one.
			Pattern:  `"token":"(?:[^.]+)\.([^.]+)\.([^"]+)`,
			String:   `"token":"bydiiuee.sdhyerhbxgygs.shdhgvfed"`,
			Expected: `"token":"bydiiuee.<redacted>.shdhgvfed"`,
		},
	} {
		t.Run("", func(t *testing.T) {
			r := regexp.MustCompile(p.Pattern)
			s := Redact(r, p.String)
			if s != p.Expected {
				log.Fatalf("%s != %s (expected); pattern: %s", s, p.Expected, p.Pattern)
			}
		})
	}
}

func TestRedactionsDegenerate(t *testing.T) {
	var (
		r = NewRedactions()
	)
	r.Redact = true

	if err := r.Add(""); err != nil {
		t.Fatal(err)
	}

	in := "normalcy prevails"
	out := r.Redactf(in)
	if in != out {
		t.Fatal(out)
	}

}
