// +build linux,amd64

package amnesia

import (
	"io"
	"os"
)

const progpre string = ".amninfected_"
const infectFile string = "./linux64_hook.bin"
const packageFile string = "./linux64_package.bin"

func symAddr(path string, symbol string) (uint64, error) {

}

func instrument(path string, addr uint64) (string, []ProgFD, error) {
	// copy binary
	orig, err := os.Open(path)
	if err != nil {
		return "", nil, err
	}
	defer s.Close()

	dst_path := progpre + path

	d, err := os.Create(dst_path)
	if err != nil {
		return "", nil, err
	}
	defer d.Close()

	_, err = io.Copy(d, orig)
	if err != nil {
		return "", nil, err
	}

	// find file location of address
	//TODO

	// get infection
	hook, err := os.Open(infectFile)

	// read what will be overwritten
	// overwrite
	// customize the package
}

func cleanInstrumentation(inf_path string, pipes []ProgFD) error {
	// TODO
	return os.Remove(inf_path)
}
