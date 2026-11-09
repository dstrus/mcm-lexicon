// Package capture is MCM Lexicon's side of the capture plugin (PLAN.md
// §11): the plugin's folder for a list's game, the request MCM Lexicon
// writes there, and the newest capture, kept added to the catalog as the
// plugin writes new ones.
package capture

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

// The files in the plugin's folder.
const (
	requestFile = "capture-request.json"
	captureFile = "capture.json"
)

// Tracker is a menu that only tracks the player's progress, with no
// settings a recording would set, and slow to run: a request skips it
// unless trackers are included.
type Tracker struct {
	Script string `json:"script"`
	Name   string `json:"name"` // as SkyUI lists it
}

// Trackers are the menus a request skips by default.
var Trackers = []Tracker{
	{Script: "DBM_DynamicMCMScript", Name: "LOTD Checklist"},
}

// documents is the player's Documents folder, where the game keeps My
// Games; a variable so tests can point it elsewhere.
var documents = documentsFolder

// watchEvery is how often the capture file is looked at.
var watchEvery = 3 * time.Second

// Service finds the capture for the list open, keeps it added to the
// catalog, and writes requests.
type Service struct {
	catalog *catalog.Service
	prefs   *prefs.Prefs
	source  func() (*modlist.Source, string) // the list open, or the catalog file open instead

	mu       sync.Mutex
	path     string    // the capture file added, "" for none
	modTime  time.Time // and when it was written, so a new one is read
	err      string    // why the capture found isn't added
	watching sync.Once
}

func NewService(c *catalog.Service, p *prefs.Prefs, source func() (*modlist.Source, string)) *Service {
	return &Service{catalog: c, prefs: p, source: source}
}

// Watch adds the newest capture now, and again whenever it changes.
func (s *Service) Watch() {
	s.Sync()
	s.watching.Do(func() {
		go func() {
			for range time.Tick(watchEvery) {
				s.Sync()
			}
		}()
	})
}

// game is the list's game as MO2 names it, "" when not known.
func (s *Service) game() string {
	if src, _ := s.source(); src != nil {
		return src.Game
	}
	return ""
}

// Folder is the plugin's folder for the list's game: MCM Lexicon in SKSE's
// log folder, Documents/My Games/<game>/SKSE, as the plugin finds it. It's
// "" when there's no My Games folder for the game, which the game makes
// when it first runs.
func Folder(game string) string {
	docs := documents()
	if docs == "" {
		return ""
	}
	for _, name := range gameFolders(game) {
		if info, err := os.Stat(filepath.Join(docs, "My Games", name)); err == nil && info.IsDir() {
			return filepath.Join(docs, "My Games", name, "SKSE", "MCM Lexicon")
		}
	}
	return ""
}

// gameFolders are the folder names under My Games for a game as MO2 names
// it, likeliest first: VR's, or Special Edition's for each store.
func gameFolders(game string) []string {
	vr := []string{"Skyrim VR"}
	se := []string{"Skyrim Special Edition", "Skyrim Special Edition GOG", "Skyrim Special Edition EPIC", "Skyrim Special Edition MS"}
	switch {
	case strings.Contains(strings.ToLower(game), "vr"):
		return vr
	case game != "":
		return se
	}
	return append(vr, se...)
}

// sameGame reports whether a capture's game (SkyrimVR or SkyrimSE) can be
// the list's, as MO2 names it; any can when the list's isn't known.
func sameGame(captured, game string) bool {
	if game == "" || captured == "" {
		return true
	}
	return strings.Contains(strings.ToLower(game), "vr") == strings.EqualFold(captured, "SkyrimVR")
}

// file is the capture file to add: the one chosen in Settings, else the
// newest in the plugin's folder.
func (s *Service) file() string {
	if f := s.prefs.Get().CaptureFile; f != "" {
		return f
	}
	if dir := Folder(s.game()); dir != "" {
		return filepath.Join(dir, captureFile)
	}
	return ""
}

// Sync adds the capture file to the catalog when it's new or changed, and
// takes it away when there's none or the player turned it off.
func (s *Service) Sync() {
	path := s.file()
	var mod time.Time
	if info, err := os.Stat(path); path != "" && err == nil {
		mod = info.ModTime()
	} else {
		path = ""
	}
	if s.prefs.Get().CaptureOff {
		path = ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if path == s.path && mod.Equal(s.modTime) {
		return
	}
	s.path, s.modTime, s.err = path, mod, ""
	if path == "" {
		s.catalog.AddCapture(nil, "")
		return
	}
	f, err := catalog.ReadCapture(path)
	switch {
	case err != nil:
		s.err = err.Error()
	case !sameGame(f.Game, s.game()):
		s.err = fmt.Sprintf("The capture is from %s, and the list is for %s.", f.Game, s.game())
	}
	if s.err != "" {
		s.catalog.AddCapture(nil, "")
		return
	}
	s.catalog.AddCapture(f, path)
	log.Printf("Capture: %s, from %s", path, f.Captured.Local().Format(time.DateTime))
}

// Request is capture-request.json: the menus to skip, by script.
type Request struct {
	Format int      `json:"format"`
	Skip   []string `json:"skip"`
}

// Status is what Settings shows of the capture.
type Status struct {
	Folder          string    `json:"folder"`             // the plugin's folder for the list's game; "" when the game's My Games folder isn't there
	Requested       time.Time `json:"requested,omitzero"` // when the request waiting for the next load was written; absent for none
	File            string    `json:"file"`               // the capture file chosen in place of the folder's; "" for the folder's
	Use             bool      `json:"use"`                // whether a capture is added at all
	IncludeTrackers bool      `json:"includeTrackers"`
	Trackers        []Tracker `json:"trackers"`
	// Capture is the capture added, as the catalog has it; absent when
	// none is. Error says why one found isn't added.
	Capture *catalog.CaptureInfo `json:"capture,omitempty"`
	Error   string               `json:"error"`
}

func (s *Service) Status() Status {
	s.Sync()
	v := s.prefs.Get()
	st := Status{Folder: Folder(s.game()), File: v.CaptureFile, Use: !v.CaptureOff, IncludeTrackers: v.IncludeTrackers, Trackers: Trackers, Capture: s.catalog.CaptureInfo()}
	if st.Folder != "" {
		if info, err := os.Stat(filepath.Join(st.Folder, requestFile)); err == nil {
			st.Requested = info.ModTime()
		}
	}
	s.mu.Lock()
	st.Error = s.err
	s.mu.Unlock()
	return st
}

// request writes a request to the plugin's folder, for the next load of a
// save. It's written while the game runs too: it's the plugin's, not the
// list's.
func (s *Service) request() error {
	dir := Folder(s.game())
	if dir == "" {
		return apperr.New(apperr.Unavailable, "There's no My Games folder for %s here; it appears once the game has run", gameName(s.game()))
	}
	req := Request{Format: catalog.CaptureFormat, Skip: []string{}}
	if !s.prefs.Get().IncludeTrackers {
		for _, t := range Trackers {
			req.Skip = append(req.Skip, t.Script)
		}
	}
	b, err := json.MarshalIndent(req, "", "\t")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return textfile.WriteAtomic(filepath.Join(dir, requestFile), append(b, '\n'), 0o644)
}

func gameName(game string) string {
	if game == "" {
		return "Skyrim"
	}
	return game
}

// SettingsRequest changes the capture's settings: whether one is added,
// whether requests include trackers, and a capture file to add in place
// of the folder's ("" for the folder's).
type SettingsRequest struct {
	Use             bool   `json:"use"`
	IncludeTrackers bool   `json:"includeTrackers"`
	File            string `json:"file"`
}

func (s *Service) Routes(m *http.ServeMux) {
	m.Handle("GET /api/capture", httpapi.JSON(func(*http.Request, httpapi.None) (Status, error) {
		return s.Status(), nil
	}))
	m.Handle("POST /api/capture/request", httpapi.JSON(func(*http.Request, httpapi.None) (Status, error) {
		if err := s.request(); err != nil {
			return Status{}, err
		}
		return s.Status(), nil
	}))
	m.Handle("DELETE /api/capture/request", httpapi.JSON(func(*http.Request, httpapi.None) (Status, error) {
		if dir := Folder(s.game()); dir != "" {
			if err := os.Remove(filepath.Join(dir, requestFile)); err != nil && !os.IsNotExist(err) {
				return Status{}, err
			}
		}
		return s.Status(), nil
	}))
	m.Handle("PUT /api/capture/settings", httpapi.JSON(func(_ *http.Request, req SettingsRequest) (Status, error) {
		file := strings.TrimSpace(req.File)
		if file != "" {
			if _, err := catalog.ReadCapture(file); err != nil {
				return Status{}, apperr.New(apperr.Invalid, "%v", err)
			}
		}
		s.prefs.Update(func(v *prefs.Values) {
			v.CaptureOff, v.IncludeTrackers, v.CaptureFile = !req.Use, req.IncludeTrackers, file
		})
		return s.Status(), nil
	}))
}
