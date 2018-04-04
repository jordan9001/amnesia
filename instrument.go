package amnesia

// returns the new path to call for the infected binary
func Instrument(path string) string {
	return instrument(path)
}
