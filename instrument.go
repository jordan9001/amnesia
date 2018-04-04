package amnesia

// returns the new path to call for the infected binary
func Instrument(path string) (string, []ProgFD, error) {
	return instrument(path)
}

func CleanInstrumentation(infpath string, pipes []ProgFD) error {
	return cleanInstrumentation(infpath, pipes)
}
