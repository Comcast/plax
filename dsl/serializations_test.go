package dsl

import "testing"

func TestSerialization(t *testing.T) {
	for name, ser := range Serializations {
		t.Run(name, func(t *testing.T) {
			payload := `{"want":"tacos"}`
			x, err := ser.Deserialize(payload)
			if err != nil {
				t.Fatal(err)
			}
			y, err := ser.Serialize(x)
			if err != nil {
				t.Fatal(err)
			}
			if payload != y {
				t.Fatal(y)
			}
		})
	}

	t.Run("illegal", func(t *testing.T) {
		if _, err := NewSerialization("graffiti"); err == nil {
			t.Fatal("expected a complaint")
		}
	})
}
