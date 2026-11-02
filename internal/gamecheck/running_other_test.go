//go:build !windows && !linux

package gamecheck

import "testing"

func TestExeNameFromWineArgs(t *testing.T) {
	for args, want := range map[string]string{
		`C:\Games\Skyrim VR\SkyrimVR.exe -forcesteamloader`:               "SkyrimVR.exe",
		`/Applications/Whisky.app/wine64-preloader C:\Steam\SkyrimSE.exe`: "SkyrimSE.exe",
		"/usr/sbin/cron": "cron",
	} {
		if got := exeName(args); got != want {
			t.Errorf("exeName(%q) = %q, want %q", args, got, want)
		}
	}
}
