package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Comcast/plax/subst"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {

	var (
		ctx        = subst.NewCtx(context.Background(), []string{"."})
		ps         = subst.NewBindings()
		ps1        = subst.NewBindings()
		delimiters = flag.String("d", "{}", "Delimeters")
		psFilename = flag.String("ps", "", "JSON file for parameters")
		bind       = flag.Bool("bind", false, "Do structuring binding instead of substitutions")
		checkIn    = flag.Bool("check-json-in", false, "Verify input is JSON")
		checkOut   = flag.Bool("check-json-out", false, "Verify output is JSON")
	)

	flag.Var(&ps1, "p", "parameters")

	flag.Parse()

	if *psFilename != "" {
		bs, err := ioutil.ReadFile(*psFilename)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(bs, &ps); err != nil {
			return err
		}
	}

	for k, v := range ps1 {
		ps.SetValue(k, v)
	}

	bs, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	if *checkIn {
		var x interface{}
		if err = json.Unmarshal(bs, &x); err != nil {
			return err
		}
	}

	if *bind {
		var x interface{}
		if err = json.Unmarshal(bs, &x); err != nil {
			return err
		}
		y, err := ps.Bind(ctx, x)
		if err != nil {
			return err
		}

		if bs, err = json.Marshal(&y); err != nil {
			return err
		}
		fmt.Printf("%s\n", bs)
		return nil
	}

	b, err := subst.NewSubber(*delimiters)
	if err != nil {
		return err
	}

	got, err := b.Sub(ctx, ps, string(bs))
	if err != nil {
		return err
	}

	if *checkOut {
		var x interface{}
		if err = json.Unmarshal([]byte(got), &x); err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(os.Stdout, "%s", got)

	return err
}
