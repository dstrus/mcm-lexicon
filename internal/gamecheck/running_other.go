//go:build !windows && !linux

package gamecheck

import (
	"os/exec"
	"strings"
)

// Skyrim only runs here through Wine (CrossOver, Whisky), which names the
// process after the Windows path of the .exe in its arguments.
func Running() bool {
	out, err := exec.Command("ps", "-axo", "args=").Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if isSkyrimExe(exeName(line)) {
			return true
		}
	}
	return false
}

// exeName is the file name of the .exe a process's arguments start with.
func exeName(args string) string {
	exe := strings.ReplaceAll(strings.TrimSpace(args), `\`, "/")
	if i := strings.Index(strings.ToLower(exe), ".exe"); i >= 0 {
		exe = exe[:i+4]
	}
	return exe[strings.LastIndex(exe, "/")+1:]
}
