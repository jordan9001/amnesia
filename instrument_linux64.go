// +build linux,amd64

package amnesia

import (
	"encoding/binary"
	"debug/elf"
	"fmt"
	"io"
	"os"
	"strconv"
	"syscall"
)

const progpre string = ".amn_"
const infectFile string = "./hook.bin"
const packageFile string = "./package.bin"

func symAddr(path string, symbol string) (uint64, error) {

	return 0, fmt.Errorf("Infection by Symbol unsupported right now")
}

func addr2fileoff(f *elf.File, addr uint64) (uint64, error) {
	for _, s := range f.Sections {
		if addr >= s.Addr && addr < (s.Addr+s.Size) {
			// found the correct section
			return (addr - s.Addr) + s.Offset, nil
		}
	}
	return 0, fmt.Errorf("Unable to find section in file at virtual address 0x%x\n", addr)
}

func Instrument(ctx *Context, suffix string) (*Context, error) {
	if ctx.Path == "" {
		return ctx, fmt.Errorf("Empty path in context to insturment")
	}

	if ctx.InfectionAddr == 0 {
		if ctx.InfectionSym == "" {
			return ctx, fmt.Errorf("No specified point of infection")
		}
		iAddr, err := symAddr(ctx.Path, ctx.InfectionSym)
		if err != nil {
			return ctx, err
		}
		ctx.InfectionAddr = iAddr
	}

	// copy binary
	orig, err := os.Open(ctx.Path)
	if err != nil {
		return ctx, err
	}
	defer orig.Close()

	dst_path := progpre + ctx.Path + suffix

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

	foff, err := addr2fileoff(f, ctx.InfectionAddr)
	f.Close()
	if err != nil {
		return ctx, err
	}

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

	hook_buf := make([]byte, hook_info.Size(), hook_info.Size()+int64(len(packageFile))) // greater cap for package path

	_, err = hook.Read(hook_buf)
	if err != nil {
		return ctx, err
	}

	// fill in package file size var
	binary.LittleEndian.PutUint64(hook_buf[len(hook_buf)-8:], uint64(len(packageFile)))

	// fill in path to package file
	hook_buf = append(hook_buf, []byte(packageFile)...)

	// read what will be overwritten

	orig_buf := make([]byte, len(hook_buf))

	n, err := d.ReadAt(orig_buf, int64(foff))
	if err != nil {
		return ctx, err
	}
	if n != len(orig_buf) {
		return ctx, fmt.Errorf("Could not insert hook at %x\n", foff)
	}

	// overwrite
	_, err = d.WriteAt(hook_buf, int64(foff))
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
	pa_len := 0x18 * len(ctx.FDs) + 8 + len(orig_buf)
	pack_app_buf := make([]byte, pa_len)

	pack_buf := make([]byte, pack_info.Size(), pack_info.Size() + int64(len(pack_app_buf)))

	_, err = pack.Read(pack_buf)
	if err != nil {
		return ctx, err
	}

	// write hook_pos
	var v_hook_pos int64
	// according to the disassember, the ret addr will point 0x49 bytes off from the patch start
	v_hook_pos = -0x49
	poff := len(pack_buf) - (8 * 3)
	binary.LittleEndian.PutUint64(pack_buf[poff:poff+8], uint64(v_hook_pos))

	// write hook_off
	var v_hook_off int64
	// distance from VAR_START to the hook size field
	v_hook_off = int64((0x18 * len(ctx.FDs)) + (3 * 8))
	poff = len(pack_buf) - (8 * 2)
	binary.LittleEndian.PutUint64(pack_buf[poff:poff+8], uint64(v_hook_off))

	// write num pipes
	var v_num_pipes int64
	v_num_pipes = int64(len(ctx.FDs))
	poff = len(pack_buf) - (8 * 1)
	binary.LittleEndian.PutUint64(pack_buf[poff:poff+8], uint64(v_num_pipes))

	// create a new context
	nctx := ctx.Copy()
	nctx.Path = dst_path

	// create the named pipes
	// append the info
	for i, _ := range nctx.FDs {
		// first create the fifo
		if nctx.FDs[i].File == "" {
			nctx.FDs[i].File = strconv.Itoa(nctx.FDs[i].FD)
			switch (nctx.FDs[i].Type) {
			case PROG_INPUT_FD:
				nctx.FDs[i].File += "INN"
			case PROG_OUTPUT_FD:
				nctx.FDs[i].File += "OUT"
			case MEM_FUZZ_FD:
				nctx.FDs[i].File += "FUZ"
			default:
				return ctx, fmt.Errorf("Unknown ProfFD type")
			}
		}
		nctx.FDs[i].File += suffix

		pipe, err := createPipe(nctx.FDs[i].File, nctx.FDs[i].Type)
		if err != nil {
			return ctx, err
		}

		nctx.FDs[i].Pipe = pipe

		// then get the serialized version
		fd_buf, err := nctx.FDs[i].Pack()
		if err != nil {
			return ctx, err
		}
		pack_buf = append(pack_buf, fd_buf...)
	}

	// append unpatch len and unpatch
	unpatch_buf := make([]byte, len(orig_buf) + 8)
	binary.LittleEndian.PutUint64(unpatch_buf, uint64(len(nctx.FDs)))
	copy(unpatch_buf[8:], orig_buf)

	pack_buf = append(pack_buf, unpatch_buf...)

	return nctx, nil
}

func createPipe(path string, t fdtype) (io.Closer, error) {
	err := syscall.Mkfifo(path, 0666)
	if err != nil {
		return nil, err
	}

	var pipe io.Closer
	if t == PROG_INPUT_FD || t == MEM_FUZZ_FD {
		pipe, err = os.OpenFile(path, os.O_WRONLY, os.ModeNamedPipe)
	} else {
		pipe, err = os.OpenFile(path, os.O_RDONLY, os.ModeNamedPipe)
	}

	if err != nil {
		return nil, err
	}

	return pipe, nil
}

func cleanupInfection(ctx *Context) error {
	for i, _ := range ctx.FDs {
			ctx.FDs[i].Pipe.Close()
			os.Remove(ctx.FDs[i].File)
	}
	return os.Remove(ctx.Path)
}
