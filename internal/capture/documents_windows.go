package capture

import "golang.org/x/sys/windows"

// documentsFolder is the Documents known folder, as SKSE finds it: wherever
// the player or OneDrive moved it.
func documentsFolder() string {
	dir, err := windows.KnownFolderPath(windows.FOLDERID_Documents, 0)
	if err != nil {
		return ""
	}
	return dir
}
