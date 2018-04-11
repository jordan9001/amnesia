package main

import (
	"log"
	"runtime"
)

func main() {
	log.Printf("Building for platform %v %v\n", runtime.GOOS, runtime.GOARCH)
}
