package dsl

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
)

func maybeUnmarshalString(s string) (string, bool) {
	var x interface{}
	if err := json.Unmarshal([]byte(s), &x); err == nil {
		if s, is := x.(string); is {
			return s, true
		}
	}
	return s, false
}

func maybeMarshalString(s string, maybe bool) (string, error) {
	if !maybe {
		return s, nil
	}
	js, err := json.Marshal(&s)
	if err != nil {
		return "", err
	}
	return string(js), nil
}

var atAtFilename = regexp.MustCompile(`"?{@@(.+?)}"?`)

func atAtSub(ctx *Ctx, s string) (string, error) {
	s, un := maybeUnmarshalString(s)

	var err error
	y := atAtFilename.ReplaceAllStringFunc(s, func(s string) string {
		m := atAtFilename.FindStringSubmatch(s)
		if len(m) != 2 {
			err = Brokenf("internal error: failed to @@ submatch on '%s'", s)
			return fmt.Sprintf("<error: %s>", err)
		}
		filename := m[1]
		var bs []byte
		if bs, err = ioutil.ReadFile(ctx.Dir + "/" + filename); err != nil {
			return fmt.Sprintf("<error: %s>", err)
		}
		return string(bs)
	})
	if err != nil {
		return "", err
	}

	if y, err = maybeMarshalString(y, un); err != nil {
		return "", err
	}

	return y, nil
}

var bangBangFilename = regexp.MustCompile(`"?{!!(.+?)!!}"?`)

func bangBangSub(ctx *Ctx, s string) (string, error) {
	s, un := maybeUnmarshalString(s)

	var err error
	y := bangBangFilename.ReplaceAllStringFunc(s, func(s string) string {
		m := bangBangFilename.FindStringSubmatch(s)
		if len(m) != 2 {
			err = Brokenf("internal error: failed to !! submatch on '%s'", s)
			return fmt.Sprintf("<error: %s>", err)
		}
		src := m[1]
		ctx.Inddf("    Expansion: Javascript '%s'", short(src))
		var x interface{}
		if x, err = JSExec(ctx, src, nil); err != nil {
			return fmt.Sprintf("<error: %s>", err)
		}
		str, is := x.(string)
		if !is {
			var js []byte
			if js, err = json.Marshal(&x); err != nil {
				return fmt.Sprintf("<error: %s>", err)
			}
			str = string(js)
		}
		return str
	})
	if err != nil {
		return "", err
	}
	if y, err = maybeMarshalString(y, un); err != nil {
		return "", err
	}

	return y, nil
}
