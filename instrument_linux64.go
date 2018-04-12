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

const progpre string = ".amn_"
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

func instrument(ctx *Context, suffix string) (*Context, error) {
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

	dst_path := progpre + path + suffix

	d, err := os.Create(dst_path)
	if err != nil {
		return ctx, err
	}
	defer d.Close()

	_, err = io.Copy(d, orig)
	if err != nil {
		return ctx, err
	}

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
	if err != nil {
		return ctx, err
	}
	defer hook.Close()

	// get size
	hook_info, err := hook.Stat()
	if err != nil {
		return ctx, err
	}

	hook_buf := make([]byte, hook_info.Size(), hook_info.Size()+len(packageFile)) // greater cap for package path

	_, err = hook.Read(hook_buf)
	if err != nil {
		return ctx, err
	}

	// fill in package file size var
	binary.LittleEndian.PutUint64(hook_buf[len(hook_buf)-8:], len(packageFile))

	// fill in path to package file
	hook_buf = append(hook_buf, []byte(packageFile)...)

	// read what will be overwritten

	orig_buf := make([]byte, len(hook_buf))

	n, err := d.ReadAt(orig_buf, foff)
	if err != nil {
		return ctx, err
	}
	if n != len(orig_buf) {
		return ctx, fmt.Errorf("Could not insert hook at %x\n", foff)
	}

	// overwrite
	_, err := d.WriteAt(hook_buf, foff)
	if err != nil {
		return ctx, err
	}

	// customize the package
	pack, err := os.Open(packageFile)
	if err != nil {
		return ctx, err
	}

	pack_info, err := pack.Stat()
	if err != nil {
		return ctx, err
	}

	// the vars to be appended are the fd_pipe infos and then un-patch len then un-patch
	pa_len = 0x18 * len(ctx.FDs) + 8 + len(orig_buf)
	pack_app_buf := make([]byte, pa_len)

	pack_buf := make([]byte, pack_info.Size(), pack_info.Size() + len(pack_app_buf))

	_, err = pack.Read(pack_buf)
	if err != nil {
		return ctx, err
	}

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
