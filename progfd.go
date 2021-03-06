package amnesia

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"syscall"
)

const (
	PROG_INPUT_FD  fdtype = 0
	PROG_OUTPUT_FD fdtype = 1
	MEM_FUZZ_FD    fdtype = 2
)

type fdtype uint8

type ProgFD struct {
	FD   int
	Type fdtype   // Could be a buff fuzz thing, a reader, or a writer

	File string    // if this is not nil, then we have a named pipe file to delete
	Pipe io.Closer // needs to be type asserted to a io.WriteCloser or io.ReadCloser
}

func (f *ProgFD) Pack() ([]byte, error) {
	if f.File == "" {
		return nil, fmt.Errorf("Tried to pack a ProgFD with no set File")
	}
	fd_buf := make([]byte, 0x18)
	// Type
	fd_buf[0] = uint8(f.Type)
	// FD
	binary.LittleEndian.PutUint32(fd_buf[1:], uint32(f.FD))
	// 19 char filename
	// makes sure there is a room for at least 1 null at the end
	if len(f.File) >= 19 {
		return nil, fmt.Errorf("ProgFD filename too long!")
	}

	copy(fd_buf[5:], []byte(f.File))

	return fd_buf, nil
}

func (f *ProgFD) Open() (io.Closer, error) {
	var pipe io.Closer
	var err error

	if f.Type == PROG_INPUT_FD || f.Type == MEM_FUZZ_FD {
		pipe, err = os.OpenFile(f.File, os.O_WRONLY, os.ModeNamedPipe)
	} else {
		pipe, err = os.OpenFile(f.File, os.O_RDONLY, os.ModeNamedPipe)
	}

	if err != nil {
		return nil, err
	}

	f.Pipe = pipe

	return pipe, nil
}

func (f *ProgFD) Close() error {
	return f.Pipe.Close()
}

func createPipe(path string, t fdtype) error {
	err := syscall.Mkfifo(path, 0666)
	if err != nil {
		return err
	}

	// it will block on opening this pipe until the other end is open as well
	// so don't open the pipe here
	return nil
}

func GetStdPipes(pfds []ProgFD) (stdin io.WriteCloser, stdout io.ReadCloser, stderr io.ReadCloser) {
	stdin = nil
	stdout = nil
	stderr = nil

	var ok bool

	for _, v := range pfds {
		if v.FD == 0 {        // stdin
			stdin, ok = v.Pipe.(io.WriteCloser)
			if !ok {
				stdin = nil
			}
		} else if v.FD == 1 { // stdout
			stdout, ok = v.Pipe.(io.ReadCloser)
			if !ok {
				stdout = nil
			}
		} else if v.FD == 2 { // stderr
			stderr, ok = v.Pipe.(io.ReadCloser)
			if !ok {
				stderr = nil
			}
		}
	}

	return stdin, stdout, stderr
}

func MemfuzzAbs(mfd ProgFD, addr uint64, buf []byte) error {
	return memfuzz(mfd, addr, buf, false)
}

func MemfuzzStackOff(mfd ProgFD, offset int64, buf []byte) error {
	return memfuzz(mfd, uint64(offset), buf, true)
}

func memfuzz(mfd ProgFD, addr uint64, buf []byte, is_stackoff bool) error {
	// check this is a mem fuz fd
	if mfd.Type != MEM_FUZZ_FD {
		return fmt.Errorf("Non-Memfuzz fd passed to Memfuzz")
	}

	pipe, ok := mfd.Pipe.(io.WriteCloser)
	if !ok {
		return fmt.Errorf("Bad pipe")
	}

	// send a memory fuzz msg
	// uint64 addr (or offset)
	// uint64 type (0 = addr, 1 = rsp offset to addr)
	// uint64 size

	// char[] buf

	header := make([]byte, 0x18) // header size
	binary.LittleEndian.PutUint64(header, addr)

	var msgtype uint64 = 0
	if is_stackoff {
		msgtype = 1
	}
	binary.LittleEndian.PutUint64(header[0x8:], msgtype)

	binary.LittleEndian.PutUint64(header[0x10:], uint64(len(buf)))

	// send header

	_, err := pipe.Write(header)
	if err != nil {
		return err
	}

	// send buf
	_, err = pipe.Write(buf)
	if err != nil {
		return err
	}

	return nil
}
