package amnesia

import (
	"syscall"
)

// Helper function ideas
/*
- Reverse RegExe for random input generation
*/

func ReportFaults(in string, fc FuzzChan) bool {
	r := <-fc.Status

	if r != nil && r.Signaled() {
		if r.Signal() != syscall.SIGKILL {
			h := Hit{
				Kind:   "Signal",
				Input:  in,
				Output: r.Signal(),
			}
			fc.Result <- h
			return true
		}
	}
	return false
}
