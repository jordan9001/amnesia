package main

import (
	"fmt"
	"github.com/jordan9001/amnesia"
	"math/rand"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage %s string to mutate\n", os.Args[1])
		return
	}

	rand.Seed(time.Now().UTC().UnixNano())

	s := strings.Join(os.Args[1:], " ")

	for i := 0; i < 30; i++ {
		fmt.Printf("%d:\t%q\n", i, amnesia.Mutate(s, i))
	}
}
