package subst

import (
	"context"
	"log"
)

type Ctx struct {
	context.Context
	IncludeDirs []string
	Tracing     bool
}

func NewCtx(ctx context.Context, dirs []string) *Ctx {
	return &Ctx{
		Context:     ctx,
		IncludeDirs: dirs,
	}
}

func (c *Ctx) Copy() *Ctx {
	return &Ctx{
		Context:     c.Context,
		IncludeDirs: c.IncludeDirs,
		Tracing:     c.Tracing,
	}
}

func (c *Ctx) trf(format string, args ...interface{}) {
	if c != nil && !c.Tracing {
		return
	}
	log.Printf(format, args...)
}

func (c *Ctx) Logf(format string, args ...interface{}) {
	log.Printf(format, args...)
}
