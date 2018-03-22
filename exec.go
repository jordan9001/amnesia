package amnesia

import (
	"context"
	"io"
	"log"
	"os/exec"
	"syscall"
)

type Context struct {
	context.Context
	WorkerCount int
	BufSize int
}

// used for reporting a crash, find, etc
// should include all necessary info to repeat the crash
// Stop can indicate fuzzing is complete for that run, even if there is no real hit
// a nil Kind indicates the hit can be discarded
type Hit struct {
	Stop bool
	Kind string
	Args []string
	Input []string
}

// ArgFunc should generate command line arguments to be passed
type ArgFunc func() []string

// CommFunc will communicate with the program
// CommFunc can report results it finds through communication (Found a Flag{...}, etc)
// Comm func can tell the worker to stop fuzzing that instance by returning a hit with Stop == true
type CommFunc func(stdin io.WriteCloser, stdout, stderr io.ReadCloser, result chan Hit, quit chan struct{})

func Fuzz(ctx Context, args FuzzFunc, comm CommFunc, quit chan struct{}) (results chan Hit) {
	// This function starts all the workers, and listens for hits

	// buffer the result chan
	if args == nil {
		args = func() []string {
			return []string{}
		}
	}

	// make the results chan we will pass back to the caller
	results := make(chan Hit, ctx.BufSize)


}

func fuzzWorker(ctx Context, args ArgFunc, comm CommFunc, result chan Hit, quit chan struct{}) {
	for {
		// this function loops running a command, and sends back any hits
		cmd := exec.ContextCommand(ctx, name, args()...)

		var comres chan Hit = nil
		var comquit chan struct{} = nil

		if comm != nil {
			stdin, err := cmd.StdinPipe()
			if err != nil {
				log.Fatal(err)
			}
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				log.Fatal(err)
			}
			stderr, err := cmd.StderrPipe()
			if err != nil {
				log.Fatal(err)
			}

			comres = make(chan Hit)
			comquit = make(chan struct{})
			go comm(stdin, stdout, stderr, comres, comquit)
		}

		cmd.Start()

		// set up a 

		for {
			select {
			case hit := <-comres:
				dostop := hit.Stop
				if hit.Kind != nil {
					//send the hit on
					hit.Stop = false
					result <- hit
				}
				if dostop {
					// stop the process and don't wait to see result
					cmd.Process.Kill()
					break
				}
			case hit := <-cmdres:
				if hit.Kind != nil {
					//send the hit on
					hit.Stop = false
					result <- hit
				}
				break
			case <-quit:
				// we are done
				if comquit != nil {
					close comquit
				}
				return
			}
		}

		// we are done, tell the communicator
		if comquit != nil {
			close(comquit)
		}
	}
}

func waitStatus2String(stat syscall.WaitStatus) string {
	//TODO
	return nil
}
