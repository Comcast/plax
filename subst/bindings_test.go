package subst

import (
	"context"
	"testing"
)

func TestBind(t *testing.T) {
	var (
		ctx = NewCtx(context.Background(), []string{"."})
	)

	t.Run("", func(t *testing.T) {
		bs := NewBindings()
		bs.SetValue("?NEED", "tacos")
		bs.SetValue("?SEND", "send")
		// Make sure keys are processed, too.
		var x interface{} = map[string]interface{}{
			"?SEND": "?NEED",
		}
		y, err := bs.Bind(ctx, x)
		if err != nil {
			t.Fatal(err)
		}
		m, is := y.(map[string]interface{})
		if !is {
			t.Fatal(x)
		}
		need, have := m["send"]
		if !have {
			t.Fatal(m)
		}
		s, is := need.(string)
		if !is {
			t.Fatal(need)
		}
		if s != "tacos" {
			t.Fatal(s)
		}
	})

}

func TestBindingsPipe(t *testing.T) {
	var (
		bs  = NewBindings()
		ctx = NewCtx(nil, nil)
		x   = map[string]interface{}{
			"request": "?like | jq .[0]",
		}
	)

	bs["?like"] = []interface{}{"tacos", "queso"}

	y, err := bs.Bind(ctx, x)
	if err != nil {
		t.Fatal(err)
	}

	if m, is := y.(map[string]interface{}); !is {
		t.Fatal(y)
	} else if z, have := m["request"]; !have {
		t.Fatal(m)
	} else if z != "tacos" {
		t.Fatal(z)
	}

}
