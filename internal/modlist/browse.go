package modlist

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"lexicon/internal/apperr"
)

// The picker lists one folder at a time: for a mod list, marking the
// subfolders that are one, or lead to one; for a catalog file, marking the
// files that are one; for a folder to save in, marking nothing.

// Mark says how a folder relates to a mod list.
type Mark string

const (
	MarkNone   Mark = ""
	MarkHere   Mark = "here"   // it's a mod list
	MarkInside Mark = "inside" // one sits below it, at Folder.Inside: a game folder's Data
)

type Folder struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Mark   Mark   `json:"mark"`
	Kind   Kind   `json:"kind,omitempty"`   // with a mark, the kind of list
	Inside string `json:"inside,omitempty"` // with MarkInside, the folder it leads to
}

// Entry is a file in a listing: shown to tell where you are, or, marked,
// a catalog file to choose.
type Entry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Mark Mark   `json:"mark"` // MarkHere for a catalog file, when looking for one
}

type Listing struct {
	Path    string   `json:"path"`
	Parent  string   `json:"parent"` // "" at the top of a drive
	Mark    Mark     `json:"mark"`   // the listed folder's own
	Kind    Kind     `json:"kind,omitempty"`
	Folders []Folder `json:"folders"`
	Files   []Entry  `json:"files"`
	Roots   []string `json:"roots"` // where browsing can start over: the drives on Windows, / elsewhere
}

// Looking is what the picker is looking for.
type Looking string

const (
	LookingList    Looking = "list"    // a mod list: a folder
	LookingCatalog Looking = "catalog" // a catalog file
	LookingCapture Looking = "capture" // a capture file, which the capture plugin writes in the game
	LookingFolder  Looking = "folder"  // a folder to save a file in
)

// Browse lists what's in dir, or in the folder MCM Lexicon runs from when
// dir is "": its subfolders and its files, marked for what's looked for.
// Hidden ones are left out.
func Browse(dir string, looking Looking) (Listing, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = programDir()
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return Listing{}, apperr.New(apperr.Invalid, "Invalid folder: %v", err)
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		if !IsDir(abs) {
			return Listing{}, apperr.New(apperr.NotFound, "Folder not found: %s", abs)
		}
		return Listing{}, apperr.New(apperr.Invalid, "Can't open %s: %v", abs, err)
	}
	out := Listing{Path: abs, Folders: []Folder{}, Files: []Entry{}, Roots: roots()}
	if looking == LookingList {
		own := mark(abs)
		out.Mark, out.Kind = own.Mark, own.Kind
	}
	if parent := filepath.Dir(abs); parent != abs {
		out.Parent = parent
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "$") {
			continue
		}
		p := filepath.Join(abs, name)
		if !IsDir(p) { // a link to a folder counts as a folder
			f := Entry{Name: name, Path: p}
			if looking == LookingCatalog && opensWith(p, "schemaVersion") || looking == LookingCapture && opensWith(p, "format") {
				f.Mark = MarkHere
			}
			out.Files = append(out.Files, f)
			continue
		}
		f := Folder{Path: p}
		if looking == LookingList {
			f = mark(p)
		}
		f.Name = name
		out.Folders = append(out.Folders, f)
	}
	sort.Slice(out.Folders, func(i, j int) bool {
		return strings.ToLower(out.Folders[i].Name) < strings.ToLower(out.Folders[j].Name)
	})
	sort.Slice(out.Files, func(i, j int) bool {
		return strings.ToLower(out.Files[i].Name) < strings.ToLower(out.Files[j].Name)
	})
	return out, nil
}

// opensWith reports whether the file at p looks like a catalog file or a
// capture: a .json file whose object opens with key, as MCM Lexicon writes
// a catalog ("schemaVersion") and the capture plugin a capture ("format").
// Only the start is read, since either can be large; opening it says more
// if it isn't one after all.
func opensWith(p, key string) bool {
	if !strings.EqualFold(filepath.Ext(p), ".json") {
		return false
	}
	f, err := os.Open(p)
	if err != nil {
		return false
	}
	defer f.Close()
	head := make([]byte, 256)
	n, _ := f.Read(head)
	t := strings.TrimLeft(strings.TrimPrefix(string(head[:n]), "\uFEFF"), " \t\r\n")
	if !strings.HasPrefix(t, "{") {
		return false
	}
	return strings.HasPrefix(strings.TrimLeft(t[1:], " \t\r\n"), `"`+key+`"`)
}

// mark is how dir relates to a mod list: an MO2 instance, a Data folder,
// or a game folder whose Data folder is one.
func mark(dir string) Folder {
	f := Folder{Path: dir}
	switch {
	case Find1(dir, "ModOrganizer.ini") != "":
		f.Mark, f.Kind = MarkHere, MO2
	case Find1(dir, "Skyrim.esm") != "":
		f.Mark, f.Kind = MarkHere, Data
	default:
		if data := Find1Dir(dir, "Data"); data != "" && Find1(data, "Skyrim.esm") != "" {
			f.Mark, f.Kind, f.Inside = MarkInside, Data, data
		}
	}
	return f
}

// Find1Dir is the subfolder of dir named name, compared without regard to
// case, or "".
func Find1Dir(dir, name string) string {
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.EqualFold(e.Name(), name) && IsDir(filepath.Join(dir, e.Name())) {
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
}

// programDir is the folder MCM Lexicon runs from, or the current one if
// that can't be told.
func programDir() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return string(filepath.Separator)
}

func roots() []string {
	if runtime.GOOS != "windows" {
		return []string{"/"}
	}
	var out []string
	for d := 'A'; d <= 'Z'; d++ {
		if root := string(d) + `:\`; IsDir(root) {
			out = append(out, root)
		}
	}
	return out
}
