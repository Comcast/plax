package subst

import (
	"testing"
)

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
