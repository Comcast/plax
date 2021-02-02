package dsl

import (
	"testing"
)

func TestRegexpMatch(t *testing.T) {
	p := `(?P<first>[a-zA-Z]+) and (?P<Last>[a-zA-Z]+)`
	bss, err := RegexpMatch(p, "Tacos and queso")
	if err != nil {
		t.Fatal(err)
	}

	if len(bss) != 1 {
		t.Fatal(bss)
	}
	bs := bss[0]

	if x, have := bs["?*first"]; !have {
		t.Fatal("?*first")
	} else {
		if s, is := x.(string); !is {
			t.Fatal(x)
		} else if s != "Tacos" {
			t.Fatal(s)
		}
	}

	if x, have := bs["?Last"]; !have {
		t.Fatal("?Last")
	} else {
		if s, is := x.(string); !is {
			t.Fatal(x)
		} else if s != "queso" {
			t.Fatal(s)
		}
	}
}
