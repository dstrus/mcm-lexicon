package recordings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
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
	Drafted  bool      `json:"drafted"`  // it has changes not yet saved
	Deleted  bool      `json:"deleted"`  // the draft deletes it
	New      bool      `json:"new"`      // it's only a draft, never saved
	Deletes  string    `json:"deletes"`  // why it can't be deleted; "" when it can
	Unlisted bool      `json:"unlisted"` // in a folder MCM Lexicon saved to that the profile doesn't enable
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
	TargetKindNewMod    TargetKind = "new-mod"   // a mod folder Save creates, which MO2 will list disabled
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
	Enabled bool       `json:"enabled"` // the game reads the folder: its mod is enabled in this profile
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
	Deleted   bool          `json:"deleted"`  // the draft deletes it: Save removes its files
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

// AddAllRequest records many options at once, in order (Record all settings
// on this page). An option the recording has already keeps its step.
type AddAllRequest struct {
	Steps []Step `json:"steps"`
}

type AddAllReply struct {
	Added int `json:"added"` // new steps
	Kept  int `json:"kept"`  // options the recording had already, left as they were
}

type AddReply struct {
	Number   int  `json:"number"`   // the step's position in the recording, from 1
	Replaced bool `json:"replaced"` // the option was in the recording already, and its step now has this value
}

// DeleteReply says where a deleted recording's files went.
type DeleteReply struct {
	Backup string `json:"backup"`
}

// Service serves recordings from the list the catalog service last
// scanned, and holds the drafts: edits kept in memory, one per recording,
// until Save. The draft is written to a file beside the settings as it
// changes and read back at the next start, so stopping the program loses
// nothing; it lasts until it's saved, discarded, or the list changes.
type Service struct {
	cat     *catalog.Service
	prefs   *prefs.Prefs
	running func() bool
	now     func() time.Time
	file    string // where the draft is kept between runs; "" keeps it in memory only

	mu      sync.Mutex
	editing bool              // changes are allowed, into the draft; off, everything is read-only
	from    string            // the list or catalog file the draft belongs to
	drafts  map[string]*Draft // the draft's recordings, new or changed, by lowercase name
	deleted map[string]bool   // the recordings the draft deletes, by lowercase name
	kept    map[string]*Draft // saved where this list doesn't see them (a new mod folder), shown as saved
	created map[string]bool   // the draft's new recordings, never saved, by lowercase name
	saved   map[string]Saved  // the last save of each, by lowercase name
}

func NewService(cat *catalog.Service, p *prefs.Prefs, running func() bool) *Service {
	if running == nil {
		running = func() bool { return false }
	}
	s := &Service{cat: cat, prefs: p, running: running, now: time.Now, file: p.Beside("draft.json")}
	s.reset()
	s.restore()
	return s
}

// reset forgets the draft. Callers hold s.mu, or have s to themselves.
func (s *Service) reset() {
	s.drafts, s.deleted, s.kept, s.saved, s.created = map[string]*Draft{}, map[string]bool{}, map[string]*Draft{}, map[string]Saved{}, map[string]bool{}
}

// held is the draft as its file keeps it: what it changes, and the list
// it belongs to.
type held struct {
	From    string            `json:"from"`
	Drafts  map[string]*Draft `json:"drafts"`
	Deleted map[string]bool   `json:"deleted,omitempty"`
	Created map[string]bool   `json:"created,omitempty"`
}

// persist writes the draft to its file, or removes the file when the draft
// is empty. Failing to write is logged, not returned: the draft is still
// held until the program stops. Callers hold s.mu.
func (s *Service) persist() {
	if s.file == "" {
		return
	}
	if len(s.drafts)+len(s.deleted) == 0 {
		if err := os.Remove(s.file); err != nil && !os.IsNotExist(err) {
			log.Printf("Could not remove the draft file: %v", err)
		}
		return
	}
	b, _ := json.MarshalIndent(held{From: s.from, Drafts: s.drafts, Deleted: s.deleted, Created: s.created}, "", "  ")
	err := os.MkdirAll(filepath.Dir(s.file), 0o755)
	if err == nil {
		err = os.WriteFile(s.file, b, 0o644)
	}
	if err != nil {
		log.Printf("Could not keep the draft for next time: %v", err)
	}
}

// restore reads the draft the last run left, if it left one. A draft with
// changes turns editing on, so they're in view and can be saved; it's
// dropped if another list is opened, as it would have been then.
func (s *Service) restore() {
	if s.file == "" {
		return
	}
	b, err := os.ReadFile(s.file)
	if err != nil {
		return
	}
	var h held
	if err := json.Unmarshal(b, &h); err != nil || h.From == "" {
		log.Printf("Could not read the draft kept at %s; it's ignored", s.file)
		return
	}
	for k, d := range h.Drafts {
		if d == nil || d.Name == "" {
			delete(h.Drafts, k)
		}
	}
	if len(h.Drafts)+len(h.Deleted) == 0 {
		return
	}
	s.from = h.From
	for k, d := range h.Drafts {
		s.drafts[strings.ToLower(k)] = d
	}
	for k, v := range h.Deleted {
		if v {
			s.deleted[strings.ToLower(k)] = true
		}
	}
	for k, v := range h.Created {
		if v && s.drafts[strings.ToLower(k)] != nil {
			s.created[strings.ToLower(k)] = true
		}
	}
	s.editing = true
	if n := len(s.drafts) + len(s.deleted); n == 1 {
		log.Print("Restored the draft from last time: changes to 1 recording, not yet saved")
	} else {
		log.Printf("Restored the draft from last time: changes to %d recordings, not yet saved", n)
	}
}

// Editing reports whether changes are allowed. MCM Lexicon starts
// read-only; the player turns editing on.
func (s *Service) Editing() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.editing
}

// Pending is how many recordings the draft changes: new, changed, or
// deleted.
func (s *Service) Pending() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.drafts) + len(s.deleted)
}

// SetEditing turns editing on or off. Turning it off with changes pending
// is refused unless discard says to drop them.
func (s *Service) SetEditing(on, discard bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !on && len(s.drafts)+len(s.deleted) > 0 {
		if !discard {
			return apperr.New(apperr.Conflict, "The draft has changes to %d recordings. Save them or discard them first", len(s.drafts)+len(s.deleted))
		}
		gone := s.dropDrafts(nil)
		s.deleted = map[string]bool{}
		s.persist()
		defer s.forgetOpen(gone)
	}
	s.editing = on
	return nil
}

// Clear forgets the draft and turns editing off, for another list.
func (s *Service) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reset()
	s.persist()
	s.editing = false
}

// mustEdit refuses a change while editing is off.
func (s *Service) mustEdit() error {
	if !s.Editing() {
		return apperr.New(apperr.Conflict, "MCM Lexicon is read-only. Turn on editing to change recordings")
	}
	return nil
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
	s.mu.Lock()
	if s.from != e.key {
		// Another list: the draft belonged to the last one.
		s.from = e.key
		s.reset()
		s.persist()
	}
	s.mu.Unlock()
	s.read(e)
	return e, nil
}

// read reads the recordings: those the list's game sees, then those in the
// folders MCM Lexicon saved to that the profile doesn't enable, marked as
// such (a recording saved to a new mod folder stays in view until MO2
// lists it and it's enabled).
func (s *Service) read(e *env) {
	if e.list == nil {
		return
	}
	e.recs = Read(e.list)
	for _, name := range s.prefs.Get().WroteTo[e.key] {
		if m, ok := e.list.Mod(name); ok && (m.Enabled || m.Layer != modlist.LayerMod) {
			continue // read with the list
		}
		for _, r := range ReadFolder(filepath.Join(e.list.Source.Path, "mods", name), name) {
			if e.recording(r.Name) == nil {
				e.recs = append(e.recs, r)
			}
		}
	}
	sort.Slice(e.recs, func(i, j int) bool { return strings.ToLower(e.recs[i].Name) < strings.ToLower(e.recs[j].Name) })
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
		if err := s.mustEdit(); err != nil {
			return Editor{}, err
		}
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
		if err := s.mustEdit(); err != nil {
			return Editor{}, err
		}
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
		key := strings.ToLower(d.Name)
		s.mu.Lock()
		if s.deleted[key] {
			s.mu.Unlock()
			return Editor{}, apperr.New(apperr.Conflict, "The draft deletes %s. Restore it to change it", d.Name)
		}
		s.drafts[key] = &d
		s.persist()
		s.mu.Unlock()
		ed, err := s.editor(e, name)
		if err == nil && !ed.Changed && ed.Recording != nil {
			// Back to what's on disk: nothing left to save.
			s.mu.Lock()
			delete(s.drafts, key)
			s.persist()
			s.mu.Unlock()
			return s.editor(e, name)
		}
		return ed, err
	}))
	m.Handle("DELETE /api/recordings/{name}/draft", httpapi.JSON(func(r *http.Request, _ httpapi.None) (Editor, error) {
		if err := s.mustEdit(); err != nil {
			return Editor{}, err
		}
		e, err := s.env(r)
		if err != nil {
			return Editor{}, err
		}
		name := r.PathValue("name")
		s.mu.Lock()
		gone := s.dropDrafts([]string{name})
		s.persist()
		s.mu.Unlock()
		s.forgetOpen(gone)
		if e.recording(name) == nil {
			return Editor{}, nil // a new recording, discarded: nothing's left
		}
		return s.editor(e, name)
	}))
	m.Handle("POST /api/recordings/{name}/steps", httpapi.JSON(func(r *http.Request, req AddRequest) (AddReply, error) {
		if err := s.mustEdit(); err != nil {
			return AddReply{}, err
		}
		e, err := s.env(r)
		if err != nil {
			return AddReply{}, err
		}
		return s.add(e, r.PathValue("name"), req.Step)
	}))
	m.Handle("POST /api/recordings/{name}/steps/all", httpapi.JSON(func(r *http.Request, req AddAllRequest) (AddAllReply, error) {
		if err := s.mustEdit(); err != nil {
			return AddAllReply{}, err
		}
		e, err := s.env(r)
		if err != nil {
			return AddAllReply{}, err
		}
		return s.addAll(e, r.PathValue("name"), req.Steps)
	}))
	m.Handle("DELETE /api/recordings/{name}", httpapi.JSON(func(r *http.Request, _ httpapi.None) (Editor, error) {
		if err := s.mustEdit(); err != nil {
			return Editor{}, err
		}
		e, err := s.env(r)
		if err != nil {
			return Editor{}, err
		}
		return s.markDeleted(e, r.PathValue("name"), true)
	}))
	m.Handle("POST /api/recordings/{name}/restore", httpapi.JSON(func(r *http.Request, _ httpapi.None) (Editor, error) {
		if err := s.mustEdit(); err != nil {
			return Editor{}, err
		}
		e, err := s.env(r)
		if err != nil {
			return Editor{}, err
		}
		return s.markDeleted(e, r.PathValue("name"), false)
	}))
	m.Handle("GET /api/changes", httpapi.JSON(func(r *http.Request, _ httpapi.None) (Changes, error) {
		e, err := s.env(r)
		if err != nil {
			return Changes{}, err
		}
		return s.changes(e), nil
	}))
	m.Handle("POST /api/changes/save", httpapi.JSON(func(r *http.Request, _ httpapi.None) (SaveReply, error) {
		if err := s.mustEdit(); err != nil {
			return SaveReply{}, err
		}
		e, err := s.env(r)
		if err != nil {
			return SaveReply{}, err
		}
		return s.saveAll(e)
	}))
	m.Handle("DELETE /api/changes", httpapi.JSON(func(r *http.Request, _ httpapi.None) (Changes, error) {
		e, err := s.env(r)
		if err != nil {
			return Changes{}, err
		}
		s.mu.Lock()
		gone := s.dropDrafts(nil)
		s.deleted = map[string]bool{}
		s.persist()
		s.mu.Unlock()
		s.forgetOpen(gone)
		return s.changes(e), nil
	}))
}

// ChangeKind is what the draft does to a recording.
type ChangeKind string

const (
	ChangeKindNew     ChangeKind = "new"
	ChangeKindChanged ChangeKind = "changed"
	ChangeKindDeleted ChangeKind = "deleted"
)

// Change is one recording the draft changes, as its editor shows it.
type Change struct {
	Kind   ChangeKind `json:"kind"`
	Editor Editor     `json:"editor"`
}

// Changes is the whole draft, for reviewing before Save.
type Changes struct {
	Recordings []Change `json:"recordings"`
	// Unsaved is why Save can't write the draft now ("" when it can):
	// "running", or "blocked" when a recording can't be written (its
	// editor says why).
	Unsaved string `json:"unsaved"`
	Files   int    `json:"files"` // how many files Save writes or removes
}

// SaveReply is what Save did, recording by recording.
type SaveReply struct {
	Saved   []SavedRecording `json:"saved"`
	Changes Changes          `json:"changes"` // what's left: nothing, unless a save failed part way
}

type SavedRecording struct {
	Name   string     `json:"name"`
	Kind   ChangeKind `json:"kind"`
	Dir    string     `json:"dir"`
	Backup string     `json:"backup"` // where the previous files went; "" when there were none
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
		s.mu.Lock()
		sum.Deleted = s.deleted[strings.ToLower(rec.Name)]
		s.mu.Unlock()
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
	s.created[strings.ToLower(name)] = true
	s.persist()
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
	drafted := held != nil
	if held == nil {
		held = s.kept[strings.ToLower(name)]
	}
	var d Draft
	if held != nil {
		d = *held
		d.Steps = append([]Step{}, held.Steps...)
	}
	saved, wasSaved := s.saved[strings.ToLower(name)]
	deleted := s.deleted[strings.ToLower(name)]
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
	ed := Editor{Recording: rec, Drafted: drafted, Deleted: deleted, Problems: []string{}, Targets: s.targets(e, d.Target)}
	files, err := Plan(&d)
	if err != nil {
		ed.Problems = append(ed.Problems, err.Error())
	}
	ed.Draft = d
	ed.Checks = Validate(&Recording{Steps: d.Steps}, e.cat)
	ed.Counts = Count(ed.Checks)
	ed.Target = s.target(e, d)
	ed.Files = []PlannedFile{}
	if deleted {
		// Save removes every file the recording has in its folder.
		disk := Disk(ed.Target.Dir, d.Name)
		var paths []string
		for path := range disk {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			ed.Files = append(ed.Files, PlannedFile{Name: filepath.Base(path), Path: path, Old: disk[path], Change: FileChangeRemoved})
		}
		ed.Changed = len(paths) > 0
	} else if ed.Target.Dir != "" && err == nil {
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
	case deleted:
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
	return sameValue(x, y, "")
}

// exactText is the keys whose text is read as written: what people see
// (a recording's name, author, and version) and what an input option is
// given. Everything else MCM Recorder compares without regard to case.
var exactText = map[string]bool{"name": true, "author": true, "version": true, "text": true}

func sameValue(a, b any, key string) bool {
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
			if w, ok := ly[k]; !ok || !sameValue(v, w, k) {
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
			if !sameValue(x[i], y[i], key) {
				return false
			}
		}
		return true
	case string:
		y, ok := b.(string)
		if exactText[key] {
			return ok && x == y
		}
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
// never a mod list's own. (Save, which does the deleting, waits for Skyrim
// to close.)
func (s *Service) deletes(e *env, r *Recording) string {
	if e.list == nil {
		return "no-list"
	}
	if !slices.ContainsFunc(s.prefs.Get().WroteTo[e.key], func(m string) bool { return strings.EqualFold(m, r.Mod) }) {
		return "not-ours"
	}
	return ""
}

// prepare is a step as adding takes it, or why it can't be written.
func prepare(step Step) (Step, error) {
	step.File = 0
	if step.Kind == KindClick {
		step.Value = step.Option
	}
	step.Wildcard = len(step.Option) > 2 && strings.HasPrefix(step.Option, "*") && strings.HasSuffix(step.Option, "*")
	probe := Draft{Name: "probe", Steps: []Step{step}}
	if _, err := Plan(&probe); err != nil {
		return step, apperr.New(apperr.Invalid, "That step can't be written: %v", err)
	}
	return step, nil
}

// addAll appends the steps the recording doesn't have yet, all or none.
func (s *Service) addAll(e *env, name string, steps []Step) (AddAllReply, error) {
	ed, err := s.editor(e, name)
	if err != nil {
		return AddAllReply{}, err
	}
	if ed.Deleted {
		return AddAllReply{}, apperr.New(apperr.Conflict, "The draft deletes %s. Restore it to add to it", ed.Draft.Name)
	}
	d := ed.Draft
	var reply AddAllReply
	for i, step := range steps {
		step, err := prepare(step)
		if err != nil {
			return AddAllReply{}, fmt.Errorf("setting %d (%s): %w; nothing was recorded", i+1, step.Option, err)
		}
		if Find(d.Steps, step) >= 0 {
			reply.Kept++
			continue
		}
		d.Steps = append(d.Steps, step)
		reply.Added++
	}
	if reply.Added > 0 {
		s.mu.Lock()
		s.drafts[strings.ToLower(d.Name)] = &d
		s.persist()
		s.mu.Unlock()
	}
	return reply, nil
}

func (s *Service) add(e *env, name string, step Step) (AddReply, error) {
	ed, err := s.editor(e, name)
	if err != nil {
		return AddReply{}, err
	}
	if ed.Deleted {
		return AddReply{}, apperr.New(apperr.Conflict, "The draft deletes %s. Restore it to add to it", ed.Draft.Name)
	}
	d := ed.Draft
	step, err = prepare(step)
	if err != nil {
		return AddReply{}, err
	}
	// The option once per recording: adding it again changes its step.
	if i := Find(d.Steps, step); i >= 0 {
		d.Steps[i].Value, d.Steps[i].Side = step.Value, step.Side
		s.mu.Lock()
		s.drafts[strings.ToLower(d.Name)] = &d
		s.persist()
		s.mu.Unlock()
		return AddReply{Number: i + 1, Replaced: true}, nil
	}
	d.Steps = append(d.Steps, step)
	s.mu.Lock()
	s.drafts[strings.ToLower(d.Name)] = &d
	s.persist()
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
	delete(s.created, key) // on disk now
	s.saved[key] = done
	s.persist()
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
		s.read(e)
	}
	if e.list == nil || e.recording(name) == nil {
		// Saved where this list doesn't see it (a new mod folder, disabled,
		// or a folder with a catalog file open): keep showing the draft.
		d := ed.Draft
		s.mu.Lock()
		s.kept[key] = &d
		s.mu.Unlock()
	}
	return s.editor(e, name)
}

// Find is the position of the step in steps that sets the same option as
// step (the same menu, page, selector, index, and kind, compared as MCM
// Recorder compares them), or -1.
func Find(steps []Step, step Step) int {
	for i, s := range steps {
		if strings.EqualFold(s.Mod, step.Mod) && strings.EqualFold(s.Page, step.Page) && strings.EqualFold(s.Option, step.Option) && s.Index == step.Index && s.Kind == step.Kind {
			return i
		}
	}
	return -1
}

// forgetOpen stops new steps going to recordings the draft no longer has:
// new ones that were discarded.
func (s *Service) forgetOpen(names []string) {
	if len(names) == 0 {
		return
	}
	s.prefs.Update(func(v *prefs.Values) {
		if slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(n, v.OpenRecording) }) {
			v.OpenRecording = ""
		}
	})
}

// dropDrafts takes recordings' changes out of the draft (all of them when
// names is nil), and returns the new ones that leaves with nothing.
// Callers hold s.mu.
func (s *Service) dropDrafts(names []string) []string {
	var gone []string
	for k, d := range s.drafts {
		if names != nil && !slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(n, k) }) {
			continue
		}
		if s.created[k] {
			gone = append(gone, d.Name)
			delete(s.created, k)
		}
		delete(s.drafts, k)
	}
	return gone
}

// markDeleted puts a recording's deletion in the draft, or takes it out.
// A new recording isn't deleted: its draft is discarded.
func (s *Service) markDeleted(e *env, name string, on bool) (Editor, error) {
	rec := e.recording(name)
	if rec == nil {
		return Editor{}, apperr.New(apperr.NotFound, "There's no saved recording %q in this list", name)
	}
	if on {
		switch s.deletes(e, rec) {
		case "not-ours":
			return Editor{}, apperr.New(apperr.Conflict, "%s is in %s, which MCM Lexicon didn't save it to, so it isn't deleted here", rec.Name, rec.Mod)
		case "no-list":
			return Editor{}, apperr.New(apperr.Unavailable, "Recordings are deleted from a mod list, and a catalog file is open")
		}
	}
	key := strings.ToLower(rec.Name)
	s.mu.Lock()
	if on {
		s.deleted[key] = true
		delete(s.drafts, key) // deleting replaces any changes to it
	} else {
		delete(s.deleted, key)
	}
	s.persist()
	s.mu.Unlock()
	return s.editor(e, rec.Name)
}

// deleteNow removes a recording's files, after backing them up.
func (s *Service) deleteNow(e *env, name string) (DeleteReply, error) {
	rec := e.recording(name)
	if rec == nil {
		return DeleteReply{}, apperr.New(apperr.NotFound, "There's no recording %q in this list", name)
	}
	if s.deletes(e, rec) == "not-ours" {
		return DeleteReply{}, apperr.New(apperr.Conflict, "%s is in %s, which MCM Lexicon didn't save it to, so it isn't deleted here", rec.Name, rec.Mod)
	}
	t := s.target(e, Draft{Name: rec.Name, Target: rec.Mod})
	backup, err := Delete(t.Dir, rec.Name, e.backups(t), s.now())
	if err != nil {
		return DeleteReply{}, err
	}
	s.mu.Lock()
	delete(s.drafts, strings.ToLower(rec.Name))
	delete(s.deleted, strings.ToLower(rec.Name))
	delete(s.saved, strings.ToLower(rec.Name))
	s.persist()
	s.mu.Unlock()
	e.list.Refresh("McmRecorder")
	s.read(e)
	s.prefs.Update(func(v *prefs.Values) {
		if strings.EqualFold(v.OpenRecording, rec.Name) {
			v.OpenRecording = ""
		}
	})
	return DeleteReply{Backup: backup}, nil
}

// changes is the whole draft, recording by recording, in name order.
func (s *Service) changes(e *env) Changes {
	s.mu.Lock()
	var names []string
	for _, d := range s.drafts {
		names = append(names, d.Name)
	}
	for k := range s.deleted {
		if r := e.recording(k); r != nil {
			names = append(names, r.Name)
		}
	}
	s.mu.Unlock()
	sort.Slice(names, func(i, j int) bool { return strings.ToLower(names[i]) < strings.ToLower(names[j]) })
	out := Changes{Recordings: []Change{}}
	for _, name := range names {
		ed, err := s.editor(e, name)
		if err != nil {
			continue
		}
		kind := ChangeKindChanged
		switch {
		case ed.Deleted:
			kind = ChangeKindDeleted
		case ed.Recording == nil:
			kind = ChangeKindNew
		}
		for _, f := range ed.Files {
			if f.Change != FileChangeSame {
				out.Files++
			}
		}
		if ed.Unsaved != "" && ed.Unsaved != "unchanged" && ed.Unsaved != "running" && out.Unsaved == "" {
			out.Unsaved = "blocked"
		}
		out.Recordings = append(out.Recordings, Change{Kind: kind, Editor: ed})
	}
	if s.running() {
		out.Unsaved = "running"
	}
	return out
}

// saveAll writes the whole draft: every new and changed recording, and
// every deletion, each backed up first. It writes nothing unless all of
// them can be written, and stops at the first failure, saying what it had
// done; what's left stays in the draft.
func (s *Service) saveAll(e *env) (SaveReply, error) {
	ch := s.changes(e)
	switch ch.Unsaved {
	case "running":
		return SaveReply{}, apperr.New(apperr.Conflict, "Skyrim is running, so nothing can be written until you close it")
	case "blocked":
		var names []string
		for _, c := range ch.Recordings {
			if u := c.Editor.Unsaved; u != "" && u != "unchanged" {
				names = append(names, c.Editor.Draft.Name)
			}
		}
		return SaveReply{}, apperr.New(apperr.Invalid, "Nothing was written: %s can't be saved yet. Its page says why", strings.Join(names, ", "))
	}
	reply := SaveReply{Saved: []SavedRecording{}}
	for _, c := range ch.Recordings {
		name := c.Editor.Draft.Name
		var err error
		switch c.Kind {
		case ChangeKindDeleted:
			var d DeleteReply
			if d, err = s.deleteNow(e, name); err == nil {
				reply.Saved = append(reply.Saved, SavedRecording{Name: name, Kind: c.Kind, Dir: c.Editor.Target.Dir, Backup: d.Backup})
				s.prefs.Update(func(v *prefs.Values) {
					if strings.EqualFold(v.OpenRecording, name) {
						v.OpenRecording = ""
					}
				})
			}
		default:
			if !c.Editor.Changed {
				s.mu.Lock()
				delete(s.drafts, strings.ToLower(name))
				s.persist()
				s.mu.Unlock()
				continue
			}
			var ed Editor
			if ed, err = s.save(e, name); err == nil && ed.Saved != nil {
				reply.Saved = append(reply.Saved, SavedRecording{Name: name, Kind: c.Kind, Dir: ed.Saved.Dir, Backup: ed.Saved.Backup})
			}
		}
		if err != nil {
			var done []string
			for _, r := range reply.Saved {
				done = append(done, r.Name)
			}
			if len(done) > 0 {
				return SaveReply{}, fmt.Errorf("saved %s, then %s failed: %w; the rest is still in the draft", strings.Join(done, ", "), name, err)
			}
			return SaveReply{}, fmt.Errorf("%s: %w; nothing was written", name, err)
		}
	}
	reply.Changes = s.changes(e)
	return reply, nil
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
		Counts: Count(checks), Problems: r.Problems, Unlisted: r.Unlisted,
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
