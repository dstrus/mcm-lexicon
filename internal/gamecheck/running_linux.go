package gamecheck

import (
	"os"
	"path/filepath"
	"strings"
)

// Under Proton/Wine the game shows up as a process named after its .exe.
func Running() bool {
	comms, _ := filepath.Glob("/proc/[0-9]*/comm")
	for _, c := range comms {
		if b, err := os.ReadFile(c); err == nil && isSkyrimExe(strings.TrimSpace(string(b))) {
			return true
		}
	}
	return false
}
