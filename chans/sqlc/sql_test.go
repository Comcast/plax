package sqlc

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Comcast/plax/dsl"
)

func TestChan(t *testing.T) {
	ctx := dsl.NewCtx(context.Background())
	c, err := NewChan(ctx, map[string]interface{}{
		"DriverName":     "sqlite",
		"DatasourceName": ":memory:",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err = c.Open(ctx); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err = c.Close(ctx); err != nil {
			t.Fatal(err)
		}
	}()

	pub := func(typ, st string, args ...interface{}) {
		op := map[string]interface{}{
			typ:    st,
			"args": args,
		}
		js, err := json.Marshal(&op)
		if err != nil {
			t.Fatal(err)
		}
		msg := dsl.Msg{
			Payload: string(js),
		}
		if err = c.Pub(ctx, msg); err != nil {
			t.Fatal(err)
		}
	}

	ch := c.Recv(ctx)

	recv := func(contains, k string, want int) {
		msg := <-ch
		if contains != "" {
			if !strings.Contains(msg.Payload, contains) {
				t.Fatalf(`"%s" doesn't contain "%s"`, msg.Payload, contains)
			}
			return
		}
		var ns map[string]int

		if err := json.Unmarshal([]byte(msg.Payload), &ns); err != nil {
			t.Fatalf("Unmarshal error on %s: %s", msg.Payload, err)
		}
		got, have := ns[k]
		if !have {
			t.Fatalf("no key %s found in %#v", k, ns)
		}
		if want != got {
			t.Fatalf("%d != %d for %s", want, got, k)
		}
	}

	pub("exec", "CREATE TABLE foo (x INTEGER)")
	recv("", "rowsAffected", 0)

	pub("exec", "INSERT INTO foo VALUES (42)")
	recv("", "rowsAffected", 1)

	pub("query", "SELECT MAX(x) AS n FROM foo")
	recv("", "n", 42)
	recv("done", "", 0)

	pub("exec", "INSERT INTO foo VALUES (43)")
	recv("", "rowsAffected", 1)

	pub("exec", "INSERT INTO foo VALUES (44)")
	recv("", "rowsAffected", 1)

	pub("query", "SELECT x FROM foo ORDER BY x")
	recv("", "x", 42)
	recv("", "x", 43)
	recv("", "x", 44)
	recv("done", "", 0)

}
