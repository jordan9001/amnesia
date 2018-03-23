package main

import (
	"github.com/jordan9001/amnesia"
	"io"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"
)

var counter int = 0
var counter_mux *sync.Mutex = &sync.Mutex{}

var charset []byte = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz!.,?")

func trySeq(stdin io.WriteCloser, stdout, stderr io.ReadCloser, fc amnesia.FuzzChan, args []string) {
	// basic send brute force sequence of readable-characters
	counter_mux.Lock()
	c := counter
	counter++
	counter_mux.Unlock()

	in := make([]byte, 0, 1024)

	for c > 0 {
		chr := charset[c%len(charset)]
		c = c / len(charset)
		in = append(in, chr)
	}

	stdin.Write(in)

	out := make([]byte, 1024)
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

const max_size int = 8

func tryRand(stdin io.WriteCloser, stdout, stderr io.ReadCloser, fc amnesia.FuzzChan, args []string) {
	// basic send of random readablecharacters
	size := rand.Intn(max_size-1) + 1
	in := make([]byte, size)
	for i := 0; i < size; i++ {
		in[i] = charset[rand.Intn(len(charset))]
	}
	stdin.Write(in)
	out := make([]byte, 1024)
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
	var ctx amnesia.Context

	ctx.WorkerCount = 15
	ctx.BufferSize = 15
	ctx.Timeout = time.Second * 6

	var quitchan chan struct{} = nil
	var args = []string{}

	resseq, _ := amnesia.Fuzz(ctx, "./chal", args, trySeq, quitchan)
	resrand, _ := amnesia.Fuzz(ctx, "./chal", args, tryRand, quitchan)

	for i := 0; i < 45; i++ {
		select {
		case h := <-resseq:
			log.Printf("Got a seq hit of type %q!\n", h.Kind)
			if h.Kind == "Flag" {
				log.Printf("\tInput %q\n", h.Input.(string))
				log.Printf("\tFlag! %q\n", h.Output.(string))
			}
		case h := <-resrand:
			log.Printf("Got a rand hit of type %q!\n", h.Kind)
			if h.Kind == "Flag" {
				log.Printf("\tInput %q\n", h.Input.(string))
				log.Printf("\tFlag! %q\n", h.Output.(string))
			}
		}
	}

	return
}
