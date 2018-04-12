package amnesia

import (
	"encoding/binary"
	"fmt"
	"io"
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
	if f.Pipe == nil {
		return nil, fmt.Errorf("Tried to pack a ProgFD with no set Pipe")
	}
	fd_buf := make([]byte, 0x18)
	// Type
	fd_buf[0] = uint8(f.Type)
	// FD
	binary.LittleEndian.PutUint32(fd_buf[1:], uint32(f.FD))
	// 19 char filename
	if len(f.File) >= 19 {
		return nil, fmt.Errorf("ProgFD filename too long!")
	}

	copy(fd_buf[5:], []byte(f.File))

	return fd_buf, nil
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
