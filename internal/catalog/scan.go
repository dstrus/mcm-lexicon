package catalog

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"lexicon/internal/mcmhelper"
	"lexicon/internal/modlist"
	"lexicon/internal/translations"
)

type Options struct {
	Language string // translations to read, as their file names spell it; ENGLISH by default
	Version  string // MCM Lexicon's, for the catalog's ListInfo
	Now      time.Time
}

// Scan reads every menu in the list. Reading a file can fail without the
// scan failing: what can't be read is listed in Skipped, with the reason.
func Scan(l *modlist.List, o Options) *Catalog {
	if o.Language == "" {
		o.Language = "ENGLISH"
	}
	if o.Now.IsZero() {
		o.Now = time.Now()
	}
	l.Index(modlist.MCMScope)
	c := &Catalog{
		SchemaVersion: SchemaVersion,
		List: ListInfo{
			Name:      path.Base(strings.ReplaceAll(l.Source.Path, `\`, "/")),
			Path:      l.Source.Path,
			Kind:      string(l.Source.Kind),
			Profile:   l.Source.Profile,
			Game:      l.Source.Game,
			Language:  strings.ToUpper(o.Language),
			ScannedAt: o.Now.UTC(),
			Version:   o.Version,
		},
		Menus:   []Menu{},
		Skipped: []Skipped{},
	}
	s := &scan{list: l, cat: c, tr: loadTranslations(l, c.List.Language), ids: map[string]bool{}}
	s.mcmHelper()
	s.papyrus()
	sort.SliceStable(c.Menus, func(i, j int) bool {
		return strings.ToLower(c.Menus[i].NameText) < strings.ToLower(c.Menus[j].NameText)
	})
	return c
}

type scan struct {
	list *modlist.List
	cat  *Catalog
	tr   *translations.Table
	ids  map[string]bool
}

// loadTranslations reads every translation file the game would load for
// language.
func loadTranslations(l *modlist.List, language string) *translations.Table {
	t := translations.New()
	for _, f := range l.Files("Interface/Translations") {
		base, lang, ok := translations.NameOf(f.Path)
		if !ok || lang != language || strings.Count(f.Path, "/") != 2 {
			continue
		}
		if b, err := os.ReadFile(f.Abs); err == nil {
			t.Add(base, b)
		}
	}
	return t
}

// mcmHelper reads every MCM/Config/<id>/config.json the game sees.
func (s *scan) mcmHelper() {
	for _, f := range s.list.Files("MCM/Config") {
		parts := strings.Split(f.Path, "/")
		if len(parts) != 4 || !strings.EqualFold(parts[3], "config.json") {
			continue
		}
		id := parts[2]
		b, err := os.ReadFile(f.Abs)
		if err != nil {
			s.skip(f, SkipReasonUnreadable, err.Error())
			continue
		}
		cfg, err := mcmhelper.Parse(b)
		if err != nil {
			s.skip(f, SkipReasonUnreadable, err.Error())
			continue
		}
		s.cat.Menus = append(s.cat.Menus, s.helperMenu(f, id, cfg))
	}
}

func (s *scan) helperMenu(f modlist.File, id string, cfg mcmhelper.Config) Menu {
	plugin := s.pluginOf(f.Mod, cfg.ModName, id)
	prefer := []string{cfg.ModName, id, strings.TrimSuffix(plugin, path.Ext(plugin))}
	m := Menu{
		ID:        s.id("mcmhelper", id),
		Name:      cfg.DisplayName,
		NameText:  s.tr.Text(cfg.DisplayName, prefer...),
		Kind:      KindMCMHelper,
		Mod:       f.Mod,
		Plugin:    plugin,
		Active:    plugin == "" || s.list.PluginActive(plugin),
		Script:    id,
		Certainty: CertaintyExact,
		Notes:     []Note{},
		Source:    SourceRef{Path: f.Path, Mod: f.Mod},
	}
	if plugin == "" {
		m.Notes = append(m.Notes, Note{Kind: NoteKindNoPlugin, Detail: f.Mod})
	}
	if copies := s.list.Copies(f.Path); len(copies) > 1 {
		for _, lost := range copies[1:] {
			m.Notes = append(m.Notes, Note{Kind: NoteKindOverridden, Detail: lost.Mod})
		}
	}
	// The mod's defaults, and the player's own values.
	defaults := s.ini("MCM/Config/" + id + "/settings.ini")
	saved := s.ini("MCM/Settings/" + cfg.ModName + ".ini")
	if cfg.ModName == "" {
		saved = s.ini("MCM/Settings/" + id + ".ini")
	}
	for _, p := range cfg.Pages {
		page := Page{Name: p.Name, Text: s.tr.Text(p.Name, prefer...), Options: []Option{}}
		for _, ctl := range p.Controls {
			if o, ok := s.helperOption(ctl, defaults, saved, prefer); ok {
				o.Source = m.Source
				page.Options = append(page.Options, o)
			}
		}
		numberDuplicates(page.Options)
		m.Pages = append(m.Pages, page)
	}
	return m
}

// helperOption is the option a control records as, or ok=false for a
// control a recording can't touch (headers, spacers, hidden toggles).
func (s *scan) helperOption(c mcmhelper.Control, defaults, saved mcmhelper.INI, prefer []string) (Option, bool) {
	o := Option{
		Label:     c.Text,
		Text:      s.tr.Text(c.Text, prefer...),
		Setting:   c.ID,
		Help:      c.Help,
		HelpText:  s.tr.Text(c.Help, prefer...),
		Certainty: CertaintyExact,
		Notes:     []Note{},
	}
	switch c.Type {
	case "toggle":
		o.Type = OptionTypeToggle
	case "slider":
		o.Type = OptionTypeSlider
		if c.Min != nil && c.Max != nil {
			o.Slider = &Range{Min: *c.Min, Max: *c.Max, Step: deref(c.Step, 1), Format: c.Format}
		}
	case "keymap":
		o.Type = OptionTypeKeymap
	case "color":
		o.Type = OptionTypeColor
	case "input":
		o.Type = OptionTypeInput
	case "enum":
		o.Type = OptionTypeMenu
		o.Control = "enum"
		o.Choices = s.choices(c.Options, prefer)
	case "stepper":
		o.Type = OptionTypeText
		o.Control = "stepper"
		o.Choices = s.choices(c.Options, prefer)
		o.Notes = append(o.Notes, Note{Kind: NoteKindStepper})
	case "text":
		o.Type = OptionTypeText
	default:
		return Option{}, false
	}
	if c.Default != nil {
		o.Default = valueOf(o.Type, c.Default)
	} else if c.ModSetting() && c.Key != "" {
		if v, ok := defaults.Get(c.Section, c.Key); ok {
			o.Default = valueOf(o.Type, v)
		}
	}
	if c.ModSetting() && c.Key != "" {
		if v, ok := saved.Get(c.Section, c.Key); ok {
			o.Saved = valueOf(o.Type, v)
		}
	}
	return o, true
}

func (s *scan) choices(raw []string, prefer []string) []Choice {
	out := make([]Choice, len(raw))
	for i, r := range raw {
		out[i] = Choice{Raw: r, Text: s.tr.Text(r, prefer...)}
	}
	return out
}

// valueOf reads v (from JSON or an ini file) as a value of type t. A menu's
// or stepper's value is its choice's position, so it's a number.
func valueOf(t OptionType, v any) *Value {
	text := strings.TrimSpace(fmt.Sprint(v))
	switch t {
	case OptionTypeToggle:
		switch x := v.(type) {
		case bool:
			return &Value{Kind: ValueKindBool, Bool: x}
		case float64:
			return &Value{Kind: ValueKindBool, Bool: x != 0}
		}
		switch strings.ToLower(text) {
		case "1", "true", "on":
			return &Value{Kind: ValueKindBool, Bool: true}
		case "0", "false", "off":
			return &Value{Kind: ValueKindBool, Bool: false}
		}
		return nil
	case OptionTypeInput:
		return &Value{Kind: ValueKindString, String: text}
	}
	if x, ok := v.(float64); ok {
		return &Value{Kind: ValueKindNumber, Number: x}
	}
	if n, err := strconv.ParseFloat(text, 64); err == nil {
		return &Value{Kind: ValueKindNumber, Number: n}
	}
	if strings.HasPrefix(strings.ToLower(text), "0x") {
		if n, err := strconv.ParseInt(text[2:], 16, 64); err == nil {
			return &Value{Kind: ValueKindNumber, Number: float64(n)}
		}
	}
	if t == OptionTypeText {
		return &Value{Kind: ValueKindString, String: text}
	}
	return nil
}

// numberDuplicates sets Index on options that share a type and label on
// their page, counting from 1 in the order they were added, as MCM
// Recorder does.
func numberDuplicates(opts []Option) {
	count := map[string]int{}
	for _, o := range opts {
		count[string(o.Type)+"\x00"+o.Label]++
	}
	seen := map[string]int{}
	for i := range opts {
		k := string(opts[i].Type) + "\x00" + opts[i].Label
		if count[k] > 1 {
			seen[k]++
			opts[i].Index = seen[k]
		}
	}
}

// pluginOf is the plugin a mod's menu most likely belongs to: the mod's
// plugin named like the menu, or its only plugin, or its first. Reading
// which quest holds the menu's script comes with reading plugins.
func (s *scan) pluginOf(mod string, names ...string) string {
	var plugins []string
	for _, f := range s.list.Files("") {
		if f.Mod == mod && !strings.Contains(f.Path, "/") && modlist.IsPlugin(f.Path) {
			plugins = append(plugins, f.Path)
		}
	}
	for _, n := range names {
		for _, p := range plugins {
			if n != "" && strings.EqualFold(strings.TrimSuffix(p, path.Ext(p)), n) {
				return p
			}
		}
	}
	if len(plugins) > 0 {
		return plugins[0]
	}
	return ""
}

func (s *scan) ini(rel string) mcmhelper.INI {
	f, ok := s.list.Open(rel)
	if !ok {
		return mcmhelper.INI{}
	}
	b, err := os.ReadFile(f.Abs)
	if err != nil {
		return mcmhelper.INI{}
	}
	return mcmhelper.ParseINI(b)
}

// papyrus lists the menus defined by Papyrus scripts, which aren't read
// yet, so the catalog says how many it's missing. A compiled script names
// the script it extends in its string table; MCM Helper's own scripts
// extend MCM_ConfigBase, and their config.json covers them.
func (s *scan) papyrus() {
	marker, helper := []byte("ski_configbase"), []byte("mcm_configbase")
	for _, f := range s.list.Files("Scripts") {
		if strings.Count(f.Path, "/") != 1 || !strings.EqualFold(path.Ext(f.Path), ".pex") {
			continue
		}
		base := strings.ToLower(strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path)))
		if base == "ski_configbase" || base == "mcm_configbase" {
			continue
		}
		b, err := os.ReadFile(f.Abs)
		if err != nil {
			continue
		}
		lower := bytes.ToLower(b)
		if bytes.Contains(lower, marker) && !bytes.Contains(lower, helper) {
			s.skip(f, SkipReasonPapyrus, "")
		}
	}
}

func (s *scan) skip(f modlist.File, reason SkipReason, detail string) {
	s.cat.Skipped = append(s.cat.Skipped, Skipped{Path: f.Path, Mod: f.Mod, Reason: reason, Detail: detail})
}

// id is a menu's ID: its kind and name, lowercased with anything but
// letters and digits as dashes, unique within the catalog.
func (s *scan) id(kind, name string) string {
	var b strings.Builder
	b.WriteString(kind + "-")
	dash := false
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	id := strings.TrimRight(b.String(), "-")
	for n := 2; s.ids[id]; n++ {
		id = fmt.Sprintf("%s-%d", strings.TrimRight(b.String(), "-"), n)
	}
	s.ids[id] = true
	return id
}

func deref(p *float64, fallback float64) float64 {
	if p == nil {
		return fallback
	}
	return *p
}
