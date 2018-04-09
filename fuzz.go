package amnesia

import (
	"context"
	"io"
	"log"
	"os/exec"
	"syscall"
	"time"
)

// used for reporting a crash, find, etc
// should include all necessary info to repeat the crash
// Input and Output can be whatever the user wants for those
type Hit struct {
	Kind   string
	Args   []string
	Input  interface{}
	Output interface{}
}

// ArgFunc should generate command line arguments to be passed
//TODO ArgFunc should also be able to fuzz Env Vars
type ArgFunc func() []string

type FuzzChan struct {
	Result chan Hit
	Status chan *syscall.WaitStatus
	Quit   chan struct{}
}

type ProgFD struct {
	FD   int
	Type uint8     // Could be a buff fuzz thing, a reader, or a writer
	File string    // if this is not nil, then we have a named pipe file to delete
	Pipe io.Closer // needs to be type asserted to a io.WriteCloser or io.ReadCloser
}

// CommFunc will communicate with the program
// CommFunc has final say about when to make and send on a Hit
// send hit on result
// get termination result on status
// quit early if quit closes
type CommFunc func(comset []ProgFD, fc FuzzChan, args []string)

func FuzzArgs(ctx Context, args ArgFunc, comm CommFunc, quit chan struct{}) (results chan Hit, err error) {
	return fuzz(ctx, path, args, comm, quit)
}

func Fuzz(ctx Context, args []string, comm CommFunc, quit chan struct{}) (results chan Hit, err error) {
	return fuzz(ctx, path, args, comm, quit)
}

func fuzz(ctx Context, args interface{}, comm CommFunc, quit chan struct{}) (results chan Hit, err error) {
	// This function starts all the workers

	// check for valid arguments
	switch args.(type) {
	case ArgFunc:
	case []string:
	default:
		log.Fatal("Invalid Args")
	}

	if ctx.WorkerCount <= 0 {
		log.Fatal("Invalid Context WorkerCount")
	}

	if ctx.BufferSize < 0 {
		log.Fatal("Invalid Context BufSize")
	}

	// make the results chan we will pass back to the caller
	// buffer the result chan
	results = make(chan Hit, ctx.BufferSize)

	for i := 0; i < ctx.WorkerCount; i++ {
		go fuzzWorker(ctx, args, comm, results, quit)
	}

	return results, nil
}

func fuzzWorker(ctx Context, args interface{}, comm CommFunc, result chan Hit, quit chan struct{}) {
	var loop bool = true
	for loop {
		// this function loops running a command, and sends back any hits
		var cmd *exec.Cmd

		var strargs []string
		switch v := args.(type) {
		case ArgFunc:
			strargs = v()
		case []string:
			strargs = v
		}

		if ctx.Timeout > 0 {
			timectx, _ := context.WithTimeout(context.Background(), ctx.Timeout)
			cmd = exec.CommandContext(timectx, path, strargs...)
		} else {
			cmd = exec.Command(path, strargs...)
		}

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

		retchan := make(chan *syscall.WaitStatus, 1)

		//TODO change this to work with the pipes thing

		go comm(stdin, stdout, stderr, FuzzChan{result, retchan, quit}, strargs) // start the fuzzer function

		err = cmd.Start()
		if err != nil {
			log.Fatalf("cmd.Start : %v\n", err)
		}

		var status *syscall.WaitStatus
		status = nil

		err = cmd.Wait() // block until program finishes or fails

		if err != nil {
			if exerr, ok := err.(*exec.ExitError); ok {
				// here we get into platform dependent stuff
				// Windows only has a WaitStatus with an ExitCode
				// but that exit code says a lot about what happened
				// Linux has a WaitStatus that implements stuff
				st, ok := exerr.Sys().(syscall.WaitStatus)
				if !ok {
					log.Fatalf("Couldn't asert syscall.WaitStatus type\n")
				}
				status = &st
			} else {
				log.Fatalf("cmd.Wait : %v\n", err)
			}
		}
		//  else program finished successfully, and we pass on a nil WaitStatus

		// by this time the CommFunc could have finished, so that is why retchan needs a buffer
		retchan <- status

		// check if we should stop
		select {
		case <-quit:
			// we are done
			return
		default:
			// keep going
		}
	}
}
