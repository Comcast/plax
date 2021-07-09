package dsl

import (
	"testing"

	"github.com/Comcast/sheens/match"
)

func TestRecvSubstitute(t *testing.T) {
	ctx, _, tst := newTest(t)

	desired := "tacos"
	tst.Bindings = Bindings{
		"?desired": desired,
	}

	r0 := &Recv{
		Topic: "{?desired}",
		Run:   "I like {?desired}.",
		// ToDo: A lot more.
	}

	r1, err := r0.Substitute(ctx, tst)
	if err != nil {
		t.Fatal(err)
	}

	if r1.Topic != desired {
		t.Fatal(r1.Topic)
	}

	if r1.Run != "I like "+desired+"." {
		t.Fatal(r1.Topic)
	}
}

func TestRecvValidateSchema(t *testing.T) {
	ctx := NewCtx(nil)
	ctx.LogLevel = "NONE"
	schema := "file://../demos/order.json"
	t.Run("happy", func(t *testing.T) {
		msg := `{"want":"chips"}`
		if err := validateSchema(ctx, schema, msg); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("sad", func(t *testing.T) {
		msg := `{"need":"queso"}`
		if err := validateSchema(ctx, schema, msg); err == nil {
			t.Fatal("'want' was required")
		}
	})
}

func TestRecvAttemptMatch(t *testing.T) {

	// ToDo: So much more ...

	ctx, _, tst := newTest(t)

	m := Msg{
		Topic:   "questions",
		Payload: `"To be or not to be?"`,
	}

	r := &Recv{
		Topic:   "questions",
		Pattern: `"?q"`,
	}

	var err error
	if r, err = r.Substitute(ctx, tst); err != nil {
		t.Fatal(err)
	}

	bss, err := r.attemptMatch(ctx, tst, m)
	if err != nil {
		t.Fatal(err)
	}

	if len(bss) != 1 {
		t.Fatal(bss)
	}
	if _, have := bss[0]["?q"]; !have {
		t.Fatal(bss[0])
	}
}

func TestRecvExtendBindings(t *testing.T) {
	var (
		ctx, _, tst = newTest(t)
		r           = &Recv{}
	)
	ctx.quiet()

	t.Run("boring", func(t *testing.T) {
		bss := []match.Bindings{
			match.Bindings{
				"one": 1,
			},
		}

		if err := r.extendBindings(ctx, tst, bss); err != nil {
			t.Fatal(nil)
		}
	})

	t.Run("extend", func(t *testing.T) {
		tst.Bindings = Bindings{
			"zero": 0,
		}

		bss := []match.Bindings{
			match.Bindings{
				"one": 1,
			},
		}

		if err := r.extendBindings(ctx, tst, bss); err != nil {
			t.Fatal(nil)
		}

		if len(tst.Bindings) != 2 {
			t.Fatal(tst.Bindings)
		}
	})

	t.Run("update", func(t *testing.T) {
		tst.Bindings = Bindings{
			"like": "kale",
		}

		bss := []match.Bindings{
			match.Bindings{
				"like": "queso",
			},
		}

		if err := r.extendBindings(ctx, tst, bss); err != nil {
			t.Fatal(nil)
		}

		if len(tst.Bindings) != 1 {
			t.Fatal(tst.Bindings)
		}

		x, have := tst.Bindings["like"]
		if !have {
			t.Fatal(tst.Bindings)
		}
		if x != "queso" {
			t.Fatal(x)
		}
	})

	t.Run("multiple", func(t *testing.T) {
		bss := []match.Bindings{
			match.Bindings{
				"one": 1,
			},
			match.Bindings{
				"two": 2,
			},
		}

		if err := r.extendBindings(ctx, tst, bss); err == nil {
			t.Fatal(len(bss))
		}
	})

}

func TestRecvCheckGuard(t *testing.T) {

	// ToDo: Much more.
	var (
		ctx, _, tst = newTest(t)
		m           = Msg{}
	)
	ctx.quiet()

	t.Run("happy", func(t *testing.T) {
		var (
			bss = []match.Bindings{
				match.Bindings{
					"likes": "queso",
				},
			}
			r = &Recv{
				Guard: `return bs["likes"] == "queso";`,
			}
		)

		tst.Bindings = Bindings(bss[0])
		happy, err := r.checkGuard(ctx, tst, m, bss)
		if err != nil {
			t.Fatal(err)
		}
		if !happy {
			t.Fatal("sad")
		}
	})

	t.Run("sad", func(t *testing.T) {
		var (
			bss = []match.Bindings{
				match.Bindings{
					"likes": "kale",
				},
			}
			r = &Recv{
				Guard: `return bs["likes"] == "queso";`,
			}
		)

		tst.Bindings = Bindings(bss[0])
		happy, err := r.checkGuard(ctx, tst, m, bss)
		if err != nil {
			t.Fatal(err)
		}
		if happy {
			t.Fatal("wrong")
		}
	})
}

func TestRecvRunRun(t *testing.T) {

	var (
		ctx, _, tst = newTest(t)
		m           = Msg{}
		bss         = []match.Bindings{
			match.Bindings{
				"likes": "queso",
			},
		}
		r = &Recv{
			Run: `test.Bindings["enjoy"] = bs["likes"];`,
		}
	)

	tst.Bindings = Bindings(bss[0])
	err := r.runRun(ctx, tst, m, bss)
	if err != nil {
		t.Fatal(err)
	}
	x, have := tst.Bindings["enjoy"]
	if !have {
		t.Fatal(tst.Bindings)
	}
	if x != "queso" {
		t.Fatal(x)
	}

}

func TestRecvVerify(t *testing.T) {
	var (
		ctx, _, tst = newTest(t)
		m           = Msg{
			Payload: `{"likes":"tacos"}`,
		}
		r = &Recv{
			Pattern: MaybeParseJSON(`{"likes":"?enjoy"}`),
			Guard:   `return bs["?enjoy"] == "tacos";`,
		}
	)

	ctx.quiet()

	var err error
	if r, err = r.Substitute(ctx, tst); err != nil {
		t.Fatal(err)
	}

	happy, err := r.verify(ctx, tst, m)
	if err != nil {
		t.Fatal(err)
	}
	if !happy {
		t.Fatal("sad")
	}
}
