package main

import (
	"github.com/jordan9001/amnesia"
	"log"
	"math/rand"
	"strings"
	"time"
	"syscall"
)

const max_size = 0x100

func tryRand(comset []amnesia.ProgFD, fc amnesia.FuzzChan, args []string) {
	stdin, stdout, _ := amnesia.GetStdPipes(comset)

	// basic send of random readablecharacters
	size := rand.Intn(max_size-2) + 2
	in := make([]byte, size)
	for i := 0; i < size-1; i++ {
		in[i] = byte(rand.Int())
	}
	in[size-1] = 0x0a

	stdin.Write(in)

	out := make([]byte, 1024)
	n, _ := stdout.Read(out)

	s := string(out[:n])

	//log.Printf("in: %q out %q\n", string(in), s)

	if strings.Contains(strings.ToLower(s), "flag") {
		// got a flag hit!
		h := amnesia.Hit{
			Kind:   "Flag",
			Input:  string(in),
			Output: s,
		}
		fc.Result <- h
	}

	amnesia.ReportFaults(string(in), fc)
}

func trySmart(comset []amnesia.ProgFD, fc amnesia.FuzzChan, args []string) {
	stdin, stdout, _ := amnesia.GetStdPipes(comset)

	size := rand.Intn(0x80)
	in := make([]byte, 0)
	for i:=0; i<size; i++ {
		in = append(in, []byte("%x")...)
	}
	in = append(in, []byte("%s\n")...)

	stdin.Write(in)

	out := make([]byte, 0x400)
	n, _ := stdout.Read(out)

	s := string(out[:n])

	if strings.Contains(strings.ToLower(s), "flag") {
		// got a flag hit!
		h := amnesia.Hit{
			Kind:   "Flag",
			Input:  string(in),
			Output: s,
		}
		fc.Result <- h
	}
}

func main() {
	var ctx *amnesia.Context = &amnesia.Context{}

	ctx.WorkerCount = 60
	ctx.BufferSize = 30
	ctx.Timeout = time.Second * 3
	ctx.Path = "./basic"

	var quitchan chan struct{} = nil
	var args = []string{}

	resRand, _ := amnesia.Fuzz(ctx, args, tryRand, quitchan)
	resSmart, _ := amnesia.Fuzz(ctx, args, trySmart, quitchan)

	var h amnesia.Hit
	for {
		select {
		case h = <-resRand:
		case h = <-resSmart:
		}
		log.Printf("Got a seq hit of type %q!\n", h.Kind)
		if h.Kind == "Flag" {
			log.Printf("\tInput %q\n", h.Input.(string))
			log.Printf("\tFlag! %q\n", h.Output.(string))
		} else if h.Kind == "Signal" {
			log.Printf("\tInput %q\n", h.Input.(string))
			log.Printf("\tSignal %v\n", h.Output.(syscall.Signal))
		}
	}

	return
}
