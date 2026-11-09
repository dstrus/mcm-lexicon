package catalog

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"lexicon/internal/mcmhelper"
	"lexicon/internal/modlist"
	"lexicon/internal/papyrus/mcm"
	"lexicon/internal/plugin"
	"lexicon/internal/translations"
)

type Options struct {
	Language string // translations to read, as their file names spell it; ENGLISH by default
	Version  string // MCM Lexicon's, for the catalog's ListInfo
	Now      time.Time
	// CacheDir is where the scan keeps what it learns of the list's
	// scripts for the next one (DefaultCacheDir); "" keeps nothing.
	CacheDir string
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
	s := &scan{list: l, cat: c, tr: loadTranslations(l, c.List.Language), ids: map[string]bool{}, cache: loadParents(o.CacheDir, l.Source.Path)}
	for _, p := range l.ArchiveProblems() {
		s.skip(p.File, SkipReasonUnreadable, p.Detail)
	}
	s.scripts()
	s.cache.save()
	s.plugins()
	s.mcmHelper()
	s.papyrus()
	plainText(c)
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

	classic    []script                // scripts whose Extends chain reaches SKI_ConfigBase
	helpers    map[string][]string     // MCM Helper's scripts (whose chain reaches MCM_ConfigBase), by mod
	quests     map[string][]attachment // quests by the lowercase name of a script attached to them
	sources    map[string][]modlist.File
	compiled   map[string]modlist.File
	forms      map[string]*form
	cache      *parents
	globals    map[string]global     // global variables by form key (plugin.FormKey)
	questForms map[string]questForm  // quests with scripts attached, by form key (plugin.FormKey)
	settings   mcm.Settings          // what PluginINIs' files hold, once read
	remotes    map[string]mcm.Remote // what each quest's scripts hold on a new game, once worked out
}

// questForm is a quest as the plugin that wins defines it, with the
// scripts attached to it: what another script's form property points to.
type questForm struct {
	scripts []attachedScript
	order   int
	active  bool
}

// attachedScript is a script on a quest, with what the plugin fills its
// properties with, and each object property's form.
type attachedScript struct {
	name    string
	props   []plugin.Property
	objects map[string]string
}

// global is a global variable's value as the plugin that wins defines it.
type global struct {
	value  float64
	order  int
	active bool
}

// attachment is a quest in a plugin with a script attached, and what the
// script's properties hold.
type attachment struct {
	plugin  string
	order   int // the plugin's place in plugins.txt; later wins
	active  bool
	quest   string
	modName string
	pages   []string
	props   []plugin.Property
	objects map[string]string // each object property's form, by lowercase name: its form key
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
		if b, err := f.Read(); err == nil {
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
		b, err := f.Read()
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
	plugin, guessed := "", false
	for _, script := range s.helpers[f.Mod] {
		if a, ok := s.quest(script); ok {
			plugin = a.plugin
			break
		}
	}
	if plugin == "" {
		plugin, guessed = s.pluginOf(f.Mod, cfg.ModName, id), true
	}
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
		Source:    SourceRef{Path: f.Path, Mod: f.Mod, Archive: f.Archive},
	}
	if plugin == "" {
		m.Notes = append(m.Notes, Note{Kind: NoteKindNoPlugin, Detail: f.Mod})
	} else if guessed {
		m.Notes = append(m.Notes, Note{Kind: NoteKindPluginGuessed, Detail: plugin})
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
		page := Page{Name: p.Name, Text: s.tr.Text(p.Name, prefer...), Options: []Option{}, Fixtures: []Fixture{}, Placed: true}
		// SkyUI's cursor, as MCM Helper drives it: each control takes the
		// next slot, one on in left-to-right fill or two on in top-to-bottom,
		// and a control's position moves the cursor first.
		cursor, step := 0, 1
		if p.TopToBottom {
			step = 2
		}
		place := func(at *int) int {
			if at != nil && *at >= 0 && *at < 128 {
				cursor = *at
			}
			if cursor < 0 {
				return -1
			}
			pos := cursor
			cursor += step
			if cursor >= 128 {
				cursor = -1
			}
			return pos
		}
		for _, ctl := range p.Controls {
			switch ctl.Type {
			case "header":
				page.Fixtures = append(page.Fixtures, Fixture{Kind: FixtureKindHeader, Label: ctl.Text, Text: s.tr.Text(ctl.Text, prefer...), Certainty: CertaintyExact, Position: place(ctl.Position)})
				continue
			case "empty":
				page.Fixtures = append(page.Fixtures, Fixture{Kind: FixtureKindEmpty, Certainty: CertaintyExact, Position: place(ctl.Position)})
				continue
			}
			if o, ok := s.helperOption(ctl, defaults, saved, prefer); ok {
				o.Source = m.Source
				o.Position = place(ctl.Position)
				if o.Position < 0 {
					page.Placed = false
				}
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
		// A text control does something on a click only through its action.
		o.Inert = c.Action == ""
		// A fixed value is the config's; one read from a setting shows
		// whatever the setting holds, which changes.
		o.ValueCertainty = CertaintyExact
		if v, ok := c.Value.(string); ok && c.SourceType == "" {
			o.Value = v
			o.ValueText = s.tr.Text(v, prefer...)
		} else if c.SourceType != "" {
			o.ValueChanges = true
			if v, ok := c.Default.(string); ok {
				o.Value, o.ValueText = v, s.tr.Text(v, prefer...)
			}
		}
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
	// A right-side click counts the text options showing its value.
	values := map[string]int{}
	for _, o := range opts {
		if o.Type == OptionTypeText && o.Value != "" {
			values[strings.ToLower(o.Value)]++
		}
	}
	shown := map[string]int{}
	for i := range opts {
		k := strings.ToLower(opts[i].Value)
		if opts[i].Type == OptionTypeText && opts[i].Value != "" && values[k] > 1 {
			shown[k]++
			opts[i].ValueIndex = shown[k]
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
	b, err := f.Read()
	if err != nil {
		return mcmhelper.INI{}
	}
	return mcmhelper.ParseINI(b)
}

// plugins reads every plugin the game sees for the quests scripts are
// attached to. The game's own masters hold no MCMs, so they're skipped.
func (s *scan) plugins() {
	s.quests = map[string][]attachment{}
	s.globals = map[string]global{}
	s.questForms = map[string]questForm{}
	s.remotes = map[string]mcm.Remote{}
	order := map[string]int{}
	for i, p := range s.list.Plugins() {
		order[strings.ToLower(p.Name)] = i
	}
	for _, f := range s.list.Files("") {
		if strings.Contains(f.Path, "/") || !modlist.IsPlugin(f.Path) || modlist.CoreMaster(f.Path) {
			continue
		}
		p, err := plugin.ReadFile(f.Abs)
		if err != nil {
			s.skip(f, SkipReasonUnreadable, err.Error())
			continue
		}
		at, listed := order[strings.ToLower(f.Path)]
		if !listed {
			at = -1
		}
		active := s.list.PluginActive(f.Path)
		// A global variable, or a later plugin's override of it: the one in
		// an active plugin over an inactive one, then the one loaded last,
		// as the game loads them.
		for _, g := range p.Globals {
			key := plugin.FormKey(p.Owner(g.FormID), g.FormID)
			if had, ok := s.globals[key]; !ok || (active && !had.active) || (active == had.active && at > had.order) {
				s.globals[key] = global{value: g.Value, order: at, active: active}
			}
		}
		for _, q := range p.Quests {
			// The quest by its form, the one that wins as the game loads
			// plugins, for a form property elsewhere that points to it.
			key := plugin.FormKey(p.Owner(q.FormID), q.FormID)
			if had, ok := s.questForms[key]; !ok || (active && !had.active) || (active == had.active && at > had.order) {
				qf := questForm{order: at, active: active}
				for _, sc := range q.Scripts {
					as := attachedScript{name: sc.Name, props: sc.Properties, objects: map[string]string{}}
					for _, prop := range sc.Properties {
						if prop.Type == plugin.PropertyObject {
							id := uint32(prop.Number)
							as.objects[strings.ToLower(prop.Name)] = plugin.FormKey(p.Owner(id), id)
						}
					}
					qf.scripts = append(qf.scripts, as)
				}
				s.questForms[key] = qf
			}
			for _, sc := range q.Scripts {
				a := attachment{plugin: f.Path, order: at, active: s.list.PluginActive(f.Path), quest: q.EditorID}
				if v, ok := sc.Property("ModName"); ok && v.Type == plugin.PropertyString {
					a.modName = v.String
				}
				if v, ok := sc.Property("Pages"); ok && v.Type == plugin.PropertyStringArray {
					a.pages = v.Strings
				}
				a.props = sc.Properties
				a.objects = map[string]string{}
				for _, prop := range sc.Properties {
					if prop.Type == plugin.PropertyObject {
						id := uint32(prop.Number)
						a.objects[strings.ToLower(prop.Name)] = plugin.FormKey(p.Owner(id), id)
					}
				}
				key := strings.ToLower(sc.Name)
				s.quests[key] = append(s.quests[key], a)
			}
		}
	}
}

// quest is the quest the game runs script on: one in an active plugin over
// an inactive one, then the plugin loaded last.
func (s *scan) quest(script string) (attachment, bool) {
	var best attachment
	found := false
	for _, a := range s.quests[strings.ToLower(script)] {
		if !found || (a.active && !best.active) || (a.active == best.active && a.order > best.order) {
			best, found = a, true
		}
	}
	return best, found
}

func (s *scan) skip(f modlist.File, reason SkipReason, detail string) {
	s.cat.Skipped = append(s.cat.Skipped, Skipped{Path: f.Path, Mod: f.Mod, Archive: f.Archive, Reason: reason, Detail: detail, Pages: []string{}})
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
