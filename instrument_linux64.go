// +build linux,amd64

package amnesia

import (
	"debug/elf"
	"fmt"
	"io"
	"log"
	"os"
	"syscall"
)

const progpre string = ".amninfected_"
const infectFile string = "./linux64_hook.bin"
const packageFile string = "./linux64_package.bin"

func symAddr(path string, symbol string) (uint64, error) {

	return 0, fmt.Errorf("Infection by Symbol unsupported right now")
}

func addr2fileoff(f elf.File, addr uint64) (uint64, error) {
	for _, s := range f.Sections {
		if addr >= s.Addr && addr < (s.Addr+s.Size) {
			// found the correct section
			return (addr - s.Addr) + s.Offset, nil
		}
	}
	return 0, fmt.Errorf("Unable to find section in file at virtual address 0x%x\n", addr)
}

func instrument(ctx *Context) (*Context, error) {
	if ctx.Path == "" {
		return ctx, fmt.Errorf("Empty path in context to insturment")
	}

	if ctx.InfectionAddr == 0 {
		if ctx.InfectionSym == "" {
			return ctx, fmt.Errorf("No specified point of infection")
		}
		ctx.InfectionAddr, err = symAddr(ctx.Path, ctx.InfectionSym)
		if err != nil {
			return ctx, err
		}
	}

	// copy binary
	orig, err := os.Open(ctx.Path)
	if err != nil {
		return ctx, err
	}
	defer s.Close()

	dst_path := progpre + path

	d, err := os.Create(dst_path)
	if err != nil {
		return ctx, err
	}

	_, err = io.Copy(d, orig)
	if err != nil {
		return ctx, err
	}

	d.Close()

	// find file location of address
	f, err := elf.Open(dst_path)
	if err != nil {
		return ctx, err
	}

	foff, err := addr2fileoff(f, addr)
	f.Close()
	if err != nil {
		return ctx, err
	}

	log.Printf("Would have infected at %x\n", foff)
	os.Exit(0)

	// get infection
	hook, err := os.Open(infectFile)

	// read what will be overwritten
	// overwrite
	// customize the package
	// have to create pipe files here for each worker
	// because each package needs the path to it's workers pipes
	return ctx, nil
}

func createPipe(path string) error {
	return syscall.Mkfifo(path, 0666)
}

func cleanInstrumentation(ctx *Context) error {
	return os.Remove(ctx.Path)
}
