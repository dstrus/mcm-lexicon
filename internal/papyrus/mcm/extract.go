// Package mcm reads an MCM's menu from its Papyrus script, in the shared
// intermediate form either front end produces: its display name, its pages,
// and on each page the options a recording can set, with their ranges,
// choices, defaults, and help where the script states them.
//
// It runs the script's initialization (OnInit, OnConfigInit and the rest,
// starting from the property values the plugin fills), then OnPageReset
// once for each page with the page known, so conditions on the page decide
// which options belong to it. Then it runs the option events (OnOptionSlider-
// Open and the others, or a state's OnSliderOpenST) on each page, linking
// the dialog calls in them to options by the option IDs the page's Add…Option
// calls returned, or by state.
package mcm

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"

	"lexicon/internal/papyrus/ir"
)

// Certainty is how sure the analysis is of an option's label.
type Certainty string

const (
	Exact   Certainty = "exact"
	Pattern Certainty = "pattern"
	Unknown Certainty = "unknown"
)

// Property is a value the plugin fills in the script's properties.
type Property struct {
	Name    string
	String  *string
	Strings []string // nil unless the property is a string array
	Number  *float64
	Global  *float64 // a global variable the property points to: what a new game starts it at
	Remote  Remote   // a form the property points to, with scripts attached: what their properties hold
}

// Remote is what the scripts attached to a form hold on a new game, by
// script name, then property name, both lowercased: what the plugin fills,
// else what the script declares, for a property its OnInit leaves alone.
type Remote map[string]map[string]RemoteValue

// RemoteValue is one of those properties' values: a number (a bool as 0 or
// 1), text, or a global variable's starting value.
type RemoteValue struct {
	Number *float64
	String *string
	Global *float64
}

type Menu struct {
	ModName      string // the display name the script or plugin sets; "" when neither does
	Pages        []Page // in order; a menu without pages has one, named ""
	PagesGuessed bool   // Pages wasn't readable, so the pages are the names OnPageReset compares against
}

type Page struct {
	Name    string // raw; "" for the default page
	Options []Option
	// Fixtures are what the page shows besides options: headers and gaps.
	Fixtures []Fixture
	// Placed is true when every option and fixture has a known position,
	// so the page can be drawn as the game draws it.
	Placed bool
}

// Fixture is a header or an empty slot on a page.
type Fixture struct {
	Kind      string // header, empty
	Label     string // a header's text, raw
	Text      string
	Certainty Certainty
	Position  int
	Line      int
}

// SkyUI's cursor: positions 0 to 127 are 64 rows of two columns (even on
// the left, odd on the right). Each page starts at 0, filling left to
// right; a script may move the cursor or fill top to bottom instead.
const (
	positions   = 128
	leftToRight = 1
	topToBottom = 2
)

type Option struct {
	Type      string // toggle, slider, menu, keymap, color, input, text
	Label     string // the selector: exact, a *pattern*, or "" when unknown
	Text      string // the label for reading: exact, or with … for runtime parts
	Certainty Certainty
	State     string // for an ST option
	Format    string // a slider's display format, from AddSliderOption
	Range     *Range
	Choices   []string // a menu's entries, raw
	Default   *float64 // a slider's default value, or a menu's default choice by position
	// Start is the value the option is shown with on a new game: what the
	// page passes Add…Option once the script's initialization has run,
	// when that's known. A number for a toggle (0 or 1), a slider, a key,
	// or a color; text for a menu's entry or an input.
	StartNumber *float64
	StartText   *string

	site     *ir.Call   // the Add…Option call that added it
	calls    []*ir.Call // the calls into the script's own functions it was added through
	Help     string
	Line     int  // where it's added
	Position int  // SkyUI's cursor position when it was added; -1 when the script decides it at runtime
	Repeats  bool // added again and again by one call in a loop the game decides, with the same label
	// Inert is true for a text option nothing happens to when it's clicked:
	// the script's click handler has no effect in any branch for it, nor
	// one for every option. A recording's click on it does nothing.
	Inert bool
	// Disabled is whether SkyUI draws the option grayed out, so a player
	// can't select it (MCM Recorder can: it calls the mod's handler without
	// looking at the flags): "always" when it's added with
	// OPTION_FLAG_DISABLED and nothing changes that, "sometimes" when its
	// flags come from a value the game decides or SetOptionFlags changes
	// them, "" when it's never disabled.
	Disabled string
	// A text option's value, shown on its right, which a click on that side
	// matches: as the label is described, from AddTextOption's second
	// argument. Changes is true when the menu shows another value at times
	// (SetTextOptionValue, or the same option added with another value).
	Value          string
	ValueText      string
	ValueCertainty Certainty
	ValueChanges   bool
	// Cycle is the values a text option shows in turn as it's clicked, in
	// the order its click handler sets them, when the handler sets them
	// from literal strings: a stepper made of a text option. Nil when the
	// option doesn't cycle, or the values can't be read.
	Cycle []string
}

// shown is a literal value a click handler gives a text option: for the
// options a branch names, or for whichever option the handler's If chain
// is about when it dispatches some other way, such as by the option's name.
type shown struct {
	ids   []int  // the options the branch is scoped to; nil when unscoped
	chain *ir.If // the If chain (an If with its ElseIfs) the call sits in; nil at the top
	arm   string // which arm of it
	text  string
}

// chainOf is the If chain the innermost step of path belongs to, as its
// top If, and the arm as a key. A compiled script's ElseIf is an Else
// holding an If, so the chain is followed up through those.
func chainOf(path []step) (*ir.If, string) {
	var ifs []step
	for _, s := range path {
		if s.at != nil {
			ifs = append(ifs, s)
		}
	}
	path = ifs
	if len(path) == 0 {
		return nil, ""
	}
	i := len(path) - 1
	arm := fmt.Sprintf("%p:%d", path[i].at, path[i].arm)
	for i > 0 && path[i-1].arm == len(path[i-1].at.Branches) && continuation(path[i-1].at.Else) == path[i].at {
		i--
	}
	return path[i].at, arm
}

type Range struct {
	Min, Max, Step float64
}

// optionFlagDisabled is SkyUI's OPTION_FLAG_DISABLED bit.
const optionFlagDisabled = 1

// disabledBy is what an option's flags argument says about it being
// disabled: "always" or "" for a known value, "sometimes" for one the
// game decides.
func disabledBy(flags Value) string {
	if !flags.numeric() {
		return "sometimes"
	}
	if int(flags.n)&optionFlagDisabled != 0 {
		return "always"
	}
	return ""
}

// The Add…Option functions and the option types they make.
var adders = map[string]string{
	"addtoggleoption": "toggle", "addslideroption": "slider", "addmenuoption": "menu",
	"addtextoption": "text", "addkeymapoption": "keymap", "addcoloroption": "color", "addinputoption": "input",
}

// The init events, in the order the analysis runs them; a later one's
// assignments win.
var initEvents = []string{"OnInit", "OnConfigInit", "OnConfigRegister", "OnVersionUpdate", "OnGameReload"}

// The functions that register for an update, and the event each brings.
var updates = map[string]string{
	"registerforsingleupdate": "OnUpdate", "registerforupdate": "OnUpdate",
	"registerforsingleupdategametime": "OnUpdateGameTime", "registerforupdategametime": "OnUpdateGameTime",
}

// The option events, and the state events that do the same for an ST option.
// The events read for what they say about options: their dialogs, help,
// and defaults, and (OnOptionSelect, OnSelectST) whether a click changes a
// text option's value.
var optionEvents = []string{"OnOptionSliderOpen", "OnOptionMenuOpen", "OnOptionHighlight", "OnOptionDefault", "OnOptionSelect"}
var stateEvents = []string{"OnSliderOpenST", "OnMenuOpenST", "OnHighlightST", "OnDefaultST", "OnSelectST"}

// budget is how many statements one run may take, so a pathological script
// can't stall a scan.
const budget = 200_000

// found is an option as the walk finds it, with what links to it.
type found struct {
	Option
	id   int
	path []step
	site *ir.Call // the Add…Option call that made it
}

// declaredType is the type a property is declared with in the chain,
// lowercased: for a form, the script it's taken as.
func declaredType(chain []*ir.Script, name string) string {
	for _, sc := range chain {
		for _, v := range sc.Properties {
			if ir.Key(v.Name) == ir.Key(name) {
				return strings.ToLower(v.Type)
			}
		}
	}
	return ""
}

// Settings is what SKSE plugins' functions return for a setting from an
// INI file the plugin keeps: by "script.function" (VRIK.VrikGetSetting),
// lowercased, then the setting's name, lowercased, or for a function that
// takes two names (MCM Helper's GetModSetting: a mod, then "key:Section"),
// both, joined by "|".
type Settings map[string]map[string]float64

// Extract reads the menu a script defines. chain is the script, then the
// ancestors whose source is at hand (SKI_ConfigBase itself excluded).
func Extract(chain []*ir.Script, props []Property) Menu {
	return ExtractWith(chain, props, nil)
}

// ExtractWith is Extract with the settings SKSE plugins' INI files hold.
// setup is a machine for the script, its variables and properties at their
// declared values, then the plugin's.
func setup(chain []*ir.Script, props []Property, settings Settings) *machine {
	m := newMachine(chain)
	m.settings = settings
	// Script variables and properties start at their declared values (the
	// root's first, so a child's declaration wins), then the plugin's.
	for i := len(chain) - 1; i >= 0; i-- {
		for _, v := range chain[i].Properties {
			m.globals[ir.Key(v.Name)] = initial(m, v, true)
		}
		for _, v := range chain[i].Variables {
			m.globals[ir.Key(v.Name)] = initial(m, v, false)
		}
	}
	// SKI_ConfigBase's cursor constants, which scripts name.
	for name, n := range map[string]float64{"left_to_right": leftToRight, "top_to_bottom": topToBottom, "option_flag_none": 0, "option_flag_disabled": optionFlagDisabled, "option_flag_hidden": 2, "option_flag_with_unmap": 4} {
		if _, ok := m.globals[name]; !ok {
			v := integer(n)
			m.globals[name] = &v
		}
	}
	for _, p := range props {
		var v Value
		switch {
		case p.Strings != nil:
			v = newArray(len(p.Strings))
			for i, s := range p.Strings {
				(*v.arr)[i] = str(s)
			}
		case p.String != nil:
			v = str(*p.String)
		case p.Number != nil:
			v = num(*p.Number)
		case p.Global != nil:
			v = global(*p.Global)
		case p.Remote != nil:
			v = Value{k: kRemote, s: declaredType(chain, p.Name), rem: p.Remote}
		default:
			continue
		}
		m.globals[ir.Key(p.Name)] = &v
	}
	return m
}

// initialize runs the script's initialization events, as a new game and a
// first opening of the menu do.
func (m *machine) initialize() {
	// The initialization events. SKI_ConfigBase's own functions do nothing
	// the menu's contents depend on, so calls to them are ignored. A script
	// that defers its setup by registering for an update (EVG Conditional
	// Idles) gets that update once the events have run. Then OnConfigOpen,
	// which SkyUI runs each time the menu opens, before any page: what it
	// sets (Accuracy's checks for what the menu needs) holds for every page.
	pending := map[string]bool{}
	var deferred []string
	m.call = func(site *ir.Call, args []Value) (Value, bool) {
		name := site.Method
		k := strings.ToLower(name)
		if event, ok := updates[k]; ok {
			if !pending[event] {
				pending[event] = true
				deferred = append(deferred, event)
			}
			return unknown, true
		}
		if isSKIFunction(name) {
			return unknown, true
		}
		return Value{}, false
	}
	for _, e := range initEvents {
		if f := m.function("", e); f != nil {
			m.budget = budget
			m.invoke(f, nil)
		}
	}
	for i := 0; i < len(deferred); i++ {
		if f := m.function("", deferred[i]); f != nil {
			m.budget = budget
			m.invoke(f, nil)
		}
	}
}

// Initialized is what a script's properties hold once its initialization
// has run as on a new game, starting from their declared values and the
// plugin's: what another script reads of them, through a property pointing
// to the form it's attached to. A property it may or may not have set, or
// sets to something the game decides, isn't there.
func Initialized(chain []*ir.Script, props []Property, settings Settings) map[string]RemoteValue {
	m := setup(chain, props, settings)
	m.newGame = true
	m.initialize()
	out := map[string]RemoteValue{}
	for _, sc := range chain {
		for _, v := range sc.Properties {
			key := ir.Key(v.Name)
			p := m.globals[key]
			if _, done := out[key]; done || p == nil || v.Computed {
				continue
			}
			if p.k == kGlobal {
				if !math.IsNaN(p.n) {
					n := p.n
					out[key] = RemoteValue{Global: &n}
				}
				continue
			}
			val, ok := onNewGame(*p)
			if !ok {
				continue
			}
			switch val.k {
			case kBool, kInt, kFloat:
				n := val.n
				out[key] = RemoteValue{Number: &n}
			case kString:
				str := val.s
				out[key] = RemoteValue{String: &str}
			}
		}
	}
	return out
}

func ExtractWith(chain []*ir.Script, props []Property, settings Settings) Menu {
	m := setup(chain, props, settings)
	m.initialize()
	if f := m.function("", "OnConfigOpen"); f != nil {
		m.budget = budget
		m.invoke(f, nil)
	}
	initial, initialState := m.snapshot(), m.state
	menu := Menu{}
	if p := m.globals["modname"]; p != nil && p.k == kString {
		menu.ModName = p.s
	}

	// The pages: Pages as initialization left it, or the names OnPageReset
	// compares its page with.
	var names []string
	if p := m.globals["pages"]; p != nil && p.k == kArray && p.arr != nil {
		for _, v := range *p.arr {
			if v.k == kString {
				names = append(names, v.s)
			}
		}
	}
	if len(names) == 0 {
		names = comparedPages(chain, m)
		menu.PagesGuessed = len(names) > 0
	}

	pageList := append([]string{""}, names...)
	for _, name := range pageList {
		m.restore(initial)
		m.state = initialState
		page := walk(m, name)
		if name == "" && len(names) > 0 && len(page.Options) == 0 {
			continue // the splash page of a menu with pages
		}
		// The page again as a new game shows it, with what's only a hint
		// (a global's value, an INI's setting) taken as known, for what
		// each option is shown with. Which options the page has stays the
		// first run's, which takes every way a setting could go.
		m.restore(initial)
		m.state = initialState
		m.newGame = true
		shownOn := walk(m, name)
		m.newGame = false
		startsFrom(&page, shownOn)
		menu.Pages = append(menu.Pages, page)
	}
	return menu
}

// startsFrom takes each option's start value from the page as a new game
// shows it, among the copies there reached through the same calls, with
// the same type and label. When the first run found as many options so
// alike (a label a page repeats), each takes the copy in the same place.
// Otherwise, as when the first run kept one copy of an option the arms of
// an If each add and a new game that can't tell which arm runs adds both,
// the copies have to agree. An option a new game doesn't show keeps what
// the first run found.
func startsFrom(page *Page, shownOn Page) {
	alike := func(a, b Option) bool {
		return slices.Equal(a.calls, b.calls) && a.Type == b.Type && a.Label == b.Label
	}
	for i := range page.Options {
		o := &page.Options[i]
		var copies []Option
		for _, s := range shownOn.Options {
			if alike(s, *o) {
				copies = append(copies, s)
			}
		}
		if len(copies) == 0 {
			continue
		}
		place, count := 0, 0
		for j, other := range page.Options {
			if alike(other, *o) {
				if j < i {
					place++
				}
				count++
			}
		}
		if count == len(copies) {
			o.StartNumber, o.StartText = copies[place].StartNumber, copies[place].StartText
			continue
		}
		o.StartNumber, o.StartText = copies[0].StartNumber, copies[0].StartText
		for _, c := range copies[1:] {
			if !sameStart(copies[0], c) {
				o.StartNumber, o.StartText = nil, nil
				break
			}
		}
	}
}

// initial is a variable's or property's value before any code runs: its
// declared value, or Papyrus's default for its type (0, 0.0, False, ""), or
// None for an array or an object. An object property the declaration
// leaves empty is one the plugin fills with a form, so it's unknown, not
// None.
func initial(m *machine, v ir.Variable, property bool) *Value {
	val := Value{k: kNone}
	switch {
	case v.Computed:
		val = unknown
	case v.Init != nil:
		val = m.eval(v.Init)
	case strings.HasSuffix(v.Type, "]"):
	default:
		switch strings.ToLower(v.Type) {
		case "int":
			val = integer(0)
		case "float":
			val = num(0)
		case "bool":
			val = boolean(false)
		case "string":
			val = str("")
		default:
			if property {
				val = unknown
			}
		}
	}
	return &val
}

// walk runs OnPageReset for one page and then the option events, and
// returns the page as it's shown.
func walk(m *machine, page string) Page {
	var opts []*found
	var fixtures []Fixture
	byState := map[string][]int{}
	next := 0
	// The cursor, as SkyUI keeps it: where the next option goes, and how
	// it moves after. Lost once the script moves it somewhere the analysis
	// can't follow.
	cursor, fill, known := 0, leftToRight, true
	place := func() int {
		if !known || cursor < 0 {
			return -1
		}
		at := cursor
		cursor += fill
		if cursor >= positions {
			cursor = -1
		}
		return at
	}
	m.globals["currentpage"] = &Value{k: kString, s: page}
	// Each arm of an If the analysis runs starts the cursor where the If
	// began, as the game runs only one; after it, the cursor is where the
	// furthest arm left it, or, when no arm may run, where it began too.
	m.arms = &cursorArms{cursor: &cursor, fill: &fill, known: &known}
	m.call = func(site *ir.Call, args []Value) (Value, bool) {
		name, line := site.Method, site.Line
		k := strings.ToLower(name)
		st := strings.HasSuffix(k, "st") && adders[strings.TrimSuffix(k, "st")] != ""
		typ := adders[strings.TrimSuffix(k, "st")]
		if adders[k] != "" {
			typ, st = adders[k], false
		}
		if typ == "" {
			switch k {
			case "setcursorposition":
				if len(args) > 0 && args[0].numeric() {
					if n := int(args[0].n); n >= 0 && n < positions {
						cursor, known = n, true
					}
				} else {
					known = false
				}
			case "setcursorfillmode":
				if len(args) > 0 && args[0].numeric() && (int(args[0].n) == leftToRight || int(args[0].n) == topToBottom) {
					fill = int(args[0].n)
				} else {
					known = false
				}
			case "setoptionflags":
				// Flags set as the page is drawn, right after the option is
				// added: they're the ones it's shown with.
				if len(args) > 1 && args[0].k == kOption {
					for _, id := range args[0].ids {
						if id >= 0 && id < len(opts) {
							opts[id].Disabled = disabledBy(args[1])
						}
					}
				}
			case "addheaderoption":
				fx := Fixture{Kind: "header", Position: place(), Line: line}
				if len(args) > 0 {
					fx.Label, fx.Text, fx.Certainty = described(args[0])
				}
				fixtures = append(fixtures, fx)
				return unknown, true
			case "addemptyoption":
				fixtures = append(fixtures, Fixture{Kind: "empty", Position: place(), Line: line})
				return unknown, true
			}
			if isSKIFunction(name) {
				return unknown, true
			}
			return Value{}, false
		}
		f := &found{id: next, path: append([]step{}, m.path...), site: site}
		for _, s := range m.path {
			if s.call != nil {
				f.Option.calls = append(f.Option.calls, s.call)
			}
		}
		next++
		f.Type, f.Line, f.Position = typ, line, place()
		f.Repeats = m.looping > 0 // in a loop the game decides the count of
		if st {
			if len(args) > 0 {
				if t, ok := args[0].text(); ok && args[0].k == kString {
					f.State = t
				}
				args = args[1:]
			}
		}
		label := unknown
		if len(args) > 0 {
			label = args[0]
		}
		describe(&f.Option, label)
		if typ == "text" {
			value := unknown
			if len(args) > 1 {
				value = args[1]
			}
			f.Value, f.ValueText, f.ValueCertainty = described(value)
		}
		if typ != "text" && len(args) > 1 {
			f.StartNumber, f.StartText = start(args[1])
		}
		if typ == "slider" && len(args) > 2 && args[2].k == kString {
			f.Format = args[2].s
		}
		// The flags follow the label and value (and a slider's format);
		// left out, they're OPTION_FLAG_NONE.
		at := 2
		if typ == "slider" {
			at = 3
		}
		if len(args) > at {
			f.Disabled = disabledBy(args[at])
		}
		opts = append(opts, f)
		if f.State != "" {
			byState[ir.Key(f.State)] = append(byState[ir.Key(f.State)], f.id)
		}
		return option(f.id), true
	}
	m.budget = budget
	m.invoke(m.function(m.state, "OnPageReset"), []Value{str(page)})
	m.arms = nil
	// As a new game shows it, only what each option is shown with counts.
	if m.newGame {
		out := Page{Name: page}
		for _, f := range opts {
			out.Options = append(out.Options, f.Option)
		}
		return out
	}
	// The state the page left the script in, where its option events are
	// looked for: Simply Balanced keeps each page's in a state of its own.
	pageState := m.state

	// The option events: dialog calls in a branch scoped to options apply
	// to those options. A literal a click handler shows a text option is
	// kept, to find the values an option cycles through.
	var shows []shown
	apply := func(name string, args []Value, ids []int) {
		if k := strings.ToLower(name); k == "settextoptionvalue" || k == "settextoptionvaluest" {
			at := 1
			if k == "settextoptionvaluest" {
				at = 0
			}
			if len(args) > at && args[at].k == kString {
				chain, arm := chainOf(m.path)
				shows = append(shows, shown{ids: ids, chain: chain, arm: arm, text: args[at].s})
			}
		}
		for _, id := range ids {
			if id < 0 || id >= len(opts) {
				continue
			}
			o := &opts[id].Option
			switch strings.ToLower(name) {
			case "setsliderdialogrange":
				if len(args) >= 2 && args[0].numeric() && args[1].numeric() {
					if o.Range == nil {
						o.Range = &Range{Step: 1}
					}
					o.Range.Min, o.Range.Max = args[0].n, args[1].n
				}
			case "setsliderdialoginterval":
				if len(args) >= 1 && args[0].numeric() {
					if o.Range == nil {
						o.Range = &Range{Step: 1}
					}
					o.Range.Step = args[0].n
				}
			case "setsliderdialogdefaultvalue", "setmenudialogdefaultindex":
				if len(args) >= 1 && args[0].numeric() {
					d := args[0].n
					o.Default = &d
				}
			case "setmenudialogoptions":
				if len(args) >= 1 && args[0].k == kArray && args[0].arr != nil {
					o.Choices = o.Choices[:0]
					for _, v := range *args[0].arr {
						if t, ok := v.text(); ok && v.k == kString {
							o.Choices = append(o.Choices, t)
						} else {
							o.Choices = append(o.Choices, "")
						}
					}
				}
			case "settextoptionvalue", "settextoptionvaluest":
				o.ValueChanges = true
			case "setoptionflags", "setoptionflagsst":
				// SetOptionFlags(option, flags), SetOptionFlagsST(flags):
				// flags other than the ones it was added with make it
				// sometimes one, sometimes the other.
				at := 1
				if strings.EqualFold(name, "SetOptionFlagsST") {
					at = 0
				}
				if len(args) > at && disabledBy(args[at]) != o.Disabled {
					o.Disabled = "sometimes"
				}
			case "setinfotext":
				if len(args) >= 1 {
					switch args[0].k {
					case kString:
						o.Help = args[0].s
					case kPattern:
						o.Help = patternText(args[0].parts)
					}
				}
			}
		}
	}
	dialog := func(fixed []int) func(*ir.Call, []Value) (Value, bool) {
		return func(site *ir.Call, args []Value) (Value, bool) {
			name := site.Method
			if !isSKIFunction(name) {
				return Value{}, false
			}
			ids := fixed
			if ids == nil && len(m.scope) > 0 {
				ids = m.scope[len(m.scope)-1]
			}
			// SetTextOptionValue and SetOptionFlags name their option outright.
			if (strings.EqualFold(name, "SetTextOptionValue") || strings.EqualFold(name, "SetOptionFlags")) && len(args) > 0 && args[0].k == kOption {
				ids = args[0].ids
			}
			apply(name, args, ids)
			return unknown, true
		}
	}
	after := m.snapshot()
	// What a click reaches: each effect the click handler has, for the
	// options its branch is about, or for every option outside one.
	responds := map[int]bool{}
	everyOption := false
	for _, e := range optionEvents {
		f := m.function(pageState, e)
		if f == nil {
			continue
		}
		m.restore(after)
		m.state = pageState
		m.call = dialog(nil)
		m.budget = budget
		m.invoke(f, []Value{{k: kParam}, {k: kParam}})
	}
	for _, state := range sortedKeys(byState) {
		ids := byState[state]
		for _, e := range stateEvents {
			f := m.function(state, e)
			if f == nil || !inState(m.chain, state, f) {
				continue
			}
			m.restore(after)
			m.state = state
			m.call = dialog(ids)
			m.budget = budget
			m.invoke(f, nil)
		}
	}
	// A pass of its own for what a click reaches, with the script's own
	// variables unknown but those holding options: whether a branch might
	// run decides it, not the value a variable starts with (SunHelm's Save
	// runs only once a profile is chosen, which "" says it never is).
	blurred := map[string]Value{}
	for k, v := range after {
		if v.k == kOption || v.k == kArray {
			blurred[k] = v
		} else {
			blurred[k] = unknown
		}
	}
	quiet := func(site *ir.Call, args []Value) (Value, bool) {
		if isSKIFunction(site.Method) {
			return unknown, true
		}
		return Value{}, false
	}
	if f := m.function(pageState, "OnOptionSelect"); f != nil {
		m.restore(blurred)
		m.state = pageState
		m.call = quiet
		m.budget = budget
		m.effect = func() {
			if len(m.scope) == 0 {
				everyOption = true
				return
			}
			for _, id := range m.scope[len(m.scope)-1] {
				responds[id] = true
			}
		}
		m.invoke(f, []Value{{k: kParam}, {k: kParam}})
		m.effect = nil
		if m.budget <= 0 {
			everyOption = true // cut short: what's left might do anything
		}
	}
	for _, state := range sortedKeys(byState) {
		f := m.function(state, "OnSelectST")
		if f == nil || !inState(m.chain, state, f) {
			continue
		}
		ids := byState[state]
		m.restore(blurred)
		m.state = state
		m.call = quiet
		m.budget = budget
		m.effect = func() {
			for _, id := range ids {
				responds[id] = true
			}
		}
		m.invoke(f, nil)
		m.effect = nil
		if m.budget <= 0 {
			for _, id := range ids {
				responds[id] = true
			}
		}
	}
	for _, o := range opts {
		if o.Type == "text" && !everyOption && !responds[o.id] {
			o.Inert = true
		}
	}
	cycles(opts, shows)
	out := Page{Name: page, Options: merge(opts), Fixtures: fixtures, Placed: true}
	for _, o := range out.Options {
		if o.Position < 0 {
			out.Placed = false
		}
	}
	for _, f := range fixtures {
		if f.Position < 0 {
			out.Placed = false
		}
	}
	return out
}

// inState reports whether f is defined in the named state, not just the
// empty one, so a state's event isn't mistaken for every option's.
func inState(chain []*ir.Script, state string, f *ir.Function) bool {
	for _, s := range chain {
		if s.State(state).Function(f.Name) == f {
			return true
		}
	}
	return false
}

// describe sets an option's label from the value passed for it.
func describe(o *Option, v Value) {
	o.Label, o.Text, o.Certainty = described(v)
}

// described is a string passed to the menu as a selector matches it, as it
// reads, and how certain both are.
func described(v Value) (selector, text string, c Certainty) {
	switch v.k {
	case kString:
		return v.s, v.s, Exact
	case kPattern:
		sel := patternSelector(v.parts)
		if sel == "" {
			return "", patternText(v.parts), Unknown
		}
		return sel, patternText(v.parts), Pattern
	default:
		if t, ok := v.text(); ok {
			return t, t, Exact
		}
		return "", "", Unknown
	}
}

// cycles gives each text option the values its click handler shows it in
// turn. The handler's If chain (an If with its ElseIfs) that sets a value
// is one cycle when each of its arms, three or more, sets one value for
// the same option: the arms are chosen by a counter the click moves. The
// cycle is the option's when the chain is in a branch scoped to the
// option, or, when the handler dispatches some other way (VRIK's by the
// option's name), when the value the option is added with is one of
// exactly one unscoped chain's values. Anything else is a message: a
// single Will Reset, a Working then Done or Error, a DONE on each of
// several buttons.
func cycles(opts []*found, shows []shown) {
	type group struct {
		ids    []int
		arms   map[string]int // values each arm sets
		values []string
	}
	var groups []*group
	byKey := map[string]*group{}
	for _, s := range shows {
		key := fmt.Sprintf("%p:%v", s.chain, s.ids)
		g := byKey[key]
		if g == nil {
			g = &group{ids: s.ids, arms: map[string]int{}}
			byKey[key] = g
			groups = append(groups, g)
		}
		g.arms[s.arm]++
		if !slices.Contains(g.values, s.text) {
			g.values = append(g.values, s.text)
		}
	}
	cyclic := func(g *group) bool {
		if len(g.arms) < 3 || len(g.values) != len(g.arms) {
			return false
		}
		for _, n := range g.arms {
			if n != 1 {
				return false
			}
		}
		return true
	}
	for _, o := range opts {
		if o.Type != "text" {
			continue
		}
		var cycle []string
		var matched int
		for _, g := range groups {
			if !cyclic(g) {
				continue
			}
			if g.ids != nil {
				if containsInt(g.ids, o.id) {
					cycle = g.values
					break
				}
				continue
			}
			if o.ValueCertainty == Exact && o.Value != "" && slices.Contains(g.values, o.Value) {
				matched++
				cycle = g.values
			}
		}
		if cycle == nil || (matched > 1 && !slices.ContainsFunc(groups, func(g *group) bool { return g.ids != nil && containsInt(g.ids, o.id) })) {
			continue
		}
		if o.ValueCertainty == Exact && o.Value != "" && !slices.Contains(cycle, o.Value) {
			cycle = append([]string{o.Value}, cycle...)
		}
		o.Cycle = append([]string{}, cycle...)
	}
}

// cursorArms keeps the page's cursor across an If's arms: each arm starts
// from the cursor as the If found it, and the If leaves it as its furthest
// arm that carries on did. An arm that returns leaves the cursor for its
// function's caller, not for the code after the If, which runs only when
// it didn't: the call ends at the furthest of those. An arm that loses
// track of the cursor loses it for the rest.
type cursorArms struct {
	cursor, fill *int
	known        *bool
	stack        []cursorIf
	calls        [][]cursorEnd // per call in progress, the ends of its arms that returned
}

type cursorIf struct {
	cursor, fill int
	known        bool
	ends         []cursorEnd
}

type cursorEnd struct {
	cursor, fill int
	known        bool
}

func (c *cursorArms) enter() {
	c.stack = append(c.stack, cursorIf{cursor: *c.cursor, fill: *c.fill, known: *c.known})
}

func (c *cursorArms) arm() {
	top := &c.stack[len(c.stack)-1]
	*c.cursor, *c.fill, *c.known = top.cursor, top.fill, top.known
}

func (c *cursorArms) done(returned bool) {
	end := cursorEnd{*c.cursor, *c.fill, *c.known}
	if returned {
		// Where the function's caller carries on from; returning from the
		// page's own OnPageReset ends the page.
		if len(c.calls) > 0 {
			c.calls[len(c.calls)-1] = append(c.calls[len(c.calls)-1], end)
		}
		return
	}
	top := &c.stack[len(c.stack)-1]
	top.ends = append(top.ends, end)
}

func (c *cursorArms) call() {
	c.calls = append(c.calls, nil)
}

func (c *cursorArms) ret() {
	ends := append(c.calls[len(c.calls)-1], cursorEnd{*c.cursor, *c.fill, *c.known})
	c.calls = c.calls[:len(c.calls)-1]
	*c.cursor, *c.fill, *c.known = furthest(ends)
}

// furthest is where the cursor is after one of several ways through: the
// furthest along, or lost when they leave it moving different ways.
func furthest(ends []cursorEnd) (int, int, bool) {
	out := ends[0]
	for _, e := range ends[1:] {
		switch {
		case !e.known || !out.known:
			out.known = false
		case e.fill != out.fill:
			out.known = false
		case e.cursor < 0 || (out.cursor >= 0 && e.cursor > out.cursor):
			out.cursor = e.cursor // -1 is past the end
		}
	}
	return out.cursor, out.fill, out.known
}

func (c *cursorArms) exit(skipped bool) {
	top := c.stack[len(c.stack)-1]
	c.stack = c.stack[:len(c.stack)-1]
	ends := top.ends
	if skipped || len(ends) == 0 {
		// No arm may run, or every one returned: what follows starts where
		// the If began.
		ends = append(ends, cursorEnd{top.cursor, top.fill, top.known})
	}
	*c.cursor, *c.fill, *c.known = furthest(ends)
}

// merge drops an option that's the same as an earlier one (type and exact
// label) when the two are in opposite arms of an If, since only one of them
// is ever shown; what the dropped one knows fills what the kept one lacks.
func merge(opts []*found) []Option {
	var kept []*found
	for _, o := range opts {
		dup := false
		for _, k := range kept {
			// One call adding the same label again (a loop over what the
			// game holds): one option, which repeats at runtime. The same
			// call reached from another call site (a helper the page calls
			// for each of several slots) is another option.
			if k.site == o.site && sameCalls(k.path, o.path) && k.Type == o.Type && k.Label == o.Label && k.Certainty == o.Certainty {
				fill(&k.Option, o.Option)
				k.Repeats = true
				dup = true
				break
			}
			if k.Type == o.Type && k.Label == o.Label && k.Certainty == Exact && o.Certainty == Exact && exclusive(k.path, o.path) {
				fill(&k.Option, o.Option)
				dup = true
				break
			}
		}
		if !dup {
			kept = append(kept, o)
		}
	}
	out := make([]Option, len(kept))
	for i, k := range kept {
		out[i] = k.Option
	}
	return out
}

// sameCalls reports whether two paths went through the same calls into the
// script's own functions.
func sameCalls(a, b []step) bool {
	var ca, cb []*ir.Call
	for _, s := range a {
		if s.call != nil {
			ca = append(ca, s.call)
		}
	}
	for _, s := range b {
		if s.call != nil {
			cb = append(cb, s.call)
		}
	}
	return slices.Equal(ca, cb)
}

// exclusive reports whether two paths part at the same If, in different
// arms: no run takes both.
func exclusive(a, b []step) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i].at != b[i].at || a[i].call != b[i].call {
			return false // different Ifs, or the same function called from two places: both may run
		}
		if a[i].arm != b[i].arm {
			return true
		}
	}
	return false
}

// start is a value an option is shown with, when it's known: a number (a
// bool as 0 or 1), or text.
func start(v Value) (*float64, *string) {
	v, ok := onNewGame(v)
	if !ok {
		return nil, nil
	}
	switch v.k {
	case kBool, kInt, kFloat:
		n := v.n
		return &n, nil
	case kString:
		s := v.s
		return nil, &s
	}
	return nil, nil
}

// sameStart reports whether two copies of an option are shown with the
// same known value.
func sameStart(a, b Option) bool {
	switch {
	case a.StartNumber != nil && b.StartNumber != nil:
		return *a.StartNumber == *b.StartNumber
	case a.StartText != nil && b.StartText != nil:
		return *a.StartText == *b.StartText
	}
	return false
}

func fill(k *Option, o Option) {
	// Shown with different values, or one not known: not known.
	if !sameStart(*k, o) {
		k.StartNumber, k.StartText = nil, nil
	}
	if k.Range == nil {
		k.Range = o.Range
	}
	if k.Choices == nil {
		k.Choices = o.Choices
	}
	if k.Default == nil {
		k.Default = o.Default
	}
	if k.Help == "" {
		k.Help = o.Help
	}
	if k.Format == "" {
		k.Format = o.Format
	}
	// The same text option shown with another value: the value changes.
	if k.Type == "text" && (k.Value != o.Value || k.ValueCertainty != o.ValueCertainty || o.ValueChanges) {
		k.ValueChanges = true
	}
	// Inert only if no copy responds to a click.
	k.Inert = k.Inert && o.Inert
	// Disabled in one arm and not the other: sometimes.
	if k.Disabled != o.Disabled {
		k.Disabled = "sometimes"
	}
	if k.Cycle == nil {
		k.Cycle = o.Cycle
	}
}

// comparedPages is the strings OnPageReset compares its page parameter or
// CurrentPage with, in the order they appear: the pages, when Pages itself
// can't be read.
func comparedPages(chain []*ir.Script, m *machine) []string {
	f := m.function("", "OnPageReset")
	if f == nil {
		return nil
	}
	param := ""
	if len(f.Params) > 0 {
		param = ir.Key(f.Params[0].Name)
	}
	seen := map[string]bool{}
	var out []string
	var visit func(e ir.Expr)
	visit = func(e ir.Expr) {
		switch e := e.(type) {
		case *ir.Binary:
			if e.Op == "==" {
				for _, pair := range [][2]ir.Expr{{e.L, e.R}, {e.R, e.L}} {
					id, isID := pair[0].(*ir.Ident)
					c, isConst := pair[1].(*ir.Const)
					if isID && isConst && c.Kind == ir.ConstString && c.String != "" && (ir.Key(id.Name) == param || ir.Key(id.Name) == "currentpage") && !seen[ir.Key(c.String)] {
						seen[ir.Key(c.String)] = true
						out = append(out, c.String)
					}
				}
			}
			visit(e.L)
			visit(e.R)
		case *ir.Unary:
			visit(e.X)
		}
	}
	// Every expression counts, not just conditions: compiled code compares
	// into a temporary and tests that.
	var walkStmts func([]ir.Stmt)
	walkStmts = func(ss []ir.Stmt) {
		for _, s := range ss {
			switch s := s.(type) {
			case *ir.Assign:
				visit(s.Value)
			case *ir.If:
				for _, b := range s.Branches {
					visit(b.Cond)
					walkStmts(b.Body)
				}
				walkStmts(s.Else)
			case *ir.While:
				visit(s.Cond)
				walkStmts(s.Body)
			}
		}
	}
	walkStmts(f.Body)
	return out
}

// SKI_ConfigBase's functions a menu calls on itself.
var skiFunctions = func() map[string]bool {
	names := []string{
		"AddHeaderOption", "AddEmptyOption", "AddTextOption", "AddToggleOption", "AddSliderOption", "AddMenuOption",
		"AddColorOption", "AddKeyMapOption", "AddInputOption", "AddTextOptionST", "AddToggleOptionST", "AddSliderOptionST",
		"AddMenuOptionST", "AddColorOptionST", "AddKeyMapOptionST", "AddInputOptionST",
		"SetCursorFillMode", "SetCursorPosition", "LoadCustomContent", "UnloadCustomContent", "SetTitleText", "SetInfoText",
		"ForcePageReset", "SetOptionFlags", "SetTextOptionValue", "SetToggleOptionValue", "SetSliderOptionValue",
		"SetMenuOptionValue", "SetColorOptionValue", "SetKeyMapOptionValue", "SetInputOptionValue", "SetOptionFlagsST",
		"SetTextOptionValueST", "SetToggleOptionValueST", "SetSliderOptionValueST", "SetMenuOptionValueST",
		"SetColorOptionValueST", "SetKeyMapOptionValueST", "SetInputOptionValueST",
		"SetSliderDialogStartValue", "SetSliderDialogDefaultValue", "SetSliderDialogRange", "SetSliderDialogInterval",
		"SetMenuDialogStartIndex", "SetMenuDialogDefaultIndex", "SetMenuDialogOptions",
		"SetColorDialogStartColor", "SetColorDialogDefaultColor", "SetInputDialogStartText",
		"ShowMessage", "RegisterForModEvent", "RegisterForKey", "UnregisterForKey",
	}
	m := map[string]bool{}
	for _, n := range names {
		m[strings.ToLower(n)] = true
	}
	return m
}()

func isSKIFunction(name string) bool { return skiFunctions[strings.ToLower(name)] }

// sortedKeys is a map's keys in order, for deterministic output.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
