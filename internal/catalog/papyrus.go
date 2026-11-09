package catalog

import (
	"path"
	"sort"
	"strings"

	"lexicon/internal/modlist"
	"lexicon/internal/papyrus/ir"
	"lexicon/internal/papyrus/mcm"
	"lexicon/internal/papyrus/pex"
	"lexicon/internal/papyrus/source"
	"lexicon/internal/plugin"
)

// script is a compiled script the game runs that makes a classic MCM.
type script struct {
	name string // as the compiled file is named
	pex  modlist.File
}

// form is a script as the analysis reads it: from its source when the mod
// ships the source the game's compiled script was built from, otherwise
// from the compiled script itself.
type form struct {
	script *ir.Script
	kind   Kind
	file   modlist.File // what was read
	stale  string       // why the shipped source wasn't used, when it's another version
}

// scripts finds the compiled scripts that make MCMs by following each
// one's Extends chain: to SKI_ConfigBase for a classic MCM, to MCM_ConfigBase
// for MCM Helper's (whose config.json defines the menu).
func (s *scan) scripts() {
	s.helpers = map[string][]string{}
	s.sources = map[string][]modlist.File{}
	s.forms = map[string]*form{}
	s.compiled = map[string]modlist.File{}
	for _, f := range s.list.Files("") {
		if strings.EqualFold(path.Ext(f.Path), ".psc") {
			k := ir.Key(strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path)))
			s.sources[k] = append(s.sources[k], f)
		}
	}
	for _, f := range s.list.Files("Scripts") {
		if strings.Count(f.Path, "/") == 1 && strings.EqualFold(path.Ext(f.Path), ".pex") {
			s.compiled[ir.Key(strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path)))] = f
		}
	}
	for _, k := range sortedKeys(s.compiled) {
		if k == "ski_configbase" || k == "mcm_configbase" {
			continue
		}
		f := s.compiled[k]
		switch s.base(k) {
		case "ski_configbase":
			s.classic = append(s.classic, script{name: strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path)), pex: f})
		case "mcm_configbase":
			s.helpers[f.Mod] = append(s.helpers[f.Mod], k)
		}
	}
}

// base follows a script's Extends chain and returns the MCM base it
// reaches, ski_configbase or mcm_configbase, or "" for neither.
func (s *scan) base(name string) string {
	seen := map[string]bool{}
	for k := name; k != "" && !seen[k]; {
		seen[k] = true
		if k == "ski_configbase" || k == "mcm_configbase" {
			return k
		}
		parent, ok := s.parent(k)
		if !ok {
			return ""
		}
		k = ir.Key(parent)
	}
	return ""
}

// parent is the script a script extends, from the cache when its compiled
// file hasn't changed since a scan read it, else from its form; ok is
// false for a script the list doesn't have.
func (s *scan) parent(name string) (string, bool) {
	if c, ok := s.compiled[name]; ok && s.cache != nil {
		if p, ok := s.cache.get(c); ok {
			return p, true
		}
	}
	f := s.form(name)
	if f == nil {
		return "", false
	}
	return f.script.Parent, true
}

// form is a script's form, read once: the compiled script the game runs,
// or its source when that's the same version (it reads the same, and
// carries the source's lines). A script with only a source counts too, as
// an ancestor a compiled script names.
func (s *scan) form(name string) *form {
	if f, ok := s.forms[name]; ok {
		return f
	}
	var out *form
	c, hasCompiled := s.compiled[name]
	var compiled *pex.File
	if hasCompiled {
		b, err := c.Read()
		if err == nil {
			compiled, err = pex.Read(b)
		}
		if err != nil {
			s.skip(c, SkipReasonUnreadable, err.Error())
			compiled = nil
		} else if s.cache != nil {
			s.cache.put(c, compiled.Script().Parent)
		}
	}
	src, text := s.source(name, c.Mod)
	switch {
	case compiled != nil && src != nil:
		if reason := compiled.OtherVersion(text); reason == "" {
			out = &form{script: src.script, kind: KindSource, file: src.file}
		} else {
			out = &form{script: compiled.Script(), kind: KindCompiled, file: c, stale: reason}
		}
	case compiled != nil:
		out = &form{script: compiled.Script(), kind: KindCompiled, file: c}
	case src != nil:
		out = &form{script: src.script, kind: KindSource, file: src.file}
	}
	s.forms[name] = out
	return out
}

type parsedSource struct {
	file   modlist.File
	script *ir.Script
}

// source is a script's parsed source and its text, preferring the copy in
// the mod whose compiled script the game runs.
func (s *scan) source(name, mod string) (*parsedSource, string) {
	files := s.sources[name]
	if len(files) == 0 {
		return nil, ""
	}
	f := files[0]
	for _, c := range files {
		if c.Mod == mod {
			f = c
			break
		}
	}
	b, err := f.Read()
	if err != nil {
		return nil, ""
	}
	r := source.Parse(string(b))
	if r.Script.Name == "" {
		return nil, ""
	}
	return &parsedSource{file: f, script: r.Script}, string(b)
}

// papyrus reads the classic MCMs: each script a quest runs, into a menu.
// One on no quest isn't a menu the game shows.
func (s *scan) papyrus() {
	for _, sc := range s.classic {
		a, attached := s.quest(sc.name)
		sk := Skipped{Path: sc.pex.Path, Mod: sc.pex.Mod, Archive: sc.pex.Archive, Pages: []string{}}
		if attached {
			sk.Name, sk.Plugin, sk.Quest, sk.Active = a.modName, a.plugin, a.quest, a.active
			sk.NameText = s.tr.Text(a.modName, strings.TrimSuffix(a.plugin, path.Ext(a.plugin)), sc.name)
			if a.pages != nil {
				sk.Pages = a.pages
			}
		}
		f := s.form(ir.Key(sc.name))
		switch {
		case !attached:
			sk.Reason = SkipReasonNoQuest
			s.cat.Skipped = append(s.cat.Skipped, sk)
		case f == nil:
			// Unreadable, and already listed as such.
		default:
			m := s.papyrusMenu(sc, a, f)
			if empty(m) {
				sk.Reason = SkipReasonEmpty
				s.cat.Skipped = append(s.cat.Skipped, sk)
				continue
			}
			s.cat.Menus = append(s.cat.Menus, m)
		}
	}
}

// papyrusMenu reads a classic MCM from its script and its ancestors'.
func (s *scan) papyrusMenu(sc script, a attachment, f *form) Menu {
	chain := []*ir.Script{f.script}
	for parent := f.script.Parent; len(chain) < 8; {
		k := ir.Key(parent)
		if k == "" || k == "ski_configbase" {
			break
		}
		p := s.form(k)
		if p == nil {
			break
		}
		chain = append(chain, p.script)
		parent = p.script.Parent
	}
	result := mcm.Extract(chain, s.properties(a))
	prefer := []string{strings.TrimSuffix(a.plugin, path.Ext(a.plugin)), sc.name}
	src := SourceRef{Path: f.file.Path, Mod: f.file.Mod, Archive: f.file.Archive}
	m := Menu{
		ID:        s.id("papyrus", sc.name),
		Name:      result.ModName,
		Kind:      f.kind,
		Mod:       sc.pex.Mod,
		Plugin:    a.plugin,
		Active:    a.active,
		Script:    sc.name,
		Certainty: CertaintyExact,
		Notes:     []Note{},
		Source:    src,
	}
	if m.Name == "" {
		m.Name = sc.name
		m.Notes = append(m.Notes, Note{Kind: NoteKindNameFromScript, Detail: sc.name})
	}
	m.NameText = s.tr.Text(m.Name, prefer...)
	if f.stale != "" {
		m.Notes = append(m.Notes, Note{Kind: NoteKindSourceStale, Detail: f.stale})
	}
	if result.PagesGuessed {
		m.Notes = append(m.Notes, Note{Kind: NoteKindPagesGuessed})
	}
	if len(chain) > 1 {
		m.Notes = append(m.Notes, Note{Kind: NoteKindFramework, Detail: chain[len(chain)-1].Name})
	}
	for _, p := range result.Pages {
		page := Page{Name: p.Name, Text: s.tr.Text(p.Name, prefer...), Options: []Option{}, Fixtures: []Fixture{}, Placed: p.Placed}
		for _, fx := range p.Fixtures {
			f := Fixture{Kind: FixtureKind(fx.Kind), Label: fx.Label, Text: fx.Text, Certainty: Certainty(fx.Certainty), Position: fx.Position}
			if fx.Kind == "header" && fx.Certainty == mcm.Exact {
				f.Text = s.tr.Text(fx.Label, prefer...)
			}
			if fx.Kind == "empty" {
				f.Certainty = CertaintyExact
			}
			page.Fixtures = append(page.Fixtures, f)
		}
		for _, o := range p.Options {
			opt := Option{
				Type:      OptionType(o.Type),
				Position:  o.Position,
				Disabled:  Disabled(o.Disabled),
				Inert:     o.Inert,
				Label:     o.Label,
				Text:      o.Text,
				State:     o.State,
				Help:      o.Help,
				HelpText:  s.tr.Text(o.Help, prefer...),
				Certainty: Certainty(o.Certainty),
				Notes:     []Note{},
				Source:    SourceRef{Path: src.Path, Mod: src.Mod, Archive: src.Archive, Line: o.Line},
			}
			if o.Certainty == mcm.Exact {
				opt.Text = s.tr.Text(o.Label, prefer...)
			}
			if o.Type == "text" {
				opt.Value, opt.ValueText, opt.ValueCertainty, opt.ValueChanges = o.Value, o.ValueText, Certainty(o.ValueCertainty), o.ValueChanges
				if o.ValueCertainty == mcm.Exact {
					opt.ValueText = s.tr.Text(o.Value, prefer...)
				}
			}
			if o.Repeats {
				opt.Notes = append(opt.Notes, Note{Kind: NoteKindRepeats})
			}
			if o.Range != nil {
				opt.Slider = &Range{Min: o.Range.Min, Max: o.Range.Max, Step: o.Range.Step, Format: o.Format}
			}
			if o.Type == "menu" {
				opt.Choices = s.choices(o.Choices, prefer)
			}
			// A text option that cycles through values as it's clicked is a
			// stepper, as MCM Helper's: a recording can only click it.
			if o.Type == "text" && len(o.Cycle) > 0 {
				opt.Control = "stepper"
				opt.Choices = s.choices(o.Cycle, prefer)
				opt.Notes = append(opt.Notes, Note{Kind: NoteKindStepper})
			}
			if o.Default != nil {
				opt.Default = &Value{Kind: ValueKindNumber, Number: *o.Default}
			}
			opt.Start = startValue(opt, o)
			switch {
			case opt.Certainty == CertaintyUnknown:
				m.Certainty = CertaintyUnknown
			case opt.Certainty == CertaintyPattern && m.Certainty == CertaintyExact:
				m.Certainty = CertaintyPattern
			}
			page.Options = append(page.Options, opt)
		}
		numberDuplicates(page.Options)
		m.Pages = append(m.Pages, page)
	}
	markLanding(&m)
	return m
}

// markLanding finds the named page a menu's default page draws exactly:
// every option of the same type and label, at the same position, added by
// the same line of the same script, and the same headers and gaps. Labels
// alone aren't enough: two pages of a left and a right side can share them.
func markLanding(m *Menu) {
	if len(m.Pages) < 2 || m.Pages[0].Name != "" || len(m.Pages[0].Options) == 0 {
		return
	}
	landing := &m.Pages[0]
	for i := 1; i < len(m.Pages); i++ {
		if p := &m.Pages[i]; samePage(*landing, *p) {
			landing.SameAs, p.Landing = p.Name, true
			return
		}
	}
}

func samePage(a, b Page) bool {
	if len(a.Options) != len(b.Options) || len(a.Fixtures) != len(b.Fixtures) {
		return false
	}
	for i, o := range a.Options {
		p := b.Options[i]
		if o.Type != p.Type || o.Label != p.Label || o.Index != p.Index || o.State != p.State || o.Position != p.Position ||
			o.Source.Path != p.Source.Path || o.Source.Line != p.Source.Line || o.Source.Line == 0 {
			return false
		}
	}
	for i, f := range a.Fixtures {
		g := b.Fixtures[i]
		if f.Kind != g.Kind || f.Label != g.Label || f.Position != g.Position {
			return false
		}
	}
	return true
}

// empty reports whether a menu has neither a name of its own nor an
// option: an old script left on a quest beside the real menu.
func empty(m Menu) bool {
	for _, p := range m.Pages {
		if len(p.Options) > 0 {
			return false
		}
	}
	for _, n := range m.Notes {
		if n.Kind == NoteKindNameFromScript {
			return true
		}
	}
	return false
}

// startValue is what an option shows on a new game, in the catalog's
// terms: on or off for a toggle, a number for a slider, a key, or a color,
// the entry for a menu (by position, as a menu's default is), and text for
// an input. A menu entry the menu doesn't list isn't one a step can hold.
func startValue(opt Option, o mcm.Option) *Value {
	switch {
	case o.StartNumber != nil && opt.Type == OptionTypeToggle:
		return &Value{Kind: ValueKindBool, Bool: *o.StartNumber != 0}
	case o.StartNumber != nil && (opt.Type == OptionTypeSlider || opt.Type == OptionTypeKeymap || opt.Type == OptionTypeColor):
		return &Value{Kind: ValueKindNumber, Number: *o.StartNumber}
	case o.StartText != nil && opt.Type == OptionTypeMenu:
		for i, c := range opt.Choices {
			if strings.EqualFold(c.Raw, *o.StartText) {
				return &Value{Kind: ValueKindNumber, Number: float64(i)}
			}
		}
	case o.StartText != nil && opt.Type == OptionTypeInput:
		return &Value{Kind: ValueKindString, String: *o.StartText}
	}
	return nil
}

// properties is a quest script's property values as the analysis takes
// them: an object property pointing to a global variable takes the value a
// new game starts the global at.
func (s *scan) properties(a attachment) []mcm.Property {
	var out []mcm.Property
	for _, p := range a.props {
		mp := mcm.Property{Name: p.Name}
		switch p.Type {
		case plugin.PropertyObject:
			g, ok := s.globals[a.objects[strings.ToLower(p.Name)]]
			if !ok {
				continue
			}
			v := g.value
			mp.Global = &v
		case plugin.PropertyString:
			s := p.String
			mp.String = &s
		case plugin.PropertyStringArray:
			mp.Strings = append([]string{}, p.Strings...)
		case plugin.PropertyInt, plugin.PropertyFloat, plugin.PropertyBool:
			n := p.Number
			mp.Number = &n
		default:
			continue
		}
		out = append(out, mp)
	}
	return out
}

// sortedKeys is a map's keys in order, for a scan that comes out the same
// every time.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Scripts is every compiled script in the list that makes an MCM, by its
// Extends chain, and the scripts on the chains between them and the MCM
// bases (a framework's), by lowercase name: what a snapshot must hold for
// the menus to read. The bases themselves aren't included.
func Scripts(l *modlist.List) map[string]bool {
	l.Index(modlist.MCMScope)
	s := &scan{list: l, cat: &Catalog{}, ids: map[string]bool{}}
	s.scripts()
	out := map[string]bool{}
	for k := range s.compiled {
		if s.base(k) == "" {
			continue
		}
		for name := k; name != "" && name != "ski_configbase" && name != "mcm_configbase" && !out[name]; {
			out[name] = true
			parent, ok := s.parent(name)
			if !ok {
				break
			}
			name = ir.Key(parent)
		}
	}
	return out
}
