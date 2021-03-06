package amnesia

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"sync"
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

type FuzzChan struct {
	Result chan Hit
	Status chan *syscall.WaitStatus
	Quit   chan struct{}
}

// ArgFunc should generate command line arguments to be passed
//TODO ArgFunc should also be able to fuzz Env Vars, and should be in the context
type ArgFunc func() []string

// CommFunc will communicate with the program
// CommFunc has final say about when to make and send on a Hit
// send hit on result
// get termination result on status
// quit early if quit closes
type CommFunc func(comset []ProgFD, fc FuzzChan, args []string)

// SetupFunc is a function that will communicate with the program before it is handed off to CommFunc
// it is useful for preping a command before a memfuzz or other infected fuzz
// the setupfunc will only ever recieve fds stdin, stdout, and stderr
type SetupFunc func(comset []ProgFD)

func FuzzArgs(ctx *Context, args ArgFunc, comm CommFunc, quit chan struct{}) (results chan Hit, err error) {
	if InfectedContext(ctx) {
		return nil, fmt.Errorf("Can not use FuzzArgs on an infected fuzzing context")
	}
	return fuzz(ctx, args, comm, quit)
}

func Fuzz(ctx *Context, args []string, comm CommFunc, quit chan struct{}) (results chan Hit, err error) {
	return fuzz(ctx, args, comm, quit)
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

		if len(ctx.FDs) == 0 {
			return nil, fmt.Errorf("Empty File Descriptor list for infected") //TODO add standard 3 with function
		}

		// infect the binary for us to do the things
		for i := 0; i < ctx.WorkerCount; i++ {
			ctxn, err := Instrument(ctx, "." + strconv.Itoa(i))
			if err != nil {
				return nil, err
			}
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

	var stderr io.ReadCloser = nil
	if ctx.Setup != nil {
		stderr, err = fserv.StderrPipe()
		if err != nil {
			log.Fatal(err)
		}
	}

	// start up the infected binary
	err = fserv.Start()
	if err != nil {
		log.Fatalf("cmd.Start : %v\n", err)
	}
	log.Printf("Infection Running\n")


	// initialize the program
	if ctx.Setup != nil {
		pfds := make([]ProgFD, 3)

		pfds[0] = ProgFD{FD: 0, Type: PROG_INPUT_FD, Pipe: stdin}
		pfds[1] = ProgFD{FD: 1, Type: PROG_OUTPUT_FD, Pipe: stdout}
		pfds[2] = ProgFD{FD: 2, Type: PROG_OUTPUT_FD, Pipe: stderr}

		ctx.Setup(pfds)
	}

	buf := make([]byte, 4)
	waitmut := &sync.Mutex{}
	i := 0

	// to attach gdb for debug
	//time.Sleep(time.Second * 30)

	for loop {
		retchan := make(chan *syscall.WaitStatus, 1)

		waitmut.Lock()
		go func() {
			// open the pipes
			for i, _ := range ctx.FDs {
				ctx.FDs[i].Open()
			}
			// pfds are in the ctx

			comm(ctx.FDs, FuzzChan{result, retchan, quit}, args) // start the fuzzer func

			// gotta close the pipes again
			for i, _ := range ctx.FDs {
				ctx.FDs[i].Close()
			}
			// let the thing know it can continue
			// We should keep track of how many programs are there, cause if we go too fast here we die real quick :(
			// ugh
			log.Printf("%d\n", i)
			i++
			waitmut.Unlock()
		}()

		_, err = stdin.Write([]byte{1})
		if err != nil {
			log.Fatal(err)
		}

		// get the response of the pid so we can kill the forked one if timeout from stdout
		_, err = stdout.Read(buf)
		if err != nil {
			log.Printf("Unable to read PID\n");
			log.Fatal(err)
		}

		var pid int32
		pid = int32(binary.LittleEndian.Uint32(buf))

		p, err := os.FindProcess(int(pid))
		if err != nil {
			log.Printf("Unable to find the process!\n")
			log.Fatal(err)
		}

		// do a timeout signal
		if ctx.Timeout != 0 {
			go func() {
				time.Sleep(ctx.Timeout)
				p.Kill()
			}()
		}

		// Go wont let us wait on the child of a child :(
		// so now the assembly code waits for us, so we just do another read
		_, err = stdout.Read(buf)
		if err != nil {
			log.Printf("Unable to read the status!\n")
			log.Fatal(err)
		}

		var status syscall.WaitStatus
		status = syscall.WaitStatus(binary.LittleEndian.Uint32(buf))
		// This isn't working
		// also my pipes fail if the program segfaults? Thats bad
		// TODO

		// by this time the CommFunc could have finished, so that is why retchan needs a buffer
		retchan <- &status

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
	fserv.Process.Kill()
	// cleanup the named_pipes passed to us
	ctx.Cleanup()
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
			cmd = exec.CommandContext(timectx, ctx.Path, strargs...)
		} else {
			cmd = exec.Command(ctx.Path, strargs...)
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
		pfds[2] = ProgFD{FD: 2, Type: PROG_OUTPUT_FD, Pipe: stderr}

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
