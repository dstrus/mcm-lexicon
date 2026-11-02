package catalog

import (
	"bytes"
	"os"
	"path"
	"sort"
	"strings"

	"lexicon/internal/modlist"
	"lexicon/internal/papyrus/ir"
	"lexicon/internal/papyrus/mcm"
	"lexicon/internal/papyrus/source"
	"lexicon/internal/plugin"
)

// script is a compiled script the game runs, and its source when the mod
// ships it.
type script struct {
	name   string // as the compiled file is named
	pex    modlist.File
	source *parsedSource
}

type parsedSource struct {
	file   modlist.File
	script *ir.Script
}

// scripts finds the compiled scripts that make MCMs, by following each
// one's Extends chain: to SKI_ConfigBase for a classic MCM, to MCM_ConfigBase
// for MCM Helper's (whose config.json defines the menu). The chain is read
// from source where a mod ships it; where it doesn't, a compiled script's
// string table names the script it extends, so a mention of either base
// stands in until compiled scripts are read.
func (s *scan) scripts() {
	s.helpers = map[string][]string{}
	s.sources = map[string][]modlist.File{}
	s.parsed = map[string]*parsedSource{}
	for _, f := range s.list.Files("") {
		if strings.EqualFold(path.Ext(f.Path), ".psc") {
			k := ir.Key(strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path)))
			s.sources[k] = append(s.sources[k], f)
		}
	}
	compiled := map[string]modlist.File{}
	for _, f := range s.list.Files("Scripts") {
		if strings.Count(f.Path, "/") == 1 && strings.EqualFold(path.Ext(f.Path), ".pex") {
			compiled[ir.Key(strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path)))] = f
		}
	}
	for _, k := range sortedKeys(compiled) {
		if k == "ski_configbase" || k == "mcm_configbase" {
			continue
		}
		f := compiled[k]
		switch s.base(k, f.Mod, compiled) {
		case "ski_configbase":
			s.classic = append(s.classic, script{name: strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path)), pex: f, source: s.source(k, f.Mod)})
		case "mcm_configbase":
			s.helpers[f.Mod] = append(s.helpers[f.Mod], k)
		}
	}
}

// base follows a script's Extends chain and returns the MCM base it
// reaches, ski_configbase or mcm_configbase, or "" for neither.
func (s *scan) base(name, mod string, compiled map[string]modlist.File) string {
	seen := map[string]bool{}
	for k := name; k != "" && !seen[k]; {
		seen[k] = true
		if k == "ski_configbase" || k == "mcm_configbase" {
			return k
		}
		if src := s.source(k, mod); src != nil {
			k = ir.Key(src.script.Parent)
			continue
		}
		f, ok := compiled[k]
		if !ok {
			return ""
		}
		b, err := os.ReadFile(f.Abs)
		if err != nil {
			return ""
		}
		lower := bytes.ToLower(b)
		switch {
		case bytes.Contains(lower, []byte("mcm_configbase")):
			return "mcm_configbase"
		case bytes.Contains(lower, []byte("ski_configbase")):
			return "ski_configbase"
		}
		return ""
	}
	return ""
}

// source is a script's parsed source, preferring the copy in the mod
// whose compiled script the game runs.
func (s *scan) source(name, mod string) *parsedSource {
	files := s.sources[name]
	if len(files) == 0 {
		return nil
	}
	f := files[0]
	for _, c := range files {
		if c.Mod == mod {
			f = c
			break
		}
	}
	key := strings.ToLower(f.Abs)
	if p, ok := s.parsed[key]; ok {
		return p
	}
	var p *parsedSource
	if b, err := os.ReadFile(f.Abs); err == nil {
		r := source.Parse(string(b))
		if r.Script.Name != "" {
			p = &parsedSource{file: f, script: r.Script}
		}
	}
	s.parsed[key] = p
	return p
}

// papyrus reads the classic MCMs: each script on a quest, from its source,
// into a menu. One with no source waits for compiled scripts to be read;
// one on no quest isn't a menu the game shows.
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
		switch {
		case !attached:
			sk.Reason = SkipReasonNoQuest
			s.cat.Skipped = append(s.cat.Skipped, sk)
		case sc.source == nil:
			sk.Reason = SkipReasonCompiled
			s.cat.Skipped = append(s.cat.Skipped, sk)
		default:
			m := s.sourceMenu(sc, a)
			if empty(m) {
				sk.Reason = SkipReasonEmpty
				s.cat.Skipped = append(s.cat.Skipped, sk)
				continue
			}
			s.cat.Menus = append(s.cat.Menus, m)
		}
	}
}

// sourceMenu reads a classic MCM from its source and its ancestors'.
func (s *scan) sourceMenu(sc script, a attachment) Menu {
	chain := []*ir.Script{sc.source.script}
	for parent := sc.source.script.Parent; len(chain) < 8; {
		k := ir.Key(parent)
		if k == "" || k == "ski_configbase" {
			break
		}
		p := s.source(k, sc.pex.Mod)
		if p == nil {
			break
		}
		chain = append(chain, p.script)
		parent = p.script.Parent
	}
	result := mcm.Extract(chain, properties(a.props))
	prefer := []string{strings.TrimSuffix(a.plugin, path.Ext(a.plugin)), sc.name}
	src := SourceRef{Path: sc.source.file.Path, Mod: sc.source.file.Mod}
	m := Menu{
		ID:        s.id("source", sc.name),
		Name:      result.ModName,
		Kind:      KindSource,
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
			if o.Range != nil {
				opt.Slider = &Range{Min: o.Range.Min, Max: o.Range.Max, Step: o.Range.Step, Format: o.Format}
			}
			if o.Type == "menu" {
				opt.Choices = s.choices(o.Choices, prefer)
			}
			if o.Default != nil {
				opt.Default = &Value{Kind: ValueKindNumber, Number: *o.Default}
			}
			if opt.Certainty != CertaintyExact && m.Certainty == CertaintyExact {
				m.Certainty = opt.Certainty
			} else if opt.Certainty == CertaintyUnknown {
				m.Certainty = CertaintyUnknown
			}
			page.Options = append(page.Options, opt)
		}
		numberDuplicates(page.Options)
		m.Pages = append(m.Pages, page)
	}
	return m
}

// empty reports whether a menu read from source has neither a name of its
// own nor an option: an old script left on a quest beside the real menu.
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
