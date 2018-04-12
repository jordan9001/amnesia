package amnesia

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
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

// CommFunc will communicate with the program
// CommFunc has final say about when to make and send on a Hit
// send hit on result
// get termination result on status
// quit early if quit closes
type CommFunc func(comset []ProgFD, fc FuzzChan, args []string)

func FuzzArgs(ctx *Context, args ArgFunc, comm CommFunc, quit chan struct{}) (results chan Hit, err error) {
	if InfectedContext(ctx) {
		return nil, fmt.Errorf("Can not use FuzzArgs on an infected fuzzing context")
	}
	return fuzz(ctx, path, args, comm, quit)
}

func Fuzz(ctx *Context, args []string, comm CommFunc, quit chan struct{}) (results chan Hit, err error) {
	return fuzz(ctx, path, args, comm, quit)
}

func fuzz(ctx *Context, args interface{}, comm CommFunc, quit chan struct{}) (results chan Hit, err error) {
	// This function starts all the workers

	// check for valid arguments

	if ctx.WorkerCount <= 0 {
		return nil, fmt.Errorf("Invalid Context WorkerCount")
	}

	if ctx.BufferSize < 0 {
		return nil, fmt.Errorf("Invalid Context BufSize")
	}

	// make the results chan we will pass back to the caller
	// buffer the result chan
	results = make(chan Hit, ctx.BufferSize)

	// This is where we should call instrument, when the ctx is finalized
	if InfectedContext(ctx) {
		strargs, ok := args.([]string)
		if !ok {
			return nil, fmt.Errorf("Invalid arguments to a infected fuzzing target")
		}

		// infect the binary for us to do the things
		pipes, ctxn, err = instrument(ctx)

		for i := 0; i < ctx.WorkerCount; i++ {
			ctxn, err := instrument(ctx, "." + strconv.Itoa(i))
			// Create the infect handling workers, handing them their named pipes to use
			go fuzzInfectedWorker(ctxn, strargs, comm, results, quit)
		}
	} else {
		switch args.(type) {
		case ArgFunc:
		case []string:
		default:
			log.Fatal("Invalid Args")
		}
		for i := 0; i < ctx.WorkerCount; i++ {
			go fuzzWorker(ctx, args, comm, results, quit)
		}
	}

	return results, nil
}

func fuzzInfectedWorker(ctx *Context, args []string, comm CommFunc, result chan Hit, quit chan struct{}) {
	var loop bool = true
	fserv := exec.Command(ctx.Path, args...)

	stdin, err := fserv.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	stdout, err := fserv.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, 4)

	for loop {
		_, err = stdin.Write([]byte{1})
		if err != nil {
			log.Fatal(err)
		}
		// get the response of the pid so we can kill the forked one if timeout from stdout
		_, err = stdout.Read(buf)
		if err != nil {
			log.Fatal(err)
		}

		var pid int32
		binary.Read(buf, binary.LittleEndian, &pid)

		p, err := os.FindProcess(pid)
		if err != nil {
			log.Fatal(err)
		}

		retchan := make(chan *syscall.WaitStatus, 1)

		// pfds are in the ctx

		go comm(ctx.FDs, FuzzChan{result, retchan, quit}, args) // start the fuzzer function

		// do a timeout signal
		if ctx.Timeout != 0 {
			go func() {
				time.Sleep(ctx.Timeout)
				p.Kill()
			}()
		}

		// TODO move linux specific code to a _linux file
		var status *syscall.WaitStatus
		status = nil

		pstate, err := os.Wait(p)
		if err != nil {
			log.Fatal(err)
		}

		if err != nil {
			if exerr, ok := err.(*exec.ExitError); ok {
				// here we get into platform dependent stuff
				// Windows only has a WaitStatus with an ExitCode
				// but that exit code says a lot about what happened
				// Linux has a WaitStatus that implements stuff
				st, ok := exerr.Sys().(syscall.WaitStatus)
				if !ok {
					log.Fatalf("Couldn't assert syscall.WaitStatus type\n")
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
			loop = false
		default:
			// keep going
		}
	}

	// kill the fork server
	fserv.Kill()
	// TODO cleanup the named_pipes passed to us
}

func fuzzWorker(ctx *Context, args interface{}, comm CommFunc, result chan Hit, quit chan struct{}) {
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

		// Handle the file descriptor things
		pfds := make([]ProgFD, 3)

		stdin, err := cmd.StdinPipe()
		if err != nil {
			log.Fatal(err)
		}
		pfds[0] = ProgFD{FD: 0, Type: PROG_INPUT_FD, Pipe: stdin}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}
		pfds[1] = ProgFD{FD: 1, Type: PROG_OUTPUT_FD, Pipe: stdout}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Fatal(err)
		}
		pfds[2] = ProgFD{FD: 2, Type: PROG_OUTPUT_FD, Pipe: stdout}

		retchan := make(chan *syscall.WaitStatus, 1)

		go comm(pfds, FuzzChan{result, retchan, quit}, strargs) // start the fuzzer function

		err = cmd.Start()
		if err != nil {
			log.Fatalf("cmd.Start : %v\n", err)
		}

		// TODO move linux specific code to a _linux file
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
					log.Fatalf("Couldn't assert syscall.WaitStatus type\n")
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
			loop = false
		default:
			// keep going
		}
	}
}
