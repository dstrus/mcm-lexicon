package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// Change is what changed between two catalogs, of a menu or an option.
type Change string

const (
	ChangeAdded     Change = "added"
	ChangeRemoved   Change = "removed"
	ChangeChanged   Change = "changed"   // a menu with changed options
	ChangeRenamed   Change = "renamed"   // an option whose label changed, by its type and place
	ChangeRange     Change = "range"     // a slider's range or step
	ChangeChoices   Change = "choices"   // a menu's entries
	ChangeDefault   Change = "default"   // its default value
	ChangeCertainty Change = "certainty" // how sure the catalog is of its label
	ChangePage      Change = "page"      // a page added or removed
)

// MenuDiff is how one menu differs.
type MenuDiff struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	NameText string       `json:"nameText"`
	Change   Change       `json:"change"`
	Options  []OptionDiff `json:"options"` // for a changed menu
}

// OptionDiff is how one option, or one page, differs.
type OptionDiff struct {
	Change   Change     `json:"change"`
	Page     string     `json:"page"` // raw
	PageText string     `json:"pageText"`
	Type     OptionType `json:"type"`
	Label    string     `json:"label"` // raw; the new label, for a rename
	Text     string     `json:"text"`
	Before   string     `json:"before"` // what it was, as text
	After    string     `json:"after"`
}

// Diff is every difference between an older and a newer catalog.
type Diff struct {
	Old    ListInfo   `json:"old"`
	New    ListInfo   `json:"new"`
	Menus  []MenuDiff `json:"menus"` // in the newer catalog's menu order, then removed menus
	Counts DiffCounts `json:"counts"`
}

type DiffCounts struct {
	MenusAdded     int `json:"menusAdded"`
	MenusRemoved   int `json:"menusRemoved"`
	OptionsAdded   int `json:"optionsAdded"`
	OptionsRemoved int `json:"optionsRemoved"`
	OptionsRenamed int `json:"optionsRenamed"`
	ValuesChanged  int `json:"valuesChanged"`
}

// Compare finds what changed from old to new. Menus are matched by ID,
// then by name; options by page, type, label, and index, with a removed
// and an added option of the same type in the same place, and labels
// alike, taken as a rename.
func Compare(old, new *Catalog) Diff {
	d := Diff{Old: old.List, New: new.List, Menus: []MenuDiff{}}
	byID, byName := map[string]*Menu{}, map[string]*Menu{}
	for i := range old.Menus {
		byID[old.Menus[i].ID] = &old.Menus[i]
		byName[strings.ToLower(old.Menus[i].Name)] = &old.Menus[i]
	}
	used := map[*Menu]bool{}
	for i := range new.Menus {
		n := &new.Menus[i]
		o := byID[n.ID]
		if o == nil || used[o] {
			o = byName[strings.ToLower(n.Name)]
		}
		if o == nil || used[o] {
			d.Menus = append(d.Menus, MenuDiff{ID: n.ID, Name: n.Name, NameText: n.NameText, Change: ChangeAdded})
			d.Counts.MenusAdded++
			continue
		}
		used[o] = true
		if opts := compareMenu(o, n); len(opts) > 0 {
			d.Menus = append(d.Menus, MenuDiff{ID: n.ID, Name: n.Name, NameText: n.NameText, Change: ChangeChanged, Options: opts})
			for _, x := range opts {
				switch x.Change {
				case ChangeAdded:
					d.Counts.OptionsAdded++
				case ChangeRemoved:
					d.Counts.OptionsRemoved++
				case ChangeRenamed:
					d.Counts.OptionsRenamed++
				case ChangePage:
				default:
					d.Counts.ValuesChanged++
				}
			}
		}
	}
	for i := range old.Menus {
		if o := &old.Menus[i]; !used[o] {
			d.Menus = append(d.Menus, MenuDiff{ID: o.ID, Name: o.Name, NameText: o.NameText, Change: ChangeRemoved})
			d.Counts.MenusRemoved++
		}
	}
	return d
}

func compareMenu(o, n *Menu) []OptionDiff {
	var out []OptionDiff
	oldPages := map[string]*Page{}
	for i := range o.Pages {
		oldPages[strings.ToLower(o.Pages[i].Name)] = &o.Pages[i]
	}
	seen := map[string]bool{}
	for i := range n.Pages {
		np := &n.Pages[i]
		key := strings.ToLower(np.Name)
		seen[key] = true
		op := oldPages[key]
		if op == nil {
			out = append(out, OptionDiff{Change: ChangePage, Page: np.Name, PageText: pageText(np), After: pageText(np)})
			for _, x := range np.Options {
				out = append(out, OptionDiff{Change: ChangeAdded, Page: np.Name, PageText: pageText(np), Type: x.Type, Label: x.Label, Text: x.Text})
			}
			continue
		}
		out = append(out, comparePage(op, np)...)
	}
	for _, op := range o.Pages {
		if !seen[strings.ToLower(op.Name)] {
			out = append(out, OptionDiff{Change: ChangePage, Page: op.Name, PageText: pageText(&op), Before: pageText(&op)})
			for _, x := range op.Options {
				out = append(out, OptionDiff{Change: ChangeRemoved, Page: op.Name, PageText: pageText(&op), Type: x.Type, Label: x.Label, Text: x.Text})
			}
		}
	}
	return out
}

// key is how an option is matched: its type, label, and index.
func key(o Option) string {
	return fmt.Sprintf("%s\x00%s\x00%d", o.Type, strings.ToLower(o.Label), o.Index)
}

func comparePage(op, np *Page) []OptionDiff {
	var out []OptionDiff
	old := map[string]int{}
	for i, o := range op.Options {
		old[key(o)] = i
	}
	matched := map[int]bool{}
	var added []int
	for i, n := range np.Options {
		j, ok := old[key(n)]
		if !ok {
			added = append(added, i)
			continue
		}
		matched[j] = true
		out = append(out, compareOption(op.Options[j], n, np)...)
	}
	var removed []int
	for j := range op.Options {
		if !matched[j] {
			removed = append(removed, j)
		}
	}
	// A removed and an added option of one type, in the same place among the
	// changes, with labels alike: one option renamed.
	renamed := map[int]bool{}
	for _, i := range added {
		n := np.Options[i]
		for _, j := range removed {
			o := op.Options[j]
			if renamed[j] || o.Type != n.Type || abs(i-j) > 2 || alike(o.Label, n.Label) < 0.5 {
				continue
			}
			renamed[j] = true
			out = append(out, OptionDiff{Change: ChangeRenamed, Page: np.Name, PageText: pageText(np), Type: n.Type, Label: n.Label, Text: n.Text, Before: o.Label, After: n.Label})
			added = without(added, i)
			break
		}
	}
	for _, i := range added {
		n := np.Options[i]
		out = append(out, OptionDiff{Change: ChangeAdded, Page: np.Name, PageText: pageText(np), Type: n.Type, Label: n.Label, Text: n.Text})
	}
	for _, j := range removed {
		if !renamed[j] {
			o := op.Options[j]
			out = append(out, OptionDiff{Change: ChangeRemoved, Page: np.Name, PageText: pageText(np), Type: o.Type, Label: o.Label, Text: o.Text})
		}
	}
	return out
}

func compareOption(o, n Option, p *Page) []OptionDiff {
	var out []OptionDiff
	add := func(c Change, before, after string) {
		if before != after {
			out = append(out, OptionDiff{Change: c, Page: p.Name, PageText: pageText(p), Type: n.Type, Label: n.Label, Text: n.Text, Before: before, After: after})
		}
	}
	add(ChangeRange, rangeText(o.Slider), rangeText(n.Slider))
	add(ChangeChoices, choicesText(o.Choices), choicesText(n.Choices))
	add(ChangeDefault, valueText(o.Default), valueText(n.Default))
	add(ChangeCertainty, string(o.Certainty), string(n.Certainty))
	return out
}

// pageText is a page as people read it: its text, or its name without one.
func pageText(p *Page) string {
	if p.Text != "" {
		return p.Text
	}
	return p.Name
}

func rangeText(r *Range) string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("%g to %g by %g", r.Min, r.Max, r.Step)
}

func choicesText(cs []Choice) string {
	raw := make([]string, len(cs))
	for i, c := range cs {
		raw[i] = c.Raw
	}
	return strings.Join(raw, ", ")
}

func valueText(v *Value) string {
	if v == nil {
		return ""
	}
	switch v.Kind {
	case ValueKindBool:
		if v.Bool {
			return "on"
		}
		return "off"
	case ValueKindString:
		return v.String
	}
	return fmt.Sprintf("%g", v.Number)
}

// alike is how alike two labels are, from 0 to 1, by the words they share.
func alike(a, b string) float64 {
	wa, wb := strings.Fields(strings.ToLower(a)), strings.Fields(strings.ToLower(b))
	if len(wa) == 0 || len(wb) == 0 {
		return 0
	}
	set := map[string]bool{}
	for _, w := range wa {
		set[w] = true
	}
	shared := 0
	for _, w := range wb {
		if set[w] {
			shared++
		}
	}
	return float64(shared) / float64(max(len(wa), len(wb)))
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func without(xs []int, x int) []int {
	out := xs[:0:0]
	for _, v := range xs {
		if v != x {
			out = append(out, v)
		}
	}
	return out
}

// Report is a diff as text, for the command line.
func Report(d Diff) string {
	var b strings.Builder
	fmt.Fprintf(&b, "From %s (%s, scanned %s)\nTo   %s (%s, scanned %s)\n\n", d.Old.Name, d.Old.Profile, d.Old.ScannedAt.Format("2006-01-02 15:04"), d.New.Name, d.New.Profile, d.New.ScannedAt.Format("2006-01-02 15:04"))
	c := d.Counts
	fmt.Fprintf(&b, "%d menus added, %d removed; %d options added, %d removed, %d renamed; %d values changed\n", c.MenusAdded, c.MenusRemoved, c.OptionsAdded, c.OptionsRemoved, c.OptionsRenamed, c.ValuesChanged)
	menus := append([]MenuDiff{}, d.Menus...)
	sort.SliceStable(menus, func(i, j int) bool { return menus[i].Change < menus[j].Change })
	for _, m := range menus {
		fmt.Fprintf(&b, "\n%s: %s\n", strings.ToUpper(string(m.Change)), m.NameText)
		for _, o := range m.Options {
			where := o.PageText
			if where != "" {
				where += " / "
			}
			switch o.Change {
			case ChangePage:
				if o.After != "" {
					fmt.Fprintf(&b, "  page added: %s\n", o.After)
				} else {
					fmt.Fprintf(&b, "  page removed: %s\n", o.Before)
				}
			case ChangeAdded, ChangeRemoved:
				fmt.Fprintf(&b, "  %s %s: %s%q\n", o.Change, o.Type, where, o.Label)
			default:
				fmt.Fprintf(&b, "  %s %s %s%q: %q -> %q\n", o.Change, o.Type, where, o.Label, o.Before, o.After)
			}
		}
	}
	return b.String()
}
