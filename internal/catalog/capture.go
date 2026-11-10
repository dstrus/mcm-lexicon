package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"lexicon/internal/translations"
)

// CaptureFormat is the capture file's format that this reads; it refuses
// another.
const CaptureFormat = 1

// Capture is what the capture plugin read from each menu in a running game
// (PLAN.md §11): capture.json, as MCMLexicon.dll writes it.
type Capture struct {
	Format   int              `json:"format"`
	Plugin   string           `json:"plugin"`  // the plugin's version, or the commit it was built from
	Game     string           `json:"game"`    // "SkyrimSE" or "SkyrimVR"
	Runtime  string           `json:"runtime"` // the game's version
	Captured time.Time        `json:"captured"`
	Menus    []CapturedMenu   `json:"menus"` // in the order SkyUI registered them
	Skipped  []CapturedSkip   `json:"skipped"`
	Stopped  string           `json:"stopped"` // why it ended early, if it did
	Running  *CapturedRunning `json:"running"` // only in a file the capture didn't finish
}

type CapturedMenu struct {
	Name   string         `json:"name"`   // as SkyUI lists it, raw: "" for one it doesn't list
	Script string         `json:"script"` // the menu's script
	Quest  string         `json:"quest"`  // its quest, as plugin.FormKey writes it
	Pages  []CapturedPage `json:"pages"`
	Error  string         `json:"error"`
}

type CapturedPage struct {
	Name    string           `json:"name"` // raw; "" for a menu without pages
	Options []CapturedOption `json:"options"`
	Error   string           `json:"error"`
}

// CapturedOption is one entry of SkyUI's option buffers. The game pools
// strings without regard to case, so Text and Str may be cased as another
// script first wrote them.
type CapturedOption struct {
	Position int     `json:"position"`
	Type     string  `json:"type"` // SkyUI's option type: empty, header, text, toggle, slider, menu, color, keymap, input
	Flags    int     `json:"flags"`
	Text     string  `json:"text"`
	Str      string  `json:"str"` // a slider's format, a menu's or text option's shown value; "None" for a none
	Num      float64 `json:"num"` // a toggle's 0 or 1, a slider's value, a key's scan code, a color
	State    string  `json:"state"`
}

type CapturedSkip struct {
	Script string `json:"script"`
	Reason string `json:"reason"`
}

type CapturedRunning struct {
	Menu   string `json:"menu"`
	Script string `json:"script"`
	Step   string `json:"step"`
}

// SkyUI's option flag for an option drawn grayed out.
const flagDisabled = 1

// ReadCapture reads a capture file. One the capture didn't finish is read
// all the same: its menus are as good as a finished one's.
func ReadCapture(path string) (*Capture, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Capture
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("%s isn't a capture file: %w", path, err)
	}
	if c.Format != CaptureFormat {
		return nil, fmt.Errorf("%s is a capture in format %d; this version of MCM Lexicon reads format %d", path, c.Format, CaptureFormat)
	}
	return &c, nil
}

func (r CaptureReport) String() string {
	return fmt.Sprintf("%d menus (%d matched, %d added), %d pages (%d matched, %d added), %d options (%d by position, %d by label, %d by pattern, %d added), %d values",
		r.Menus, r.MenusMatched, r.MenusAdded, r.Pages, r.PagesMatched, r.PagesAdded,
		r.Options, r.ByPosition, r.ByLabel, r.ByPattern, r.OptionsAdded, r.Values)
}

// ApplyCapture is c with what the capture read from path added: each
// option it matches gets the game's value, and the game's label where the
// files only partly know it, and the pages, options, and menus the files
// don't have are added as captured. c itself isn't changed, so a scan and
// its cache stay as the files say.
func ApplyCapture(c *Catalog, f *Capture, path string) *Catalog {
	out := *c
	out.Menus = slices.Clone(c.Menus)
	for i := range out.Menus {
		m := &out.Menus[i]
		m.Pages = slices.Clone(m.Pages)
		for j := range m.Pages {
			m.Pages[j].Options = slices.Clone(m.Pages[j].Options)
			m.Pages[j].Fixtures = slices.Clone(m.Pages[j].Fixtures)
		}
	}
	a := applier{cat: &out, ids: map[string]bool{}, tr: c.tr}
	for _, m := range out.Menus {
		a.ids[m.ID] = true
	}

	matched := map[int]bool{}
	for _, cm := range f.Menus {
		if cm.Name == "" {
			continue // SkyUI doesn't list it
		}
		a.report.Menus++
		i := a.findMenu(cm, matched)
		if i < 0 {
			a.addMenu(cm)
			continue
		}
		matched[i] = true
		a.report.MenusMatched++
		a.translating(out.Menus[i].Plugin)
		a.applyMenu(&out.Menus[i], cm)
	}
	if a.report.MenusAdded > 0 {
		sort.SliceStable(out.Menus, func(i, j int) bool {
			return strings.ToLower(out.Menus[i].NameText) < strings.ToLower(out.Menus[j].NameText)
		})
	}
	out.Capture = InfoOf(f, path)
	out.Capture.Report = a.report
	return &out
}

// InfoOf is what a catalog says of the capture read from path, before it's
// matched against one: everything but the report.
func InfoOf(f *Capture, path string) *CaptureInfo {
	info := &CaptureInfo{Path: path, Captured: f.Captured, Game: f.Game, Plugin: f.Plugin, Stopped: f.Stopped}
	if f.Running != nil {
		info.Unfinished = f.Running.Menu + ": " + f.Running.Step
	}
	return info
}

type applier struct {
	cat    *Catalog
	ids    map[string]bool
	tr     *translations.Table // the scan's; nil for a catalog file
	prefer []string            // the translation files to look in first: the menu's plugin's
	report CaptureReport
}

// text is a captured label for reading, as the scan gives the files' own:
// translated where it's a $KEY the list translates, without its markup.
func (a *applier) text(s string) string {
	if a.tr != nil {
		s = a.tr.Text(s, a.prefer...)
	}
	return plain(s)
}

// translating sets the translation files a menu's text looks in first.
func (a *applier) translating(plugin string) {
	a.prefer = nil
	if plugin != "" {
		a.prefer = []string{strings.TrimSuffix(plugin, filepath.Ext(plugin))}
	}
}

// findMenu is the catalog menu a captured one is: the one with its script,
// or, when no menu or several have it, the one with its name.
func (a *applier) findMenu(cm CapturedMenu, taken map[int]bool) int {
	var byScript []int
	for i, m := range a.cat.Menus {
		if !taken[i] && strings.EqualFold(m.Script, cm.Script) {
			byScript = append(byScript, i)
		}
	}
	if len(byScript) == 1 {
		return byScript[0]
	}
	among := byScript
	if len(among) == 0 {
		among = make([]int, 0, len(a.cat.Menus))
		for i := range a.cat.Menus {
			if !taken[i] {
				among = append(among, i)
			}
		}
	}
	for _, i := range among {
		if strings.EqualFold(a.cat.Menus[i].Name, cm.Name) {
			return i
		}
	}
	return -1
}

func (a *applier) applyMenu(m *Menu, cm CapturedMenu) {
	for _, cp := range cm.Pages {
		if cp.Error != "" {
			continue
		}
		a.report.Pages++
		i := slices.IndexFunc(m.Pages, func(p Page) bool { return strings.EqualFold(p.Name, cp.Name) })
		if i < 0 {
			m.Pages = append(m.Pages, a.newPage(cp))
			a.report.PagesAdded++
			continue
		}
		a.report.PagesMatched++
		a.applyPage(&m.Pages[i], cp)
	}
}

func (a *applier) addMenu(cm CapturedMenu) {
	a.translating(pluginOf(cm.Quest))
	m := Menu{
		ID:        a.id(cm.Name),
		Name:      cm.Name,
		NameText:  a.text(cm.Name),
		Kind:      KindCaptured,
		Plugin:    pluginOf(cm.Quest),
		Active:    true,
		Script:    cm.Script,
		Pages:     []Page{},
		Certainty: CertaintyCaptured,
		Notes:     []Note{},
	}
	for _, cp := range cm.Pages {
		if cp.Error != "" {
			continue
		}
		a.report.Pages++
		a.report.PagesAdded++
		m.Pages = append(m.Pages, a.newPage(cp))
	}
	a.cat.Menus = append(a.cat.Menus, m)
	a.report.MenusAdded++
}

// applyPage matches each captured option to one of the page's: by where
// the page draws it (when an exact label agrees), then by its label, then
// by a pattern it fits. One that matches none is added.
func (a *applier) applyPage(p *Page, cp CapturedPage) {
	taken := make([]bool, len(p.Options))
	// find is the first option not yet matched for which ok is true, so
	// options that share a label match in order, as Index numbers them.
	find := func(ok func(Option) bool) int {
		for i, o := range p.Options {
			if !taken[i] && ok(o) {
				return i
			}
		}
		return -1
	}
	added := false
	for _, co := range cp.Options {
		t, ok := capturedType(co.Type)
		if !ok {
			continue // a header, or an empty slot
		}
		a.report.Options++
		i, by := find(func(o Option) bool {
			return o.Type == t && o.Position >= 0 && o.Position == co.Position &&
				(o.Certainty != CertaintyExact || strings.EqualFold(o.Label, co.Text))
		}), &a.report.ByPosition
		if i < 0 {
			i, by = find(func(o Option) bool {
				return o.Type == t && o.Certainty == CertaintyExact && strings.EqualFold(o.Label, co.Text)
			}), &a.report.ByLabel
		}
		if i < 0 {
			i, by = find(func(o Option) bool {
				return o.Type == t && o.Certainty == CertaintyPattern && fits(o.Label, co.Text)
			}), &a.report.ByPattern
		}
		if i < 0 {
			p.Options = append(p.Options, a.newOption(co, t))
			taken = append(taken, true)
			added = true
			continue
		}
		*by++
		taken[i] = true
		o := &p.Options[i]
		if v := gameValue(*o, co); v != nil {
			o.Game = v
			a.report.Values++
		}
		if o.Certainty != CertaintyExact && co.Text != "" && !strings.EqualFold(o.Label, co.Text) {
			o.GameLabel, o.GameText = co.Text, a.text(co.Text)
		}
	}
	if added {
		numberDuplicates(p.Options)
	}
}

func (a *applier) newPage(cp CapturedPage) Page {
	p := Page{Name: cp.Name, Text: a.text(cp.Name), Options: []Option{}, Fixtures: []Fixture{}, Placed: true}
	for _, co := range cp.Options {
		if co.Type == "header" {
			p.Fixtures = append(p.Fixtures, Fixture{Kind: FixtureKindHeader, Label: co.Text, Text: a.text(co.Text), Certainty: CertaintyCaptured, Position: co.Position})
			continue
		}
		if t, ok := capturedType(co.Type); ok {
			a.report.Options++
			p.Options = append(p.Options, a.newOption(co, t))
		}
	}
	numberDuplicates(p.Options)
	return p
}

func (a *applier) newOption(co CapturedOption, t OptionType) Option {
	o := Option{
		Type:      t,
		Label:     co.Text,
		Text:      a.text(co.Text),
		State:     co.State,
		Certainty: CertaintyCaptured,
		Position:  co.Position,
		Notes:     []Note{},
	}
	if co.Flags&flagDisabled != 0 {
		o.Disabled = DisabledSometimes // so it was in this game; the files don't say whether it always is
	}
	if t == OptionTypeText && shown(co.Str) {
		o.Value, o.ValueText, o.ValueCertainty = co.Str, a.text(co.Str), CertaintyCaptured
	}
	if v := gameValue(o, co); v != nil {
		o.Game = v
		a.report.Values++
	}
	a.report.OptionsAdded++
	return o
}

// id is an added menu's ID, as scan.id makes them, under the kind
// "captured".
func (a *applier) id(name string) string {
	s := scan{ids: a.ids}
	return s.id(string(KindCaptured), name)
}

// capturedType is a captured option's type as a recording names it;
// false for a header or an empty slot, which aren't options.
func capturedType(t string) (OptionType, bool) {
	switch t {
	case "toggle", "slider", "menu", "keymap", "color", "input", "text":
		return OptionType(t), true
	}
	return "", false
}

// shown reports whether a captured string holds something: Papyrus writes
// a none as "None", and SkyUI passes none for the strings an option type
// doesn't use.
func shown(s string) bool { return s != "" && s != "None" }

// gameValue is the value an option showed in the captured game, in the
// catalog's terms: on or off, a number, the entry by position for a menu
// (or a stepper) whose entries are known and its text otherwise, and text.
func gameValue(o Option, co CapturedOption) *Value {
	switch o.Type {
	case OptionTypeToggle:
		return &Value{Kind: ValueKindBool, Bool: co.Num != 0}
	case OptionTypeSlider, OptionTypeKeymap, OptionTypeColor:
		return &Value{Kind: ValueKindNumber, Number: co.Num}
	case OptionTypeInput:
		if co.Str == "None" {
			return nil
		}
		return &Value{Kind: ValueKindString, String: co.Str}
	case OptionTypeMenu, OptionTypeText:
		if i := slices.IndexFunc(o.Choices, func(c Choice) bool { return strings.EqualFold(c.Raw, co.Str) }); i >= 0 {
			return &Value{Kind: ValueKindNumber, Number: float64(i)}
		}
		if shown(co.Str) && len(o.Choices) == 0 {
			return &Value{Kind: ValueKindString, String: co.Str}
		}
	}
	return nil
}

// fits reports whether text fits a pattern label: *part* fits any text
// containing part, as MCM Recorder matches it.
func fits(label, text string) bool {
	if len(label) > 2 && strings.HasPrefix(label, "*") && strings.HasSuffix(label, "*") {
		return strings.Contains(strings.ToLower(text), strings.ToLower(label[1:len(label)-1]))
	}
	return strings.EqualFold(label, text)
}

// pluginOf is the plugin a capture's quest key names.
func pluginOf(quest string) string {
	if i := strings.LastIndexByte(quest, ':'); i > 0 {
		return quest[:i]
	}
	return ""
}
