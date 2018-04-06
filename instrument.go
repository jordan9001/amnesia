package amnesia

func InstrumentSym(path, sym string) (string, []ProgFD, error) {
	return instrument(path, symAddr(path, sym))
}

// returns the new path to call for the infected binary
func InstrumentAddr(path string, addr uint64) (string, []ProgFD, error) {
	return instrument(path, addr)
}

func CleanInstrumentation(infpath string, pipes []ProgFD) error {
	return cleanInstrumentation(infpath, pipes)
}
