package recordings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"lexicon/internal/apperr"
	"lexicon/internal/textfile"
)

// Draft is a recording being edited: its info and its steps, in replay
// order, and where Save writes it. Nothing is on disk until Save.
type Draft struct {
	Name    string `json:"name"`  // the file name MCM Recorder plays it by
	Title   string `json:"title"` // the info file's name; Name when empty
	Version string `json:"version"`
	Author  string `json:"author"`
	Autorun bool   `json:"autorun"`
	Steps   []Step `json:"steps"`
	// Target is the mod folder Save writes into, by name ("overwrite" for
	// MO2's overwrite folder); with a catalog file open, a folder's path.
	Target string `json:"target"`
}

// DraftOf is a saved recording as a draft, to edit.
func DraftOf(r *Recording, target string) Draft {
	d := Draft{Name: r.Name, Title: r.Title, Version: r.Version, Author: r.Author, Autorun: r.Autorun, Steps: append([]Step{}, r.Steps...), Target: target}
	if d.Title == "" {
		d.Title = r.Name
	}
	return d
}

// OutFile is a file Save writes, relative to the target folder.
type OutFile struct {
	Name string `json:"name"` // such as 0001_Dirt & Blood.json, or the info file's name
	Path string `json:"path"` // McmRecorder/…, with forward slashes
	Text string `json:"text"`
}

// PathPart is a name as it goes into a file name. MCM Recorder turns / and
// \ into _; the other characters Windows refuses in a file name go the same
// way, since the game couldn't make such a file either. The file's name is
// only its order and a reminder: the steps inside say which menu they're for.
func PathPart(name string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		if r < 32 {
			return '_'
		}
		return r
	}, strings.TrimSpace(name))
}

// Plan is the files Save writes for d: its info file, then a step file
// for each run of steps for one menu, numbered from 0001. It also sets
// each step's File to its file's position among the step files.
func Plan(d *Draft) ([]OutFile, error) {
	name := PathPart(d.Name)
	if name == "" || name == "." || name == ".." {
		return nil, apperr.New(apperr.Invalid, "A recording needs a name")
	}
	title := d.Title
	if title == "" {
		title = d.Name
	}
	version := d.Version
	if version == "" {
		version = "1.0.0"
	}
	autorun := "false"
	if d.Autorun {
		autorun = "true"
	}
	info, err := encode([]pair{{"author", d.Author}, {"autorun", autorun}, {"name", title}, {"version", version}})
	if err != nil {
		return nil, err
	}
	out := []OutFile{{Name: name + ".json", Path: "McmRecorder/" + name + ".json", Text: info}}
	// A run of steps for one menu (named without regard to case, as Papyrus
	// compares) is one file, named for the run's first step.
	var actions []string
	first := 0
	flush := func(mod string) {
		n := len(out)
		file := fmt.Sprintf("%04d_%s.json", n, PathPart(mod))
		out = append(out, OutFile{Name: file, Path: "McmRecorder/" + name + "/" + file, Text: "[\n" + strings.Join(actions, ",\n") + "\n]"})
		actions = nil
	}
	for i := range d.Steps {
		s := &d.Steps[i]
		if i > 0 && !strings.EqualFold(s.Mod, d.Steps[i-1].Mod) {
			flush(d.Steps[first].Mod)
			first = i
		}
		s.File = len(out) - 1
		a, err := action(*s)
		if err != nil {
			return nil, fmt.Errorf("step %d: %w", i+1, err)
		}
		actions = append(actions, indent(a))
	}
	if len(actions) > 0 {
		flush(d.Steps[first].Mod)
	}
	return out, nil
}

// action is a step as MCM Recorder writes it: lowercase keys in the order
// JContainers sorts them, the page left out for a menu's default page and
// the index when it doesn't pick among duplicates.
func action(s Step) (string, error) {
	if strings.TrimSpace(s.Mod) == "" {
		return "", errors.New("it names no menu")
	}
	var value any
	switch s.Kind {
	case KindToggle:
		v := strings.ToLower(s.Value)
		if v != "on" && v != "off" {
			return "", fmt.Errorf("a toggle is on or off, not %q", s.Value)
		}
		value = v
	case KindSlider:
		f, err := strconv.ParseFloat(strings.TrimSpace(s.Value), 64)
		if err != nil {
			return "", fmt.Errorf("a slider's value is a number, not %q", s.Value)
		}
		value = sliderNumber(f)
	case KindShortcut, KindColor:
		n, err := strconv.ParseInt(strings.TrimSpace(s.Value), 0, 64)
		if err != nil {
			return "", fmt.Errorf("a %s's value is a whole number, not %q", s.Kind, s.Value)
		}
		value = json.Number(strconv.FormatInt(n, 10))
	case KindChoose, KindText, KindClick:
		value = s.Value
	default:
		return "", fmt.Errorf("it doesn't say what it does (%q)", s.Kind)
	}
	pairs := []pair{{"mod", s.Mod}, {string(s.Kind), value}}
	if s.Kind == KindClick {
		pairs[1].value = s.Option // a click's selector is what it clicks
	} else {
		pairs = append(pairs, pair{"option", s.Option})
	}
	if s.Page != "" {
		pairs = append(pairs, pair{"page", s.Page})
	}
	if s.Index > 0 {
		pairs = append(pairs, pair{"index", json.Number(strconv.Itoa(s.Index))})
	}
	if s.Side != "" {
		pairs = append(pairs, pair{"side", s.Side})
	}
	if s.State != "" {
		pairs = append(pairs, pair{"state", s.State})
	}
	// How long to look for the option, as the file spelled it.
	for _, k := range waits {
		w := strings.TrimSpace(*s.wait(k))
		if w == "" {
			continue
		}
		if f, err := strconv.ParseFloat(w, 64); err != nil || f < 0 {
			return "", fmt.Errorf("its %s is a number of seconds, not %q", k, w)
		}
		pairs = append(pairs, pair{k, json.Number(w)})
	}
	sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].key < pairs[j].key })
	return encode(pairs)
}

// sliderNumber writes a slider's value as MCM Recorder does: always with a
// decimal point (25.0).
func sliderNumber(f float64) json.Number {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return json.Number(s)
}

type pair struct {
	key   string
	value any
}

// encode writes an object with its keys in the order given, indented by
// two spaces as JContainers writes, and its text as UTF-8 (no \u escapes
// for < > &).
func encode(pairs []pair) (string, error) {
	var b strings.Builder
	b.WriteString("{\n")
	for i, p := range pairs {
		var v bytes.Buffer
		enc := json.NewEncoder(&v)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(p.value); err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "  %q: %s", p.key, strings.TrimRight(v.String(), "\n"))
		if i < len(pairs)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("}")
	return b.String(), nil
}

func indent(s string) string {
	return "  " + strings.ReplaceAll(s, "\n", "\n  ")
}

// Disk is what a target folder holds now of a recording: its info file
// and step files, by path relative to the folder.
func Disk(dir, name string) map[string]string {
	out := map[string]string{}
	name = PathPart(name)
	if b, err := os.ReadFile(filepath.Join(dir, "McmRecorder", name+".json")); err == nil {
		out["McmRecorder/"+name+".json"] = string(b)
	}
	sub := filepath.Join(dir, "McmRecorder", name)
	entries, _ := os.ReadDir(sub)
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".json") {
			continue
		}
		if b, err := os.ReadFile(filepath.Join(sub, e.Name())); err == nil {
			out["McmRecorder/"+name+"/"+e.Name()] = string(b)
		}
	}
	return out
}

// Saved is what Save did.
type Saved struct {
	At      time.Time `json:"at"`
	Dir     string    `json:"dir"`     // the folder written to
	Written []string  `json:"written"` // the files, relative to it
	Backup  string    `json:"backup"`  // the folder the previous files went to; "" when there were none
}

// Save writes d into dir: the info file McmRecorder/<Name>.json and the
// step files in McmRecorder/<Name>/, replacing what was there. The previous
// files are copied to backups/<Name>/<time>/ first. The step files are
// written to a folder beside the recording's and swapped in, so the game
// never finds half a recording.
func Save(dir string, d Draft, backups string, now time.Time) (Saved, error) {
	files, err := Plan(&d)
	if err != nil {
		return Saved{}, err
	}
	name := PathPart(d.Name)
	root := filepath.Join(dir, "McmRecorder")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return Saved{}, err
	}
	saved := Saved{At: now, Dir: dir}
	backup, err := backUp(dir, name, backups, now, BackupSave)
	if err != nil {
		return Saved{}, fmt.Errorf("backing up %s: %w", name, err)
	}
	saved.Backup = backup

	stage, err := os.MkdirTemp(root, "."+name+".new-*")
	if err != nil {
		return Saved{}, err
	}
	defer os.RemoveAll(stage)
	for _, f := range files[1:] {
		if err := os.WriteFile(filepath.Join(stage, f.Name), []byte(f.Text), 0o644); err != nil {
			return Saved{}, err
		}
	}
	steps := filepath.Join(root, name)
	old := ""
	if _, err := os.Stat(steps); err == nil {
		old = filepath.Join(root, "."+name+".old-"+now.Format("20060102-150405.000"))
		if err := os.Rename(steps, old); err != nil {
			return Saved{}, err
		}
	}
	if err := os.Rename(stage, steps); err != nil {
		if old != "" {
			os.Rename(old, steps) // put the previous steps back
		}
		return Saved{}, err
	}
	if old != "" {
		os.RemoveAll(old)
	}
	if err := textfile.WriteAtomic(filepath.Join(root, files[0].Name), []byte(files[0].Text), 0o644); err != nil {
		return Saved{}, err
	}
	for _, f := range files {
		saved.Written = append(saved.Written, f.Path)
	}
	return saved, nil
}

// Delete removes a recording's files from dir, after copying them to
// backups as Save does, and returns the backup's folder.
func Delete(dir, name, backups string, now time.Time) (string, error) {
	name = PathPart(name)
	backup, err := backUp(dir, name, backups, now, BackupDelete)
	if err != nil {
		return "", fmt.Errorf("backing up %s: %w", name, err)
	}
	if backup == "" {
		return "", apperr.New(apperr.NotFound, "There's no recording %q in %s", name, dir)
	}
	root := filepath.Join(dir, "McmRecorder")
	if err := os.RemoveAll(filepath.Join(root, name)); err != nil {
		return backup, err
	}
	if err := os.Remove(filepath.Join(root, name+".json")); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return backup, err
	}
	return backup, nil
}

// backUp copies a recording's files in dir to backups/<name>/<time>/, as
// they sit under McmRecorder, with a note saying where they came from and
// why, and returns that folder; "" when the recording has no files there.
func backUp(dir, name, backups string, now time.Time, reason BackupReason) (string, error) {
	files := Disk(dir, name)
	if len(files) == 0 {
		return "", nil
	}
	dest := filepath.Join(backups, name, now.Format("20060102-150405.000"))
	for n := 2; ; n++ {
		if _, err := os.Stat(dest); errors.Is(err, fs.ErrNotExist) {
			break
		}
		dest = filepath.Join(backups, name, fmt.Sprintf("%s-%d", now.Format("20060102-150405.000"), n))
	}
	for rel, text := range files {
		p := filepath.Join(dest, filepath.FromSlash(strings.TrimPrefix(rel, "McmRecorder/")))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			return "", err
		}
	}
	note, _ := json.MarshalIndent(backupNote{Reason: reason, Mod: filepath.Base(dir), Dir: dir, At: now}, "", "  ")
	if err := os.WriteFile(filepath.Join(dest, backupNoteName), note, 0o644); err != nil {
		return "", err
	}
	return dest, nil
}
