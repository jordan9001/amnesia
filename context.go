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
	Path	    string
	WorkerCount int
	BufferSize  int
	Timeout     time.Duration
	FDs	    []ProgFD
}


//TODO make a bunch of context genorators
// Should also do early checks, like checking the file exists?
// Have a MemFuzz context creator
func StandardContext() Context {
	//TODO
}
