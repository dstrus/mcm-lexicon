package gamecheck

import "testing"

func TestIsSkyrimExe(t *testing.T) {
	for name, want := range map[string]bool{
		"SkyrimVR.exe":         true,
		"skyrimse.exe":         true,
		"SKYRIM.EXE":           true,
		"SkyrimSELauncher.exe": false,
		"ModOrganizer.exe":     false,
		"":                     false,
	} {
		if got := isSkyrimExe(name); got != want {
			t.Errorf("isSkyrimExe(%q) = %v, want %v", name, got, want)
		}
	}
}
