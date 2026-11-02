package catalog

import (
	"strings"
	"testing"

	"lexicon/internal/modlist"
	"lexicon/internal/testcorpus"
)

func TestCompare(t *testing.T) {
	old := &Catalog{Menus: []Menu{
		{ID: "a", Name: "Alpha", NameText: "Alpha", Pages: []Page{{Name: "Main", Options: []Option{
			{Type: OptionTypeToggle, Label: "Keep"},
			{Type: OptionTypeToggle, Label: "Show Messages"},
			{Type: OptionTypeSlider, Label: "Chance", Slider: &Range{Min: 0, Max: 100, Step: 1}},
			{Type: OptionTypeMenu, Label: "Mode", Choices: []Choice{{Raw: "Off"}, {Raw: "On"}}},
			{Type: OptionTypeToggle, Label: "Going away"},
		}}, {Name: "Old Page", Options: []Option{{Type: OptionTypeToggle, Label: "Lost"}}}}},
		{ID: "gone", Name: "Gone", NameText: "Gone"},
		{ID: "renamed-id", Name: "Same Name", NameText: "Same Name"},
	}}
	new := &Catalog{Menus: []Menu{
		{ID: "a", Name: "Alpha", NameText: "Alpha", Pages: []Page{{Name: "Main", Options: []Option{
			{Type: OptionTypeToggle, Label: "Keep"},
			{Type: OptionTypeToggle, Label: "Show All Messages"},
			{Type: OptionTypeSlider, Label: "Chance", Slider: &Range{Min: 0, Max: 50, Step: 1}},
			{Type: OptionTypeMenu, Label: "Mode", Choices: []Choice{{Raw: "Off"}, {Raw: "On"}, {Raw: "Auto"}}},
			{Type: OptionTypeKeymap, Label: "Brand new"},
		}}}},
		{ID: "new-id", Name: "same name", NameText: "Same Name"},
		{ID: "fresh", Name: "Fresh", NameText: "Fresh"},
	}}
	d := Compare(old, new)
	var got []string
	for _, m := range d.Menus {
		got = append(got, string(m.Change)+" "+m.Name)
		for _, o := range m.Options {
			got = append(got, "  "+string(o.Change)+" "+o.Label+" "+o.Before+">"+o.After)
		}
	}
	want := []string{
		"changed Alpha",
		"  range Chance 0 to 100 by 1>0 to 50 by 1",
		"  choices Mode Off, On>Off, On, Auto",
		"  renamed Show All Messages Show Messages>Show All Messages",
		"  added Brand new >",
		"  removed Going away >",
		"  page  Old Page>",
		"  removed Lost >",
		"added Fresh",
		"removed Gone",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("diff:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	c := d.Counts
	if c.MenusAdded != 1 || c.MenusRemoved != 1 || c.OptionsAdded != 1 || c.OptionsRemoved != 2 || c.OptionsRenamed != 1 || c.ValuesChanged != 2 {
		t.Errorf("counts = %+v", c)
	}
	if r := Report(d); !strings.Contains(r, "renamed toggle Main / \"Show All Messages\": \"Show Messages\" -> \"Show All Messages\"") {
		t.Errorf("report:\n%s", r)
	}
}

// The two real lists share many mods; comparing them must work and find
// what one has and the other doesn't, and a catalog compared with itself
// differs in nothing.
func TestCompareRealSnapshots(t *testing.T) {
	snaps := testcorpus.Snapshots(t)
	var cats []*Catalog
	for _, snap := range snaps {
		src, err := modlist.Check(snap.Dir, "")
		if err != nil {
			t.Fatal(err)
		}
		l, err := modlist.Read(src)
		if err != nil {
			t.Fatal(err)
		}
		c := Scan(l, Options{})
		if d := Compare(c, c); len(d.Menus) != 0 {
			t.Errorf("%s compared with itself: %+v", snap.Name, d.Menus)
		}
		cats = append(cats, c)
	}
	if len(cats) < 2 {
		return
	}
	d := Compare(cats[0], cats[1])
	t.Logf("%s to %s: %+v", snaps[0].Name, snaps[1].Name, d.Counts)
	if d.Counts.MenusAdded == 0 || d.Counts.MenusRemoved == 0 {
		t.Errorf("two different lists compared alike: %+v", d.Counts)
	}
}
