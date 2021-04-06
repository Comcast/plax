package dsl

import (
	"context"
	"strings"
	"testing"
)

func TestStringSub(t *testing.T) {
	var (
		ctx = NewCtx(context.Background())
		bs  = NewBindings()
		s   = `'{@@bindings.go}'`
	)

	got, err := bs.StringSub(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, `'/*`) {
		t.Fatal(got)
	}
}
