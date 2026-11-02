package recordings

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lexicon/internal/catalog"
	"lexicon/internal/modlist"
	"lexicon/internal/prefs"
	"lexicon/internal/testcorpus"
)

type harness struct {
	t       *testing.T
	mux     *http.ServeMux
	svc     *Service
	running bool
	inst    *testcorpus.Instance
}

func newHarness(t *testing.T) *harness {
	inst := testcorpus.New(t, "MGO", "Default")
	inst.Mod("Be Seated", true, map[string]string{
		"MCM/Config/BeSeated/config.json": `{"modName": "BeSeated", "displayName": "Be Seated", "content": [{"text": "Prompts", "type": "toggle", "id": "bPrompts:Main"}, {"text": "Height", "type": "slider", "id": "fHeight:Main", "valueOptions": {"min": 0, "max": 10, "step": 1}}]}`,
		"BeSeated.esp":                    testcorpus.Plugin(),
	})
	inst.Mod("MGO Recordings", true, map[string]string{
		"McmRecorder/MGO.json":                `{"name": "MGO", "author": "Moyce", "autorun": "True", "version": "1.0.0"}`,
		"McmRecorder/MGO/0001_Be Seated.json": `[{"Mod": "Be Seated", "option": "Prompts", "toggle": "off"}]`,
	})
	inst.Plugins("*BeSeated.esp")
	src, err := modlist.Check(inst.Dir, "")
	if err != nil {
		t.Fatal(err)
	}
	cat := catalog.NewService("test")
	cat.UseList(src)
	h := &harness{t: t, mux: http.NewServeMux(), inst: inst}
	h.svc = NewService(cat, prefs.Load(""), func() bool { return h.running })
	h.svc.now = func() time.Time { return time.Date(2026, 11, 2, 23, 4, 0, 0, time.Local) }
	h.svc.Routes(h.mux)
	return h
}

func (h *harness) call(method, path string, body any, out any) int {
	h.t.Helper()
	var r *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		r = httptest.NewRequest(method, path, strings.NewReader(string(b)))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	h.mux.ServeHTTP(w, r)
	if out != nil && w.Code == 200 {
		if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
			h.t.Fatalf("%s %s: %v\n%s", method, path, err, w.Body)
		}
	}
	return w.Code
}

func TestNewRecordingAddSaveDelete(t *testing.T) {
	h := newHarness(t)
	var ed Editor
	// Read-only until editing is turned on.
	if code := h.call("POST", "/api/recordings", CreateRequest{Name: "My/Settings", Author: "Me"}, nil); code == 200 {
		t.Fatal("a recording was created while read-only")
	}
	if err := h.svc.SetEditing(true, false); err != nil {
		t.Fatal(err)
	}
	if code := h.call("POST", "/api/recordings", CreateRequest{Name: "My/Settings", Author: "Me"}, &ed); code != 200 {
		t.Fatalf("create: %d", code)
	}
	if ed.Draft.Name != "My_Settings" || ed.Draft.Title != "My/Settings" || ed.Target.Kind != TargetKindNewMod || ed.Target.Name != DefaultTarget || ed.Recording != nil {
		t.Errorf("created = %+v", ed)
	}
	if code := h.call("POST", "/api/recordings", CreateRequest{Name: "MGO"}, nil); code == 200 {
		t.Error("a second MGO was created")
	}
	var added AddReply
	for i, st := range []Step{
		{Mod: "Be Seated", Option: "Prompts", Kind: KindToggle, Value: "on"},
		{Mod: "Be Seated", Option: "Height", Kind: KindSlider, Value: "4"},
	} {
		if code := h.call("POST", "/api/recordings/my_settings/steps", AddRequest{Step: st}, &added); code != 200 || added.Number != i+1 {
			t.Fatalf("add %d: %d, %+v", i+1, code, added)
		}
	}
	if code := h.call("POST", "/api/recordings/My_Settings/steps", AddRequest{Step: Step{Mod: "Be Seated", Option: "Height", Kind: KindSlider, Value: "high"}}, nil); code == 200 {
		t.Error("a slider step with a word for its value was added")
	}
	h.call("GET", "/api/recordings/My_Settings", nil, &ed)
	if len(ed.Draft.Steps) != 2 || ed.Counts.Matched != 2 || !ed.Changed || ed.Unsaved != "" || len(ed.Files) != 2 || ed.Files[1].Change != FileChangeNew {
		t.Fatalf("editor = %+v", ed)
	}
	var list ListReply
	h.call("GET", "/api/recordings", nil, &list)
	if len(list.Recordings) != 2 || !list.Recordings[1].New || list.Recordings[1].Steps != 2 {
		t.Errorf("list = %+v", list.Recordings)
	}

	var ch Changes
	h.call("GET", "/api/changes", nil, &ch)
	if len(ch.Recordings) != 1 || ch.Recordings[0].Kind != ChangeKindNew || ch.Files != 2 || h.svc.Pending() != 1 {
		t.Fatalf("changes = %+v", ch)
	}
	if err := h.svc.SetEditing(false, false); err == nil {
		t.Error("editing turned off with changes in the draft")
	}
	h.running = true
	if code := h.call("POST", "/api/changes/save", nil, nil); code == 200 {
		t.Error("saved while Skyrim runs")
	}
	h.running = false
	var saved SaveReply
	if code := h.call("POST", "/api/changes/save", nil, &saved); code != 200 || len(saved.Saved) != 1 || len(saved.Changes.Recordings) != 0 {
		t.Fatalf("save: %d, %+v", code, saved)
	}
	h.call("GET", "/api/recordings/My_Settings", nil, &ed)
	dir := filepath.Join(h.inst.Dir, "mods", DefaultTarget)
	if b, err := os.ReadFile(filepath.Join(dir, "McmRecorder", "My_Settings", "0001_Be Seated.json")); err != nil || !strings.Contains(string(b), `"slider": 4.0`) {
		t.Errorf("written: %s, %v", b, err)
	}
	// The new mod folder isn't in the profile until MO2 adds it, so the
	// draft stays on screen, saved.
	if ed.Saved == nil || ed.Saved.Backup != "" || ed.Target.Kind != TargetKindNewMod || ed.Changed {
		t.Errorf("after saving = %+v", ed)
	}

	// The list's own recording: edited and saved in place, backed up first,
	// and never deleted.
	h.call("GET", "/api/recordings/MGO", nil, &ed)
	if ed.Target.Name != "MGO Recordings" || ed.Target.Kind != TargetKindMod || ed.Changed || ed.Deletes != "not-ours" {
		t.Fatalf("MGO = %+v", ed.Target)
	}
	d := ed.Draft
	d.Steps[0].Value = "on"
	if code := h.call("PUT", "/api/recordings/MGO/draft", d, &ed); code != 200 || !ed.Changed || !ed.Drafted {
		t.Fatalf("draft: %d, %+v", code, ed)
	}
	// "True" and "true", "Mod" and "mod", read alike: only the step changed.
	if ed.Files[0].Change != FileChangeSame || ed.Files[1].Change != FileChangeChanged {
		t.Errorf("files = %s, %s", ed.Files[0].Change, ed.Files[1].Change)
	}
	h.call("POST", "/api/changes/save", nil, &saved)
	h.call("GET", "/api/recordings/MGO", nil, &ed)
	if len(saved.Saved) != 1 || saved.Saved[0].Backup == "" || ed.Drafted || ed.Changed || ed.Recording == nil || ed.Recording.Steps[0].Value != "on" {
		t.Fatalf("saved MGO = %+v", ed)
	}
	if _, err := os.Stat(filepath.Join(h.inst.Dir, "lexicon-backups", "MGO")); err != nil {
		t.Errorf("no backup beside the list: %v", err)
	}
	// Saved there now, so it's MCM Lexicon's to delete: in the draft first,
	// where it can be restored, then by Save.
	if code := h.call("DELETE", "/api/recordings/MGO", nil, &ed); code != 200 || !ed.Deleted || ed.Files[0].Change != FileChangeRemoved {
		t.Fatalf("delete: %d, %+v", code, ed)
	}
	h.call("POST", "/api/recordings/MGO/restore", nil, &ed)
	if ed.Deleted || h.svc.Pending() != 0 {
		t.Errorf("restored = %+v", ed)
	}
	h.call("DELETE", "/api/recordings/MGO", nil, nil)
	if code := h.call("POST", "/api/changes/save", nil, &saved); code != 200 || saved.Saved[0].Kind != ChangeKindDeleted || saved.Saved[0].Backup == "" {
		t.Errorf("deleting save: %d, %+v", code, saved)
	}
	if _, err := os.Stat(filepath.Join(h.inst.Dir, "mods", "MGO Recordings", "McmRecorder", "MGO.json")); err == nil {
		t.Error("MGO's info file is still there")
	}
	h.call("GET", "/api/recordings", nil, &list)
	for _, r := range list.Recordings {
		if r.Name == "MGO" {
			t.Error("MGO is still listed")
		}
	}
}

func TestLeavingEditingDiscards(t *testing.T) {
	h := newHarness(t)
	h.svc.SetEditing(true, false)
	h.call("POST", "/api/recordings", CreateRequest{Name: "Scratch"}, nil)
	if err := h.svc.SetEditing(false, true); err != nil || h.svc.Pending() != 0 || h.svc.Editing() {
		t.Fatalf("after discarding: %v, %d pending", err, h.svc.Pending())
	}
	if code := h.call("GET", "/api/recordings/Scratch", nil, nil); code == 200 {
		t.Error("the discarded recording is still there")
	}
}

func TestTargetThatLoses(t *testing.T) {
	h := newHarness(t)
	h.inst.Mod("Later", true, map[string]string{"McmRecorder/MGO.json": `{"name": "MGO"}`})
	src, _ := modlist.Check(h.inst.Dir, "")
	cat := catalog.NewService("test")
	cat.UseList(src)
	h.svc = NewService(cat, prefs.Load(""), nil)
	h.svc.SetEditing(true, false)
	h.mux = http.NewServeMux()
	h.svc.Routes(h.mux)
	var ed Editor
	h.call("GET", "/api/recordings/MGO", nil, &ed)
	d := ed.Draft
	d.Target = "MGO Recordings"
	h.call("PUT", "/api/recordings/MGO/draft", d, &ed)
	if ed.Target.LosesTo != "Later" {
		t.Errorf("target = %+v", ed.Target)
	}
	d.Target = "../escape"
	if code := h.call("PUT", "/api/recordings/MGO/draft", d, nil); code == 200 {
		t.Error("a target outside the mods folder was taken")
	}
}

// The steps from the report: a new recording made while adding, the same
// option added twice, then everything discarded.
func TestAddingAgainChangesTheStepAndDiscardingForgetsTheRecording(t *testing.T) {
	h := newHarness(t)
	h.svc.SetEditing(true, false)
	var ed Editor
	h.call("POST", "/api/recordings", CreateRequest{Name: "Fresh"}, &ed)
	h.svc.prefs.Update(func(v *prefs.Values) { v.OpenRecording = "Fresh" })
	prompts := Step{Mod: "Be Seated", Option: "Prompts", Kind: KindToggle, Value: "on"}
	var added AddReply
	h.call("POST", "/api/recordings/Fresh/steps", AddRequest{Step: prompts}, &added)
	prompts.Value = "off"
	if code := h.call("POST", "/api/recordings/Fresh/steps", AddRequest{Step: prompts}, &added); code != 200 || added.Number != 1 || !added.Replaced {
		t.Fatalf("adding again: %d, %+v", code, added)
	}
	h.call("GET", "/api/recordings/Fresh", nil, &ed)
	if len(ed.Draft.Steps) != 1 || ed.Draft.Steps[0].Value != "off" {
		t.Errorf("steps = %+v", ed.Draft.Steps)
	}
	if code := h.call("DELETE", "/api/changes", nil, nil); code != 200 {
		t.Fatal(code)
	}
	if open := h.svc.prefs.Get().OpenRecording; open != "" {
		t.Errorf("the discarded recording is still open: %q", open)
	}
	// Discarding changes to a saved recording leaves it open.
	h.svc.prefs.Update(func(v *prefs.Values) { v.OpenRecording = "MGO" })
	h.call("POST", "/api/recordings/MGO/steps", AddRequest{Step: prompts}, nil)
	h.call("DELETE", "/api/changes", nil, nil)
	if open := h.svc.prefs.Get().OpenRecording; open != "MGO" {
		t.Errorf("open = %q", open)
	}
}
