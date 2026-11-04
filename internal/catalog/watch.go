package catalog

import (
	"crypto/sha256"
	"log"
	"os"
	"path/filepath"
	"time"

	"lexicon/internal/modlist"
)

// The profile's files that say which mods and plugins the game loads: when
// MO2 (or, for a Data folder, Vortex or the launcher) writes one with
// something new, the catalog is out of date.
var watchedFiles = []string{"modlist.txt", "plugins.txt"}

// watchEvery is how often the profile's files are looked at. A change is
// acted on once two looks in a row agree, so MO2 has finished writing.
var watchEvery = 2 * time.Second

// sums is what each watched file held, by its path: a hash of its
// contents, or "" when it isn't there.
type sums map[string]string

// profileFiles is the watched files of src: its MO2 profile's, or for a
// Data folder, the game's own plugins.txt under AppData, which Vortex and
// the game's launcher keep.
func profileFiles(src *modlist.Source) []string {
	if src == nil {
		return nil
	}
	if src.Kind == modlist.Data {
		if p := modlist.GamePluginsFile(src.Path); p != "" {
			return []string{p}
		}
		return nil
	}
	if src.Kind != modlist.MO2 {
		return nil
	}
	var out []string
	for _, name := range watchedFiles {
		out = append(out, filepath.Join(src.Path, "profiles", src.Profile, name))
	}
	return out
}

func sumsOf(paths []string) sums {
	out := sums{}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			out[p] = ""
			continue
		}
		h := sha256.Sum256(b)
		out[p] = string(h[:])
	}
	return out
}

func (a sums) same(b sums) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}

// watch rescans the list when its profile's modlist.txt or plugins.txt
// changes: when a mod is installed, removed, enabled, disabled, or moved,
// or a plugin is, in MO2. It compares contents, not times, since MO2
// rewrites the files unchanged as well. It runs for the life of the
// service; with a catalog file open, or a Data folder, it waits.
func (s *Service) watch() {
	var pending sums // a change seen once, waiting for a second look to agree
	for range time.Tick(watchEvery) {
		s.mu.Lock()
		src, scanned := s.source, s.scanned
		s.mu.Unlock()
		paths := profileFiles(src)
		if len(paths) == 0 || scanned == nil {
			pending = nil
			continue
		}
		now := sumsOf(paths)
		switch {
		case now.same(scanned):
			pending = nil
		case pending == nil || !now.same(pending):
			pending = now
		default:
			pending = nil
			log.Printf("The profile's mod list or plugins changed in %s; rescanning", src.Path)
			s.Rescan()
		}
	}
}
