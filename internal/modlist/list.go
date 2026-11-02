package modlist

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"lexicon/internal/apperr"
	"lexicon/internal/bsa"
)

// Layer is where a mod's files sit in the order the game sees them.
type Layer string

const (
	LayerData      Layer = "data"      // the game's own Data folder: below every mod
	LayerMod       Layer = "mod"       // a mod under mods/, in the profile's order
	LayerOverwrite Layer = "overwrite" // MO2's overwrite folder: above every mod
)

// Mod is one entry of the list, as the profile orders it.
type Mod struct {
	Name      string `json:"name"` // the folder's name under mods/; "overwrite" or "Data" for those layers
	Dir       string `json:"dir"`  // absolute; "" for an unmanaged entry without a folder
	Enabled   bool   `json:"enabled"`
	Unmanaged bool   `json:"unmanaged"` // "*" in modlist.txt: DLC or Creation Club content, which lives in Data
	Priority  int    `json:"priority"`  // as MO2 numbers them: 0 is the lowest, and a higher number wins
	Layer     Layer  `json:"layer"`
}

// File is a file as the game sees it: one copy, and the mod it comes from.
// A file packed in an archive has the archive's name, and Abs is the
// archive's; Read reads either kind.
type File struct {
	Path    string    `json:"path"` // its virtual path with forward slashes, spelled as this copy spells it
	Abs     string    `json:"abs"`
	Mod     string    `json:"mod"`
	Archive string    `json:"archive,omitempty"` // the archive it's packed in, as the game names it; "" for a loose file
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`

	packed *bsa.Archive
	entry  bsa.File
}

// Read is the file's contents, from disk or from its archive.
func (f File) Read() ([]byte, error) {
	if f.packed != nil {
		return f.packed.ReadFile(f.entry)
	}
	return os.ReadFile(f.Abs)
}

// ArchiveProblem is an archive the game loads that couldn't be read.
type ArchiveProblem struct {
	File   File
	Detail string
}

// Plugin is a line of plugins.txt.
type Plugin struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// List is a mod list read from disk: its mods in load order, which plugins
// are active, and, once Index has run, the files the game sees.
type List struct {
	Source   Source
	Mods     []Mod  // in load order: the lowest priority first, overwrite last
	DataPath string // the game folder ModOrganizer.ini names, whether or not it's here
	DataDir  string // the game's Data folder, when it's here; "" otherwise

	plugins  []Plugin        // plugins.txt in order; nil when there is no plugins.txt
	byName   map[string]bool // active plugins by lowercase name
	index    map[string][]File
	archives []File           // the archives the game loads, in load order
	problems []ArchiveProblem // the ones of those that couldn't be read
}

// Core masters are always loaded, so MO2 leaves them out of plugins.txt.
var coreMasters = map[string]bool{"skyrim.esm": true, "update.esm": true, "dawnguard.esm": true, "hearthfires.esm": true, "dragonborn.esm": true, "skyrimvr.esm": true}

// Read reads the mod list src names: for an MO2 instance, the profile's
// modlist.txt and plugins.txt, the overwrite folder, and the game's Data
// folder when ModOrganizer.ini says where it is and it's here; for a Data
// folder, the folder itself. Files aren't read until Index.
func Read(src Source) (*List, error) {
	l := &List{Source: src}
	if src.Kind == Data {
		l.Mods = []Mod{{Name: "Data", Dir: src.Path, Enabled: true, Layer: LayerData}}
		l.DataDir = src.Path
		return l, nil
	}
	profile := filepath.Join(src.Path, "profiles", src.Profile)
	lines, err := readLines(filepath.Join(profile, "modlist.txt"))
	if err != nil {
		return nil, apperr.New(apperr.Invalid, "Profile %q has no modlist.txt: %v", src.Profile, err)
	}
	if game := readINI(Find1(src.Path, "ModOrganizer.ini"))["gamePath"]; game != "" {
		l.DataPath = filepath.Join(fromWindows(game), "Data")
		if IsDir(l.DataPath) {
			l.DataDir = l.DataPath
			l.Mods = append(l.Mods, Mod{Name: "Data", Dir: l.DataDir, Enabled: true, Layer: LayerData})
		}
	}
	// modlist.txt lists the highest priority first; MO2 numbers from the
	// bottom, counting every entry, separators included.
	var entries []string
	for _, line := range lines {
		if line != "" && !strings.HasPrefix(line, "#") {
			entries = append(entries, line)
		}
	}
	mods := filepath.Join(src.Path, "mods")
	for i := len(entries) - 1; i >= 0; i-- {
		name := entries[i][1:]
		if strings.HasSuffix(name, "_separator") {
			continue
		}
		m := Mod{Name: name, Priority: len(entries) - 1 - i, Layer: LayerMod}
		switch entries[i][0] {
		case '+':
			m.Enabled = true
		case '*':
			m.Enabled, m.Unmanaged = true, true
		case '-':
		default:
			continue
		}
		if dir := filepath.Join(mods, name); IsDir(dir) {
			m.Dir = dir
		}
		l.Mods = append(l.Mods, m)
	}
	if dir := filepath.Join(src.Path, "overwrite"); IsDir(dir) {
		l.Mods = append(l.Mods, Mod{Name: "overwrite", Dir: dir, Enabled: true, Priority: len(entries), Layer: LayerOverwrite})
	}
	if plugins, err := readLines(filepath.Join(profile, "plugins.txt")); err == nil {
		l.plugins = []Plugin{}
		l.byName = map[string]bool{}
		for _, line := range plugins {
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			p := Plugin{Name: strings.TrimPrefix(line, "*"), Active: strings.HasPrefix(line, "*")}
			l.plugins = append(l.plugins, p)
			l.byName[strings.ToLower(p.Name)] = p.Active
		}
	}
	return l, nil
}

// Plugins is plugins.txt in order, or nil when the profile has none (in
// which case every plugin counts as active).
func (l *List) Plugins() []Plugin { return l.plugins }

// CoreMaster reports whether name is one of the game's own masters, which
// are always loaded.
func CoreMaster(name string) bool { return coreMasters[strings.ToLower(name)] }

// PluginActive reports whether the game loads the plugin named name: its
// mark in plugins.txt, or always for the game's own masters. Without a
// plugins.txt, every plugin counts as active.
func (l *List) PluginActive(name string) bool {
	if l.plugins == nil {
		return true
	}
	key := strings.ToLower(name)
	if active, listed := l.byName[key]; listed {
		return active
	}
	return coreMasters[key]
}

// Mod is the mod named name (compared without regard to case), if listed.
func (l *List) Mod(name string) (Mod, bool) {
	for _, m := range l.Mods {
		if strings.EqualFold(m.Name, name) {
			return m, true
		}
	}
	return Mod{}, false
}

// Scope is which of a mod's files Index reads: whole subtrees by their
// path under the mod, top-level files by name, and the subtrees read from
// the archives the game loads (PLAN.md §2.6). Reading only what MCM
// Lexicon needs keeps a scan of 1,800 mods quick.
type Scope struct {
	Subtrees []string
	TopLevel func(name string) bool
	Archived []string
}

// MCMScope covers what MCMs are made of: scripts and their sources, MCM
// Helper's configs and settings, recordings, translations, and plugins.
// Settings and recordings are read only loose, since what writes them
// can't write into an archive.
var MCMScope = Scope{
	Subtrees: []string{"Scripts", "Source", "MCM", "McmRecorder", "Interface/Translations"},
	TopLevel: IsPlugin,
	Archived: []string{"Scripts", "Source", "MCM/Config", "Interface/Translations"},
}

// IsArchive reports whether name is a Bethesda archive's.
func IsArchive(name string) bool { return strings.EqualFold(filepath.Ext(name), ".bsa") }

// Clutter reports whether a file is one an operating system leaves in
// folders it shows (Finder's .DS_Store and ._ files, Windows's Thumbs.db
// and desktop.ini), which no game or mod reads.
func Clutter(name string) bool {
	switch strings.ToLower(name) {
	case ".ds_store", "thumbs.db", "desktop.ini":
		return true
	}
	return strings.HasPrefix(name, "._")
}

// IsPlugin reports whether name is a plugin file's.
func IsPlugin(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".esp", ".esm", ".esl":
		return true
	}
	return false
}

// Index reads the files in scope from every enabled mod, in load order, so
// Open and Files can say which copy the game sees. Folder and file names
// are matched without regard to case, as Windows does. Files in the
// archives the game loads sit beneath every loose file.
func (l *List) Index(scope Scope) {
	l.index = map[string][]File{}
	l.archives, l.problems = nil, nil
	top := map[string][]File{} // every mod's top-level plugins and archives, for finding what loads
	for _, m := range l.Mods {
		if !m.Enabled || m.Dir == "" {
			continue
		}
		entries, _ := os.ReadDir(m.Dir)
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			abs := filepath.Join(m.Dir, e.Name())
			if scope.TopLevel != nil && scope.TopLevel(e.Name()) {
				l.add(m, e.Name(), abs)
			}
			if IsPlugin(e.Name()) || IsArchive(e.Name()) {
				top[strings.ToLower(e.Name())] = append(top[strings.ToLower(e.Name())], stat(File{Path: e.Name(), Abs: abs, Mod: m.Name}))
			}
		}
		for _, sub := range scope.Subtrees {
			dir, rel, ok := Resolve(m.Dir, sub)
			if !ok || !IsDir(dir) {
				continue
			}
			filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() || Clutter(d.Name()) {
					return nil
				}
				inside, _ := filepath.Rel(dir, p)
				l.add(m, rel+"/"+filepath.ToSlash(inside), p)
				return nil
			})
		}
	}
	if len(scope.Archived) > 0 {
		l.indexArchives(top, scope.Archived)
	}
}

func (l *List) add(m Mod, rel, abs string) {
	f := stat(File{Path: rel, Abs: abs, Mod: m.Name})
	key := strings.ToLower(rel)
	l.index[key] = append(l.index[key], f)
}

func stat(f File) File {
	if info, err := os.Stat(f.Abs); err == nil {
		f.Size, f.ModTime = info.Size(), info.ModTime()
	}
	return f
}

// indexArchives adds the files under subtrees from each archive the game
// loads, beneath the loose files: an archive's files lose to any loose
// copy, and to those of an archive loaded after it.
func (l *List) indexArchives(top map[string][]File, subtrees []string) {
	l.archives = loadedArchives(top, l.loadOrder(top))
	packed := map[string][]File{}
	for _, af := range l.archives {
		a, err := bsa.Open(af.Abs)
		if err != nil {
			l.problems = append(l.problems, ArchiveProblem{File: af, Detail: err.Error()})
			continue
		}
		for _, e := range a.Files {
			if !under(e.Path, subtrees) {
				continue
			}
			f := File{Path: e.Path, Abs: af.Abs, Mod: af.Mod, Archive: af.Path, Size: int64(e.Size), ModTime: af.ModTime, packed: a, entry: e}
			packed[e.Path] = append(packed[e.Path], f)
		}
	}
	for k, copies := range packed {
		l.index[k] = append(copies, l.index[k]...)
	}
}

func under(p string, subtrees []string) bool {
	for _, s := range subtrees {
		if s := key(s) + "/"; strings.HasPrefix(p, s) {
			return true
		}
	}
	return false
}

// loadOrder is the plugins present and active, in the order the game
// loads them: the game's own masters, then plugins.txt's order. Without a
// plugins.txt, every plugin present loads, in the order of the mods
// holding them.
func (l *List) loadOrder(top map[string][]File) []string {
	var out []string
	seen := map[string]bool{}
	addIf := func(name string) {
		k := strings.ToLower(name)
		if seen[k] || !IsPlugin(k) || len(top[k]) == 0 || !l.PluginActive(name) {
			return
		}
		seen[k] = true
		out = append(out, k)
	}
	for _, m := range []string{"skyrim.esm", "skyrimvr.esm", "update.esm", "dawnguard.esm", "hearthfires.esm", "dragonborn.esm"} {
		addIf(m)
	}
	if l.plugins != nil {
		for _, p := range l.plugins {
			addIf(p.Name)
		}
		return out
	}
	var rest []File
	for k, copies := range top {
		if IsPlugin(k) {
			rest = append(rest, copies[len(copies)-1])
		}
	}
	priority := map[string]int{}
	for i, m := range l.Mods {
		priority[m.Name] = i
	}
	sort.SliceStable(rest, func(i, j int) bool {
		if pi, pj := priority[rest[i].Mod], priority[rest[j].Mod]; pi != pj {
			return pi < pj
		}
		return strings.ToLower(rest[i].Path) < strings.ToLower(rest[j].Path)
	})
	for _, f := range rest {
		addIf(f.Path)
	}
	return out
}

// loadedArchives is the archives each plugin in order brings in: the
// game loads <plugin>.bsa and <plugin> - Textures.bsa, as the winning
// copies of those files.
func loadedArchives(top map[string][]File, order []string) []File {
	var out []File
	for _, p := range order {
		base := strings.TrimSuffix(p, filepath.Ext(p))
		for _, name := range []string{base + ".bsa", base + " - textures.bsa"} {
			if copies := top[name]; len(copies) > 0 {
				out = append(out, copies[len(copies)-1])
			}
		}
	}
	return out
}

// Archives is the archives the game loads, in load order, once Index has
// run with archives in scope.
func (l *List) Archives() []File { return l.archives }

// ArchiveProblems is the archives the game loads that couldn't be read.
func (l *List) ArchiveProblems() []ArchiveProblem { return l.problems }

// Open is the copy of rel the game sees, if any. rel is a path under Data
// with either kind of slash, in any case.
func (l *List) Open(rel string) (File, bool) {
	copies := l.index[key(rel)]
	if len(copies) == 0 {
		return File{}, false
	}
	return copies[len(copies)-1], true
}

// Copies is every copy of rel in the list: the one the game sees first,
// then the ones it overrides, highest priority first.
func (l *List) Copies(rel string) []File {
	copies := l.index[key(rel)]
	out := make([]File, len(copies))
	for i, c := range copies {
		out[len(copies)-1-i] = c
	}
	return out
}

// Files is every file the game sees under the virtual folder prefix (or
// everything indexed, for ""), in path order.
func (l *List) Files(prefix string) []File {
	p := key(prefix)
	if p != "" && !strings.HasSuffix(p, "/") {
		p += "/"
	}
	var out []File
	for k, copies := range l.index {
		if strings.HasPrefix(k, p) {
			out = append(out, copies[len(copies)-1])
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Path) < strings.ToLower(out[j].Path) })
	return out
}

func key(rel string) string {
	return strings.ToLower(strings.Trim(strings.ReplaceAll(rel, `\`, "/"), "/"))
}

// Resolve finds rel under root, matching each part of the path without
// regard to case, and returns the path as it's spelled on disk: absolute,
// and relative with forward slashes.
func Resolve(root, rel string) (abs, spelled string, ok bool) {
	abs = root
	var parts []string
	for _, part := range strings.Split(strings.ReplaceAll(rel, `\`, "/"), "/") {
		if part == "" {
			continue
		}
		entries, err := os.ReadDir(abs)
		if err != nil {
			return "", "", false
		}
		found := ""
		for _, e := range entries {
			if strings.EqualFold(e.Name(), part) {
				found = e.Name()
				if e.Name() == part {
					break // an exact match over a case-insensitive one
				}
			}
		}
		if found == "" {
			return "", "", false
		}
		abs = filepath.Join(abs, found)
		parts = append(parts, found)
	}
	return abs, strings.Join(parts, "/"), true
}

// fromWindows turns a path as ModOrganizer.ini writes it (escaped
// backslashes) into one this OS can use. A Windows drive path stays as it
// is on another OS; it just won't be found.
func fromWindows(p string) string {
	p = strings.ReplaceAll(p, `\\`, `\`)
	if runtime.GOOS != "windows" {
		return strings.ReplaceAll(p, `\`, "/")
	}
	return p
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		lines = append(lines, strings.TrimRight(strings.TrimPrefix(sc.Text(), "\ufeff"), "\r"))
	}
	return lines, sc.Err()
}
