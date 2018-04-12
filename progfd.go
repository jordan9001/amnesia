package amnesia

import (
	"encodeing/binary"
	"fmd"
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

func (f *ProgFD) Pack() []byte, err {
	if f.Pipe == nil {
		return nil, fmt.Errorf("Tried to pack a ProgFD with no set Pipe")
	}
	fd_buf = make([]byte, 0x18)
	// Type
	fd_buf[0] = uint8(nctx.FDs[i].Type)
	// FD
	binary.LiddleEndian.PutUint32(fd_buf[1:], uint32(nctx.FDs[i].
	// 19 char filename
	if len(f.File) >= 19 {
		return nil, fmt.Errorf("ProgFD filename too long!")
	}

	copy(fd_buf[5:], []byte(f.File))

	return fd_buf, nil
}
