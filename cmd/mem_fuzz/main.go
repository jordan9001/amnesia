package main

import (
	"github.com/jordan9001/amnesia"
	"io"
	"log"
	"strings"
	"syscall"
	"time"
	"math/rand"
)

var letters []byte = []byte("JQDPU")

func memfuzz(comset []amnesia.ProgFD, fc amnesia.FuzzChan, args []string) {

	// send the membuffer message
	answer := make([]byte, 0x100)

	//answer[0x80] = 'J'
	//answer[0] = 'Q'
	//answer[1] = 'D'
	//answer[2] = 'D'
	//answer[8] = 'P'
	//answer[255] = 'U'

	for i := 0; i < 0x100; i++ {
		answer[i] = letters[rand.Intn(len(letters))]
	}

	var offset int64 = 8 // second item on the stack is a pointer to the buffer

	amnesia.MemfuzzStackOff(comset[1], offset, answer);

	// see what answer we get from that

	response := make([]byte, 1024)

	readpipe := comset[0].Pipe.(io.ReadCloser)
	n, err := readpipe.Read(response)
	if err != nil {
		log.Printf("No fun!\n")
		log.Fatal(err)
	}
	if strings.Contains(string(response[:n]), "Success") {
		log.Printf("Got a response! : %q\n", string(response[:n]))
		log.Printf("Input was %q\n", string(answer))
		log.Fatal("Done!")
	}
}

func main() {
	var ctx *amnesia.Context = &amnesia.Context{}

	ctx.WorkerCount = 3
	ctx.BufferSize = 1
	ctx.Timeout = time.Second * 3
	ctx.Path = "./target"
	ctx.InfectionAddr = 0x00000000004005ad

	stdout := amnesia.ProgFD{1, amnesia.PROG_OUTPUT_FD, "", nil}
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
