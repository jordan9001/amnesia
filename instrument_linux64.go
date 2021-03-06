// +build linux,amd64

package amnesia

import (
	"encoding/binary"
	"debug/elf"
	"fmt"
	"io"
	"os"
	"strconv"
	"path"
)

const progpre string = "amn_"
var infectFile string = "./hook.bin"
var packageFile string = "./package.bin"

func symAddr(sympath string, symbol string) (uint64, error) {

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

func SetHookPath(fpath string) {
	infectFile = fpath
}

func SetPackagePath(fpath string) {
	packageFile = fpath
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

	dst_path := path.Dir(ctx.Path) + "/" + progpre + path.Base(ctx.Path) + suffix

	d, err := os.OpenFile(dst_path, os.O_RDWR|os.O_CREATE, 0755)
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

	// set the path for our new packageFile
	pkpath := path.Dir(packageFile) + "/" + progpre + path.Base(packageFile) + suffix

	// read what will be overwritten
	hook_size := 0x56 + len(pkpath) + 1

	orig_buf := make([]byte, hook_size)

	n, err := d.ReadAt(orig_buf, int64(foff))
	if err != nil {
		return ctx, err
	}
	if n != len(orig_buf) {
		return ctx, fmt.Errorf("Could not insert hook at %x\n", foff)
	}

	// open the generic package
	pack, err := os.Open(packageFile)
	if err != nil {
		return ctx, err
	}
	defer pack.Close()

	// make the package buf
	nctx, pack_buf, err := makePackBuf(ctx, pack, orig_buf, suffix)
	if err != nil {
		return ctx, err
	}
	nctx.Path = dst_path

	// get generic hook
	hook, err := os.Open(infectFile)
	if err != nil {
		return ctx, err
	}
	defer hook.Close()

	// make the hook buf
	hook_buf, err := makeHookBuf(nctx, hook, uint64(len(pack_buf)), pkpath)
	if err != nil {
		return ctx, err
	}

	// overwrite in copy of the bin
	_, err = d.WriteAt(hook_buf, int64(foff))
	if err != nil {
		return ctx, err
	}

	// write out package file
	infpk, err := os.Create(pkpath)
	if err != nil {
		return ctx, err
	}
	defer infpk.Close()

	_, err = infpk.Write(pack_buf)
	if err != nil {
		return ctx, err
	}

	return nctx, nil
}

func makePackBuf(ctx *Context, pack *os.File, orig_buf []byte, suffix string) (*Context, []byte, error) {

	pack_info, err := pack.Stat()
	if err != nil {
		return ctx, nil, err
	}

	pack_end_size := (0x18 * len(ctx.FDs)) + len(orig_buf) + 8
	pack_buf := make([]byte, pack_info.Size(), pack_info.Size() + int64(pack_end_size))

	_, err = pack.Read(pack_buf)
	if err != nil {
		return ctx, nil, err
	}

	// write hook_pos
	var v_hook_pos int64
	// according to the disassember, the ret addr will point 0x49 bytes off from the patch start
	v_hook_pos = 0x41
	poff := len(pack_buf) - (8 * 4)
	binary.LittleEndian.PutUint64(pack_buf[poff:poff+8], uint64(v_hook_pos))

	// write hook_off
	var v_hook_off int64
	// distance from VAR_START to the hook size field
	v_hook_off = int64((0x18 * len(ctx.FDs)) + (4 * 8)) // update this number if you add more vars
	poff = len(pack_buf) - (8 * 3)
	binary.LittleEndian.PutUint64(pack_buf[poff:poff+8], uint64(v_hook_off))

	// write mprot_off
	var v_pagesz_off int64
	v_pagesz_off = int64(os.Getpagesize())
	// turn it into a mask
	v_pagesz_off = (-1 ^ (v_pagesz_off - 1))
	poff = len(pack_buf) - (8 * 2)
	binary.LittleEndian.PutUint64(pack_buf[poff:poff+8], uint64(v_pagesz_off))

	// write num pipes
	var v_num_pipes int64
	v_num_pipes = int64(len(ctx.FDs))
	poff = len(pack_buf) - (8 * 1)
	binary.LittleEndian.PutUint64(pack_buf[poff:poff+8], uint64(v_num_pipes))

	// create a new context
	nctx := ctx.Copy()

	// create the named pipes
	// append the info
	for i, _ := range nctx.FDs {
		// first create the fifo
		if nctx.FDs[i].File == "" {
			nctx.FDs[i].File = progpre + strconv.Itoa(nctx.FDs[i].FD)
			switch (nctx.FDs[i].Type) {
			case PROG_INPUT_FD:
				nctx.FDs[i].File += "INN"
			case PROG_OUTPUT_FD:
				nctx.FDs[i].File += "OUT"
			case MEM_FUZZ_FD:
				nctx.FDs[i].File += "FUZ"
			default:
				return ctx, nil, fmt.Errorf("Unknown ProfFD type")
			}
		}
		nctx.FDs[i].File += suffix

		err := createPipe(nctx.FDs[i].File, nctx.FDs[i].Type)
		if err != nil {
			return ctx, nil, err
		}

		// then get the serialized version
		fd_buf, err := nctx.FDs[i].Pack()
		if err != nil {
			return ctx, nil, err
		}
		pack_buf = append(pack_buf, fd_buf...)
	}

	unpatch_buf := make([]byte, len(orig_buf) + 8)
	binary.LittleEndian.PutUint64(unpatch_buf, uint64(len(orig_buf)))
	copy(unpatch_buf[8:], orig_buf)
	// append unpatch len and unpatch
	pack_buf = append(pack_buf, unpatch_buf...)

	return nctx, pack_buf, nil
}

func makeHookBuf(ctx *Context, hook *os.File, packageSize uint64, packagePath string) ([]byte, error) {
	// get size
	hook_info, err := hook.Stat()
	if err != nil {
		return nil, err
	}

	packagePath += "\x00" // add a null

	// +1 for null
	hook_buf := make([]byte, hook_info.Size(), hook_info.Size()+int64(len(packagePath)))

	_, err = hook.Read(hook_buf)
	if err != nil {
		return nil, err
	}

	// fill in package file size var
	binary.LittleEndian.PutUint64(hook_buf[len(hook_buf)-8:], packageSize)

	// fill in path to package file
	hook_buf = append(hook_buf, []byte(packagePath)...)

	return hook_buf, nil
}

func cleanupInfection(ctx *Context) error {
	for i, _ := range ctx.FDs {
			ctx.FDs[i].Pipe.Close()
			os.Remove(ctx.FDs[i].File)
	}
	return os.Remove(ctx.Path)
}
