package dsl

import (
	"strings"
	"testing"
)

func TestWantsRedaction(t *testing.T) {
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

func TestAddRedaction(t *testing.T) {
	var (
		ctx    = NewCtx(nil)
		logger = NewTestLogger()
	)
	ctx.Logger = logger
	ctx.Redact = true

	if err := ctx.AddRedaction("secret[0-9]+"); err != nil {
		t.Fatal(err)
	}

	ctx.Redactf("don't say this secret42")
	line := logger.lines[0]
	if strings.Contains(line, "secret") {
		t.Fatal(line)
	}

}
