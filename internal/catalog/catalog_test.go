package catalog

import (
	"encoding/binary"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"lexicon/internal/modlist"
	"lexicon/internal/testcorpus"
)

func utf16LE(s string) string {
	u := utf16.Encode([]rune(s))
	b := make([]byte, 2+2*len(u))
	b[0], b[1] = 0xFF, 0xFE
	for i, c := range u {
		binary.LittleEndian.PutUint16(b[2+2*i:], c)
	}
	return string(b)
}

const collision = `{
	"modName": "PhysicalCollisionVR",
	"displayName": "$PCVR_Name",
	"pages": [
		{"pageDisplayName": "Main", "content": [
			{"text": "General", "type": "header"},
			{"id": "bEnabled:General", "text": "Enable", "type": "toggle", "valueOptions": {"sourceType": "ModSettingBool"}},
			{"id": "fRadius:Hands", "text": "Radius ", "type": "slider", "valueOptions": {"sourceType": "ModSettingFloat", "min": 1, "max": 10, "step": 0.5}},
			{"id": "bDup1:General", "text": "Same", "type": "toggle", "valueOptions": {"sourceType": "ModSettingBool"}},
			{"id": "bDup2:General", "text": "Same", "type": "toggle", "valueOptions": {"sourceType": "ModSettingBool"}},
			{"type": "empty"}
		]},
		{"pageDisplayName": "$PCVR_Hands", "content": [
			{"id": "iMode:Hands", "text": "Mode", "type": "enum", "valueOptions": {"sourceType": "ModSettingInt", "options": ["Off", "$PCVR_On"]}},
			{"id": "uUnits:Hands", "text": "Units", "type": "stepper", "valueOptions": {"sourceType": "ModSettingInt", "options": ["Imperial", "Metric"]}},
			{"id": "iTint:Hands", "text": "Tint", "type": "color", "valueOptions": {"sourceType": "ModSettingInt", "defaultValue": 8323582}},
			{"text": "Apply", "type": "text", "action": {"type": "CallFunction", "function": "Go"}},
			{"type": "hiddenToggle", "valueOptions": {"sourceType": "PropertyValueBool"}}
		]}
	]
}`

func build(t *testing.T) *modlist.List {
	t.Helper()
	inst := testcorpus.New(t, "[NoDelete] MGO", "VR")
	inst.Mod("Physical Collision VR", true, map[string]string{
		"PhysicalCollisionVR.esp":                                "plugin",
		"MCM/Config/PhysicalCollisionVR/config.json":             collision,
		"MCM/Config/PhysicalCollisionVR/settings.ini":            "[General]\nbEnabled = 1\nbDup1 = 0\n[Hands]\nfRadius = 4.0\niMode = 1\nuUnits = 0\n",
		"Interface/Translations/PhysicalCollisionVR_ENGLISH.txt": utf16LE("$PCVR_Name\tPhysical Collision VR\r\n$PCVR_Hands\tHands\r\n$PCVR_On\tOn\r\n"),
		"Interface/Translations/PhysicalCollisionVR_GERMAN.txt":  utf16LE("$PCVR_Name\tPhysische Kollision\r\n"),
	})
	inst.Mod("Prisma UI Additem Menu", true, map[string]string{
		"PrismaUIAddItemMenu.esp":                    "plugin",
		"MCM/Config/PrismaUIAddItemMenu/config.json": `{"modName": "PrismaUIAddItemMenu", "displayName": "Prisma UI AddItem Menu", "content": [{"id": "Toggle:General", "text": "Toggle Hotkey", "type": "keymap", "valueOptions": {"sourceType": "GlobalValue", "defaultValue": 59}}]}`,
	})
	inst.Mod("Inactive Thing", true, map[string]string{
		"Inactive.esp":                    "plugin",
		"MCM/Config/Inactive/config.json": `{"modName": "Inactive", "displayName": "Inactive Thing", "content": [{"id": "b:G", "text": "On", "type": "toggle", "valueOptions": {"sourceType": "ModSettingBool"}}]}`,
	})
	inst.Mod("Broken", true, map[string]string{"MCM/Config/Broken/config.json": "{ not json"})
	inst.Mod("Classic", true, map[string]string{
		"Scripts/Classic_MCM.pex":    "\xfa\x57\xc0\xde Classic_MCM SKI_ConfigBase OnPageReset",
		"Scripts/Helper_MCM.pex":     "\xfa\x57\xc0\xde Helper_MCM MCM_ConfigBase SKI_ConfigBase",
		"Scripts/SKI_ConfigBase.pex": "SKI_ConfigBase",
		"Scripts/Unrelated.pex":      "nothing here",
	})
	inst.Overwrite(map[string]string{"MCM/Settings/PhysicalCollisionVR.ini": "[Hands]\nfRadius = 6.5\niMode = 1\n[General]\nbEnabled = 1\n"})
	inst.Plugins("*PhysicalCollisionVR.esp", "*PrismaUIAddItemMenu.esp", "Inactive.esp")
	src, err := modlist.Check(inst.Dir, "")
	if err != nil {
		t.Fatal(err)
	}
	l, err := modlist.Read(src)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func menu(t *testing.T, c *Catalog, text string) Menu {
	t.Helper()
	for _, m := range c.Menus {
		if m.NameText == text {
			return m
		}
	}
	t.Fatalf("no menu %q in %d menus", text, len(c.Menus))
	return Menu{}
}

func TestScanReadsMCMHelperMenus(t *testing.T) {
	c := Scan(build(t), Options{Version: "test", Now: time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)})
	var names []string
	for _, m := range c.Menus {
		names = append(names, m.NameText)
	}
	if want := "Inactive Thing, Physical Collision VR, Prisma UI AddItem Menu"; strings.Join(names, ", ") != want {
		t.Errorf("menus = %q, want %q", names, want)
	}
	if c.List.Profile != "VR" || c.List.Language != "ENGLISH" || c.List.Version != "test" || c.SchemaVersion != SchemaVersion {
		t.Errorf("list = %+v", c.List)
	}

	pc := menu(t, c, "Physical Collision VR")
	if pc.Name != "$PCVR_Name" || pc.ID != "mcmhelper-physicalcollisionvr" || pc.Kind != KindMCMHelper || pc.Mod != "Physical Collision VR" || pc.Plugin != "PhysicalCollisionVR.esp" || !pc.Active || pc.Certainty != CertaintyExact {
		t.Errorf("menu = %+v", pc)
	}
	if pc.Source.Path != "MCM/Config/PhysicalCollisionVR/config.json" {
		t.Errorf("source = %+v", pc.Source)
	}
	if len(pc.Pages) != 2 || pc.Pages[0].Name != "Main" || pc.Pages[1].Name != "$PCVR_Hands" || pc.Pages[1].Text != "Hands" {
		t.Fatalf("pages = %+v", pc.Pages)
	}

	main := pc.Pages[0].Options
	if len(main) != 4 {
		t.Fatalf("Main has %d options, want 4 (headers and spacers aren't options): %+v", len(main), main)
	}
	if e := main[0]; e.Type != OptionTypeToggle || e.Label != "Enable" || e.Index != 0 || e.Default == nil || !e.Default.Bool || e.Saved == nil || !e.Saved.Bool || e.Setting != "bEnabled:General" {
		t.Errorf("Enable = %+v", e)
	}
	r := main[1]
	if r.Type != OptionTypeSlider || r.Label != "Radius " || r.Slider == nil || r.Slider.Min != 1 || r.Slider.Max != 10 || r.Slider.Step != 0.5 {
		t.Errorf("Radius = %+v", r)
	}
	if r.Default == nil || r.Default.Number != 4 || r.Saved == nil || r.Saved.Number != 6.5 {
		t.Errorf("Radius default %+v, saved %+v", r.Default, r.Saved)
	}
	if main[2].Index != 1 || main[3].Index != 2 {
		t.Errorf("duplicate labels numbered %d and %d, want 1 and 2", main[2].Index, main[3].Index)
	}

	hands := pc.Pages[1].Options
	if len(hands) != 4 {
		t.Fatalf("Hands has %d options: %+v", len(hands), hands)
	}
	if m := hands[0]; m.Type != OptionTypeMenu || m.Control != "enum" || len(m.Choices) != 2 || m.Choices[1] != (Choice{Raw: "$PCVR_On", Text: "On"}) || m.Default.Number != 1 || m.Saved.Number != 1 {
		t.Errorf("Mode = %+v", m)
	}
	if s := hands[1]; s.Type != OptionTypeText || s.Control != "stepper" || len(s.Choices) != 2 || len(s.Notes) != 1 || s.Notes[0].Kind != NoteKindStepper {
		t.Errorf("Units = %+v", s)
	}
	if col := hands[2]; col.Type != OptionTypeColor || col.Default == nil || col.Default.Number != 8323582 || col.Saved != nil {
		t.Errorf("Tint = %+v", col)
	}
	if a := hands[3]; a.Type != OptionTypeText || a.Label != "Apply" || a.Default != nil {
		t.Errorf("Apply = %+v", a)
	}

	pr := menu(t, c, "Prisma UI AddItem Menu")
	if len(pr.Pages) != 1 || pr.Pages[0].Name != "" {
		t.Fatalf("Prisma's pages = %+v", pr.Pages)
	}
	if k := pr.Pages[0].Options[0]; k.Type != OptionTypeKeymap || k.Label != "Toggle Hotkey" || k.Default == nil || k.Default.Number != 59 {
		t.Errorf("Toggle Hotkey = %+v", k)
	}
	if in := menu(t, c, "Inactive Thing"); in.Active {
		t.Error("a menu whose plugin is inactive is marked active")
	}

	// What couldn't be read: the broken config, and the classic script.
	var skipped []string
	for _, s := range c.Skipped {
		skipped = append(skipped, string(s.Reason)+" "+s.Path)
	}
	if want := "unreadable MCM/Config/Broken/config.json, papyrus Scripts/Classic_MCM.pex"; strings.Join(skipped, ", ") != want {
		t.Errorf("skipped = %q, want %q", skipped, want)
	}
}

func TestOtherLanguage(t *testing.T) {
	c := Scan(build(t), Options{Language: "german"})
	if m := menu(t, c, "Physische Kollision"); m.Pages[1].Text != "$PCVR_Hands" {
		t.Errorf("a key the German file lacks should stay raw: %q", m.Pages[1].Text)
	}
}

func TestFileRoundTrip(t *testing.T) {
	c := Scan(build(t), Options{})
	path := filepath.Join(t.TempDir(), "mgo.json")
	if err := WriteFile(path, c); err != nil {
		t.Fatal(err)
	}
	back, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := Marshal(c)
	b, _ := Marshal(back)
	if string(a) != string(b) {
		t.Error("a catalog read back differs from the one written")
	}
	os.WriteFile(path, []byte(`{"schemaVersion": 99, "menus": []}`), 0o644)
	if _, err := ReadFile(path); err == nil || !strings.Contains(err.Error(), "newer") {
		t.Errorf("a newer catalog: %v", err)
	}
	os.WriteFile(path, []byte(`{"menus": []}`), 0o644)
	if _, err := ReadFile(path); err == nil {
		t.Error("a file without a schema version was read as a catalog")
	}
	if err := WriteFile(filepath.Join(t.TempDir(), "mgo.txt"), c); err == nil {
		t.Error("a catalog was written to a file not named .json")
	}
}

func TestSearch(t *testing.T) {
	c := Scan(build(t), Options{})
	results, total := Search(c, "  RADIUS  ", 10)
	if total != 1 || len(results) != 1 || results[0].Option.Label != "Radius " || results[0].MenuID != "mcmhelper-physicalcollisionvr" || results[0].Page != 0 || results[0].Position != 1 {
		t.Errorf("radius: %d %+v", total, results)
	}
	// Every word must match, in the option, its page, or its menu.
	if _, total := Search(c, "hands mode", 10); total != 1 {
		t.Errorf("hands mode: %d results", total)
	}
	if results, total := Search(c, "physical", 2); total != 8 || len(results) != 2 {
		t.Errorf("physical: %d results, %d returned", total, len(results))
	}
	if results, total := Search(c, "   ", 10); total != 0 || len(results) != 0 {
		t.Errorf("an empty search found %d", total)
	}
}

func TestSummarize(t *testing.T) {
	c := Scan(build(t), Options{})
	s := Summarize(menu(t, c, "Physical Collision VR"))
	if s.Pages != 2 || s.Options != 8 || s.Uncertain != 0 {
		t.Errorf("summary = %+v", s)
	}
	if s := Summarize(menu(t, c, "Prisma UI AddItem Menu")); s.Pages != 0 || s.Options != 1 {
		t.Errorf("a menu without pages: %+v", s)
	}
}

func TestService(t *testing.T) {
	l := build(t)
	svc := NewService("test")
	r := httptest.NewRequest("GET", "/api/menus", nil)
	if _, err := svc.Catalog(r); err == nil {
		t.Error("a catalog before anything was chosen")
	}
	svc.UseList(l.Source)
	c, err := svc.Catalog(r) // waits for the scan
	if err != nil || len(c.Menus) != 3 {
		t.Fatalf("after scanning: %v, %v", c, err)
	}
	if st := svc.State(); st.Scanning || st.Menus != 3 || st.ScannedAt.IsZero() {
		t.Errorf("state = %+v", st)
	}
	path := filepath.Join(t.TempDir(), "other.json")
	c.Menus = c.Menus[:1]
	if err := WriteFile(path, c); err != nil {
		t.Fatal(err)
	}
	if err := svc.UseFile(path); err != nil {
		t.Fatal(err)
	}
	if c, _ := svc.Catalog(r); len(c.Menus) != 1 {
		t.Errorf("the catalog file wasn't used: %d menus", len(c.Menus))
	}
	svc.Clear()
	if _, err := svc.Catalog(r); err == nil {
		t.Error("a catalog after clearing")
	}
}

// The MCM Helper checks from PLAN.md §7, against the real lists.
func TestRealSnapshots(t *testing.T) {
	for _, snap := range testcorpus.Snapshots(t) {
		t.Run(snap.Name, func(t *testing.T) {
			src, err := modlist.Check(snap.Dir, "")
			if err != nil {
				t.Fatal(err)
			}
			l, err := modlist.Read(src)
			if err != nil {
				t.Fatal(err)
			}
			c := Scan(l, Options{})
			helper := 0
			for _, m := range c.Menus {
				if m.Kind == KindMCMHelper {
					helper++
				}
				for _, p := range m.Pages {
					for _, o := range p.Options {
						// Real menus have these (SkyUI's favorite group is
						// an enum with no label); worth knowing, not wrong.
						if o.Label == "" && o.Type != OptionTypeText {
							t.Logf("%s / %s: a %s without a label", m.NameText, p.Text, o.Type)
						}
					}
				}
			}
			for _, s := range c.Skipped {
				if s.Reason == SkipReasonUnreadable {
					t.Errorf("couldn't read %s (%s): %s", s.Path, s.Mod, s.Detail)
				}
			}
			t.Logf("%d MCM Helper menus, %d Papyrus menus not read yet", helper, len(c.Skipped))
			if helper == 0 {
				t.Fatal("no MCM Helper menus")
			}
			if !strings.Contains(snap.Name, "mgo") {
				return
			}
			pc := menu(t, c, "Physical Collision VR")
			var pages []string
			for _, p := range pc.Pages {
				pages = append(pages, p.Text)
			}
			if want := "Main, Hands, Weapons, Weapon vs weapon, What has collision, Enemy weapons, Parry and block, Arrows and spells, Your body, Sparks and feel, Beta features, Excluded items"; strings.Join(pages, ", ") != want {
				t.Errorf("Physical Collision VR's pages = %q", pages)
			}
			pr := menu(t, c, "Prisma UI AddItem Menu")
			found := false
			for _, o := range pr.Pages[0].Options {
				if o.Label == "Toggle Hotkey" {
					found = true
					if o.Type != OptionTypeKeymap || o.Default == nil || o.Default.Number != 59 {
						t.Errorf("Toggle Hotkey = %+v", o)
					}
				}
			}
			if !found {
				t.Error("Prisma UI AddItem Menu has no Toggle Hotkey")
			}
		})
	}
}
