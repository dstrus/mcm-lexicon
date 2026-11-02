// Package recordings reads MCM Recorder's recordings from a list and checks
// each step against the catalog: whether the menu, page, and option it
// names are there, and whether its value is one the option allows.
//
// A recording named N is McmRecorder/N.json (its info: title, version,
// author, autorun) and the step files in McmRecorder/N/, played in name
// order, each a JSON array of actions. Keys are read without regard to
// case, as JContainers does.
package recordings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"lexicon/internal/modlist"
)

// Recording is a recording as the game sees it: its info file and its step
// files, each the copy that wins.
type Recording struct {
	Name     string    `json:"name"`  // the file name MCM Recorder plays it by
	Title    string    `json:"title"` // the info file's name, which MCM Recorder shows
	Version  string    `json:"version"`
	Author   string    `json:"author"`
	Autorun  bool      `json:"autorun"`
	Mod      string    `json:"mod"`  // the mod holding its info file, or its first step file without one
	Info     string    `json:"info"` // the info file's path; "" without one
	Files    []File    `json:"files"`
	Losing   []string  `json:"losing"` // other mods with a copy of its info file, which lose
	Problems []Problem `json:"problems"`
	Steps    []Step    `json:"steps"` // every step, in replay order
	// Unlisted is a recording in a folder MCM Lexicon saved to that the
	// profile doesn't enable (a new mod MO2 hasn't listed, or one left
	// disabled): it's on disk, but the game won't play it.
	Unlisted bool     `json:"unlisted"`
	byFile   [][]Step // the steps of each file
}

// File is a step file.
type File struct {
	Name  string `json:"name"` // such as 0001_Dirt & Blood.json
	Path  string `json:"path"` // as the game sees it
	Mod   string `json:"mod"`
	Steps int    `json:"steps"`
	abs   string
}

// Kind is the key a step's value is under, which says what it does.
type Kind string

const (
	KindToggle   Kind = "toggle"
	KindSlider   Kind = "slider"
	KindChoose   Kind = "choose"
	KindShortcut Kind = "shortcut"
	KindColor    Kind = "color"
	KindText     Kind = "text" // an input option's text
	KindClick    Kind = "click"
)

// The kinds in the order MCM Recorder checks for them.
var kinds = []Kind{KindClick, KindToggle, KindChoose, KindText, KindShortcut, KindColor, KindSlider}

// Step is one action.
type Step struct {
	File     int    `json:"file"` // its file's position in Files
	Mod      string `json:"mod"`
	Page     string `json:"page"`   // "" for a menu's default page
	Option   string `json:"option"` // the selector; a click step's is its click value
	Index    int    `json:"index"`  // 1-based; 0 when the step doesn't say
	Kind     Kind   `json:"kind"`
	Value    string `json:"value"`    // as the file writes it: "on", 25.0, a menu entry, a scan code
	Side     string `json:"side"`     // a click step's side, when it says
	State    string `json:"state"`    // an ST option's state, when it says
	Wildcard bool   `json:"wildcard"` // the selector is *part*, which matches any label containing part
}

// ProblemKind says what's wrong with a recording's files.
type ProblemKind string

const (
	ProblemNoInfo     ProblemKind = "no-info"    // step files without an info file
	ProblemUnreadable ProblemKind = "unreadable" // a file that isn't what MCM Recorder reads; Detail says why
	ProblemNoKind     ProblemKind = "no-kind"    // a step with none of the value keys, which MCM Recorder can't play
)

type Problem struct {
	Kind   ProblemKind `json:"kind"`
	Path   string      `json:"path"`
	Detail string      `json:"detail"`
}

// Read reads every recording in the list, in name order. The list must be
// indexed.
func Read(l *modlist.List) []*Recording {
	return read(l.Files("McmRecorder"), l.Copies)
}

// ReadFolder reads the recordings in one mod folder's McmRecorder, as if
// it were the only mod: for a folder the list doesn't read (disabled, or
// not listed yet).
func ReadFolder(dir, mod string) []*Recording {
	root, rel, ok := modlist.Resolve(dir, "McmRecorder")
	if !ok || !modlist.IsDir(root) {
		return nil
	}
	var files []modlist.File
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || modlist.Clutter(d.Name()) {
			return nil
		}
		inside, _ := filepath.Rel(root, p)
		files = append(files, modlist.File{Path: rel + "/" + filepath.ToSlash(inside), Abs: p, Mod: mod})
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return strings.ToLower(files[i].Path) < strings.ToLower(files[j].Path) })
	recs := read(files, func(string) []modlist.File { return []modlist.File{{}} })
	for _, r := range recs {
		r.Unlisted = true
	}
	return recs
}

// read builds recordings from the McmRecorder files the game sees; copies
// is every copy of a path, the winning one first.
func read(files []modlist.File, copies func(string) []modlist.File) []*Recording {
	byName := map[string]*Recording{}
	get := func(name string) *Recording {
		k := strings.ToLower(name)
		if r, ok := byName[k]; ok {
			return r
		}
		r := &Recording{Name: name, Files: []File{}, Losing: []string{}, Problems: []Problem{}, Steps: []Step{}}
		byName[k] = r
		return r
	}
	for _, f := range files {
		parts := strings.Split(f.Path, "/")
		if !strings.EqualFold(path.Ext(f.Path), ".json") || strings.HasPrefix(parts[1], ".") {
			continue
		}
		switch len(parts) {
		case 2:
			r := get(strings.TrimSuffix(parts[1], path.Ext(parts[1])))
			r.Info, r.Mod = f.Path, f.Mod
			readInfo(r, f)
			for _, c := range copies(f.Path)[1:] {
				r.Losing = append(r.Losing, c.Mod)
			}
		case 3:
			r := get(parts[1])
			r.Files = append(r.Files, File{Name: parts[2], Path: f.Path, Mod: f.Mod, abs: f.Abs})
		}
	}
	var out []*Recording
	for _, r := range byName {
		if r.Info == "" {
			r.Problems = append(r.Problems, Problem{Kind: ProblemNoInfo, Path: "McmRecorder/" + r.Name + ".json"})
			if len(r.Files) > 0 {
				r.Mod = r.Files[0].Mod
			}
		}
		// Played in name order; files with the same number (MGO's own has
		// two 0001s) by the rest of the name.
		sort.SliceStable(r.Files, func(i, j int) bool { return strings.ToLower(r.Files[i].Name) < strings.ToLower(r.Files[j].Name) })
		for i := range r.Files {
			steps := readSteps(r, i)
			r.Files[i].Steps = len(steps)
			r.byFile = append(r.byFile, steps)
			r.Steps = append(r.Steps, steps...)
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}

func readInfo(r *Recording, f modlist.File) {
	b, err := f.Read()
	if err != nil {
		r.Problems = append(r.Problems, Problem{Kind: ProblemUnreadable, Path: f.Path, Detail: err.Error()})
		return
	}
	var m map[string]any
	if err := json.Unmarshal(bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF}), &m); err != nil {
		r.Problems = append(r.Problems, Problem{Kind: ProblemUnreadable, Path: f.Path, Detail: "not a JSON object: " + err.Error()})
		return
	}
	info := lower(m)
	r.Title = text(info["name"])
	r.Version = text(info["version"])
	r.Author = text(info["author"])
	// autorun is a string MCM Recorder writes as "false"; MGO's own says "True".
	r.Autorun = strings.EqualFold(text(info["autorun"]), "true")
}

func readSteps(r *Recording, file int) []Step {
	f := r.Files[file]
	b, err := os.ReadFile(f.abs)
	if err != nil {
		r.Problems = append(r.Problems, Problem{Kind: ProblemUnreadable, Path: f.Path, Detail: err.Error()})
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})))
	dec.UseNumber() // a slider's 25.0 stays 25.0
	var actions []map[string]any
	if err := dec.Decode(&actions); err != nil {
		r.Problems = append(r.Problems, Problem{Kind: ProblemUnreadable, Path: f.Path, Detail: "not a JSON array of actions: " + err.Error()})
		return nil
	}
	var out []Step
	for i, raw := range actions {
		a := lower(raw)
		s := Step{File: file, Mod: text(a["mod"]), Page: text(a["page"]), Option: text(a["option"]), Side: text(a["side"]), State: text(a["state"])}
		if n, err := strconv.Atoi(text(a["index"])); err == nil {
			s.Index = n
		}
		for _, k := range kinds {
			if v, ok := a[string(k)]; ok {
				s.Kind, s.Value = k, text(v)
				break
			}
		}
		if s.Kind == "" {
			r.Problems = append(r.Problems, Problem{Kind: ProblemNoKind, Path: f.Path, Detail: fmt.Sprint(i + 1)})
		}
		if s.Kind == KindClick {
			s.Option = s.Value // a click step's selector is what it clicks
		}
		s.Wildcard = len(s.Option) > 2 && strings.HasPrefix(s.Option, "*") && strings.HasSuffix(s.Option, "*")
		out = append(out, s)
	}
	return out
}

// lower is a map with its keys lowercased.
func lower(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[strings.ToLower(k)] = v
	}
	return out
}

// text is a JSON value as its text: a string as itself, a number as the
// file spells it.
func text(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case json.Number:
		return x.String()
	}
	return fmt.Sprint(v)
}

// Text is a step file's contents, for showing as written.
func (r *Recording) Text(file int) (string, error) {
	if file < 0 || file >= len(r.Files) {
		return "", fmt.Errorf("no file %d", file)
	}
	b, err := os.ReadFile(r.Files[file].abs)
	return string(b), err
}
