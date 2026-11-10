package capture

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lexicon/internal/catalog"
	"lexicon/internal/modlist"
	"lexicon/internal/prefs"
)

func captureOf(game string) string {
	return `{"format": 1, "game": "` + game + `", "captured": "2026-11-09T18:11:43Z", "menus": [
		{"name": "Test", "script": "Test_MCM", "quest": "test.esp:d62", "pages": [{"name": "", "options": [
			{"position": 0, "type": "toggle", "text": "Enable", "str": "None", "num": 1}
		]}]}
	]}`
}

// fixture is a Service for a VR list, with Documents in a temporary folder
// and a catalog of one menu open.
func fixture(t *testing.T) (*Service, string, *prefs.Prefs) {
	t.Helper()
	docs := t.TempDir()
	was := documents
	documents = func() string { return docs }
	t.Cleanup(func() { documents = was })

	cat := filepath.Join(t.TempDir(), "catalog.json")
	err := catalog.WriteFile(cat, &catalog.Catalog{SchemaVersion: catalog.SchemaVersion, Menus: []catalog.Menu{{
		ID: "papyrus-test", Name: "Test", NameText: "Test", Script: "Test_MCM",
		Pages: []catalog.Page{{Options: []catalog.Option{{Type: catalog.OptionTypeToggle, Label: "Enable", Certainty: catalog.CertaintyExact}}}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	c := catalog.NewService("test")
	if err := c.UseFile(cat); err != nil {
		t.Fatal(err)
	}
	p := prefs.Load(filepath.Join(t.TempDir(), "settings.json"))
	src := &modlist.Source{Game: "Skyrim VR"}
	return NewService(c, p, func() (*modlist.Source, string) { return src, "" }), docs, p
}

func TestFolder(t *testing.T) {
	_, docs, _ := fixture(t)
	if f := Folder("Skyrim VR"); f != "" {
		t.Errorf("a folder before the game made My Games: %q", f)
	}
	os.MkdirAll(filepath.Join(docs, "My Games", "Skyrim Special Edition GOG"), 0o755)
	if f := Folder("Skyrim VR"); f != "" {
		t.Errorf("VR found Special Edition's folder: %q", f)
	}
	want := filepath.Join(docs, "My Games", "Skyrim Special Edition GOG", "SKSE", "MCM Lexicon")
	if f := Folder("Skyrim Special Edition"); f != want {
		t.Errorf("GOG's folder: %q", f)
	}
	if f := Folder(""); f != want {
		t.Errorf("a game not known finds any: %q", f)
	}
}

func TestRequest(t *testing.T) {
	s, docs, p := fixture(t)
	if err := s.request(); err == nil {
		t.Error("wrote a request with no My Games folder")
	}
	dir := filepath.Join(docs, "My Games", "Skyrim VR", "SKSE", "MCM Lexicon")
	os.MkdirAll(filepath.Dir(filepath.Dir(dir)), 0o755)
	read := func() Request {
		t.Helper()
		if err := s.request(); err != nil {
			t.Fatal(err)
		}
		var r Request
		b, _ := os.ReadFile(filepath.Join(dir, requestFile))
		if err := json.Unmarshal(b, &r); err != nil {
			t.Fatal(err)
		}
		return r
	}
	if r := read(); r.Format != 1 || len(r.Skip) != 1 || r.Skip[0] != "DBM_DynamicMCMScript" {
		t.Errorf("request: %+v", r)
	}
	if st := s.Status(); st.Requested.IsZero() || st.Folder != dir {
		t.Errorf("status after a request: %+v", st)
	}
	p.Update(func(v *prefs.Values) { v.IncludeTrackers = true })
	if r := read(); len(r.Skip) != 0 {
		t.Errorf("with trackers included: %+v", r)
	}
}

func TestSync(t *testing.T) {
	s, docs, p := fixture(t)
	dir := filepath.Join(docs, "My Games", "Skyrim VR", "SKSE", "MCM Lexicon")
	os.MkdirAll(dir, 0o755)
	if st := s.Status(); st.Capture != nil || st.Error != "" || !st.Use {
		t.Errorf("before a capture: %+v", st)
	}
	os.WriteFile(filepath.Join(dir, captureFile), []byte(captureOf("SkyrimVR")), 0o644)
	st := s.Status()
	if st.Capture == nil || st.Capture.Report.MenusMatched != 1 || st.Capture.Report.Values != 1 {
		t.Fatalf("the folder's capture wasn't added: %+v", st)
	}

	p.Update(func(v *prefs.Values) { v.CaptureOff = true })
	if st := s.Status(); st.Capture != nil || st.Use {
		t.Errorf("turned off: %+v", st)
	}
	p.Update(func(v *prefs.Values) { v.CaptureOff = false })

	// A capture from the other edition isn't this list's.
	other := filepath.Join(t.TempDir(), "se.json")
	os.WriteFile(other, []byte(captureOf("SkyrimSE")), 0o644)
	p.Update(func(v *prefs.Values) { v.CaptureFile = other })
	if st := s.Status(); st.Capture != nil || !strings.Contains(st.Error, "SkyrimSE") || st.File != other {
		t.Errorf("another edition's capture: %+v", st)
	}
}

func TestSettingsRoute(t *testing.T) {
	s, _, p := fixture(t)
	m := httptestMux(s)
	bad := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(bad, []byte(`{"format": 9}`), 0o644)
	if code := put(m, `{"use": true, "file": `+jsonString(bad)+`}`); code < 400 {
		t.Errorf("chose a file that isn't a capture: %d", code)
	}
	if code := put(m, `{"use": false, "includeTrackers": true}`); code != 200 {
		t.Errorf("settings: %d", code)
	}
	if v := p.Get(); !v.CaptureOff || !v.IncludeTrackers || v.CaptureFile != "" {
		t.Errorf("settings kept: %+v", v)
	}
}

func httptestMux(s *Service) *http.ServeMux {
	m := http.NewServeMux()
	s.Routes(m)
	return m
}

func put(m http.Handler, body string) int {
	r := httptest.NewRequest("PUT", "/api/capture/settings", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	return w.Code
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// Before the first scan ends there's no catalog to match a capture against:
// Status says it's read and waiting, rather than that there's none.
func TestStatusBeforeTheScan(t *testing.T) {
	_, docs, p := fixture(t)
	dir := filepath.Join(docs, "My Games", "Skyrim VR", "SKSE", "MCM Lexicon")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, captureFile), []byte(captureOf("SkyrimVR")), 0o644)
	src := &modlist.Source{Game: "Skyrim VR"}
	s := NewService(catalog.NewService("test"), p, func() (*modlist.Source, string) { return src, "" })
	st := s.Status()
	if st.Capture == nil || !st.Pending || st.Capture.Captured.IsZero() || st.Error != "" {
		t.Errorf("before the scan: %+v", st)
	}
}

func TestMissingChosenFile(t *testing.T) {
	s, _, p := fixture(t)
	p.Update(func(v *prefs.Values) { v.CaptureFile = filepath.Join(t.TempDir(), "gone.json") })
	if st := s.Status(); st.Capture != nil || !strings.Contains(st.Error, "isn't there") {
		t.Errorf("a chosen file that's gone: %+v", st)
	}
	p.Update(func(v *prefs.Values) { v.CaptureOff = true })
	if st := s.Status(); st.Error != "" {
		t.Errorf("turned off, a missing file doesn't matter: %+v", st)
	}
}
