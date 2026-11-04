package recordings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
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

// Summary is a recording without its steps, for the list of recordings.
type Summary struct {
	Name      string    `json:"name"`
	Title     string    `json:"title"`
	Version   string    `json:"version"`
	Author    string    `json:"author"`
	Autorun   bool      `json:"autorun"`
	Mod       string    `json:"mod"`
	Info      string    `json:"info"`
	Losing    []string  `json:"losing"`
	Steps     int       `json:"steps"`
	Files     int       `json:"files"`
	Menus     int       `json:"menus"` // the menus its steps name
	Counts    Counts    `json:"counts"`
	Problems  []Problem `json:"problems"`
	Drafted   bool      `json:"drafted"`   // it has changes not yet saved
	Deleted   bool      `json:"deleted"`   // the draft deletes it
	New       bool      `json:"new"`       // it's only a draft, never saved
	Deletes   string    `json:"deletes"`   // why it can't be deleted; "" when it can
	Unlisted  bool      `json:"unlisted"`  // in a folder MCM Lexicon saved to that the profile doesn't enable
	Restoring bool      `json:"restoring"` // the draft puts it back from a backup
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
	// Covers is the other mod whose step file of the same name this empty
	// one hides, so the game doesn't play it; "" for the recording's own.
	Covers string `json:"covers,omitempty"`
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
	// FromBackup is the backup the draft puts back: Save writes its files
	// exactly, after backing up what's there.
	FromBackup *Backup `json:"fromBackup,omitempty"`
}

// CreateRequest makes a new recording, as a draft.
type CreateRequest struct {
	Name    string `json:"name"`
	Author  string `json:"author"`
	Autorun bool   `json:"autorun"`
}

// FileTextRequest is one of a recording's files, typed.
type FileTextRequest struct {
	Text string `json:"text"`
}

// AddRequest adds a step to a recording's draft.
type AddRequest struct {
	Step Step `json:"step"`
	// Clicks, for a click step on a stepper, is how many times the
	// recording clicks it: that many identical steps, in one run where the
	// first was, or appended; 0 takes them out. Nil adds or changes one
	// step, as for any other option.
	Clicks *int `json:"clicks,omitempty"`
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

// AddFromRequest adds every step of another recording, in order, to pull
// recordings together: as AddAllRequest, the options this one has already
// keep their steps.
type AddFromRequest struct {
	From string `json:"from"` // the other recording's name
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
	// fromBackups are the recordings the draft puts back from a backup, by
	// lowercase name: Save writes the backup's files exactly.
	fromBackups map[string]fromBackup

	// The draft's history, for undo and redo: the state it was in before
	// each change, and the changes undone. Kept in memory only; a Save,
	// which writes, starts it afresh.
	last       snapshot
	undo, redo []snapshot
}

// snapshot is the draft at one moment: what undo puts back.
type snapshot struct {
	drafts      map[string]Draft
	deleted     map[string]bool
	created     map[string]bool
	fromBackups map[string]fromBackup
}

// fromBackup is a recording the draft puts back from one of its backups.
type fromBackup struct {
	Name string `json:"name"` // the recording, as it's spelled
	ID   string `json:"id"`   // the backup
	Mod  string `json:"mod"`  // the folder it goes back into
}

// pending is how many recordings the draft changes. Callers hold s.mu.
func (s *Service) pending() int {
	return len(s.drafts) + len(s.deleted) + len(s.fromBackups)
}

// historyLimit is the most changes undo goes back through.
const historyLimit = 200

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
	s.fromBackups = map[string]fromBackup{}
	s.forget()
}

// snap is the draft now, copied.
func (s *Service) snap() snapshot {
	out := snapshot{drafts: map[string]Draft{}, deleted: maps.Clone(s.deleted), created: maps.Clone(s.created), fromBackups: maps.Clone(s.fromBackups)}
	for k, d := range s.drafts {
		c := *d
		c.Steps = slices.Clone(d.Steps)
		out.drafts[k] = c
	}
	return out
}

// put makes the draft what n holds.
func (s *Service) put(n snapshot) {
	s.drafts = map[string]*Draft{}
	for k, d := range n.drafts {
		c := d
		c.Steps = slices.Clone(d.Steps)
		s.drafts[k] = &c
	}
	s.deleted, s.created = maps.Clone(n.deleted), maps.Clone(n.created)
	if s.deleted == nil {
		s.deleted = map[string]bool{}
	}
	if s.created == nil {
		s.created = map[string]bool{}
	}
	s.fromBackups = maps.Clone(n.fromBackups)
	if s.fromBackups == nil {
		s.fromBackups = map[string]fromBackup{}
	}
}

func (a snapshot) same(b snapshot) bool {
	x, _ := json.Marshal([]any{a.drafts, a.deleted, a.created, a.fromBackups})
	y, _ := json.Marshal([]any{b.drafts, b.deleted, b.created, b.fromBackups})
	return string(x) == string(y)
}

// remember puts the state before a change on the undo history, when the
// change changed anything, and drops what was undone. Callers hold s.mu.
func (s *Service) remember() {
	now := s.snap()
	if now.same(s.last) {
		return
	}
	s.undo = append(s.undo, s.last)
	if len(s.undo) > historyLimit {
		s.undo = s.undo[len(s.undo)-historyLimit:]
	}
	s.redo = nil
	s.last = now
}

// amend folds a further change into the last one, as one change for undo:
// what undo puts back is still the state before both. A change that ends
// where it began leaves no history. Callers hold s.mu.
func (s *Service) amend() {
	s.last = s.snap()
	if n := len(s.undo); n > 0 && s.undo[n-1].same(s.last) {
		s.undo = s.undo[:n-1]
	}
	s.write()
}

// forget starts the history afresh from the draft as it is: after a
// write, which can't be undone, or for another list. Callers hold s.mu.
func (s *Service) forget() {
	s.undo, s.redo, s.last = nil, nil, s.snap()
}

// History is how many changes undo and redo can step through.
func (s *Service) History() (undo, redo int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.undo), len(s.redo)
}

// step undoes the last change (back) or redoes the last undone one.
func (s *Service) step(back bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	from, to := &s.undo, &s.redo
	if !back {
		from, to = to, from
	}
	if len(*from) == 0 {
		if back {
			return apperr.New(apperr.Conflict, "There's nothing to undo")
		}
		return apperr.New(apperr.Conflict, "There's nothing to redo")
	}
	n := (*from)[len(*from)-1]
	*from = (*from)[:len(*from)-1]
	*to = append(*to, s.snap())
	s.put(n)
	s.last = n
	s.write()
	return nil
}

// held is the draft as its file keeps it: what it changes, and the list
// it belongs to.
type held struct {
	From        string                `json:"from"`
	Drafts      map[string]*Draft     `json:"drafts"`
	Deleted     map[string]bool       `json:"deleted,omitempty"`
	Created     map[string]bool       `json:"created,omitempty"`
	FromBackups map[string]fromBackup `json:"fromBackups,omitempty"`
}

// persist records a change to the draft: on the undo history, and in its
// file. Callers hold s.mu.
func (s *Service) persist() {
	// Changing a recording, or deleting it, replaces putting it back from a
	// backup.
	for k := range s.fromBackups {
		if s.drafts[k] != nil || s.deleted[k] {
			delete(s.fromBackups, k)
		}
	}
	s.remember()
	s.write()
}

// write writes the draft to its file, or removes the file when the draft
// is empty. Failing to write is logged, not returned: the draft is still
// held until the program stops. Callers hold s.mu.
func (s *Service) write() {
	if s.file == "" {
		return
	}
	if s.pending() == 0 {
		if err := os.Remove(s.file); err != nil && !os.IsNotExist(err) {
			log.Printf("Could not remove the draft file: %v", err)
		}
		return
	}
	b, _ := json.MarshalIndent(held{From: s.from, Drafts: s.drafts, Deleted: s.deleted, Created: s.created, FromBackups: s.fromBackups}, "", "  ")
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
	if len(h.Drafts)+len(h.Deleted)+len(h.FromBackups) == 0 {
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
	for k, b := range h.FromBackups {
		if b.ID != "" {
			s.fromBackups[strings.ToLower(k)] = b
		}
	}
	s.editing = true
	s.forget()
	if n := s.pending(); n == 1 {
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
	return s.pending()
}

// SetEditing turns editing on or off. Turning it off with changes pending
// is refused unless discard says to drop them.
func (s *Service) SetEditing(on, discard bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !on && s.pending() > 0 {
		if !discard {
			return apperr.New(apperr.Conflict, "The draft has changes to %d recordings. Save them or discard them first", s.pending())
		}
		gone := s.dropDrafts(nil)
		s.deleted = map[string]bool{}
		s.fromBackups = map[string]fromBackup{}
		s.persist()
		s.forget() // leaving editing: nothing to undo while read-only
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
		return s.replaceDraft(e, r.PathValue("name"), d)
	}))
	// The text of one file as it will be written, typed: read as MCM
	// Recorder reads it, and put in the draft in its place.
	m.Handle("PUT /api/recordings/{name}/files/{file}", httpapi.JSON(func(r *http.Request, req FileTextRequest) (Editor, error) {
		if err := s.mustEdit(); err != nil {
			return Editor{}, err
		}
		e, err := s.env(r)
		if err != nil {
			return Editor{}, err
		}
		file, err := strconv.Atoi(r.PathValue("file"))
		if err != nil || file < 0 {
			return Editor{}, apperr.New(apperr.NotFound, "There's no file %q", r.PathValue("file"))
		}
		return s.putText(e, r.PathValue("name"), file, req.Text)
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
		if req.Clicks != nil {
			return s.setClicks(e, r.PathValue("name"), req.Step, *req.Clicks)
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
	m.Handle("POST /api/recordings/{name}/steps/from", httpapi.JSON(func(r *http.Request, req AddFromRequest) (AddAllReply, error) {
		if err := s.mustEdit(); err != nil {
			return AddAllReply{}, err
		}
		e, err := s.env(r)
		if err != nil {
			return AddAllReply{}, err
		}
		return s.addFrom(e, r.PathValue("name"), req.From)
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
	// A recording's backups, and putting one back.
	m.Handle("GET /api/recordings/{name}/backups", httpapi.JSON(func(r *http.Request, _ httpapi.None) (BackupsReply, error) {
		e, err := s.env(r)
		if err != nil {
			return BackupsReply{}, err
		}
		return s.backupsOf(e, r.PathValue("name")), nil
	}))
	m.Handle("PUT /api/recordings/{name}/backups/{id}", httpapi.JSON(func(r *http.Request, _ httpapi.None) (Editor, error) {
		if err := s.mustEdit(); err != nil {
			return Editor{}, err
		}
		e, err := s.env(r)
		if err != nil {
			return Editor{}, err
		}
		return s.restoreFrom(e, r.PathValue("name"), r.PathValue("id"))
	}))
	m.Handle("GET /api/backups", httpapi.JSON(func(r *http.Request, _ httpapi.None) (DeletedReply, error) {
		e, err := s.env(r)
		if err != nil {
			return DeletedReply{}, err
		}
		return s.deletedWithBackups(e), nil
	}))
	// Undo and redo step through the draft's history of changes.
	for path, back := range map[string]bool{"POST /api/changes/undo": true, "POST /api/changes/redo": false} {
		m.Handle(path, httpapi.JSON(func(r *http.Request, _ httpapi.None) (Changes, error) {
			if err := s.mustEdit(); err != nil {
				return Changes{}, err
			}
			e, err := s.env(r)
			if err != nil {
				return Changes{}, err
			}
			if err := s.step(back); err != nil {
				return Changes{}, err
			}
			return s.changes(e), nil
		}))
	}
	m.Handle("DELETE /api/changes", httpapi.JSON(func(r *http.Request, _ httpapi.None) (Changes, error) {
		e, err := s.env(r)
		if err != nil {
			return Changes{}, err
		}
		s.mu.Lock()
		gone := s.dropDrafts(nil)
		s.deleted = map[string]bool{}
		s.fromBackups = map[string]fromBackup{}
		s.persist()
		s.mu.Unlock()
		s.forgetOpen(gone)
		return s.changes(e), nil
	}))
}

// ChangeKind is what the draft does to a recording.
type ChangeKind string

const (
	ChangeKindNew      ChangeKind = "new"
	ChangeKindChanged  ChangeKind = "changed"
	ChangeKindDeleted  ChangeKind = "deleted"
	ChangeKindRestored ChangeKind = "restored" // put back from a backup
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
	restoring := maps.Clone(s.fromBackups)
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
		if _, ok := restoring[strings.ToLower(rec.Name)]; ok && !sum.Drafted && !sum.Deleted {
			sum.Drafted, sum.Restoring = true, true
		}
		delete(restoring, strings.ToLower(rec.Name))
		sum.Deletes = s.deletes(e, rec)
		reply.Recordings = append(reply.Recordings, sum)
	}
	// A deleted recording the draft puts back from a backup.
	for k, fb := range restoring {
		if drafts[k] != nil {
			continue
		}
		ed, err := s.editor(e, fb.Name)
		if err != nil {
			continue
		}
		rec := &Recording{Name: ed.Draft.Name, Title: ed.Draft.Title, Author: ed.Draft.Author, Version: ed.Draft.Version, Autorun: ed.Draft.Autorun, Steps: ed.Draft.Steps, Losing: []string{}, Problems: []Problem{}, Files: []File{}}
		sum := summarize(rec, e.cat)
		sum.Drafted, sum.Restoring, sum.Mod, sum.Deletes = true, true, ed.Draft.Target, "new"
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

// defaultTarget is where a new recording saves: the save folder the
// settings name, or with a catalog file open, the folder chosen last time.
func (s *Service) defaultTarget(e *env) string {
	switch {
	case e.list != nil && e.list.Source.Kind == modlist.Data:
		return ""
	case e.list != nil:
		return s.prefs.Get().SaveMod()
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
			return s.prefs.Get().SaveMod(), nil
		}
		if !ModName(t) {
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
		// Another copy of the info file that the game sees instead. Not for
		// a new mod folder: MO2 adds it at the bottom of the list, where it
		// loads last and wins.
		if t.Kind == TargetKindNewMod {
			t.Exists = isDir(t.Dir)
			return t
		}
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
	fb, restoring := s.fromBackups[strings.ToLower(name)]
	s.mu.Unlock()
	if restoring && !drafted && !deleted {
		return s.editorFromBackup(e, rec, fb, saved, wasSaved)
	}
	if rec == nil && held == nil {
		if e.list == nil {
			return Editor{}, apperr.New(apperr.NotFound, "There's no new recording %q. With a catalog file open, recordings are made here and saved to a folder", name)
		}
		return Editor{}, apperr.New(apperr.NotFound, "There's no recording %q in this list", name)
	}
	// Untouched, a recording has nothing to save, though the folder it would
	// save to (the save folder, for a recording from another mod) may not
	// hold it yet.
	untouched := held == nil
	if untouched {
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
	cover := map[string]string{}
	if err == nil && !deleted {
		cover = s.covers(e, d, ed.Target, files)
		files = Cover(files, d.Name, cover)
	}
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
			p := PlannedFile{Name: f.Name, Path: f.Path, Text: f.Text, Change: FileChangeNew, Covers: cover[f.Name]}
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
			ed.Files = append(ed.Files, PlannedFile{Name: f.Name, Path: f.Path, Text: f.Text, Change: FileChangeNew, Covers: cover[f.Name]})
		}
		ed.Changed = true
	}
	if untouched && !deleted {
		ed.Changed = false
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
// the copy of its info file the game reads, when MCM Lexicon saved it
// there, else the save folder, so recordings from other mods are left as
// they came and the copy in the save folder, loaded after, takes their
// place.
func (s *Service) homeOf(e *env, r *Recording) string {
	switch {
	case e.list != nil && e.list.Source.Kind == modlist.Data:
		return ""
	case e.list != nil && r.Mod != "" && s.wrote(e, r.Mod):
		return r.Mod
	case e.list != nil:
		return s.defaultTarget(e)
	case r.Mod != "":
		return r.Mod
	default:
		return s.defaultTarget(e)
	}
}

// wrote reports whether MCM Lexicon has saved recordings into mod.
func (s *Service) wrote(e *env, mod string) bool {
	return slices.ContainsFunc(s.prefs.Get().WroteTo[e.key], func(m string) bool { return strings.EqualFold(m, mod) })
}

// ModName reports whether name can be a mod folder's: no character
// Windows refuses in a file name, and more than dots and spaces.
func ModName(name string) bool {
	return !strings.ContainsAny(name, `/\:*?"<>|`) && strings.Trim(name, ". ") != ""
}

// covers is the step files other mods put in the recording's folder that
// Save doesn't replace by name, each with the mod it's from: the game sees
// one folder made of every mod's, so it would play them beside the
// recording's own. A mod's files can't be removed from under it, so Save
// writes an empty file of the same name over each, which plays nothing.
// That holds while the target loads after them (Target.LosesTo says when
// it doesn't).
func (s *Service) covers(e *env, d Draft, t Target, files []OutFile) map[string]string {
	if e.list == nil || e.list.Source.Kind == modlist.Data || t.Name == "" {
		return nil
	}
	planned := map[string]bool{}
	for _, f := range files {
		planned[strings.ToLower(f.Name)] = true
	}
	out := map[string]string{}
	for _, f := range e.list.Files("McmRecorder/" + PathPart(d.Name)) {
		name := path.Base(f.Path)
		if strings.Count(f.Path, "/") != 2 || !strings.EqualFold(path.Ext(name), ".json") || strings.HasPrefix(name, ".") || planned[strings.ToLower(name)] {
			continue
		}
		for _, c := range e.list.Copies(f.Path) {
			if !strings.EqualFold(c.Mod, t.Name) {
				out[name] = c.Mod
				break
			}
		}
	}
	return out
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
	addIf(s.prefs.Get().SaveMod())
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
	if !s.wrote(e, r.Mod) {
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

// replaceDraft puts d in the draft as name's recording, dropping it if it
// matches what's on disk.
func (s *Service) replaceDraft(e *env, name string, d Draft) (Editor, error) {
	if _, err := s.editor(e, name); err != nil {
		return Editor{}, err
	}
	if d.Steps == nil {
		d.Steps = []Step{}
	}
	var err error
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
	if err == nil && s.settled(e, ed) {
		// Back to what's on disk: nothing left to save. Part of the same
		// change, as undo sees it.
		s.mu.Lock()
		delete(s.drafts, key)
		s.amend()
		s.mu.Unlock()
		return s.editor(e, name)
	}
	return ed, err
}

// putText replaces one of a recording's files with text typed for it: file
// 0 is the info file, and file n the nth step file Save would write, whose
// run of steps the text's steps take the place of.
func (s *Service) putText(e *env, name string, file int, text string) (Editor, error) {
	ed, err := s.editor(e, name)
	if err != nil {
		return Editor{}, err
	}
	if ed.Deleted {
		return Editor{}, apperr.New(apperr.Conflict, "The draft deletes %s. Restore it to change it", ed.Draft.Name)
	}
	d := ed.Draft
	d.Steps = append([]Step{}, d.Steps...)
	if file == 0 {
		info, err := ParseInfo(text)
		if err != nil {
			return Editor{}, err
		}
		d.Title, d.Version, d.Author, d.Autorun = info.Title, info.Version, info.Author, info.Autorun
		return s.replaceDraft(e, name, d)
	}
	steps, err := ParseSteps(text)
	if err != nil {
		return Editor{}, err
	}
	first, last := -1, -1
	for i, st := range d.Steps {
		if st.File == file-1 {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		return Editor{}, apperr.New(apperr.NotFound, "%s has no step file %d", ed.Draft.Name, file)
	}
	rest := append([]Step{}, d.Steps[last+1:]...)
	d.Steps = append(append(d.Steps[:first], steps...), rest...)
	return s.replaceDraft(e, name, d)
}

// maxClicks is the most clicks one step can stand for: more than any
// stepper's values, and few enough that a slip can't write thousands.
const maxClicks = 64

// setClicks makes the recording click a stepper n times: its steps for the
// option become n identical clicks, in one run where the first of them
// was, or at the end; 0 takes them out. MCM Recorder plays every action in
// a file in turn, so each step is one click.
func (s *Service) setClicks(e *env, name string, step Step, n int) (AddReply, error) {
	if step.Kind != KindClick {
		return AddReply{}, apperr.New(apperr.Invalid, "Only a click step is recorded as a number of clicks")
	}
	if n < 0 || n > maxClicks {
		return AddReply{}, apperr.New(apperr.Invalid, "A stepper is clicked from 0 to %d times, not %d", maxClicks, n)
	}
	ed, err := s.editor(e, name)
	if err != nil {
		return AddReply{}, err
	}
	if ed.Deleted {
		return AddReply{}, apperr.New(apperr.Conflict, "The draft deletes %s. Restore it to add to it", ed.Draft.Name)
	}
	step, err = prepare(step)
	if err != nil {
		return AddReply{}, err
	}
	d := ed.Draft
	at := -1
	var kept []Step
	for _, st := range d.Steps {
		if Find([]Step{st}, step) == 0 {
			if at < 0 {
				at = len(kept)
				// The run keeps how long its first click looked.
				step.Timeout, step.Interval, step.PageLoad = st.Timeout, st.Interval, st.PageLoad
			}
			continue
		}
		kept = append(kept, st)
	}
	if at < 0 {
		at = len(kept)
	}
	run := make([]Step, n)
	for i := range run {
		run[i] = step
	}
	d.Steps = append(append(append([]Step{}, kept[:at]...), run...), kept[at:]...)
	if _, err := s.replaceDraft(e, name, d); err != nil {
		return AddReply{}, err
	}
	return AddReply{Number: at + 1, Replaced: n == 0}, nil
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
	reply, err := merge(&d, steps)
	if err != nil {
		return AddAllReply{}, err
	}
	if reply.Added > 0 {
		s.mu.Lock()
		s.drafts[strings.ToLower(d.Name)] = &d
		s.persist()
		s.mu.Unlock()
	}
	return reply, nil
}

// addFrom adds every step of the recording from to name's draft, as
// addAll does: the steps from's draft holds, when it has one.
func (s *Service) addFrom(e *env, name, from string) (AddAllReply, error) {
	if strings.EqualFold(name, from) {
		return AddAllReply{}, apperr.New(apperr.Invalid, "A recording can't add its own steps to itself")
	}
	src, err := s.editor(e, from)
	if err != nil {
		return AddAllReply{}, err
	}
	if src.Deleted {
		return AddAllReply{}, apperr.New(apperr.Conflict, "The draft deletes %s. Restore it to add its steps", src.Draft.Name)
	}
	return s.addAll(e, name, src.Draft.Steps)
}

// merge appends to d the steps for options it doesn't have, in order. An
// option it has keeps its step; a run of identical clicks (a stepper
// clicked that many times) for one it doesn't is appended whole.
func merge(d *Draft, steps []Step) (AddAllReply, error) {
	var reply AddAllReply
	added := map[string]bool{}
	for i, step := range steps {
		step, err := prepare(step)
		if err != nil {
			return AddAllReply{}, fmt.Errorf("setting %d (%s): %w; nothing was recorded", i+1, step.Option, err)
		}
		k := strings.ToLower(step.Mod+"\x00"+step.Page+"\x00"+step.Option+"\x00"+string(step.Kind)) + fmt.Sprint(step.Index)
		if !added[k] && Find(d.Steps, step) >= 0 {
			reply.Kept++
			continue
		}
		added[k] = true
		d.Steps = append(d.Steps, step)
		reply.Added++
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
		s.settle(e, name)
		return AddReply{Number: i + 1, Replaced: true}, nil
	}
	d.Steps = append(d.Steps, step)
	s.mu.Lock()
	s.drafts[strings.ToLower(d.Name)] = &d
	s.persist()
	s.mu.Unlock()
	return AddReply{Number: len(d.Steps)}, nil
}

// settle drops a recording's draft when it's back to what's on disk, as
// part of the change that brought it there.
func (s *Service) settle(e *env, name string) {
	ed, err := s.editor(e, name)
	if err != nil || !s.settled(e, ed) {
		return
	}
	s.mu.Lock()
	delete(s.drafts, strings.ToLower(ed.Draft.Name))
	s.amend()
	s.mu.Unlock()
}

// settled reports whether a saved recording's draft has nothing left to
// save: a deletion or a backup that changes no file, or changes that say
// only what the recording does.
func (s *Service) settled(e *env, ed Editor) bool {
	switch {
	case ed.Recording == nil:
		return false
	case ed.Deleted || ed.FromBackup != nil:
		return !ed.Changed
	}
	return s.unchanged(e, ed)
}

// unchanged reports whether a draft of a saved recording says only what
// the recording does, and saves where an untouched one would: the files
// Save would write are the ones it writes from the recording as read.
func (s *Service) unchanged(e *env, ed Editor) bool {
	d, was := ed.Draft, DraftOf(ed.Recording, s.homeOf(e, ed.Recording))
	if !strings.EqualFold(d.Target, was.Target) {
		return false
	}
	a, err := Plan(&d)
	if err != nil {
		return false
	}
	b, err := Plan(&was)
	if err != nil || len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Path != b[i].Path || (a[i].Text != b[i].Text && !sameJSON(a[i].Text, b[i].Text)) {
			return false
		}
	}
	return true
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
	var done Saved
	if ed.FromBackup != nil {
		// The backup's files, exactly, in place of what's there.
		files := map[string]string{}
		for _, f := range ed.Files {
			if f.Change != FileChangeRemoved {
				files[f.Path] = f.Text
			}
		}
		done, err = Put(ed.Target.Dir, ed.Draft.Name, files, e.backups(ed.Target), s.now())
	} else {
		cover := map[string]string{}
		for _, f := range ed.Files {
			if f.Covers != "" {
				cover[f.Name] = f.Covers
			}
		}
		done, err = Save(ed.Target.Dir, ed.Draft, cover, e.backups(ed.Target), s.now())
	}
	if err != nil {
		return Editor{}, err
	}
	key := strings.ToLower(ed.Draft.Name)
	s.mu.Lock()
	delete(s.drafts, key)
	delete(s.fromBackups, key)
	delete(s.created, key) // on disk now
	s.saved[key] = done
	s.persist()
	s.forget() // a write can't be undone
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
	for k := range s.fromBackups {
		if names == nil || slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(n, k) }) {
			delete(s.fromBackups, k)
		}
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
	s.forget() // a write can't be undone
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
	for k, b := range s.fromBackups {
		if s.drafts[k] == nil && !s.deleted[k] {
			names = append(names, b.Name)
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
		case ed.FromBackup != nil:
			kind = ChangeKindRestored
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
				delete(s.fromBackups, strings.ToLower(name))
				s.persist()
				s.forget() // a write can't be undone
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
