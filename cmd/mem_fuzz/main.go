package main

import (
	"github.com/jordan9001/amnesia"
	"log"
	"os"
	"syscall"
	"time"
)

func prefuzz(comset []amnesia.ProgFD) {
	log.Printf("Sending initial communication with program before the fork server\n")
	stdin, stdout, _ := amnesia.GetStdPipes(comset)

	welcome_msg := make([]byte, 1024)
	n, err := stdout.Read(welcome_msg)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s\n", string(welcome_msg[:n]))

	stdin.Write([]byte("AAAA"))

	log.Printf("Initialized program\n")
}

func memfuzz(comset []amnesia.ProgFD, fc amnesia.FuzzChan, args []string) {

	log.Printf("Got to memfuzz!\n")
	os.Exit(0)
}

func main() {
	var ctx *amnesia.Context = &amnesia.Context{}

	ctx.WorkerCount = 1
	ctx.BufferSize = 1
	ctx.Timeout = time.Second * 3
	ctx.Path = "./target"
	ctx.InfectionAddr = 0x0000000000400603

	ctx.Setup = prefuzz

	stdout := amnesia.ProgFD{0, amnesia.PROG_OUTPUT_FD, "", nil}
	memfuz := amnesia.ProgFD{-1, amnesia.MEM_FUZZ_FD, "", nil}
	ctx.FDs = []amnesia.ProgFD{stdout, memfuz}

	var quitchan chan struct{} = nil

	res, err := amnesia.Fuzz(ctx, []string{}, memfuzz, quitchan)
	if err != nil {
		log.Fatal(err)
	}

	var h amnesia.Hit
	for {
		h = <-res
		log.Printf("Got a seq hit of type %q!\n", h.Kind)
		if h.Kind == "Success" {
			log.Printf("\tInput %q\n", h.Input.(string))
			log.Printf("\tFlag! %q\n", h.Output.(string))
		} else if h.Kind == "Signal" {
			log.Printf("\tInput %q\n", h.Input.(string))
			log.Printf("\tSignal %v\n", h.Output.(syscall.Signal))
		}
	}

	return
}
