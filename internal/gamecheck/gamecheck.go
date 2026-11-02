// Package gamecheck reports whether Skyrim is running. The game reads
// recordings from its mod list while it runs, so MCM Lexicon refuses to
// write them then.
package gamecheck

import "strings"

// Executables that mean Skyrim is running.
var skyrimExes = []string{"skyrimse.exe", "skyrimvr.exe", "skyrim.exe"}

func isSkyrimExe(name string) bool {
	name = strings.ToLower(name)
	for _, exe := range skyrimExes {
		if name == exe {
			return true
		}
	}
	return false
}
