package recordings

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"lexicon/internal/apperr"
	"lexicon/internal/modlist"
	"lexicon/internal/textfile"
)

// BackupReason is what made a backup.
type BackupReason string

const (
	BackupSave    BackupReason = "save"    // a Save replaced the files
	BackupDelete  BackupReason = "delete"  // a Save deleted the recording
	BackupRestore BackupReason = "restore" // a Save put a backup back in their place
)

// backupNoteName is the note in each backup saying where its files came
// from and why; a backup from before the notes has none.
const backupNoteName = "lexicon-backup.json"

type backupNote struct {
	Reason BackupReason `json:"reason"`
	Mod    string       `json:"mod"` // the mod folder the files were in
	Dir    string       `json:"dir"`
	At     time.Time    `json:"at"`
}

// Backup is a recording's files as a Save or a delete found them, before
// it wrote: a folder under the list's lexicon-backups.
type Backup struct {
	ID     string       `json:"id"`     // its folder's name, the time it was made
	At     time.Time    `json:"at"`     // when it was made
	Reason BackupReason `json:"reason"` // "" for a backup from before backups said
	Mod    string       `json:"mod"`    // the mod folder the files were in; "" when the backup doesn't say
	Steps  int          `json:"steps"`
	Files  int          `json:"files"`
	dir    string       // where it is
}

// backupTime reads a backup folder's name, which is the time it was made
// (20061102-150405.000, with -2 and so on for two in the same moment).
func backupTime(id string) time.Time {
	const layout = "20060102-150405.000"
	if len(id) < len(layout) {
		return time.Time{}
	}
	t, err := time.ParseInLocation(layout, id[:len(layout)], time.Local)
	if err != nil {
		return time.Time{}
	}
	return t
}

// Backups is every backup of the recording name under root, newest first.
func Backups(root, name string) []Backup {
	entries, err := os.ReadDir(filepath.Join(root, PathPart(name)))
	if err != nil {
		return []Backup{}
	}
	out := []Backup{}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		b := Backup{ID: e.Name(), At: backupTime(e.Name()), dir: filepath.Join(root, PathPart(name), e.Name())}
		if raw, err := os.ReadFile(filepath.Join(b.dir, backupNoteName)); err == nil {
			var n backupNote
			if json.Unmarshal(raw, &n) == nil {
				b.Reason, b.Mod = n.Reason, n.Mod
				if !n.At.IsZero() {
					b.At = n.At
				}
			}
		}
		files := backupFiles(b.dir, name)
		b.Files = len(files)
		if r := backupRecording(b.dir, name, b.Mod); r != nil {
			b.Steps = len(r.Steps)
		}
		out = append(out, b)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out
}

// findBackup is the backup id of name under root.
func findBackup(root, name, id string) (Backup, error) {
	if id == "" || strings.ContainsAny(id, `/\`) || id == "." || id == ".." {
		return Backup{}, apperr.New(apperr.Invalid, "That isn't a backup's name")
	}
	for _, b := range Backups(root, name) {
		if b.ID == id {
			return b, nil
		}
	}
	return Backup{}, apperr.New(apperr.NotFound, "There's no backup %s of %s", id, name)
}

// backupFiles is a backup's files by the path each had under the mod
// folder (McmRecorder/<name>.json, McmRecorder/<name>/<step file>), as
// Disk gives a recording's: the exact text to put back.
func backupFiles(dir, name string) map[string]string {
	out := map[string]string{}
	name = PathPart(name)
	if b, err := os.ReadFile(filepath.Join(dir, name+".json")); err == nil {
		out["McmRecorder/"+name+".json"] = string(b)
	}
	entries, _ := os.ReadDir(filepath.Join(dir, name))
	for _, e := range entries {
		if e.IsDir() || modlist.Clutter(e.Name()) {
			continue
		}
		if b, err := os.ReadFile(filepath.Join(dir, name, e.Name())); err == nil {
			out["McmRecorder/"+name+"/"+e.Name()] = string(b)
		}
	}
	return out
}

// backupRecording reads a backup as a recording, to show its steps.
func backupRecording(dir, name, mod string) *Recording {
	name = PathPart(name)
	var files []modlist.File
	if p := filepath.Join(dir, name+".json"); fileExists(p) {
		files = append(files, modlist.File{Path: "McmRecorder/" + name + ".json", Abs: p, Mod: mod})
	}
	entries, _ := os.ReadDir(filepath.Join(dir, name))
	for _, e := range entries {
		if !e.IsDir() && !modlist.Clutter(e.Name()) {
			files = append(files, modlist.File{Path: "McmRecorder/" + name + "/" + e.Name(), Abs: filepath.Join(dir, name, e.Name()), Mod: mod})
		}
	}
	sort.Slice(files, func(i, j int) bool { return strings.ToLower(files[i].Path) < strings.ToLower(files[j].Path) })
	recs := read(files, func(string) []modlist.File { return []modlist.File{{}} })
	if len(recs) == 0 {
		return nil
	}
	return recs[0]
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// Put writes files, a recording's exact files by their path under the mod
// folder, into dir in place of what the recording has there: a backup put
// back. Like Save, it backs the current files up first and swaps the step
// files in whole; a file the backup doesn't have is removed.
func Put(dir, name string, files map[string]string, backups string, now time.Time) (Saved, error) {
	name = PathPart(name)
	root := filepath.Join(dir, "McmRecorder")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return Saved{}, err
	}
	saved := Saved{At: now, Dir: dir}
	backup, err := backUp(dir, name, backups, now, BackupRestore)
	if err != nil {
		return Saved{}, err
	}
	saved.Backup = backup
	stage, err := os.MkdirTemp(root, "."+name+".new-*")
	if err != nil {
		return Saved{}, err
	}
	defer os.RemoveAll(stage)
	prefix := "McmRecorder/" + name + "/"
	var paths []string
	for p := range files {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	for _, p := range paths {
		if rest, ok := strings.CutPrefix(p, prefix); ok && !strings.Contains(rest, "/") {
			if err := os.WriteFile(filepath.Join(stage, rest), []byte(files[p]), 0o644); err != nil {
				return Saved{}, err
			}
			saved.Written = append(saved.Written, p)
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
			os.Rename(old, steps)
		}
		return Saved{}, err
	}
	if old != "" {
		os.RemoveAll(old)
	}
	info := "McmRecorder/" + name + ".json"
	if text, ok := files[info]; ok {
		if err := textfile.WriteAtomic(filepath.Join(root, name+".json"), []byte(text), 0o644); err != nil {
			return Saved{}, err
		}
		saved.Written = append([]string{info}, saved.Written...)
	} else if err := os.Remove(filepath.Join(root, name+".json")); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Saved{}, err
	}
	return saved, nil
}

// restoreTarget is the folder a backup goes back into: the recording's
// own while it's there, else the folder its backup came from, else the
// one new recordings go to.
func (s *Service) restoreTarget(e *env, rec *Recording, mod string) string {
	switch {
	case rec != nil:
		return s.homeOf(e, rec)
	case mod != "":
		return mod
	}
	return s.defaultTarget(e)
}

// editorFromBackup is a recording as the draft puts it back from a backup:
// the backup's steps, and its files exactly, against what's on disk.
func (s *Service) editorFromBackup(e *env, rec *Recording, fb fromBackup, saved Saved, wasSaved bool) (Editor, error) {
	mod := s.restoreTarget(e, rec, fb.Mod)
	target := s.target(e, Draft{Name: fb.Name, Target: mod})
	b, err := findBackup(e.backups(target), fb.Name, fb.ID)
	if err != nil {
		return Editor{}, err
	}
	d := Draft{Name: fb.Name, Steps: []Step{}, Target: mod}
	if r := backupRecording(b.dir, fb.Name, mod); r != nil {
		d = DraftOf(r, mod)
		d.Name = fb.Name
	}
	ed := Editor{Recording: rec, Drafted: true, Draft: d, FromBackup: &b, Problems: []string{}, Targets: s.targets(e, mod), Target: target}
	ed.Checks = Validate(&Recording{Steps: d.Steps}, e.cat)
	ed.Counts = Count(ed.Checks)
	ed.Files = []PlannedFile{}
	files := backupFiles(b.dir, fb.Name)
	disk := map[string]string{}
	if target.Dir != "" {
		disk = Disk(target.Dir, fb.Name)
	}
	var paths []string
	for p := range files {
		paths = append(paths, p)
	}
	sort.Slice(paths, func(i, j int) bool {
		// The info file first, then the step files, as Save lists them.
		ii, jj := !strings.Contains(strings.TrimPrefix(paths[i], "McmRecorder/"), "/"), !strings.Contains(strings.TrimPrefix(paths[j], "McmRecorder/"), "/")
		if ii != jj {
			return ii
		}
		return paths[i] < paths[j]
	})
	for _, p := range paths {
		f := PlannedFile{Name: filepath.Base(p), Path: p, Text: files[p], Change: FileChangeNew}
		if old, ok := disk[p]; ok {
			f.Old, f.Change = old, FileChangeChanged
			if old == files[p] {
				f.Change = FileChangeSame
			}
			delete(disk, p)
		}
		ed.Files = append(ed.Files, f)
	}
	var gone []string
	for p := range disk {
		gone = append(gone, p)
	}
	sort.Strings(gone)
	for _, p := range gone {
		ed.Files = append(ed.Files, PlannedFile{Name: filepath.Base(p), Path: p, Old: disk[p], Change: FileChangeRemoved})
	}
	for _, f := range ed.Files {
		if f.Change != FileChangeSame {
			ed.Changed = true
		}
	}
	switch {
	case s.running():
		ed.Unsaved = "running"
	case target.Kind == TargetKindNone:
		ed.Unsaved = "no-target"
	case !ed.Changed:
		ed.Unsaved = "unchanged"
	}
	if wasSaved {
		ed.Saved = &saved
	}
	if rec != nil {
		ed.Deletes = s.deletes(e, rec)
	} else {
		ed.Deletes = "new"
	}
	return ed, nil
}

// BackupView is a backup with its files as it would put them back, against
// what's on disk now.
type BackupView struct {
	Backup  `tstype:",extends"`
	Planned []PlannedFile `json:"planned"`
}

type BackupsReply struct {
	Backups []BackupView `json:"backups"` // newest first
	// Restoring is the backup the draft puts back, by ID; "" for none.
	Restoring string `json:"restoring"`
}

// DeletedBackups is a recording the list no longer has, with its backups.
type DeletedBackups struct {
	Name   string `json:"name"`
	Latest Backup `json:"latest"`
	Count  int    `json:"count"`
}

type DeletedReply struct {
	Recordings []DeletedBackups `json:"recordings"`
}

// spelled is name as its backups folder spells it, for a recording the
// list no longer has; name when there's no such folder.
func spelled(root, name string) string {
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if e.IsDir() && strings.EqualFold(e.Name(), PathPart(name)) {
			return e.Name()
		}
	}
	return name
}

// backupsOf is every backup of name, each with its files against what's
// on disk where it would go back.
func (s *Service) backupsOf(e *env, name string) BackupsReply {
	rec := e.recording(name)
	reply := BackupsReply{Backups: []BackupView{}}
	s.mu.Lock()
	if fb, ok := s.fromBackups[strings.ToLower(name)]; ok && s.drafts[strings.ToLower(name)] == nil && !s.deleted[strings.ToLower(name)] {
		reply.Restoring = fb.ID
	}
	s.mu.Unlock()
	if rec != nil {
		name = rec.Name
	}
	root := e.backups(s.target(e, Draft{Name: name, Target: s.restoreTarget(e, rec, "")}))
	if rec == nil {
		name = spelled(root, name)
	}
	for _, b := range Backups(root, name) {
		target := s.target(e, Draft{Name: name, Target: s.restoreTarget(e, rec, b.Mod)})
		disk := map[string]string{}
		if target.Dir != "" {
			disk = Disk(target.Dir, name)
		}
		v := BackupView{Backup: b, Planned: []PlannedFile{}}
		files := backupFiles(b.dir, name)
		var paths []string
		for p := range files {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		for _, p := range paths {
			f := PlannedFile{Name: filepath.Base(p), Path: p, Text: files[p], Change: FileChangeNew}
			if old, ok := disk[p]; ok {
				f.Old, f.Change = old, FileChangeChanged
				if old == files[p] {
					f.Change = FileChangeSame
				}
				delete(disk, p)
			}
			v.Planned = append(v.Planned, f)
		}
		var gone []string
		for p := range disk {
			gone = append(gone, p)
		}
		sort.Strings(gone)
		for _, p := range gone {
			v.Planned = append(v.Planned, PlannedFile{Name: filepath.Base(p), Path: p, Old: disk[p], Change: FileChangeRemoved})
		}
		reply.Backups = append(reply.Backups, v)
	}
	return reply
}

// restoreFrom puts the backup id of name in the draft, in place of any
// other change to it: Save then writes the backup's files exactly. A
// backup that matches what's on disk leaves nothing to save.
func (s *Service) restoreFrom(e *env, name, id string) (Editor, error) {
	rec := e.recording(name)
	if rec != nil {
		name = rec.Name
	}
	root := e.backups(s.target(e, Draft{Name: name, Target: s.restoreTarget(e, rec, "")}))
	if rec == nil {
		name = spelled(root, name)
	}
	b, err := findBackup(root, name, id)
	if err != nil {
		return Editor{}, err
	}
	key := strings.ToLower(name)
	s.mu.Lock()
	delete(s.drafts, key)
	delete(s.deleted, key)
	s.fromBackups[key] = fromBackup{Name: name, ID: b.ID, Mod: s.restoreTarget(e, rec, b.Mod)}
	s.persist()
	s.mu.Unlock()
	ed, err := s.editor(e, name)
	if err == nil && !ed.Changed && rec != nil {
		s.mu.Lock()
		delete(s.fromBackups, key)
		s.amend()
		s.mu.Unlock()
		return s.editor(e, name)
	}
	return ed, err
}

// deletedWithBackups is each recording the list no longer has that has a
// backup, newest first.
func (s *Service) deletedWithBackups(e *env) DeletedReply {
	reply := DeletedReply{Recordings: []DeletedBackups{}}
	if e.list == nil {
		return reply
	}
	root := e.backups(Target{})
	entries, _ := os.ReadDir(root)
	for _, d := range entries {
		if !d.IsDir() || strings.HasPrefix(d.Name(), ".") || e.recording(d.Name()) != nil {
			continue
		}
		s.mu.Lock()
		_, drafted := s.drafts[strings.ToLower(d.Name())]
		s.mu.Unlock()
		if drafted {
			continue
		}
		list := Backups(root, d.Name())
		if len(list) == 0 {
			continue
		}
		reply.Recordings = append(reply.Recordings, DeletedBackups{Name: d.Name(), Latest: list[0], Count: len(list)})
	}
	sort.Slice(reply.Recordings, func(i, j int) bool { return reply.Recordings[i].Latest.ID > reply.Recordings[j].Latest.ID })
	return reply
}
