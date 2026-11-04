package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lexicon/internal/modlist"
	"lexicon/internal/textfile"
)

// cacheVersion is the cache file's format. A file of another version is
// read as empty, so a change to what's kept starts the cache again.
const cacheVersion = 1

// DefaultCacheDir is where scans keep their cache: MCM Lexicon's folder
// in the system's cache folder, or "" when there's none.
func DefaultCacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "MCM Lexicon")
}

// parents is what a scan remembers of a list's compiled scripts between
// scans: the script each one extends, by file, as long as the file is the
// same size and age. Finding the menus means following every compiled
// script's Extends chain, which otherwise reads all of them, thousands in
// a large list, each time. Only scripts that read cleanly are kept, so an
// unreadable one is read, and reported, every time.
type parents struct {
	path    string // the cache file; "" for none
	entries map[string]parentEntry
	seen    map[string]bool
	changed bool
}

type parentEntry struct {
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	Parent  string    `json:"parent"`
}

type parentsFile struct {
	Version int                    `json:"version"`
	Scripts map[string]parentEntry `json:"scripts"`
}

// loadParents reads the cache kept for the list at listPath in dir. With
// dir "", nothing is kept.
func loadParents(dir, listPath string) *parents {
	p := &parents{entries: map[string]parentEntry{}, seen: map[string]bool{}}
	if dir == "" {
		return p
	}
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(listPath))))
	p.path = filepath.Join(dir, "scripts-"+hex.EncodeToString(sum[:8])+".json")
	var f parentsFile
	if b, err := os.ReadFile(p.path); err == nil && json.Unmarshal(b, &f) == nil && f.Version == cacheVersion && f.Scripts != nil {
		p.entries = f.Scripts
	}
	return p
}

// key is a compiled script's file as the cache knows it: its path on disk,
// and for one packed in an archive, its path in the archive.
func parentKey(f modlist.File) string {
	if f.Archive != "" {
		return f.Abs + "|" + f.Path
	}
	return f.Abs
}

// get is the parent a script's file had when last read, if it's unchanged.
func (p *parents) get(f modlist.File) (string, bool) {
	k := parentKey(f)
	e, ok := p.entries[k]
	if !ok || e.Size != f.Size || !e.ModTime.Equal(f.ModTime) {
		return "", false
	}
	p.seen[k] = true
	return e.Parent, true
}

// put remembers the parent read from a script's file.
func (p *parents) put(f modlist.File, parent string) {
	k := parentKey(f)
	p.seen[k] = true
	if e, ok := p.entries[k]; ok && e.Size == f.Size && e.ModTime.Equal(f.ModTime) && e.Parent == parent {
		return
	}
	p.entries[k] = parentEntry{Size: f.Size, ModTime: f.ModTime, Parent: parent}
	p.changed = true
}

// save writes the cache, keeping only the scripts this scan looked at, so
// files gone from the list leave it. A cache that can't be written is
// only slower next time.
func (p *parents) save() {
	if p.path == "" {
		return
	}
	for k := range p.entries {
		if !p.seen[k] {
			delete(p.entries, k)
			p.changed = true
		}
	}
	if !p.changed {
		return
	}
	b, err := json.Marshal(parentsFile{Version: cacheVersion, Scripts: p.entries})
	if err != nil || os.MkdirAll(filepath.Dir(p.path), 0o755) != nil {
		return
	}
	textfile.WriteAtomic(p.path, b, 0o644)
}
