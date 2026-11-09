//go:build !windows

package capture

import (
	"os"
	"path/filepath"
)

// documentsFolder is Documents in the home folder. The game runs on
// Windows, so on macOS or Linux there's usually no My Games there, and a
// capture is chosen as a file.
func documentsFolder() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Documents")
}
