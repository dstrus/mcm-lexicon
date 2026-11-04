package recordings

import (
	"encoding/json"
	"maps"
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
	list    string // the list's path, as the settings key what's saved by list
}

// ours marks mod as a folder MCM Lexicon saved recordings to, as if from
// an earlier session, so its recordings save back in place.
func (h *harness) ours(mod string) {
	h.svc.prefs.Update(func(v *prefs.Values) {
		if v.WroteTo == nil {
			v.WroteTo = map[string][]string{}
		}
		v.WroteTo[h.list] = append(v.WroteTo[h.list], mod)
	})
}

// defaultMod is the save folder with the settings left as they start.
const defaultMod = prefs.NoDeletePrefix + prefs.DefaultSaveName

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
	h := &harness{t: t, mux: http.NewServeMux(), inst: inst, list: src.Path}
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
	if ed.Draft.Name != "My_Settings" || ed.Draft.Title != "My/Settings" || ed.Target.Kind != TargetKindNewMod || ed.Target.Name != defaultMod || ed.Recording != nil {
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
	dir := filepath.Join(h.inst.Dir, "mods", defaultMod)
	if b, err := os.ReadFile(filepath.Join(dir, "McmRecorder", "My_Settings", "0001_Be Seated.json")); err != nil || !strings.Contains(string(b), `"slider": 4.0`) {
		t.Errorf("written: %s, %v", b, err)
	}
	// The new mod folder isn't in the profile until MO2 adds it, so the
	// draft stays on screen, saved.
	if ed.Saved == nil || ed.Saved.Backup != "" || ed.Target.Kind != TargetKindNewMod || ed.Changed {
		t.Errorf("after saving = %+v", ed)
	}

	// A recording in a folder saved to before: edited and saved in place,
	// backed up first.
	h.ours("MGO Recordings")
	h.call("GET", "/api/recordings/MGO", nil, &ed)
	if ed.Target.Name != "MGO Recordings" || ed.Target.Kind != TargetKindMod || ed.Changed || ed.Deletes != "" {
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
	// MCM Lexicon's to delete: in the draft first, where it can be
	// restored, then by Save.
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

func TestRecordAllKeepsWhatsRecorded(t *testing.T) {
	h := newHarness(t)
	h.svc.SetEditing(true, false)
	h.call("POST", "/api/recordings", CreateRequest{Name: "All"}, nil)
	h.call("POST", "/api/recordings/All/steps", AddRequest{Step: Step{Mod: "Be Seated", Option: "Prompts", Kind: KindToggle, Value: "off"}}, nil)
	var r AddAllReply
	code := h.call("POST", "/api/recordings/All/steps/all", AddAllRequest{Steps: []Step{
		{Mod: "Be Seated", Option: "Prompts", Kind: KindToggle, Value: "on"},
		{Mod: "Be Seated", Option: "Height", Kind: KindSlider, Value: "4"},
	}}, &r)
	if code != 200 || r.Added != 1 || r.Kept != 1 {
		t.Fatalf("record all: %d, %+v", code, r)
	}
	var ed Editor
	h.call("GET", "/api/recordings/All", nil, &ed)
	if len(ed.Draft.Steps) != 2 || ed.Draft.Steps[0].Value != "off" {
		t.Errorf("steps = %+v", ed.Draft.Steps)
	}
	// One that can't be written stops the lot.
	code = h.call("POST", "/api/recordings/All/steps/all", AddAllRequest{Steps: []Step{
		{Mod: "Be Seated", Option: "Depth", Kind: KindSlider, Value: "4"},
		{Mod: "Be Seated", Option: "Bad", Kind: KindSlider, Value: "lots"},
	}}, nil)
	h.call("GET", "/api/recordings/All", nil, &ed)
	if code == 200 || len(ed.Draft.Steps) != 2 {
		t.Errorf("a bad setting: %d, %d steps", code, len(ed.Draft.Steps))
	}
}

// The info is edited in the draft; a name or author changed only in case is
// a change, autorun's True for true isn't.
func TestEditingTheInfo(t *testing.T) {
	h := newHarness(t)
	h.ours("MGO Recordings")
	h.svc.SetEditing(true, false)
	var ed Editor
	h.call("GET", "/api/recordings/MGO", nil, &ed)
	d := ed.Draft
	d.Author = "moyce"
	h.call("PUT", "/api/recordings/MGO/draft", d, &ed)
	if !ed.Changed || ed.Files[0].Change != FileChangeChanged {
		t.Fatalf("author in lowercase: changed %v, %s", ed.Changed, ed.Files[0].Change)
	}
	d.Author, d.Title, d.Version, d.Autorun = "Moyce", "MGO Defaults", "1.1.0", false
	h.call("PUT", "/api/recordings/MGO/draft", d, &ed)
	for _, want := range []string{`"name": "MGO Defaults"`, `"version": "1.1.0"`, `"autorun": "false"`} {
		if !strings.Contains(ed.Files[0].Text, want) {
			t.Errorf("info lacks %s:\n%s", want, ed.Files[0].Text)
		}
	}
	if ed.Draft.Name != "MGO" {
		t.Errorf("the file name changed to %q", ed.Draft.Name)
	}
}

// A recording saved to a new mod folder stays listed, marked, after Save,
// after a restart, and once MO2 lists the folder disabled.
func TestSavedToANewFolderStaysListed(t *testing.T) {
	h := newHarness(t)
	h.svc.SetEditing(true, false)
	h.call("POST", "/api/recordings", CreateRequest{Name: "Fresh"}, nil)
	h.call("POST", "/api/recordings/Fresh/steps", AddRequest{Step: Step{Mod: "Be Seated", Option: "Prompts", Kind: KindToggle, Value: "on"}}, nil)
	if code := h.call("POST", "/api/changes/save", nil, nil); code != 200 {
		t.Fatalf("save: %d", code)
	}
	listed := func(svc *Service) *Summary {
		t.Helper()
		h.svc = svc
		var list ListReply
		h.call("GET", "/api/recordings", nil, &list)
		for i, r := range list.Recordings {
			if r.Name == "Fresh" {
				return &list.Recordings[i]
			}
		}
		return nil
	}
	if r := listed(h.svc); r == nil || !r.Unlisted || r.New || r.Drafted || r.Steps != 1 || r.Mod != defaultMod {
		t.Fatalf("after saving: %+v", r)
	}
	// A restart: the draft is gone, the folder's still read.
	mux := http.NewServeMux()
	restarted := NewService(h.svc.cat, h.svc.prefs, nil)
	restarted.Routes(mux)
	h.mux = mux
	if r := listed(restarted); r == nil || !r.Unlisted {
		t.Fatalf("after a restart: %+v", r)
	}
	// MO2 lists the folder, disabled; it's still shown, and can be added to.
	h.inst.Mod(defaultMod, false, nil)
	src, _ := modlist.Check(h.inst.Dir, "")
	cat := catalog.NewService("test")
	cat.UseList(src)
	disabled := NewService(cat, h.svc.prefs, nil)
	disabled.SetEditing(true, false)
	h.mux = http.NewServeMux()
	disabled.Routes(h.mux)
	if r := listed(disabled); r == nil || !r.Unlisted {
		t.Fatalf("listed disabled: %+v", r)
	}
	var added AddReply
	if code := h.call("POST", "/api/recordings/Fresh/steps", AddRequest{Step: Step{Mod: "Be Seated", Option: "Height", Kind: KindSlider, Value: "2"}}, &added); code != 200 || added.Number != 2 {
		t.Errorf("adding to it: %d, %+v", code, added)
	}
	var ed Editor
	h.call("GET", "/api/recordings/Fresh", nil, &ed)
	if ed.Target.Kind != TargetKindMod || ed.Target.Enabled || ed.Target.Name != defaultMod {
		t.Errorf("its target: %+v", ed.Target)
	}
}

// The draft survives a restart: what it holds comes back, editing is on
// so it's in view, and saving it removes the file it was kept in.
func TestDraftKeptAcrossRestarts(t *testing.T) {
	h := newHarness(t)
	p := prefs.Load(filepath.Join(t.TempDir(), "settings.json"))
	h.svc = NewService(h.svc.cat, p, nil)
	h.mux = http.NewServeMux()
	h.svc.Routes(h.mux)
	h.svc.SetEditing(true, false)
	var ed Editor
	if code := h.call("POST", "/api/recordings", CreateRequest{Name: "Fresh"}, &ed); code != 200 {
		t.Fatalf("creating: %d", code)
	}
	var added AddReply
	if code := h.call("POST", "/api/recordings/Fresh/steps", AddRequest{Step: Step{Mod: "Be Seated", Option: "Prompts", Kind: KindToggle, Value: "off"}}, &added); code != 200 {
		t.Fatalf("adding: %d", code)
	}
	file := p.Beside("draft.json")
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("the draft file: %v", err)
	}

	restarted := NewService(h.svc.cat, p, nil)
	if !restarted.Editing() || restarted.Pending() != 1 {
		t.Fatalf("after a restart: editing %v, pending %d", restarted.Editing(), restarted.Pending())
	}
	h.svc, h.mux = restarted, http.NewServeMux()
	restarted.Routes(h.mux)
	if code := h.call("GET", "/api/recordings/Fresh", nil, &ed); code != 200 || ed.Recording != nil || len(ed.Draft.Steps) != 1 || ed.Draft.Steps[0].Option != "Prompts" {
		t.Fatalf("the new recording after a restart: %d, %+v", code, ed)
	}
	var saved SaveReply
	if code := h.call("POST", "/api/changes/save", nil, &saved); code != 200 || len(saved.Saved) != 1 {
		t.Fatalf("saving after a restart: %d, %+v", code, saved)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("the draft file after saving: %v", err)
	}
	if again := NewService(h.svc.cat, p, nil); again.Editing() || again.Pending() != 0 {
		t.Errorf("after saving and a restart: editing %v, pending %d", again.Editing(), again.Pending())
	}

	// A draft that belonged to another list is dropped when this one is read.
	h.svc.SetEditing(true, false)
	if code := h.call("POST", "/api/recordings", CreateRequest{Name: "Other"}, &ed); code != 200 {
		t.Fatalf("creating again: %d", code)
	}
	b, _ := os.ReadFile(file)
	os.WriteFile(file, []byte(strings.Replace(string(b), `"from": "`, `"from": "elsewhere`, 1)), 0o644)
	moved := NewService(h.svc.cat, p, nil)
	h.svc, h.mux = moved, http.NewServeMux()
	moved.Routes(h.mux)
	var list ListReply
	h.call("GET", "/api/recordings", nil, &list)
	if moved.Pending() != 0 {
		t.Errorf("another list's draft: pending %d", moved.Pending())
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("another list's draft file: %v", err)
	}
}

// A step file's text, typed, takes the place of that file's run of steps
// in the draft, and the info file's text sets the recording's info.
func TestPutFileText(t *testing.T) {
	h := newHarness(t)
	h.ours("MGO Recordings")
	h.svc.SetEditing(true, false)
	var ed Editor
	h.call("GET", "/api/recordings/MGO", nil, &ed)
	if len(ed.Files) < 2 {
		t.Fatalf("files: %+v", ed.Files)
	}
	// The one step file: Be Seated's toggle becomes two steps.
	code := h.call("PUT", "/api/recordings/MGO/files/1", FileTextRequest{Text: `[{"mod": "Be Seated", "option": "Prompts", "toggle": "on"}, {"mod": "Be Seated", "option": "Height", "slider": 3.0}]`}, &ed)
	if code != 200 || len(ed.Draft.Steps) != 2 || ed.Draft.Steps[1].Option != "Height" || ed.Draft.Steps[1].Value != "3.0" || !ed.Drafted {
		t.Fatalf("after typing steps: %d, %+v", code, ed.Draft.Steps)
	}
	if code := h.call("PUT", "/api/recordings/MGO/files/1", FileTextRequest{Text: `[{"mod": "Be Seated", "option": "Prompts", "togle": "on"}]`}, &ed); code != 400 {
		t.Errorf("a slip: %d", code)
	}
	if code := h.call("PUT", "/api/recordings/MGO/files/3", FileTextRequest{Text: `[]`}, &ed); code != 404 {
		t.Errorf("no such file: %d", code)
	}
	code = h.call("PUT", "/api/recordings/MGO/files/0", FileTextRequest{Text: `{"author": "Me", "autorun": "false", "name": "MGO", "version": "2.0.0"}`}, &ed)
	if code != 200 || ed.Draft.Author != "Me" || ed.Draft.Autorun || ed.Draft.Version != "2.0.0" {
		t.Fatalf("after typing the info: %d, %+v", code, ed.Draft)
	}
	// Typed back to what's on disk: nothing left to save.
	h.call("PUT", "/api/recordings/MGO/files/0", FileTextRequest{Text: ed.Files[0].Old}, &ed)
	h.call("PUT", "/api/recordings/MGO/files/1", FileTextRequest{Text: `[{"Mod": "Be Seated", "option": "Prompts", "toggle": "off"}]`}, &ed)
	if ed.Drafted || ed.Changed {
		t.Errorf("typed back: drafted %v, changed %v", ed.Drafted, ed.Changed)
	}
}

// A stepper's clicks are a run of identical click steps: set, changed,
// and taken out, in place, leaving the other steps as they were.
func TestSetClicks(t *testing.T) {
	h := newHarness(t)
	h.svc.SetEditing(true, false)
	click := Step{Mod: "Be Seated", Option: "Mode", Kind: KindClick}
	n := func(k int) *int { return &k }
	var reply AddReply
	if code := h.call("POST", "/api/recordings/MGO/steps", AddRequest{Step: click, Clicks: n(2)}, &reply); code != 200 || reply.Number != 2 {
		t.Fatalf("two clicks: %d, %+v", code, reply)
	}
	var ed Editor
	h.call("GET", "/api/recordings/MGO", nil, &ed)
	clicks := func() (int, []string) {
		var c int
		var order []string
		for _, s := range ed.Draft.Steps {
			order = append(order, s.Option)
			if s.Kind == KindClick && s.Option == "Mode" {
				c++
			}
		}
		return c, order
	}
	if c, order := clicks(); c != 2 || len(order) != 3 || order[0] != "Prompts" {
		t.Fatalf("after two: %d, %v", c, order)
	}
	h.call("POST", "/api/recordings/MGO/steps", AddRequest{Step: Step{Mod: "Be Seated", Option: "Height", Kind: KindSlider, Value: "2"}}, &reply)
	h.call("POST", "/api/recordings/MGO/steps", AddRequest{Step: click, Clicks: n(3)}, &reply)
	h.call("GET", "/api/recordings/MGO", nil, &ed)
	if c, order := clicks(); c != 3 || order[1] != "Mode" || order[3] != "Mode" || order[4] != "Height" {
		t.Fatalf("after three, in place: %d, %v", c, order)
	}
	// A run's waits, typed on its first click, are kept as the count changes.
	ed.Draft.Steps[1].Timeout = "60"
	h.call("PUT", "/api/recordings/MGO/draft", ed.Draft, &ed)
	h.call("POST", "/api/recordings/MGO/steps", AddRequest{Step: click, Clicks: n(2)}, &reply)
	h.call("GET", "/api/recordings/MGO", nil, &ed)
	if ed.Draft.Steps[1].Timeout != "60" || ed.Draft.Steps[2].Timeout != "60" {
		t.Errorf("the run's timeout: %q, %q", ed.Draft.Steps[1].Timeout, ed.Draft.Steps[2].Timeout)
	}
	h.call("POST", "/api/recordings/MGO/steps", AddRequest{Step: click, Clicks: n(0)}, &reply)
	h.call("GET", "/api/recordings/MGO", nil, &ed)
	if c, order := clicks(); c != 0 || len(order) != 2 {
		t.Fatalf("after none: %d, %v", c, order)
	}
	if code := h.call("POST", "/api/recordings/MGO/steps", AddRequest{Step: Step{Mod: "Be Seated", Option: "Prompts", Kind: KindToggle, Value: "on"}, Clicks: n(2)}, &reply); code != 400 {
		t.Errorf("clicks on a toggle: %d", code)
	}
	if code := h.call("POST", "/api/recordings/MGO/steps", AddRequest{Step: click, Clicks: n(1000)}, &reply); code != 400 {
		t.Errorf("a thousand clicks: %d", code)
	}
}

// Undo steps back through the draft's changes, one at a time, and redo
// forward again; a new change drops what was undone, and Save starts the
// history afresh.
func TestUndoRedo(t *testing.T) {
	h := newHarness(t)
	h.svc.SetEditing(true, false)
	toggle := func(v string) {
		h.call("POST", "/api/recordings/MGO/steps", AddRequest{Step: Step{Mod: "Be Seated", Option: "Prompts", Kind: KindToggle, Value: v}}, &AddReply{})
	}
	value := func() string {
		var ed Editor
		h.call("GET", "/api/recordings/MGO", nil, &ed)
		return ed.Draft.Steps[0].Value
	}
	history := func() (int, int) { return h.svc.History() }
	if u, r := history(); u != 0 || r != 0 {
		t.Fatalf("at the start: %d, %d", u, r)
	}
	toggle("on")
	h.call("POST", "/api/recordings/MGO/steps", AddRequest{Step: Step{Mod: "Be Seated", Option: "Height", Kind: KindSlider, Value: "4"}}, &AddReply{})
	if u, _ := history(); u != 2 || value() != "on" {
		t.Fatalf("after two changes: undo %d, %s", u, value())
	}
	var ch Changes
	if code := h.call("POST", "/api/changes/undo", nil, &ch); code != 200 {
		t.Fatalf("undo: %d", code)
	}
	var ed Editor
	h.call("GET", "/api/recordings/MGO", nil, &ed)
	if len(ed.Draft.Steps) != 1 || ed.Draft.Steps[0].Value != "on" {
		t.Fatalf("after one undo: %+v", ed.Draft.Steps)
	}
	h.call("POST", "/api/changes/undo", nil, &ch)
	if h.svc.Pending() != 0 || value() != "off" {
		t.Fatalf("after two undos: pending %d, %s", h.svc.Pending(), value())
	}
	if code := h.call("POST", "/api/changes/undo", nil, &ch); code != 409 {
		t.Errorf("nothing to undo: %d", code)
	}
	h.call("POST", "/api/changes/redo", nil, &ch)
	if u, r := history(); u != 1 || r != 1 || value() != "on" {
		t.Fatalf("after a redo: %d, %d, %s", u, r, value())
	}
	// Toggled back to what's on disk: one change, which undo takes back.
	toggle("off")
	if u, r := history(); u != 2 || r != 0 || h.svc.Pending() != 0 {
		t.Fatalf("back to disk: %d, %d, pending %d", u, r, h.svc.Pending())
	}
	h.call("POST", "/api/changes/undo", nil, &ch)
	if value() != "on" {
		t.Fatalf("undoing the return: %s", value())
	}
	if code := h.call("POST", "/api/changes/save", nil, &SaveReply{}); code != 200 {
		t.Fatalf("save: %d", code)
	}
	if u, r := history(); u != 0 || r != 0 {
		t.Errorf("after saving: %d, %d", u, r)
	}
}

// A backup goes back through the draft: chosen, it's a change Save writes
// exactly, which undo takes back; a deleted recording comes back from the
// backup its deletion made.
func TestRestoreFromBackup(t *testing.T) {
	h := newHarness(t)
	h.ours("MGO Recordings")
	h.svc.SetEditing(true, false)
	dir := filepath.Join(h.inst.Dir, "mods", "MGO Recordings")
	original := Disk(dir, "MGO")
	// A change, saved: the save backs the original up.
	h.call("POST", "/api/recordings/MGO/steps", AddRequest{Step: Step{Mod: "Be Seated", Option: "Prompts", Kind: KindToggle, Value: "on"}}, &AddReply{})
	if code := h.call("POST", "/api/changes/save", nil, &SaveReply{}); code != 200 {
		t.Fatalf("save: %d", code)
	}
	if maps.Equal(Disk(dir, "MGO"), original) {
		t.Fatal("the save changed nothing")
	}
	var list BackupsReply
	h.call("GET", "/api/recordings/MGO/backups", nil, &list)
	if len(list.Backups) != 1 || list.Backups[0].Reason != BackupSave || list.Backups[0].Mod != "MGO Recordings" || list.Restoring != "" {
		t.Fatalf("backups: %+v", list)
	}
	changed := 0
	for _, f := range list.Backups[0].Planned {
		if f.Change != FileChangeSame {
			changed++
		}
	}
	if changed == 0 {
		t.Errorf("the backup's files show no change from disk: %+v", list.Backups[0].Planned)
	}
	id := list.Backups[0].ID
	// Chosen: a change in the draft, with the backup's steps.
	var ed Editor
	if code := h.call("PUT", "/api/recordings/MGO/backups/"+id, nil, &ed); code != 200 || ed.FromBackup == nil || !ed.Changed || ed.Draft.Steps[0].Value != "off" {
		t.Fatalf("restoring: %d, %+v", code, ed.FromBackup)
	}
	var ch Changes
	h.call("GET", "/api/changes", nil, &ch)
	if h.svc.Pending() != 1 || len(ch.Recordings) != 1 || ch.Recordings[0].Kind != ChangeKindRestored {
		t.Fatalf("changes: pending %d, %+v", h.svc.Pending(), ch.Recordings)
	}
	h.call("GET", "/api/recordings/MGO/backups", nil, &list)
	if list.Restoring != id {
		t.Errorf("restoring: %q", list.Restoring)
	}
	// Undone and redone.
	h.call("POST", "/api/changes/undo", nil, &ch)
	if h.svc.Pending() != 0 {
		t.Fatalf("after undo: pending %d", h.svc.Pending())
	}
	h.call("POST", "/api/changes/redo", nil, &ch)
	// Saved: the original files, exactly, and a backup of what they replaced.
	if code := h.call("POST", "/api/changes/save", nil, &SaveReply{}); code != 200 {
		t.Fatalf("save the restore: %d", code)
	}
	if disk := Disk(dir, "MGO"); !maps.Equal(disk, original) {
		t.Errorf("after restoring: %v", disk)
	}
	h.call("GET", "/api/recordings/MGO/backups", nil, &list)
	if len(list.Backups) != 2 || list.Backups[0].Reason != BackupRestore {
		t.Errorf("after restoring, backups: %+v", list.Backups)
	}
	// Deleted (MCM Lexicon saved to its folder, so it may), then back.
	if code := h.call("DELETE", "/api/recordings/MGO", nil, &ed); code != 200 {
		t.Fatalf("delete: %d", code)
	}
	h.call("POST", "/api/changes/save", nil, &SaveReply{})
	if len(Disk(dir, "MGO")) != 0 {
		t.Fatal("still on disk after deleting")
	}
	var gone DeletedReply
	h.call("GET", "/api/backups", nil, &gone)
	if len(gone.Recordings) != 1 || gone.Recordings[0].Name != "MGO" || gone.Recordings[0].Latest.Reason != BackupDelete || gone.Recordings[0].Count != 3 {
		t.Fatalf("deleted with backups: %+v", gone)
	}
	if code := h.call("PUT", "/api/recordings/MGO/backups/"+gone.Recordings[0].Latest.ID, nil, &ed); code != 200 || ed.Recording != nil || ed.Target.Name != "MGO Recordings" {
		t.Fatalf("restoring a deleted recording: %d, %+v", code, ed.Target)
	}
	var recs ListReply
	h.call("GET", "/api/recordings", nil, &recs)
	found := false
	for _, r := range recs.Recordings {
		if r.Name == "MGO" && r.Restoring {
			found = true
		}
	}
	if !found {
		t.Errorf("the list doesn't show MGO being restored: %+v", recs.Recordings)
	}
	h.call("POST", "/api/changes/save", nil, &SaveReply{})
	if disk := Disk(dir, "MGO"); !maps.Equal(disk, original) {
		t.Errorf("after restoring a deleted recording: %v", disk)
	}
}

// A recording from another mod saves to the save folder, leaving the mod's files
// as they came. Its step file that the copy doesn't replace by name gets
// an empty one over it, so the game, seeing one folder made of both,
// doesn't play it. The save folder's name follows the settings.
func TestRecordingFromAnotherModSavesToTheSaveFolder(t *testing.T) {
	h := newHarness(t)
	h.svc.SetEditing(true, false)
	var ed Editor
	h.call("GET", "/api/recordings/MGO", nil, &ed)
	if ed.Target.Name != defaultMod || ed.Target.Kind != TargetKindNewMod || ed.Target.LosesTo != "" || ed.Changed || ed.Drafted || ed.Deletes != "not-ours" {
		t.Fatalf("untouched: %+v, changed %v", ed.Target, ed.Changed)
	}
	// Its one step moved to another menu: the mod's 0001_Be Seated.json
	// isn't replaced, so it's covered.
	d := ed.Draft
	d.Steps[0].Mod = "Other"
	h.call("PUT", "/api/recordings/MGO/draft", d, &ed)
	var names []string
	cover := ""
	for _, f := range ed.Files {
		names = append(names, f.Name)
		if f.Covers != "" {
			cover = f.Name + " over " + f.Covers + ": " + f.Text
		}
	}
	if strings.Join(names, ", ") != "MGO.json, 0001_Other.json, 0001_Be Seated.json" || cover != "0001_Be Seated.json over MGO Recordings: []" {
		t.Fatalf("files: %v, %s", names, cover)
	}
	// Put back as it was: nothing to save, though the save folder doesn't
	// hold it.
	d.Steps[0].Mod = "Be Seated"
	h.call("PUT", "/api/recordings/MGO/draft", d, &ed)
	if ed.Drafted || ed.Changed || h.svc.Pending() != 0 {
		t.Fatalf("put back: drafted %v, changed %v, pending %d", ed.Drafted, ed.Changed, h.svc.Pending())
	}
	d.Steps[0].Mod = "Other"
	h.call("PUT", "/api/recordings/MGO/draft", d, &ed)
	if code := h.call("POST", "/api/changes/save", nil, &SaveReply{}); code != 200 {
		t.Fatalf("save: %d", code)
	}
	mine := Disk(filepath.Join(h.inst.Dir, "mods", defaultMod), "MGO")
	theirs := Disk(filepath.Join(h.inst.Dir, "mods", "MGO Recordings"), "MGO")
	if mine["McmRecorder/MGO/0001_Be Seated.json"] != "[]" || mine["McmRecorder/MGO/0001_Other.json"] == "" || len(theirs) != 2 || !strings.Contains(theirs["McmRecorder/MGO/0001_Be Seated.json"], "Be Seated") {
		t.Errorf("mine %v, theirs %v", mine, theirs)
	}

	// Named in the settings, without the prefix.
	off := false
	h.svc.prefs.Update(func(v *prefs.Values) { v.SaveName, v.NoDelete = "My Recordings", &off })
	h.call("GET", "/api/recordings/MGO", nil, &ed)
	if ed.Target.Name != "My Recordings" {
		t.Errorf("renamed: %+v", ed.Target)
	}
}

// Another recording's steps are added to one in order, the options it has
// already keeping their steps, and a stepper's run of clicks whole, so
// recordings can be pulled together. The other recording is left as it is.
func TestAddStepsFromAnotherRecording(t *testing.T) {
	h := newHarness(t)
	h.inst.Mod("Theirs", true, map[string]string{
		"McmRecorder/Theirs.json":                `{"name": "Theirs"}`,
		"McmRecorder/Theirs/0001_Be Seated.json": `[{"mod": "Be Seated", "option": "Prompts", "toggle": "on"}, {"mod": "Be Seated", "option": "Height", "slider": 7.0}]`,
		"McmRecorder/Theirs/0002_Other.json":     `[{"mod": "Other", "option": "Next", "click": "Next"}, {"mod": "Other", "option": "Next", "click": "Next"}]`,
	})
	src, _ := modlist.Check(h.inst.Dir, "")
	cat := catalog.NewService("test")
	cat.UseList(src)
	h.svc = NewService(cat, prefs.Load(""), nil)
	h.svc.SetEditing(true, false)
	h.mux = http.NewServeMux()
	h.svc.Routes(h.mux)
	h.call("POST", "/api/recordings", CreateRequest{Name: "Mine"}, nil)
	h.call("POST", "/api/recordings/Mine/steps", AddRequest{Step: Step{Mod: "Be Seated", Option: "Prompts", Kind: KindToggle, Value: "off"}}, nil)
	var r AddAllReply
	if code := h.call("POST", "/api/recordings/Mine/steps/from", AddFromRequest{From: "Theirs"}, &r); code != 200 || r.Added != 3 || r.Kept != 1 {
		t.Fatalf("from Theirs: %d, %+v", code, r)
	}
	var ed Editor
	h.call("GET", "/api/recordings/Mine", nil, &ed)
	var got []string
	for _, s := range ed.Draft.Steps {
		got = append(got, s.Option+"="+s.Value)
	}
	if strings.Join(got, ", ") != "Prompts=off, Height=7.0, Next=Next, Next=Next" {
		t.Errorf("steps: %v", got)
	}
	h.call("GET", "/api/recordings/Theirs", nil, &ed)
	if ed.Drafted || len(ed.Draft.Steps) != 4 {
		t.Errorf("Theirs changed: drafted %v, %d steps", ed.Drafted, len(ed.Draft.Steps))
	}
	if code := h.call("POST", "/api/recordings/Mine/steps/from", AddFromRequest{From: "Mine"}, nil); code == 200 {
		t.Error("added to itself")
	}
	if code := h.call("POST", "/api/recordings/Mine/steps/from", AddFromRequest{From: "Nope"}, nil); code != 404 {
		t.Errorf("a missing recording: %d", code)
	}
	// One undo takes the lot back.
	h.call("POST", "/api/changes/undo", nil, &Changes{})
	h.call("GET", "/api/recordings/Mine", nil, &ed)
	if len(ed.Draft.Steps) != 1 {
		t.Errorf("after undo: %d steps", len(ed.Draft.Steps))
	}
}
