package recordings

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"lexicon/internal/apperr"
	"lexicon/internal/catalog"
	"lexicon/internal/httpapi"
	"lexicon/internal/modlist"
	"lexicon/internal/prefs"
	"lexicon/internal/textfile"
)

// DefaultTarget is the mod folder new recordings are saved to until the
// player chooses another.
const DefaultTarget = "MCM Lexicon Recordings"

// Summary is a recording without its steps, for the list of recordings.
type Summary struct {
	Name     string    `json:"name"`
	Title    string    `json:"title"`
	Version  string    `json:"version"`
	Author   string    `json:"author"`
	Autorun  bool      `json:"autorun"`
	Mod      string    `json:"mod"`
	Info     string    `json:"info"`
	Losing   []string  `json:"losing"`
	Steps    int       `json:"steps"`
	Files    int       `json:"files"`
	Menus    int       `json:"menus"` // the menus its steps name
	Counts   Counts    `json:"counts"`
	Problems []Problem `json:"problems"`
	Drafted  bool      `json:"drafted"` // it has changes not yet saved
	New      bool      `json:"new"`     // it's only a draft, never saved
	Deletes  string    `json:"deletes"` // why it can't be deleted; "" when it can
}

type ListReply struct {
	Recordings []Summary `json:"recordings"`
	// Readable is false when a catalog file is open: recordings live in a
	// list's files, so only new ones, made here, are listed.
	Readable bool   `json:"readable"`
	Author   string `json:"author"` // the author new recordings are written with, remembered
}

// PlannedFile is a file Save writes, beside what the target folder holds
// now, or one Save removes.
type PlannedFile struct {
	Name   string     `json:"name"`
	Path   string     `json:"path"` // relative to the target folder
	Text   string     `json:"text"` // "" for one Save removes
	Old    string     `json:"old"`  // what's on disk now; "" when it's new
	Change FileChange `json:"change"`
}

type FileChange string

const (
	FileChangeSame    FileChange = "same"
	FileChangeChanged FileChange = "changed"
	FileChangeNew     FileChange = "new"
	FileChangeRemoved FileChange = "removed"
)

// TargetKind is what kind of folder a recording saves to.
type TargetKind string

const (
	TargetKindMod       TargetKind = "mod"       // a mod folder in the list
	TargetKindNewMod    TargetKind = "new-mod"   // a mod folder Save creates, which MO2 will list unticked
	TargetKindOverwrite TargetKind = "overwrite" // MO2's overwrite folder
	TargetKindData      TargetKind = "data"      // the game's Data folder, for a list that is one
	TargetKindFolder    TargetKind = "folder"    // a folder chosen with a catalog file open
	TargetKindNone      TargetKind = "none"      // none chosen yet (a catalog file is open)
)

// Target is where Save writes, and what the game will make of it.
type Target struct {
	Name    string     `json:"name"` // the mod's name, or the folder's path
	Dir     string     `json:"dir"`  // the folder's path
	Kind    TargetKind `json:"kind"`
	Enabled bool       `json:"enabled"` // the game reads the folder: its mod is ticked in this profile
	Exists  bool       `json:"exists"`  // the folder is on disk (a new mod folder once Save has made it)
	LosesTo string     `json:"losesTo"` // a mod whose copy of the recording wins over this folder's
}

// Editor is a recording as the editor shows it: the draft (or the saved
// recording, when there's no draft), each step's check, and the files Save
// would write.
type Editor struct {
	Recording *Recording    `json:"recording"` // as the game sees it now; nil for one never saved
	Draft     Draft         `json:"draft"`
	Drafted   bool          `json:"drafted"` // a draft is held
	Changed   bool          `json:"changed"` // Save would change the target folder's files
	Checks    []Check       `json:"checks"`  // one per draft step
	Counts    Counts        `json:"counts"`
	Files     []PlannedFile `json:"files"`
	Target    Target        `json:"target"`
	Saved     *Saved        `json:"saved"`    // the last save this session
	Unsaved   string        `json:"unsaved"`  // why Save can't write now; "" when it can
	Deletes   string        `json:"deletes"`  // why it can't be deleted; "" when it can
	Targets   []string      `json:"targets"`  // mod folders to choose from: those with recordings, and the default
	Problems  []string      `json:"problems"` // what's wrong with the draft, so Save can't write it
}

// CreateRequest makes a new recording, as a draft.
type CreateRequest struct {
	Name    string `json:"name"`
	Author  string `json:"author"`
	Autorun bool   `json:"autorun"`
}

// AddRequest adds a step to a recording's draft.
type AddRequest struct {
	Step Step `json:"step"`
}

type AddReply struct {
	Number int `json:"number"` // the step's position in the recording, from 1
}

// DeleteReply says where a deleted recording's files went.
type DeleteReply struct {
	Backup string `json:"backup"`
}

// Service serves recordings from the list the catalog service last
// scanned, and holds the drafts: edits kept in memory, one per recording,
// until Save. Drafts last until the program stops or the list changes.
type Service struct {
	cat     *catalog.Service
	prefs   *prefs.Prefs
	running func() bool
	now     func() time.Time

	mu     sync.Mutex
	from   string            // the list or catalog file the drafts belong to
	drafts map[string]*Draft // by lowercase name
	saved  map[string]Saved  // the last save of each, by lowercase name
}

func NewService(cat *catalog.Service, p *prefs.Prefs, running func() bool) *Service {
	if running == nil {
		running = func() bool { return false }
	}
	return &Service{cat: cat, prefs: p, running: running, now: time.Now, drafts: map[string]*Draft{}, saved: map[string]Saved{}}
}

// env is what a request works against: the catalog, the list (nil with a
// catalog file open), and the recordings read from it.
type env struct {
	cat  *catalog.Catalog
	list *modlist.List
	file string // the catalog file open instead of a list
	recs []*Recording
	key  string // the list's or file's path, for prefs and drafts
}

func (s *Service) env(r *http.Request) (*env, error) {
	c, err := s.cat.Catalog(r)
	if err != nil {
		return nil, err
	}
	l, err := s.cat.List(r)
	if err != nil {
		return nil, err
	}
	src, file := s.cat.Source()
	e := &env{cat: c, list: l, file: file}
	switch {
	case src != nil:
		e.key = src.Path
	default:
		e.key = file
	}
	if l != nil {
		e.recs = Read(l)
	}
	s.mu.Lock()
	if s.from != e.key {
		s.from, s.drafts, s.saved = e.key, map[string]*Draft{}, map[string]Saved{}
	}
	s.mu.Unlock()
	return e, nil
}

func (e *env) recording(name string) *Recording {
	for _, r := range e.recs {
		if strings.EqualFold(r.Name, name) {
			return r
		}
	}
	return nil
}

func (s *Service) Routes(m *http.ServeMux) {
	m.Handle("GET /api/recordings", httpapi.JSON(func(r *http.Request, _ httpapi.None) (ListReply, error) {
		e, err := s.env(r)
		if err != nil {
			return ListReply{}, err
		}
		return s.list(e), nil
	}))
	m.Handle("POST /api/recordings", httpapi.JSON(func(r *http.Request, req CreateRequest) (Editor, error) {
		e, err := s.env(r)
		if err != nil {
			return Editor{}, err
		}
		return s.create(e, req)
	}))
	m.Handle("GET /api/recordings/{name}", httpapi.JSON(func(r *http.Request, _ httpapi.None) (Editor, error) {
		e, err := s.env(r)
		if err != nil {
			return Editor{}, err
		}
		return s.editor(e, r.PathValue("name"))
	}))
	m.Handle("PUT /api/recordings/{name}/draft", httpapi.JSON(func(r *http.Request, d Draft) (Editor, error) {
		e, err := s.env(r)
		if err != nil {
			return Editor{}, err
		}
		name := r.PathValue("name")
		if _, err := s.editor(e, name); err != nil {
			return Editor{}, err
		}
		if d.Steps == nil {
			d.Steps = []Step{}
		}
		d.Name = s.canonical(e, name)
		if d.Target, err = s.checkTarget(e, d.Target); err != nil {
			return Editor{}, err
		}
		s.mu.Lock()
		s.drafts[strings.ToLower(d.Name)] = &d
		s.mu.Unlock()
		return s.editor(e, name)
	}))
	m.Handle("DELETE /api/recordings/{name}/draft", httpapi.JSON(func(r *http.Request, _ httpapi.None) (Editor, error) {
		e, err := s.env(r)
		if err != nil {
			return Editor{}, err
		}
		name := r.PathValue("name")
		s.mu.Lock()
		delete(s.drafts, strings.ToLower(name))
		s.mu.Unlock()
		if e.recording(name) == nil {
			return Editor{}, nil // a new recording, discarded: nothing's left
		}
		return s.editor(e, name)
	}))
	m.Handle("POST /api/recordings/{name}/steps", httpapi.JSON(func(r *http.Request, req AddRequest) (AddReply, error) {
		e, err := s.env(r)
		if err != nil {
			return AddReply{}, err
		}
		return s.add(e, r.PathValue("name"), req.Step)
	}))
	m.Handle("POST /api/recordings/{name}/save", httpapi.JSON(func(r *http.Request, _ httpapi.None) (Editor, error) {
		e, err := s.env(r)
		if err != nil {
			return Editor{}, err
		}
		return s.save(e, r.PathValue("name"))
	}))
	m.Handle("DELETE /api/recordings/{name}", httpapi.JSON(func(r *http.Request, _ httpapi.None) (DeleteReply, error) {
		e, err := s.env(r)
		if err != nil {
			return DeleteReply{}, err
		}
		return s.delete(e, r.PathValue("name"))
	}))
}

func (s *Service) list(e *env) ListReply {
	reply := ListReply{Recordings: []Summary{}, Readable: e.list != nil, Author: s.prefs.Get().Author}
	s.mu.Lock()
	drafts := map[string]*Draft{}
	for k, d := range s.drafts {
		drafts[k] = d
	}
	s.mu.Unlock()
	for _, rec := range e.recs {
		sum := summarize(rec, e.cat)
		if d := drafts[strings.ToLower(rec.Name)]; d != nil {
			sum.Drafted = true
			delete(drafts, strings.ToLower(rec.Name))
		}
		sum.Deletes = s.deletes(e, rec)
		reply.Recordings = append(reply.Recordings, sum)
	}
	for _, d := range drafts {
		rec := &Recording{Name: d.Name, Title: d.Title, Author: d.Author, Version: d.Version, Autorun: d.Autorun, Steps: d.Steps, Losing: []string{}, Problems: []Problem{}, Files: []File{}}
		sum := summarize(rec, e.cat)
		sum.Drafted, sum.New, sum.Mod = true, true, d.Target
		sum.Deletes = "new"
		reply.Recordings = append(reply.Recordings, sum)
	}
	sort.Slice(reply.Recordings, func(i, j int) bool {
		return strings.ToLower(reply.Recordings[i].Name) < strings.ToLower(reply.Recordings[j].Name)
	})
	return reply
}

// canonical is a recording's name as it's spelled on disk or in its draft.
func (s *Service) canonical(e *env, name string) string {
	if r := e.recording(name); r != nil {
		return r.Name
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if d := s.drafts[strings.ToLower(name)]; d != nil {
		return d.Name
	}
	return name
}

func (s *Service) create(e *env, req CreateRequest) (Editor, error) {
	title := strings.TrimSpace(req.Name)
	name := PathPart(title)
	if name == "" || strings.Trim(name, ".") == "" {
		return Editor{}, apperr.New(apperr.Invalid, "A recording needs a name")
	}
	if e.recording(name) != nil {
		return Editor{}, apperr.New(apperr.Conflict, "There's already a recording named %q in this list", name)
	}
	s.mu.Lock()
	_, drafted := s.drafts[strings.ToLower(name)]
	s.mu.Unlock()
	if drafted {
		return Editor{}, apperr.New(apperr.Conflict, "There's already a new recording named %q", name)
	}
	author := strings.TrimSpace(req.Author)
	if author == "" {
		author = s.prefs.Get().Author
	} else {
		s.prefs.Update(func(v *prefs.Values) { v.Author = author })
	}
	d := &Draft{Name: name, Title: title, Version: "1.0.0", Author: author, Autorun: req.Autorun, Steps: []Step{}, Target: s.defaultTarget(e)}
	s.mu.Lock()
	s.drafts[strings.ToLower(name)] = d
	s.mu.Unlock()
	return s.editor(e, name)
}

// defaultTarget is where a new recording saves: the folder chosen last
// time, or MCM Lexicon's own mod folder.
func (s *Service) defaultTarget(e *env) string {
	switch {
	case e.list != nil && e.list.Source.Kind == modlist.Data:
		return ""
	case e.list != nil:
		if t := s.prefs.Get().SaveTarget; t != "" && !filepath.IsAbs(t) {
			return t
		}
		return DefaultTarget
	default:
		if t := s.prefs.Get().SaveTarget; filepath.IsAbs(t) {
			return t
		}
		return ""
	}
}

// checkTarget is a target as given, checked: a mod folder's name in an MO2
// instance (one in the list, overwrite, or a new one Save creates), or an
// absolute folder with a catalog file open. A Data folder has no choice.
func (s *Service) checkTarget(e *env, t string) (string, error) {
	t = strings.TrimSpace(t)
	switch {
	case e.list != nil && e.list.Source.Kind == modlist.Data:
		return "", nil
	case e.list != nil:
		if t == "" {
			return DefaultTarget, nil
		}
		if strings.ContainsAny(t, `/\:*?"<>|`) || strings.Trim(t, ". ") == "" {
			return "", apperr.New(apperr.Invalid, "%q isn't a mod folder's name", t)
		}
		if m, ok := e.list.Mod(t); ok {
			return m.Name, nil
		}
		return t, nil
	default:
		if t == "" {
			return "", nil
		}
		if !filepath.IsAbs(t) {
			return "", apperr.New(apperr.Invalid, "Choose a folder by its full path, such as C:\\Mods\\My Recordings")
		}
		return filepath.Clean(t), nil
	}
}

// target is where a recording's draft saves, and what the game will make
// of it.
func (s *Service) target(e *env, d Draft) Target {
	switch {
	case e.list != nil && e.list.Source.Kind == modlist.Data:
		return Target{Name: "Data", Dir: e.list.Source.Path, Kind: TargetKindData, Enabled: true}
	case e.list != nil:
		t := Target{Name: d.Target}
		if strings.EqualFold(d.Target, "overwrite") {
			t.Name, t.Kind, t.Enabled = "overwrite", TargetKindOverwrite, true
			t.Dir = filepath.Join(e.list.Source.Path, "overwrite")
		} else if m, ok := e.list.Mod(d.Target); ok && m.Layer == modlist.LayerMod {
			t.Name, t.Kind, t.Enabled = m.Name, TargetKindMod, m.Enabled
			t.Dir = filepath.Join(e.list.Source.Path, "mods", m.Name)
		} else {
			t.Kind = TargetKindNewMod
			t.Dir = filepath.Join(e.list.Source.Path, "mods", d.Target)
		}
		// Another copy of the info file that the game sees instead.
		rank := map[string]int{}
		for i, m := range e.list.Mods {
			rank[strings.ToLower(m.Name)] = i
		}
		mine, listed := rank[strings.ToLower(t.Name)]
		for _, c := range e.list.Copies("McmRecorder/" + PathPart(d.Name) + ".json") {
			if strings.EqualFold(c.Mod, t.Name) {
				continue
			}
			if r, ok := rank[strings.ToLower(c.Mod)]; ok && (!listed || r > mine) {
				t.LosesTo = c.Mod
				break
			}
		}
		t.Exists = isDir(t.Dir)
		return t
	default:
		if d.Target == "" {
			return Target{Kind: TargetKindNone}
		}
		return Target{Name: d.Target, Dir: d.Target, Kind: TargetKindFolder, Enabled: true, Exists: isDir(d.Target)}
	}
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// backups is where Save and Delete keep the files they replace: beside the
// list, where the game doesn't look.
func (e *env) backups(t Target) string {
	switch {
	case e.list != nil && e.list.Source.Kind == modlist.Data:
		return filepath.Join(filepath.Dir(e.list.Source.Path), "lexicon-backups")
	case e.list != nil:
		return filepath.Join(e.list.Source.Path, "lexicon-backups")
	default:
		return filepath.Join(t.Dir, "lexicon-backups")
	}
}

func (s *Service) editor(e *env, name string) (Editor, error) {
	rec := e.recording(name)
	s.mu.Lock()
	held := s.drafts[strings.ToLower(name)]
	var d Draft
	if held != nil {
		d = *held
		d.Steps = append([]Step{}, held.Steps...)
	}
	saved, wasSaved := s.saved[strings.ToLower(name)]
	s.mu.Unlock()
	if rec == nil && held == nil {
		if e.list == nil {
			return Editor{}, apperr.New(apperr.NotFound, "There's no new recording %q. With a catalog file open, recordings are made here and saved to a folder", name)
		}
		return Editor{}, apperr.New(apperr.NotFound, "There's no recording %q in this list", name)
	}
	if held == nil {
		d = DraftOf(rec, s.homeOf(e, rec))
	}
	ed := Editor{Recording: rec, Drafted: held != nil, Problems: []string{}, Targets: s.targets(e, d.Target)}
	files, err := Plan(&d)
	if err != nil {
		ed.Problems = append(ed.Problems, err.Error())
	}
	ed.Draft = d
	ed.Checks = Validate(&Recording{Steps: d.Steps}, e.cat)
	ed.Counts = Count(ed.Checks)
	ed.Target = s.target(e, d)
	ed.Files = []PlannedFile{}
	if ed.Target.Dir != "" && err == nil {
		disk := Disk(ed.Target.Dir, d.Name)
		for _, f := range files {
			p := PlannedFile{Name: f.Name, Path: f.Path, Text: f.Text, Change: FileChangeNew}
			if old, ok := disk[f.Path]; ok {
				p.Old = old
				p.Change = FileChangeChanged
				if textfile.Normalize(old) == f.Text || sameJSON(old, f.Text) {
					p.Change = FileChangeSame
				}
				delete(disk, f.Path)
			}
			ed.Files = append(ed.Files, p)
		}
		var gone []string
		for path := range disk {
			gone = append(gone, path)
		}
		sort.Strings(gone)
		for _, path := range gone {
			ed.Files = append(ed.Files, PlannedFile{Name: filepath.Base(path), Path: path, Old: disk[path], Change: FileChangeRemoved})
		}
		for _, f := range ed.Files {
			if f.Change != FileChangeSame {
				ed.Changed = true
			}
		}
	} else if err == nil {
		for _, f := range files {
			ed.Files = append(ed.Files, PlannedFile{Name: f.Name, Path: f.Path, Text: f.Text, Change: FileChangeNew})
		}
		ed.Changed = true
	}
	switch {
	case s.running():
		ed.Unsaved = "running"
	case ed.Target.Kind == TargetKindNone:
		ed.Unsaved = "no-target"
	case len(ed.Problems) > 0:
		ed.Unsaved = "problems"
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

// sameJSON reports whether two files say the same thing to MCM Recorder,
// which reads keys without regard to case, in any order, and compares text
// without regard to case: a file written by the game or by hand that Save
// would only reformat isn't a change.
func sameJSON(a, b string) bool {
	var x, y any
	if json.Unmarshal(bytes.TrimPrefix([]byte(a), []byte{0xEF, 0xBB, 0xBF}), &x) != nil || json.Unmarshal([]byte(b), &y) != nil {
		return false
	}
	return sameValue(x, y)
}

func sameValue(a, b any) bool {
	switch x := a.(type) {
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok {
			return false
		}
		lx, ly := lower(x), lower(y)
		if len(lx) != len(ly) {
			return false
		}
		for k, v := range lx {
			if w, ok := ly[k]; !ok || !sameValue(v, w) {
				return false
			}
		}
		return true
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !sameValue(x[i], y[i]) {
				return false
			}
		}
		return true
	case string:
		y, ok := b.(string)
		return ok && strings.EqualFold(x, y)
	default:
		return a == b // numbers as float64: 25 and 25.0 alike
	}
}

// homeOf is the folder a saved recording saves back to: the mod holding
// the copy of its info file the game reads.
func (s *Service) homeOf(e *env, r *Recording) string {
	switch {
	case e.list != nil && e.list.Source.Kind == modlist.Data:
		return ""
	case r.Mod != "":
		return r.Mod
	default:
		return s.defaultTarget(e)
	}
}

// targets is the mod folders offered for saving: those holding recordings,
// the default, and the current one.
func (s *Service) targets(e *env, current string) []string {
	if e.list == nil || e.list.Source.Kind == modlist.Data {
		return []string{}
	}
	seen := map[string]bool{}
	var out []string
	addIf := func(name string) {
		if name != "" && !seen[strings.ToLower(name)] {
			seen[strings.ToLower(name)] = true
			out = append(out, name)
		}
	}
	for _, f := range e.list.Files("McmRecorder") {
		addIf(f.Mod)
	}
	addIf(DefaultTarget)
	addIf(current)
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i]) < strings.ToLower(out[j]) })
	return out
}

// deletes is why a recording can't be deleted, or "" when it can: MCM
// Lexicon deletes only recordings in folders it has saved recordings to,
// never a mod list's own, and not while Skyrim runs.
func (s *Service) deletes(e *env, r *Recording) string {
	if e.list == nil {
		return "no-list"
	}
	if !slices.ContainsFunc(s.prefs.Get().WroteTo[e.key], func(m string) bool { return strings.EqualFold(m, r.Mod) }) {
		return "not-ours"
	}
	if s.running() {
		return "running"
	}
	return ""
}

func (s *Service) add(e *env, name string, step Step) (AddReply, error) {
	ed, err := s.editor(e, name)
	if err != nil {
		return AddReply{}, err
	}
	d := ed.Draft
	step.File = 0
	if step.Kind == KindClick {
		step.Value = step.Option
	}
	step.Wildcard = len(step.Option) > 2 && strings.HasPrefix(step.Option, "*") && strings.HasSuffix(step.Option, "*")
	probe := Draft{Name: "probe", Steps: []Step{step}}
	if _, err := Plan(&probe); err != nil {
		return AddReply{}, apperr.New(apperr.Invalid, "That step can't be written: %v", err)
	}
	d.Steps = append(d.Steps, step)
	s.mu.Lock()
	s.drafts[strings.ToLower(d.Name)] = &d
	s.mu.Unlock()
	return AddReply{Number: len(d.Steps)}, nil
}

func (s *Service) save(e *env, name string) (Editor, error) {
	ed, err := s.editor(e, name)
	if err != nil {
		return Editor{}, err
	}
	switch ed.Unsaved {
	case "running":
		return Editor{}, apperr.New(apperr.Conflict, "Skyrim is running, so nothing can be written until you close it")
	case "no-target":
		return Editor{}, apperr.New(apperr.Invalid, "Choose a folder to save the recording to")
	case "problems":
		return Editor{}, apperr.New(apperr.Invalid, "The recording can't be written: %s", strings.Join(ed.Problems, "; "))
	}
	if err := os.MkdirAll(ed.Target.Dir, 0o755); err != nil {
		return Editor{}, err
	}
	done, err := Save(ed.Target.Dir, ed.Draft, e.backups(ed.Target), s.now())
	if err != nil {
		return Editor{}, err
	}
	key := strings.ToLower(ed.Draft.Name)
	s.mu.Lock()
	delete(s.drafts, key)
	s.saved[key] = done
	s.mu.Unlock()
	s.prefs.Update(func(v *prefs.Values) {
		if ed.Recording == nil {
			v.SaveTarget = ed.Draft.Target
		}
		if e.list != nil {
			if v.WroteTo == nil {
				v.WroteTo = map[string][]string{}
			}
			if !slices.ContainsFunc(v.WroteTo[e.key], func(m string) bool { return strings.EqualFold(m, ed.Target.Name) }) {
				v.WroteTo[e.key] = append(v.WroteTo[e.key], ed.Target.Name)
			}
		}
	})
	if e.list != nil {
		e.list.Refresh("McmRecorder")
		e.recs = Read(e.list)
	}
	if e.list == nil || e.recording(name) == nil {
		// Saved where this list doesn't see it (a new mod folder, unticked,
		// or a folder with a catalog file open): keep showing the draft.
		d := ed.Draft
		s.mu.Lock()
		s.drafts[key] = &d
		s.mu.Unlock()
	}
	return s.editor(e, name)
}

func (s *Service) delete(e *env, name string) (DeleteReply, error) {
	rec := e.recording(name)
	if rec == nil {
		return DeleteReply{}, apperr.New(apperr.NotFound, "There's no recording %q in this list", name)
	}
	switch s.deletes(e, rec) {
	case "running":
		return DeleteReply{}, apperr.New(apperr.Conflict, "Skyrim is running, so nothing can be deleted until you close it")
	case "not-ours":
		return DeleteReply{}, apperr.New(apperr.Conflict, "%s is in %s, which MCM Lexicon didn't save it to, so it isn't deleted here", rec.Name, rec.Mod)
	case "no-list":
		return DeleteReply{}, apperr.New(apperr.Unavailable, "Recordings are deleted from a mod list, and a catalog file is open")
	}
	t := s.target(e, Draft{Name: rec.Name, Target: rec.Mod})
	backup, err := Delete(t.Dir, rec.Name, e.backups(t), s.now())
	if err != nil {
		return DeleteReply{}, err
	}
	s.mu.Lock()
	delete(s.drafts, strings.ToLower(rec.Name))
	delete(s.saved, strings.ToLower(rec.Name))
	s.mu.Unlock()
	e.list.Refresh("McmRecorder")
	s.prefs.Update(func(v *prefs.Values) {
		if strings.EqualFold(v.OpenRecording, rec.Name) {
			v.OpenRecording = ""
		}
	})
	return DeleteReply{Backup: backup}, nil
}

func summarize(r *Recording, c *catalog.Catalog) Summary {
	checks := Validate(r, c)
	menus := map[string]bool{}
	for _, s := range r.Steps {
		menus[strings.ToLower(s.Mod)] = true
	}
	return Summary{
		Name: r.Name, Title: r.Title, Version: r.Version, Author: r.Author, Autorun: r.Autorun,
		Mod: r.Mod, Info: r.Info, Losing: r.Losing, Steps: len(r.Steps), Files: len(r.Files), Menus: len(menus),
		Counts: Count(checks), Problems: r.Problems,
	}
}

// CompareRequest names the catalog file to compare the current catalog
// with, as the older of the two.
type CompareRequest struct {
	Path string `json:"path"`
}

// Break is a recorded step that matched in the older catalog and doesn't
// in the current one: what an update broke.
type Break struct {
	Recording string `json:"recording"`
	Step      int    `json:"step"` // its position in the recording, from 1
	Mod       string `json:"mod"`
	Page      string `json:"page"`
	Option    string `json:"option"`
	Kind      Kind   `json:"kind"`
	Check     Check  `json:"check"` // against the current catalog
}

type CompareReply struct {
	Diff   catalog.Diff `json:"diff"`
	Breaks []Break      `json:"breaks"`
}

// CompareRoutes serves comparing the current catalog with a catalog file.
func CompareRoutes(m *http.ServeMux, svc *catalog.Service) {
	m.Handle("POST /api/compare", httpapi.JSON(func(r *http.Request, req CompareRequest) (CompareReply, error) {
		cur, err := svc.Catalog(r)
		if err != nil {
			return CompareReply{}, err
		}
		old, err := catalog.ReadFile(strings.TrimSpace(req.Path))
		if err != nil {
			return CompareReply{}, err
		}
		reply := CompareReply{Diff: catalog.Compare(old, cur), Breaks: []Break{}}
		l, err := svc.List(r)
		if err != nil || l == nil {
			return reply, err
		}
		for _, rec := range Read(l) {
			before, after := Validate(rec, old), Validate(rec, cur)
			for i, s := range rec.Steps {
				if before[i].Status == StatusMatched && after[i].Status != StatusMatched {
					reply.Breaks = append(reply.Breaks, Break{Recording: rec.Name, Step: i + 1, Mod: s.Mod, Page: s.Page, Option: s.Option, Kind: s.Kind, Check: after[i]})
				}
			}
		}
		return reply, nil
	}))
}
