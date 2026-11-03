package catalog

import (
	"encoding/binary"
	"encoding/json"
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
		"PhysicalCollisionVR.esp":                                testcorpus.Plugin(),
		"Scripts/PCVR_MCM.pex":                                   testcorpus.Compiled("PCVR_MCM", "MCM_ConfigBase"),
		"Collision Patch.esp":                                    testcorpus.Plugin(testcorpus.Quest{EditorID: "PCVR_MCMQuest", Script: "PCVR_MCM"}),
		"MCM/Config/PhysicalCollisionVR/config.json":             collision,
		"MCM/Config/PhysicalCollisionVR/settings.ini":            "[General]\nbEnabled = 1\nbDup1 = 0\n[Hands]\nfRadius = 4.0\niMode = 1\nuUnits = 0\n",
		"Interface/Translations/PhysicalCollisionVR_ENGLISH.txt": utf16LE("$PCVR_Name\tPhysical Collision VR\r\n$PCVR_Hands\tHands\r\n$PCVR_On\tOn\r\n"),
		"Interface/Translations/PhysicalCollisionVR_GERMAN.txt":  utf16LE("$PCVR_Name\tPhysische Kollision\r\n"),
	})
	inst.Mod("Prisma UI Additem Menu", true, map[string]string{
		"PrismaUIAddItemMenu.esp":                    testcorpus.Plugin(),
		"MCM/Config/PrismaUIAddItemMenu/config.json": `{"modName": "PrismaUIAddItemMenu", "displayName": "Prisma UI AddItem Menu", "content": [{"id": "Toggle:General", "text": "Toggle Hotkey", "type": "keymap", "valueOptions": {"sourceType": "GlobalValue", "defaultValue": 59}}]}`,
	})
	inst.Mod("Inactive Thing", true, map[string]string{
		"Inactive.esp":                    testcorpus.Plugin(),
		"MCM/Config/Inactive/config.json": `{"modName": "Inactive", "displayName": "Inactive Thing", "content": [{"id": "b:G", "text": "On", "type": "toggle", "valueOptions": {"sourceType": "ModSettingBool"}}]}`,
	})
	inst.Mod("Broken", true, map[string]string{"MCM/Config/Broken/config.json": "{ not json"})
	inst.Mod("Classic", true, map[string]string{
		"Scripts/Classic_MCM.pex":    testcorpus.Compiled("Classic_MCM", "SKI_ConfigBase"),
		"Classic.esp":                testcorpus.Plugin(testcorpus.Quest{EditorID: "ClassicMCMQuest", Script: "classic_mcm", ModName: "Classic Menu", Pages: []string{"General", "$CL_More"}}),
		"Broken.esp":                 "not a plugin",
		"Scripts/Helper_MCM.pex":     testcorpus.Compiled("Helper_MCM", "MCM_ConfigBase"),
		"Scripts/SKI_ConfigBase.pex": testcorpus.Compiled("SKI_ConfigBase", "Quest"),
		"Scripts/Unrelated.pex":      testcorpus.Compiled("Unrelated", "Quest"),
	})
	inst.Overwrite(map[string]string{"MCM/Settings/PhysicalCollisionVR.ini": "[Hands]\nfRadius = 6.5\niMode = 1\n[General]\nbEnabled = 1\n"})
	// A classic menu with its source: its name comes from the plugin, its
	// pages and options from the script.
	inst.Mod("Source Menu", true, map[string]string{
		"Scripts/Src_MCM.pex":                    testcorpus.Compiled("Src_MCM", "SKI_ConfigBase", "$SRC_Main", "Shiny", "Level ", "Makes it shiny."),
		"Source/Scripts/Src_MCM.psc":             "ScriptName Src_MCM extends SKI_ConfigBase\nint t\nEvent OnConfigInit()\n\tPages = new string[1]\n\tPages[0] = \"$SRC_Main\"\nEndEvent\nEvent OnPageReset(string page)\n\tif page == Pages[0]\n\t\tt = AddToggleOption(\"Shiny\", true)\n\t\tAddSliderOption(\"Level \", 3, \"{0}\")\n\tendif\nEndEvent\nEvent OnOptionHighlight(int o)\n\tif o == t\n\t\tSetInfoText(\"Makes it shiny.\")\n\tendif\nEndEvent\n",
		"Src.esp":                                testcorpus.Plugin(testcorpus.Quest{EditorID: "SrcQuest", Script: "Src_MCM", ModName: "$SRC_Name"}),
		"Interface/Translations/Src_ENGLISH.txt": utf16LE("$SRC_Name\tSource Menu\r\n$SRC_Main\tMain\r\n"),
		// A script built on SKI_ConfigBase that no quest runs, and one that
		// only mentions it, extending something else.
		"Scripts/Orphan_MCM.pex":        testcorpus.Compiled("Orphan_MCM", "SKI_ConfigBase"),
		"Scripts/Source/Orphan_MCM.psc": "ScriptName Orphan_MCM extends SKI_ConfigBase\n",
		"Scripts/Helper.pex":            testcorpus.Compiled("Helper", "Quest", "SKI_ConfigBase"),
		"Scripts/Source/Helper.psc":     "ScriptName Helper extends Quest\nSKI_ConfigBase Property Menu Auto\n",
	})
	inst.Plugins("*PhysicalCollisionVR.esp", "*Collision Patch.esp", "*PrismaUIAddItemMenu.esp", "Inactive.esp", "Classic.esp", "*Broken.esp", "*Src.esp")
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
	if want := "Classic Menu, Inactive Thing, Physical Collision VR, Prisma UI AddItem Menu, Source Menu"; strings.Join(names, ", ") != want {
		t.Errorf("menus = %q, want %q", names, want)
	}
	if c.List.Profile != "VR" || c.List.Language != "ENGLISH" || c.List.Version != "test" || c.SchemaVersion != SchemaVersion {
		t.Errorf("list = %+v", c.List)
	}

	pc := menu(t, c, "Physical Collision VR")
	if pc.Name != "$PCVR_Name" || pc.ID != "mcmhelper-physicalcollisionvr" || pc.Kind != KindMCMHelper || pc.Mod != "Physical Collision VR" || pc.Plugin != "Collision Patch.esp" || !pc.Active || pc.Certainty != CertaintyExact || len(pc.Notes) != 0 {
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
	if in := menu(t, c, "Inactive Thing"); in.Active || in.Plugin != "Inactive.esp" || len(in.Notes) != 1 || in.Notes[0].Kind != NoteKindPluginGuessed {
		t.Errorf("a menu whose plugin is inactive, found by its name: %+v", in)
	}

	// What couldn't be read: the broken config, and the classic script.
	var skipped []string
	for _, s := range c.Skipped {
		skipped = append(skipped, string(s.Reason)+" "+s.Path)
	}
	if want := "unreadable Broken.esp, unreadable MCM/Config/Broken/config.json, no-quest Scripts/Orphan_MCM.pex"; strings.Join(skipped, ", ") != want {
		t.Errorf("skipped = %q, want %q", skipped, want)
	}
	// A menu with only its compiled script: its name and pages from its
	// quest, matched to the script without regard to case.
	if cl := menu(t, c, "Classic Menu"); cl.Kind != KindCompiled || cl.Plugin != "Classic.esp" || cl.Active || len(cl.Pages) != 2 || cl.Pages[1].Name != "$CL_More" || cl.Source.Path != "Scripts/Classic_MCM.pex" {
		t.Errorf("the compiled menu = %+v", cl)
	}
	if sm := menu(t, c, "Source Menu"); sm.Kind != KindSource || sm.Source.Path != "Source/Scripts/Src_MCM.psc" {
		t.Errorf("a menu whose source matches its compiled script is read from source: %+v", sm)
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
	if err != nil || len(c.Menus) != 5 {
		t.Fatalf("after scanning: %v, %v", c, err)
	}
	if st := svc.State(); st.Scanning || st.Menus != 5 || st.ScannedAt.IsZero() {
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
			named := 0
			for _, s := range c.Skipped {
				if s.Reason == SkipReasonUnreadable {
					t.Errorf("couldn't read %s (%s): %s", s.Path, s.Mod, s.Detail)
				}
				if s.Name != "" {
					named++
				}
				if s.Reason == SkipReasonNoQuest {
					t.Logf("not a menu (no quest runs it): %s (%s)", s.Path, s.Mod)
				}
			}
			t.Logf("%d of %d skipped scripts named by their quests", named, len(c.Skipped))
			// What the source menus hold, and what isn't exact, and why.
			source, compiled, options, inexact := 0, 0, 0, 0
			for _, m := range c.Menus {
				switch m.Kind {
				case KindSource:
					source++
				case KindCompiled:
					compiled++
				default:
					continue
				}
				for _, n := range m.Notes {
					t.Logf("%s: note %s %s", m.NameText, n.Kind, n.Detail)
				}
				for _, p := range m.Pages {
					options += len(p.Options)
					for _, o := range p.Options {
						if o.Certainty != CertaintyExact {
							inexact++
							t.Logf("%s / %s: %s %s %q (line %d)", m.NameText, p.Text, o.Certainty, o.Type, o.Text, o.Source.Line)
						}
					}
				}
			}
			t.Logf("%d menus read from source and %d from compiled scripts: %d options, %d not exact", source, compiled, options, inexact)
			checkRecordings(t, l, c)
			for _, m := range c.Menus {
				for _, n := range m.Notes {
					if n.Kind == NoteKindPluginGuessed {
						t.Logf("%s: plugin guessed as %s", m.NameText, n.Detail)
					}
				}
			}
			t.Logf("%d MCM Helper menus, %d Papyrus menus not read yet", helper, len(c.Skipped))
			if helper == 0 {
				t.Fatal("no MCM Helper menus")
			}
			if !strings.Contains(snap.Name, "mgo") {
				return
			}
			// The golden cases of PLAN.md §7 that read from source.
			golden := []struct{ mod, page, typ, label, choice string }{
				{"Dirt & Blood", "Settings", "menu", "Dirt on Player", "Deactivated"},
				{"Dirt & Blood", "Settings", "toggle", "Dirt on NPCs Based on Profession", ""},
				{"Dirt & Blood", "Settings", "toggle", "Dirt on Bandits", ""},
				{"EVG Conditional Idles", "Toggle - NPCs", "toggle", "Cold", ""},
				{"EVG Conditional Idles", "Toggle - NPCs", "toggle", "Modesty Male", ""},
				{"EVG Conditional Idles", "Toggle - NPCs", "toggle", "Modesty Female", ""},
				{"EVG Conditional Idles", "Toggle - NPCs", "toggle", "Injured", ""},
				{"Experience", "$EXP_General", "toggle", "$EXP_ShowMessages", ""},
				// From compiled scripts.
				{"Be Seated", "", "toggle", "Enable / Disable Sitting Prompts", ""},
				{"Sofia Follower", "Sofia Settings", "slider", "Sofia Health: ", ""},
				{"Sofia Follower", "Sofia Settings", "slider", "Sofia Stamina: ", ""},
				{"Sofia Follower", "Sofia Settings", "slider", "Sofia Magicka: ", ""},
				{"Sofia Follower", "Sofia Settings", "toggle", "Disable Comments Dialogue", ""},
			}
			// Nether's Follower Framework, a large compiled MCM, reads whole.
			nff := 0
			for _, m := range c.Menus {
				if m.Kind == KindCompiled && strings.Contains(m.Mod, "Nether") {
					for _, p := range m.Pages {
						nff += len(p.Options)
					}
					t.Logf("Nether's Follower Framework: %s, %d pages", m.NameText, len(m.Pages))
				}
			}
			if nff < 100 {
				t.Errorf("Nether's Follower Framework has %d options", nff)
			}
			for _, g := range golden {
				o, ok := find(c, g.mod, g.page, OptionType(g.typ), g.label, 0)
				if !ok || o.Certainty != CertaintyExact {
					t.Errorf("%s / %s / %s %q: found %v, %+v", g.mod, g.page, g.typ, g.label, ok, o)
					continue
				}
				if g.choice != "" && !hasChoice(o, g.choice) {
					t.Errorf("%s / %q: no choice %q in %+v", g.mod, g.label, g.choice, o.Choices)
				}
			}
			if o, _ := find(c, "Experience", "$EXP_General", OptionTypeToggle, "$EXP_ShowMessages", 0); o.Text != "Show Messages" {
				t.Errorf("Experience's toggle reads %q", o.Text)
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

// find is the option a recording step names: by its menu's name, page,
// type, label, and index, as MCM Recorder matches them.
func find(c *Catalog, mod, page string, typ OptionType, label string, index int) (Option, bool) {
	for _, m := range c.Menus {
		if m.Name != mod {
			continue
		}
		for _, p := range m.Pages {
			if p.Name != page {
				continue
			}
			for _, o := range p.Options {
				if o.Type == typ && o.Label == label && (o.Index == index || (index == 0 && o.Index == 1)) {
					return o, true
				}
			}
		}
	}
	return Option{}, false
}

// hasChoice compares as Papyrus does, without regard to case: R.A.S.S.'s
// recording in Nordic chooses "off" from a menu that offers "Off".
func hasChoice(o Option, raw string) bool {
	for _, ch := range o.Choices {
		if strings.EqualFold(ch.Raw, raw) {
			return true
		}
	}
	return false
}

// checkRecordings resolves every step of every recording in the list
// against the catalog: a step for a menu read from source must find an
// exact option, and a menu choice must be one of its entries.
func checkRecordings(t *testing.T, l *modlist.List, c *Catalog) {
	t.Helper()
	kinds := map[string]Kind{}
	for _, m := range c.Menus {
		kinds[m.Name] = m.Kind
	}
	types := map[string]OptionType{"toggle": OptionTypeToggle, "slider": OptionTypeSlider, "choose": OptionTypeMenu, "shortcut": OptionTypeKeymap, "color": OptionTypeColor, "text": OptionTypeInput, "click": OptionTypeText}
	checked, resolved := 0, 0
	for _, f := range l.Files("McmRecorder") {
		if strings.Count(f.Path, "/") != 2 || !strings.EqualFold(filepath.Ext(f.Path), ".json") {
			continue
		}
		b, _ := os.ReadFile(f.Abs)
		var steps []map[string]any
		if json.Unmarshal(b, &steps) != nil {
			continue
		}
		for _, raw := range steps {
			st := map[string]any{}
			for k, v := range raw {
				st[strings.ToLower(k)] = v
			}
			mod, _ := st["mod"].(string)
			if kinds[mod] != KindSource && kinds[mod] != KindCompiled {
				continue
			}
			page, _ := st["page"].(string)
			label, _ := st["option"].(string)
			index := 0
			if n, ok := st["index"].(float64); ok {
				index = int(n)
			}
			var typ OptionType
			var choice string
			for k, ty := range types {
				if v, ok := st[k]; ok {
					typ = ty
					if k == "choose" {
						choice, _ = v.(string)
					}
				}
			}
			checked++
			o, ok := find(c, mod, page, typ, label, index)
			switch {
			case !ok:
				t.Errorf("%s: %s / %q / %s %q (index %d) isn't in the catalog", f.Path, mod, page, typ, label, index)
			case o.Certainty != CertaintyExact:
				t.Errorf("%s: %s / %q is %s", f.Path, mod, label, o.Certainty)
			case choice != "" && len(o.Choices) > 0 && !hasChoice(o, choice):
				t.Errorf("%s: %s / %q has no choice %q", f.Path, mod, label, choice)
			default:
				resolved++
			}
		}
	}
	t.Logf("%d of %d recorded steps for Papyrus menus resolved exactly", resolved, checked)
}

// Right-side clicks count the text options showing the same value.
func TestValueIndex(t *testing.T) {
	opts := []Option{
		{Type: OptionTypeText, Value: "Run"},
		{Type: OptionTypeText, Label: "Other", Value: "run"},
		{Type: OptionTypeText, Value: "Stop"},
	}
	numberDuplicates(opts)
	if opts[0].ValueIndex != 1 || opts[1].ValueIndex != 2 || opts[2].ValueIndex != 0 {
		t.Errorf("value indexes = %d, %d, %d", opts[0].ValueIndex, opts[1].ValueIndex, opts[2].ValueIndex)
	}
}
