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

func TestBindingsSet(t *testing.T) {
	bs := NewBindings()
	check := func(err error, key, want string) {
		if err != nil {
			t.Fatalf("key %s -> <%v> != %s", key, err, want)
		}
		got, have := (*bs)[key]
		if !have {
			t.Fatalf("key %s -> %s != %s", key, got, want)
		}
	}

	err := bs.Set(`like="tacos"`)
	check(err, "like", "tacos")

	err = bs.Set(`like=tacos`)
	check(err, "like", "tacos")

	if err = bs.Set(`like=42`); err != nil {
		t.Fatal(err)
	} else {
		x, have := (*bs)["like"]
		if !have {
			t.Fatal("like")
		}
		switch vv := x.(type) {
		case float64:
			if vv != 42 {
				t.Fatal(vv)
			}
		default:
			t.Fatalf("%T: %v", x, x)
		}
	}

	if err = bs.Set(`like={"want":"chips"}`); err != nil {
		t.Fatal(err)
	} else {
		x, have := (*bs)["like"]
		if !have {
			t.Fatal("like")
		}
		switch vv := x.(type) {
		case map[string]interface{}:
			x, have := vv["want"]
			if !have {
				t.Fatal(err)
			}
			switch vv := x.(type) {
			case string:
				if vv != "chips" {
					t.Fatal(vv)
				}
			default:
				t.Fatalf("%T: %v", x, x)
			}
		default:
			t.Fatalf("%T: %v", x, x)
		}
	}

	if err = bs.Set(`liketacos`); err == nil {
		t.Fatal("should have complained")
	}
}
