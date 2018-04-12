package amnesia

import (
	"context"
	"io"
	"log"
	"os/exec"
	"syscall"
	"time"
)

type Context struct {
	Path          string
	WorkerCount   int
	BufferSize    int
	Timeout       time.Duration
	FDs           []ProgFD
	InfectionAddr uint64
	InfectionSym  string
}

// when fuzzing first thing is to make a context.
// Then you can add fd pipes and memfuzz things, if you want

func InfectedContext(ctx *Context) bool {
	return (len(ctx.FDs) > 0 || ctx.InfectionAddr != 0 || ctx.InfectionSym != "")
}

func CleanupContext(ctx *Context) {
	// cleans up leftover named pipes and infected files
	if InfectedContext(ctx) {
		cleanInstrumentation(ctx)
	}
}

func (c *Context) Copy() *Context {
	var nc Context

	nc = *c

	nfds = make([]ProgFD, len(c.FDs))

	for i, v := range c.FDs {
		nfds[i] = v
	}

	nc.FDs = nfds

	return &nc
}
