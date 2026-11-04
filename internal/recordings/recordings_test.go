package recordings

import (
	"fmt"
	"strings"
	"testing"

	"lexicon/internal/catalog"
	"lexicon/internal/modlist"
	"lexicon/internal/testcorpus"
)

func list(t *testing.T, inst *testcorpus.Instance) *modlist.List {
	t.Helper()
	src, err := modlist.Check(inst.Dir, "")
	if err != nil {
		t.Fatal(err)
	}
	l, err := modlist.Read(src)
	if err != nil {
		t.Fatal(err)
	}
	l.Index(modlist.MCMScope)
	return l
}

func TestRead(t *testing.T) {
	inst := testcorpus.New(t, "mgo", "VR")
	inst.Mod("Old Recordings", true, map[string]string{
		"McmRecorder/Mine.json": `{"name": "Old", "autorun": "false"}`,
	})
	inst.Mod("[NoDelete] Davey's MCM Recordings", true, map[string]string{
		// MGO's own: Mod in capitals, autorun "True", two files numbered 0001.
		"McmRecorder/Mine.json":                   "\ufeff" + `{"author": "Me", "autorun": "True", "name": "My Settings", "Version": "1.0.0"}`,
		"McmRecorder/Mine/0001_VRIK.json":         `[{"Mod": "VRIK", "option": "Head Bobbing Height", "slider": 0.0}]`,
		"McmRecorder/Mine/0001_Be Seated.json":    `[{"mod": "Be Seated", "option": "Enable / Disable Sitting Prompts", "toggle": "off"}]`,
		"McmRecorder/Mine/0002_Dirt & Blood.json": `[{"mod": "Dirt & Blood", "page": "Settings", "option": "Dirt on Player", "choose": "Deactivated"}, {"mod": "Dirt & Blood", "page": "Settings", "click": "Restart Player Effects", "side": "left"}, {"mod": "Dirt & Blood", "page": "Settings", "option": "*Bandits*", "toggle": "on", "index": 2}]`,
		"McmRecorder/Broken.json":                 `{"name": "Broken"}`,
		"McmRecorder/Broken/0001_X.json":          `{"not": "an array"}`,
		"McmRecorder/Broken/0002_Y.json":          `[{"mod": "Y", "option": "Nothing to do"}]`,
		"McmRecorder/Loose/0001_Z.json":           `[]`,
		"McmRecorder/.debug/dump.json":            `{}`,
	})
	recs := Read(list(t, inst))
	var names []string
	for _, r := range recs {
		names = append(names, r.Name)
	}
	if strings.Join(names, ",") != "Broken,Loose,Mine" {
		t.Fatalf("recordings = %q", names)
	}
	mine := recs[2]
	if mine.Title != "My Settings" || mine.Author != "Me" || mine.Version != "1.0.0" || !mine.Autorun || mine.Mod != "[NoDelete] Davey's MCM Recordings" {
		t.Errorf("info = %+v", mine)
	}
	if len(mine.Losing) != 1 || mine.Losing[0] != "Old Recordings" {
		t.Errorf("losing copies = %q", mine.Losing)
	}
	var files []string
	for _, f := range mine.Files {
		files = append(files, f.Name)
	}
	if strings.Join(files, ",") != "0001_Be Seated.json,0001_VRIK.json,0002_Dirt & Blood.json" {
		t.Errorf("files in replay order = %q", files)
	}
	if len(mine.Steps) != 5 {
		t.Fatalf("steps = %+v", mine.Steps)
	}
	if s := mine.Steps[1]; s.Mod != "VRIK" || s.Kind != KindSlider || s.Value != "0.0" || s.File != 1 {
		t.Errorf("a slider keeps its spelling: %+v", s)
	}
	if s := mine.Steps[3]; s.Kind != KindClick || s.Option != "Restart Player Effects" || s.Side != "left" {
		t.Errorf("a click's selector is what it clicks: %+v", s)
	}
	if s := mine.Steps[4]; !s.Wildcard || s.Index != 2 {
		t.Errorf("a wildcard with an index: %+v", s)
	}
	broken := recs[0]
	var problems []string
	for _, p := range broken.Problems {
		problems = append(problems, string(p.Kind)+" "+p.Path)
	}
	if strings.Join(problems, ",") != "unreadable McmRecorder/Broken/0001_X.json,no-kind McmRecorder/Broken/0002_Y.json" {
		t.Errorf("problems = %q", problems)
	}
	if p := recs[1].Problems; len(p) != 1 || p[0].Kind != ProblemNoInfo {
		t.Errorf("a folder without an info file: %+v", p)
	}
}

func cat() *catalog.Catalog {
	return &catalog.Catalog{Menus: []catalog.Menu{
		{ID: "dirt", Name: "Dirt & Blood", NameText: "Dirt & Blood", Active: true, Certainty: catalog.CertaintyExact, Pages: []catalog.Page{{Name: "Settings", Text: "Settings", Options: []catalog.Option{
			{Type: catalog.OptionTypeMenu, Label: "Dirt on Player", Choices: []catalog.Choice{{Raw: "Deactivated"}, {Raw: "Often"}}},
			{Type: catalog.OptionTypeToggle, Label: "Dirt on Bandits"},
			{Type: catalog.OptionTypeToggle, Label: "Bandits too", Index: 0},
			{Type: catalog.OptionTypeSlider, Label: "Chance", Slider: &catalog.Range{Min: 0, Max: 100}},
			{Type: catalog.OptionTypeText, Label: "Restart Player Effects"},
		}}}},
		{ID: "off", Name: "Turned Off", Active: false, Certainty: catalog.CertaintyExact, Pages: []catalog.Page{{}}},
		{ID: "vague", Name: "Vague", Active: true, Certainty: catalog.CertaintyPattern, Pages: []catalog.Page{{Name: "", Options: []catalog.Option{{Type: catalog.OptionTypeToggle, Label: "*Slot*", Certainty: catalog.CertaintyPattern}}}}},
		{ID: "other", Name: "Other", Active: true, Certainty: catalog.CertaintyExact, Pages: []catalog.Page{
			{Name: "A", Options: []catalog.Option{{Type: catalog.OptionTypeToggle, Label: "Moved"}}},
			{Name: "B", Options: []catalog.Option{{Type: catalog.OptionTypeToggle, Label: "Here"}}},
		}},
	}}
}

func TestValidate(t *testing.T) {
	steps := []Step{
		{Mod: "Dirt & Blood", Page: "Settings", Option: "Dirt on Player", Kind: KindChoose, Value: "Deactivated"},
		{Mod: "Dirt & Blood", Page: "Settings", Option: "Dirt on Player", Kind: KindChoose, Value: "Never"},
		{Mod: "Dirt & Blood", Page: "Settings", Option: "Chance", Kind: KindSlider, Value: "150.0"},
		{Mod: "Dirt & Blood", Page: "Settings", Option: "*Bandits*", Kind: KindToggle, Value: "on", Index: 2, Wildcard: true},
		{Mod: "Dirt & Blood", Page: "Settings", Option: "Restart Player Effects", Kind: KindClick, Value: "Restart Player Effects"},
		{Mod: "dirt & blood", Page: "settings", Option: "dirt on player", Kind: KindChoose, Value: "deactivated"},
		{Mod: "Dirt & Blood", Page: "Settings", Option: "Dirt On Bandits ", Kind: KindToggle, Value: "on"},
		{Mod: "Dirt & Blood", Page: "Settings", Option: "Dirt on Bandits", Kind: KindToggle, Value: "on", Index: 3},
		{Mod: "Dirt & Blood", Page: "Gone", Option: "Dirt on Bandits", Kind: KindToggle, Value: "on"},
		{Mod: "Not Installed", Option: "x", Kind: KindToggle},
		{Mod: "Turned Off", Option: "x", Kind: KindToggle},
		{Mod: "Vague", Option: "Slot 3", Kind: KindToggle},
		{Mod: "Other", Page: "B", Option: "Moved", Kind: KindToggle},
	}
	checks := Validate(&Recording{Steps: steps}, cat())
	want := []string{"matched", "not-allowed choice", "not-allowed range", "matched", "matched", "matched", "not-found option", "not-found index", "not-found page", "mod-missing", "mod-missing inactive", "unchecked option", "not-found option"}
	for i, ch := range checks {
		got := string(ch.Status)
		if ch.Why != "" {
			got += " " + string(ch.Why)
		}
		if got != want[i] {
			t.Errorf("step %d (%s): %s, want %s", i+1, steps[i].Option, got, want[i])
		}
	}
	if o := checks[3].Option; o == nil || o.Label != "Bandits too" {
		t.Errorf("a wildcard's second match: %+v", o)
	}
	if s := checks[6].Suggestion; s == nil || s.Label != "Dirt on Bandits" {
		t.Errorf("a near miss suggests the label: %+v", s)
	}
	if s := checks[8].Suggestion; s == nil || s.Page != "Settings" {
		t.Errorf("a missing page suggests where the option is: %+v", s)
	}
	if s := checks[12].Suggestion; s == nil || s.Page != "A" || s.Label != "Moved" {
		t.Errorf("an option on another page: %+v", s)
	}
	if c := Count(checks); c.Matched != 4 || c.NotAllowed != 2 || c.NotFound != 4 || c.ModMissing != 2 || c.Unchecked != 1 {
		t.Errorf("counts = %+v", c)
	}
}

// The generic acceptance check of PLAN.md §7: every step of every recording
// in a real list resolves exactly, or its mod isn't in the list.
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
			c := catalog.Scan(l, catalog.Options{})
			recs := Read(l)
			total := 0
			var counts Counts
			for _, r := range recs {
				for _, p := range r.Problems {
					t.Errorf("%s: %s %s %s", r.Name, p.Kind, p.Path, p.Detail)
				}
				checks := Validate(r, c)
				for i, ch := range checks {
					s := r.Steps[i]
					switch ch.Status {
					case StatusMatched:
					case StatusModMissing:
						t.Logf("%s: %q isn't in this list (%s)", r.Name, s.Mod, ch.Why)
					default:
						t.Errorf("%s step %d: %s / %q / %s %q: %s %s", r.Name, i+1, s.Mod, s.Page, s.Kind, s.Option, ch.Status, ch.Why)
					}
				}
				c := Count(checks)
				counts.Matched += c.Matched
				counts.ModMissing += c.ModMissing
				total += len(r.Steps)
			}
			t.Logf("%d recordings, %d steps: %d matched, %d for mods not in the list", len(recs), total, counts.Matched, counts.ModMissing)
		})
	}
}

// A right-side click matches a text option by its value.
func TestRightSideClicks(t *testing.T) {
	fixed := func(label, value string) catalog.Option {
		return catalog.Option{Type: catalog.OptionTypeText, Label: label, Value: value, ValueCertainty: catalog.CertaintyExact}
	}
	c := &catalog.Catalog{Menus: []catalog.Menu{
		{ID: "fixed", Name: "Fixed", Active: true, Certainty: catalog.CertaintyExact, Pages: []catalog.Page{{Name: "P", Options: []catalog.Option{
			fixed("", "Reset everything"), fixed("", "Run"), fixed("Other", "Run"),
		}}}},
		{ID: "moving", Name: "Moving", Active: true, Certainty: catalog.CertaintyExact, Pages: []catalog.Page{{Name: "P", Options: []catalog.Option{
			{Type: catalog.OptionTypeText, Label: "Status", Value: "Enabled", ValueCertainty: catalog.CertaintyExact, ValueChanges: true},
		}}}},
	}}
	click := func(mod, value string, index int) Step {
		return Step{Mod: mod, Page: "P", Option: value, Kind: KindClick, Value: value, Side: "right", Index: index, Wildcard: strings.HasPrefix(value, "*")}
	}
	steps := []Step{
		click("Fixed", "reset everything", 0), // matched, as Papyrus compares
		click("Fixed", "Run", 2),              // the second option showing Run
		click("Fixed", "*Reset*", 0),          // a wildcard
		click("Fixed", "Nothing", 0),          // not shown anywhere: won't replay
		click("Fixed", "Run", 3),              // only two show it
		click("Moving", "Enabled", 0),         // a value it shows at times
		click("Moving", "Disabled", 0),        // perhaps the other one: can't tell
	}
	got := Validate(&Recording{Steps: steps}, c)
	want := []string{"matched", "matched 2", "matched", "not-found value", "not-found index", "matched", "unchecked value"}
	for i, ch := range got {
		s := string(ch.Status)
		if ch.Why != "" {
			s += " " + string(ch.Why)
		}
		if ch.Status == StatusMatched && ch.Position > 0 {
			s += fmt.Sprintf(" %d", ch.Position)
		}
		if s != want[i] {
			t.Errorf("step %d: %s, want %s", i+1, s, want[i])
		}
	}
}

// Typed text is read strictly: what MCM Recorder reads passes, in any key
// case, and a slip is refused with the step's number.
func TestParseSteps(t *testing.T) {
	steps, err := ParseSteps("\uFEFF[\n {\"Mod\": \"VRIK\", \"option\": \"Head Bobbing Height\", \"slider\": 0.5, \"page\": \"VRIK Options\", \"index\": 2},\n {\"mod\": \"Be Seated\", \"click\": \"Reset\", \"side\": \"right\"}\n]")
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 2 || steps[0].Kind != KindSlider || steps[0].Value != "0.5" || steps[0].Index != 2 || steps[0].Page != "VRIK Options" || steps[1].Kind != KindClick || steps[1].Option != "Reset" || steps[1].Side != "right" {
		t.Errorf("steps = %+v", steps)
	}
	for text, want := range map[string]string{
		`{"mod": "x"}`: "array of actions",
		`[{"mod": "x", "option": "y", "toggle": "on"}] x`:               "more text",
		`[{"mod": "x", "option": "y", "togle": "on"}]`:                  `MCM Recorder doesn't read, "togle"`,
		`[{"mod": "x", "option": "y"}]`:                                 "doesn't say what it does",
		`[{"mod": "x", "option": "y", "toggle": "on", "slider": 1.0}]`:  "does one thing",
		`[{"option": "y", "toggle": "on"}]`:                             "names no menu",
		`[{"mod": "x", "toggle": "on"}]`:                                "names no option",
		`[{"mod": "x", "option": "y", "toggle": "yes"}]`:                "on or off",
		`[{"mod": "x", "option": "y", "slider": "wide"}]`:               "a number",
		`[{"mod": 3, "option": "y", "toggle": "on"}]`:                   "in quotes",
		`[{"mod": "x", "option": "y", "toggle": "on", "index": "two"}]`: "whole number",
	} {
		if _, err := ParseSteps(text); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want %q", text, err, want)
		}
	}
	if steps, err := ParseSteps("[]"); err != nil || len(steps) != 0 {
		t.Errorf("empty: %v, %v", steps, err)
	}
}

func TestParseInfo(t *testing.T) {
	info, err := ParseInfo(`{"author": "Moyce", "autorun": "True", "name": "MGO", "version": "1.0.0"}`)
	if err != nil || info != (Info{Title: "MGO", Version: "1.0.0", Author: "Moyce", Autorun: true}) {
		t.Errorf("info = %+v, %v", info, err)
	}
	if info, err := ParseInfo(`{"Autorun": false, "Name": "x"}`); err != nil || info.Autorun || info.Title != "x" {
		t.Errorf("bool autorun: %+v, %v", info, err)
	}
	for text, want := range map[string]string{
		`[]`:                         "JSON object",
		`{"name": "x", "colour": 1}`: "isn't a key MCM Recorder reads",
		`{"autorun": "maybe"}`:       "true or false",
		`{"version": 2}`:             "in quotes",
	} {
		if _, err := ParseInfo(text); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want %q", text, err, want)
		}
	}
}

// How long MCM Recorder looks for an option is kept as the file spells it:
// read from a file, typed, and written back.
func TestWaits(t *testing.T) {
	steps, err := ParseSteps(`[{"mod": "Be Seated", "option": "Height", "slider": 2.0, "timeout": 60, "interval": 0.25, "pageload": 10.0}]`)
	if err != nil {
		t.Fatal(err)
	}
	s := steps[0]
	if s.Timeout != "60" || s.Interval != "0.25" || s.PageLoad != "10.0" {
		t.Fatalf("typed: %+v", s)
	}
	out, err := Plan(&Draft{Name: "Waits", Steps: steps})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"timeout": 60`, `"interval": 0.25`, `"pageload": 10.0`} {
		if !strings.Contains(out[1].Text, want) {
			t.Errorf("written without %s:\n%s", want, out[1].Text)
		}
	}
	again, err := ParseSteps(out[1].Text)
	if err != nil || again[0].Timeout != "60" || again[0].Interval != "0.25" || again[0].PageLoad != "10.0" {
		t.Errorf("read back: %+v, %v", again, err)
	}
	for text, want := range map[string]string{
		`[{"mod": "x", "option": "y", "toggle": "on", "timeout": "60"}]`: "number of seconds",
		`[{"mod": "x", "option": "y", "toggle": "on", "pageload": -1}]`:  "number of seconds",
	} {
		if _, err := ParseSteps(text); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want %q", text, err, want)
		}
	}
}

// MCM Recorder plays a step on an option the menu grays out, since it
// calls the mod's handler without looking at the option's flags: the step
// matches, saying so, unless its value is one the option can't take.
func TestGrayedOutStepPlays(t *testing.T) {
	c := &catalog.Catalog{Menus: []catalog.Menu{{
		Name: "Experience", NameText: "Experience", Active: true, Certainty: catalog.CertaintyExact,
		Pages: []catalog.Page{{Name: "$EXP_General", Text: "General", Options: []catalog.Option{
			{Type: catalog.OptionTypeText, Label: "$EXP_PluginVersion", Certainty: catalog.CertaintyExact, Disabled: catalog.DisabledAlways},
			{Type: catalog.OptionTypeSlider, Label: "Level", Certainty: catalog.CertaintyExact, Disabled: catalog.DisabledAlways, Slider: &catalog.Range{Min: 1, Max: 10, Step: 1}},
		}}},
	}}}
	checks := Validate(&Recording{Steps: []Step{
		{Mod: "Experience", Page: "$EXP_General", Option: "$EXP_PluginVersion", Kind: KindClick, Value: "$EXP_PluginVersion"},
		{Mod: "Experience", Page: "$EXP_General", Option: "Level", Kind: KindSlider, Value: "20"},
	}}, c)
	if checks[0].Status != StatusMatched || checks[0].Why != WhyDisabled {
		t.Errorf("a click on a grayed-out option: %s, %s", checks[0].Status, checks[0].Why)
	}
	if checks[1].Status != StatusNotAllowed || checks[1].Why != WhyRange {
		t.Errorf("a grayed-out slider out of range: %s, %s", checks[1].Status, checks[1].Why)
	}
}

// A click on a text option the menu doesn't respond to matches, saying it
// does nothing, grayed out or not.
func TestInertClickDoesNothing(t *testing.T) {
	c := &catalog.Catalog{Menus: []catalog.Menu{{
		Name: "Wet", NameText: "Wet", Active: true, Certainty: catalog.CertaintyExact,
		Pages: []catalog.Page{{Name: "Misc", Text: "Misc", Options: []catalog.Option{
			{Type: catalog.OptionTypeText, Label: "Version", Certainty: catalog.CertaintyExact, Inert: true},
			{Type: catalog.OptionTypeText, Label: "Should Run", Certainty: catalog.CertaintyExact, Inert: true, Disabled: catalog.DisabledAlways},
			{Type: catalog.OptionTypeText, Label: "Export", Certainty: catalog.CertaintyExact},
		}}},
	}}}
	checks := Validate(&Recording{Steps: []Step{
		{Mod: "Wet", Page: "Misc", Option: "Version", Kind: KindClick, Value: "Version"},
		{Mod: "Wet", Page: "Misc", Option: "Should Run", Kind: KindClick, Value: "Should Run"},
		{Mod: "Wet", Page: "Misc", Option: "Export", Kind: KindClick, Value: "Export"},
	}}, c)
	for i, want := range []Why{WhyInert, WhyInert, ""} {
		if checks[i].Status != StatusMatched || checks[i].Why != want {
			t.Errorf("step %d: %s, %q, want matched, %q", i+1, checks[i].Status, checks[i].Why, want)
		}
	}
}
