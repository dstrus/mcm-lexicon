// Package textfile is how MCM Lexicon changes files the game reads. Every
// write replaces the file in one step, a copy of the old contents is kept
// first, and a file's line endings stay as they were.
package textfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WriteAtomic replaces path's contents in one step, so a failure part-way
// through never leaves the game a half-written file.
func WriteAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), perm); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Backup saves data as <base>.<time>.bak, creating folders as needed, and
// returns the file it wrote. If that name is taken (two backups within the
// same millisecond), it adds -2, -3… rather than overwrite the earlier one.
func Backup(base string, data []byte, perm os.FileMode) (string, error) {
	if err := os.MkdirAll(filepath.Dir(base), 0o755); err != nil {
		return "", err
	}
	stamp := base + "." + time.Now().Format(BackupStamp)
	for n := 1; ; n++ {
		dest := stamp + ".bak"
		if n > 1 {
			dest = fmt.Sprintf("%s-%d.bak", stamp, n)
		}
		f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		if _, err := f.Write(data); err != nil {
			f.Close()
			os.Remove(dest)
			return "", err
		}
		return dest, f.Close()
	}
}

// BackupStamp is the time format in a backup's name, in local time.
const BackupStamp = "20060102-150405.000"

// Normalize turns Windows line endings into \n, for text that's read or
// compared.
func Normalize(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }

// MatchLineEndings gives text the line endings old uses: Windows ones if old
// has any, otherwise \n.
func MatchLineEndings(text, old string) string {
	text = Normalize(text)
	if strings.Contains(old, "\r\n") {
		return strings.ReplaceAll(text, "\n", "\r\n")
	}
	return text
}

// Remove deletes path, then each folder above it that's left empty, up to
// but not including root.
func Remove(path, root string) error {
	if err := os.Remove(path); err != nil {
		return err
	}
	root = filepath.Clean(root)
	for dir := filepath.Dir(path); dir != root && strings.HasPrefix(dir, root+string(filepath.Separator)); dir = filepath.Dir(dir) {
		if os.Remove(dir) != nil { // not empty
			break
		}
	}
	return nil
}
