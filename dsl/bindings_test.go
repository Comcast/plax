package dsl

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"testing"
)

func TestSubstitute(t *testing.T) {
	var (
		ctx = NewCtx(context.Background())
		tst = NewTest(ctx, "a", nil)
	)

	t.Run("basic", func(t *testing.T) {
		// We bind variables that require recursive
		// subsitution.  Note that these variables (by
		// definition) start with '?'.
		tst.Bindings = map[string]interface{}{
			"?want":  "{?queso}",
			"?queso": "queso",
		}
		s, err := tst.Bindings.StringSub(ctx, `!!"I want " + "{?want}."`)
		if err != nil {
			t.Fatal(err)
		}
		if s != "I want queso." {
			t.Fatal(s)
		}
	})

	t.Run("constantEmbedded", func(t *testing.T) {
		// Same basic test but we using a "binding" for a
		// constant (without the '?' prefix).
		tst.Bindings = map[string]interface{}{
			// Bind 'want' to a string that itself
			// references a binding variable.
			"want": "{?queso}",
			// Bind the variable referenced above.
			"?queso": "queso",
		}
		s, err := tst.Bindings.StringSub(ctx, `!!"I want " + "{want}."`)
		if err != nil {
			t.Fatal(err)
		}
		if s != "I want queso." {
			t.Fatal(s)
		}
	})

	t.Run("constantStructured", func(t *testing.T) {
		// Parameter-like subsitution: Bind a "parameter",
		// which has no "?" but does have "{}".
		tst.Bindings = map[string]interface{}{
			// Bind 'want' to a string that itself
			// references a binding variable.
			"{want}": "{?this}",
			// Bind the variable referenced above.
			"{?this}": "queso",
		}
		x := MaybeParseJSON(`{"need":"{want}"}`)
		var y interface{}
		if err := tst.Bindings.SubX(ctx, x, &y); err != nil {
			t.Fatal(err)
		}
		log.Printf("DEBUG y %s", JSON(y))
		js1, err := json.Marshal(&x)
		if err != nil {
			t.Fatal(err)
		}
		js2, err := json.Marshal(&y)
		if err != nil {
			t.Fatal(err)
		}
		if string(js1) != string(js2) {
			t.Fatal(string(js2))
		}
	})

	t.Run("deepstring", func(t *testing.T) {

		var (
			src, target struct {
				Foo struct {
					Bar string
				}
			}
			js = `{"Foo":{"Bar":"I want {?want}."}}`
		)

		if err := json.Unmarshal([]byte(js), &src); err != nil {
			t.Fatal(err)
		}

		tst.Bindings = map[string]interface{}{
			"?want": "queso",
		}
		if err := tst.Bindings.SubX(ctx, src, &target); err != nil {
			t.Fatal(err)
		}
		if s := target.Foo.Bar; s != "I want queso." {
			t.Fatal(s)
		}
	})

}

func TestSubstituteOnce(t *testing.T) {
	var (
		ctx = NewCtx(context.Background())
		tst = NewTest(ctx, "a", nil)
	)

	t.Run("badjs", func(t *testing.T) {
		if _, err := tst.Bindings.StringSubOnce(ctx, "!!nope"); err == nil {
			t.Fatal("should have complained")
		}
	})

	t.Run("jsobj", func(t *testing.T) {
		if s, err := tst.Bindings.StringSubOnce(ctx, `!!({"want":"tacos"})`); err != nil {
			t.Fatal(err)
		} else if s != `{"want":"tacos"}` {
			t.Fatal(s)
		}
	})

	t.Run("jsnots", func(t *testing.T) {
		if _, err := tst.Bindings.StringSubOnce(ctx, `!!function() {}`); err == nil {
			t.Fatal("should have complained")
		}
	})

	t.Run("file", func(t *testing.T) {
		if s, err := tst.Bindings.StringSubOnce(ctx, `@@test_test.go`); err != nil {
			t.Fatal(err)
		} else if len(s) < 1000 {
			t.Fatal(len(s))
		}
	})

	t.Run("filebad", func(t *testing.T) {
		if _, err := tst.Bindings.StringSubOnce(ctx, `@@nope`); err == nil {
			t.Fatal("should have complained")
		}
	})

	tst.Bindings = map[string]interface{}{
		"?need": "chips",
	}

	t.Run("filegood", func(t *testing.T) {
		s, err := tst.Bindings.StringSubOnce(ctx, `@@test_test.go`)
		if err != nil {
			t.Fatal(err)
		}
		// The following comment should be substituted!
		//
		// {?need}
		if strings.Contains(s, "{?need}") {
			t.Fatal("?need")
		}
	})
}

func TestBindingsSet(t *testing.T) {
	bs := NewBindings()
	check := func(err error, key, want string) {
		if err != nil {
			t.Fatalf("key %s -> <%v> != %s", key, err, want)
		}
		got, have := bs[key]
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
		x, have := bs["like"]
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
		x, have := bs["like"]
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
