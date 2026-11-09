package catalog

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"lexicon/internal/modlist"
	"lexicon/internal/testcorpus"
)

// A capture file as the plugin writes it: one menu the catalog has, with a
// page it has and one it lacks; one menu it lacks; and one SkyUI doesn't
// list. The game's casing differs from the scripts' in places, as pooled
// strings do.
const captureJSON = `{
	"format": 1,
	"plugin": "0.2.0",
	"game": "SkyrimVR",
	"runtime": "1.4.15.0",
	"captured": "2026-11-09T18:11:43Z",
	"menus": [
		{"name": "Test", "script": "test_mcm", "quest": "test.esp:d62", "pages": [
			{"name": "general", "options": [
				{"position": 0, "type": "toggle", "flags": 0, "text": "Enable", "str": "None", "num": 1, "state": ""},
				{"position": 1, "type": "header", "flags": 0, "text": "Movement", "str": "None", "num": 0, "state": ""},
				{"position": 2, "type": "menu", "flags": 0, "text": "Mode", "str": "slow", "num": 0, "state": ""},
				{"position": 3, "type": "slider", "flags": 0, "text": "speed", "str": "{1}", "num": 2.5, "state": ""},
				{"position": 5, "type": "text", "flags": 0, "text": "Lydia", "str": "None", "num": 0, "state": ""},
				{"position": 8, "type": "text", "flags": 0, "text": "Version: 1.2", "str": "", "num": 0, "state": ""},
				{"position": 10, "type": "toggle", "flags": 0, "text": "Repeat", "str": "None", "num": 0, "state": ""},
				{"position": 11, "type": "toggle", "flags": 0, "text": "Repeat", "str": "None", "num": 1, "state": ""},
				{"position": 12, "type": "keymap", "flags": 5, "text": "Hotkey", "str": "None", "num": 42, "state": "HotkeyST"}
			]},
			{"name": "Advanced", "options": [
				{"position": 0, "type": "header", "flags": 0, "text": "Debug", "str": "None", "num": 0, "state": ""},
				{"position": 2, "type": "input", "flags": 0, "text": "Log name", "str": "lexicon", "num": 0, "state": ""}
			]},
			{"name": "Broken", "error": "timed out"}
		]},
		{"name": "Other", "script": "Other_MCM", "quest": "other.esp:801", "pages": [
			{"name": "", "options": [
				{"position": 0, "type": "toggle", "flags": 0, "text": "On", "str": "None", "num": 0, "state": ""}
			]}
		]},
		{"name": "", "script": "Hidden_MCM", "quest": "hidden.esp:800", "pages": [{"name": "", "options": []}]}
	],
	"skipped": [{"script": "DBM_DynamicMCMScript", "reason": "asked to skip"}]
}`

func captureCatalog() *Catalog {
	return &Catalog{
		SchemaVersion: SchemaVersion,
		Menus: []Menu{{
			ID: "papyrus-test", Name: "Test", NameText: "Test", Kind: KindSource, Script: "Test_MCM",
			Pages: []Page{{Name: "General", Text: "General", Placed: true, Options: []Option{
				{Type: OptionTypeToggle, Label: "Enable", Certainty: CertaintyExact, Position: 0},
				{Type: OptionTypeMenu, Label: "Mode", Certainty: CertaintyExact, Position: 2, Choices: []Choice{{Raw: "Fast"}, {Raw: "Slow"}}},
				{Type: OptionTypeSlider, Label: "Speed", Certainty: CertaintyExact, Position: -1},
				{Type: OptionTypeText, Label: "", Certainty: CertaintyUnknown, Position: 5},
				{Type: OptionTypeText, Label: "*Version*", Certainty: CertaintyPattern, Position: -1},
				{Type: OptionTypeToggle, Label: "Repeat", Certainty: CertaintyExact, Position: 6, Index: 1},
				{Type: OptionTypeToggle, Label: "Repeat", Certainty: CertaintyExact, Position: 7, Index: 2},
			}}},
		}},
	}
}

func readTestCapture(t *testing.T) (*Capture, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "capture.json")
	if err := os.WriteFile(path, []byte(captureJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := ReadCapture(path)
	if err != nil {
		t.Fatal(err)
	}
	return f, path
}

func TestApplyCapture(t *testing.T) {
	f, path := readTestCapture(t)
	if f.Game != "SkyrimVR" || len(f.Menus) != 3 || f.Captured.IsZero() {
		t.Fatalf("read: %+v", f)
	}
	base := captureCatalog()
	c := ApplyCapture(base, f, path)

	want := CaptureReport{
		Menus: 2, MenusMatched: 1, MenusAdded: 1,
		Pages: 3, PagesMatched: 1, PagesAdded: 2,
		Options: 10, ByPosition: 3, ByLabel: 3, ByPattern: 1, OptionsAdded: 3,
		Values: 8,
	}
	if c.Capture == nil || c.Capture.Report != want || c.Capture.Path != path || c.Capture.Game != "SkyrimVR" {
		t.Errorf("report:\n got %+v\nwant %+v", c.Capture, want)
	}
	if base.Capture != nil || base.Menus[0].Pages[0].Options[0].Game != nil || len(base.Menus) != 1 || len(base.Menus[0].Pages) != 1 {
		t.Error("the catalog given was changed")
	}

	// Menus stay in reading order, the added one among them.
	if len(c.Menus) != 2 || c.Menus[0].Name != "Other" || c.Menus[1].Name != "Test" {
		t.Fatalf("menus: %+v", c.Menus)
	}
	other := c.Menus[0]
	if other.Kind != KindCaptured || other.Certainty != CertaintyCaptured || other.Plugin != "other.esp" || other.ID != "captured-other" ||
		len(other.Pages) != 1 || len(other.Pages[0].Options) != 1 || other.Pages[0].Options[0].Game.Bool {
		t.Errorf("added menu: %+v", other)
	}

	opts := c.Menus[1].Pages[0].Options
	game := func(i int) Value {
		if opts[i].Game == nil {
			t.Errorf("%s has no game value", opts[i].Label)
			return Value{}
		}
		return *opts[i].Game
	}
	if v := game(0); v.Kind != ValueKindBool || !v.Bool {
		t.Errorf("toggle by position: %+v", v)
	}
	if v := game(1); v.Kind != ValueKindNumber || v.Number != 1 {
		t.Errorf("menu entry by position, as its choices number it: %+v", v)
	}
	if v := game(2); v.Number != 2.5 {
		t.Errorf("slider by label, without regard to case: %+v", v)
	}
	if opts[3].GameLabel != "Lydia" || opts[3].Game != nil {
		t.Errorf("unknown label by position: label %q, value %+v (None isn't a value)", opts[3].GameLabel, opts[3].Game)
	}
	if opts[4].GameLabel != "Version: 1.2" {
		t.Errorf("pattern: label %q", opts[4].GameLabel)
	}
	if game(5).Bool || !game(6).Bool {
		t.Error("options sharing a label match in order")
	}
	added := opts[7]
	if len(opts) != 8 || added.Certainty != CertaintyCaptured || added.Type != OptionTypeKeymap || added.Position != 12 ||
		added.State != "HotkeyST" || added.Disabled != DisabledSometimes || game(7).Number != 42 {
		t.Errorf("added option: %+v", opts[7:])
	}

	adv := c.Menus[1].Pages[1]
	if len(c.Menus[1].Pages) != 2 || adv.Name != "Advanced" || !adv.Placed || len(adv.Fixtures) != 1 || adv.Fixtures[0].Label != "Debug" ||
		len(adv.Options) != 1 || adv.Options[0].Game.String != "lexicon" {
		t.Errorf("added page: %+v (a page that timed out isn't added)", c.Menus[1].Pages[1:])
	}
}

func TestReadCaptureRefusesOtherFormats(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capture.json")
	os.WriteFile(path, []byte(`{"format": 2, "menus": []}`), 0o644)
	if _, err := ReadCapture(path); err == nil {
		t.Error("read a capture in format 2")
	}
	os.WriteFile(path, []byte(`not json`), 0o644)
	if _, err := ReadCapture(path); err == nil {
		t.Error("read a file that isn't JSON")
	}
}

func TestServiceCapture(t *testing.T) {
	dir := t.TempDir()
	cat := filepath.Join(dir, "catalog.json")
	if err := WriteFile(cat, captureCatalog()); err != nil {
		t.Fatal(err)
	}
	_, path := readTestCapture(t)
	svc := NewService("test")
	r := httptest.NewRequest("GET", "/api/menus", nil)
	if err := svc.UseFile(cat); err != nil {
		t.Fatal(err)
	}
	if err := svc.UseCapture(path); err != nil {
		t.Fatal(err)
	}
	c, err := svc.Catalog(r)
	if err != nil || c.Capture == nil || len(c.Menus) != 2 {
		t.Fatalf("with the capture: %v, %v", c, err)
	}
	if again, _ := svc.Catalog(r); again != c {
		t.Error("the capture was added again for the same catalog")
	}
	svc.UseCapture("")
	if c, _ := svc.Catalog(r); c.Capture != nil || len(c.Menus) != 1 {
		t.Errorf("without the capture: %+v", c.Capture)
	}
	if err := svc.UseCapture(filepath.Join(dir, "missing.json")); err == nil {
		t.Error("used a capture file that isn't there")
	}
}

// A snapshot with a capture.json from its list's game: every menu SkyUI
// listed matches one the files give, and most options match one.
func TestRealCapture(t *testing.T) {
	for _, snap := range testcorpus.Snapshots(t) {
		path := filepath.Join(snap.Root, "capture.json")
		if _, err := os.Stat(path); err != nil {
			continue
		}
		t.Run(snap.Name, func(t *testing.T) {
			src, err := modlist.Check(snap.Dir, "")
			if err != nil {
				t.Fatal(err)
			}
			l, err := modlist.Read(src)
			if err != nil {
				t.Fatal(err)
			}
			f, err := ReadCapture(path)
			if err != nil {
				t.Fatal(err)
			}
			r := ApplyCapture(Scan(l, Options{}), f, path).Capture.Report
			t.Logf("capture: %s", r)
			if r.MenusMatched != r.Menus {
				t.Errorf("%d of %d captured menus matched", r.MenusMatched, r.Menus)
			}
			if matched := r.ByPosition + r.ByLabel + r.ByPattern; matched*100 < r.Options*75 {
				t.Errorf("%d of %d captured options matched", matched, r.Options)
			}
		})
	}
}
