package amnesia

import (
	"time"
)

type Context struct {
	Path          string
	WorkerCount   int
	BufferSize    int
	Timeout       time.Duration

	// the below items only refer to infected setups
	FDs           []ProgFD
	InfectionAddr uint64
	InfectionSym  string
	Setup         SetupFunc
}
// TODO argsfunc and args should maybe be in the context?

// when fuzzing first thing is to make a context.
// Then you can add fd pipes and memfuzz things, if you want

func InfectedContext(ctx *Context) bool {
	return (len(ctx.FDs) > 0 || ctx.InfectionAddr != 0 || ctx.InfectionSym != "")
}

func (c *Context) Cleanup() {
	// cleans up leftover named pipes and infected files
	// should be called on the infected generated contexts after the worker is cleaning up
	if InfectedContext(c) {
		cleanupInfection(c)
	}
}

func (c *Context) Copy() *Context {
	var nc Context

	nc = *c

	nfds := make([]ProgFD, len(c.FDs))

	for i, v := range c.FDs {
		nfds[i] = v
	}

	nc.FDs = nfds

	return &nc
}
