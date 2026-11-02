package catalog

import (
	"os"
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
		f := s.form(k)
		if f == nil {
			return ""
		}
		k = ir.Key(f.script.Parent)
	}
	return ""
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
		var err error
		if compiled, err = pex.ReadFile(c.Abs); err != nil {
			s.skip(c, SkipReasonUnreadable, err.Error())
			compiled = nil
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
	b, err := os.ReadFile(f.Abs)
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
		sk := Skipped{Path: sc.pex.Path, Mod: sc.pex.Mod, Pages: []string{}}
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
	result := mcm.Extract(chain, properties(a.props))
	prefer := []string{strings.TrimSuffix(a.plugin, path.Ext(a.plugin)), sc.name}
	src := SourceRef{Path: f.file.Path, Mod: f.file.Mod}
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
		page := Page{Name: p.Name, Text: s.tr.Text(p.Name, prefer...), Options: []Option{}}
		for _, o := range p.Options {
			opt := Option{
				Type:      OptionType(o.Type),
				Label:     o.Label,
				Text:      o.Text,
				State:     o.State,
				Help:      o.Help,
				HelpText:  s.tr.Text(o.Help, prefer...),
				Certainty: Certainty(o.Certainty),
				Notes:     []Note{},
				Source:    SourceRef{Path: src.Path, Mod: src.Mod, Line: o.Line},
			}
			if o.Certainty == mcm.Exact {
				opt.Text = s.tr.Text(o.Label, prefer...)
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
			if o.Default != nil {
				opt.Default = &Value{Kind: ValueKindNumber, Number: *o.Default}
			}
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
	return m
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

// properties is a quest script's property values as the analysis takes
// them.
func properties(props []plugin.Property) []mcm.Property {
	var out []mcm.Property
	for _, p := range props {
		mp := mcm.Property{Name: p.Name}
		switch p.Type {
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
			f := s.form(name)
			if f == nil {
				break
			}
			name = ir.Key(f.script.Parent)
		}
	}
	return out
}
