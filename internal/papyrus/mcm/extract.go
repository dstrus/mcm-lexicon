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
}

type Menu struct {
	ModName      string // the display name the script or plugin sets; "" when neither does
	Pages        []Page // in order; a menu without pages has one, named ""
	PagesGuessed bool   // Pages wasn't readable, so the pages are the names OnPageReset compares against
}

type Page struct {
	Name    string // raw; "" for the default page
	Options []Option
}

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
	Help      string
	Line      int  // where it's added
	Repeats   bool // added again and again by one call in a loop the game decides, with the same label
}

type Range struct {
	Min, Max, Step float64
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
var optionEvents = []string{"OnOptionSliderOpen", "OnOptionMenuOpen", "OnOptionHighlight", "OnOptionDefault"}
var stateEvents = []string{"OnSliderOpenST", "OnMenuOpenST", "OnHighlightST", "OnDefaultST"}

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

// Extract reads the menu a script defines. chain is the script, then the
// ancestors whose source is at hand (SKI_ConfigBase itself excluded).
func Extract(chain []*ir.Script, props []Property) Menu {
	m := newMachine(chain)
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
		default:
			continue
		}
		m.globals[ir.Key(p.Name)] = &v
	}
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
	if f := m.function("", "OnConfigOpen"); f != nil {
		m.budget = budget
		m.invoke(f, nil)
	}
	initial := m.snapshot()
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
		opts := walk(m, name)
		if name == "" && len(names) > 0 && len(opts) == 0 {
			continue // the splash page of a menu with pages
		}
		menu.Pages = append(menu.Pages, Page{Name: name, Options: opts})
	}
	return menu
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
// returns the page's options.
func walk(m *machine, page string) []Option {
	var opts []*found
	byState := map[string][]int{}
	next := 0
	m.globals["currentpage"] = &Value{k: kString, s: page}
	m.call = func(site *ir.Call, args []Value) (Value, bool) {
		name, line := site.Method, site.Line
		k := strings.ToLower(name)
		st := strings.HasSuffix(k, "st") && adders[strings.TrimSuffix(k, "st")] != ""
		typ := adders[strings.TrimSuffix(k, "st")]
		if adders[k] != "" {
			typ, st = adders[k], false
		}
		if typ == "" {
			if isSKIFunction(name) {
				return unknown, true
			}
			return Value{}, false
		}
		f := &found{id: next, path: append([]step{}, m.path...), site: site}
		next++
		f.Type, f.Line = typ, line
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
		if typ == "slider" && len(args) > 2 && args[2].k == kString {
			f.Format = args[2].s
		}
		opts = append(opts, f)
		if f.State != "" {
			byState[ir.Key(f.State)] = append(byState[ir.Key(f.State)], f.id)
		}
		return option(f.id), true
	}
	m.budget = budget
	m.invoke(m.function("", "OnPageReset"), []Value{str(page)})

	// The option events: dialog calls in a branch scoped to options apply
	// to those options.
	apply := func(name string, args []Value, ids []int) {
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
			apply(name, args, ids)
			return unknown, true
		}
	}
	after := m.snapshot()
	for _, e := range optionEvents {
		f := m.function("", e)
		if f == nil {
			continue
		}
		m.restore(after)
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
			m.call = dialog(ids)
			m.budget = budget
			m.invoke(f, nil)
		}
	}
	return merge(opts)
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
	switch v.k {
	case kString:
		o.Label, o.Text, o.Certainty = v.s, v.s, Exact
	case kPattern:
		o.Label, o.Text, o.Certainty = patternSelector(v.parts), patternText(v.parts), Pattern
		if o.Label == "" {
			o.Certainty = Unknown
		}
	default:
		if t, ok := v.text(); ok {
			o.Label, o.Text, o.Certainty = t, t, Exact
			return
		}
		o.Label, o.Text, o.Certainty = "", "", Unknown
	}
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
			// game holds): one option, which repeats at runtime.
			if k.site == o.site && k.Type == o.Type && k.Label == o.Label && k.Certainty == o.Certainty {
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

// exclusive reports whether two paths part at the same If, in different
// arms: no run takes both.
func exclusive(a, b []step) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i].at != b[i].at {
			return false
		}
		if a[i].arm != b[i].arm {
			return true
		}
	}
	return false
}

func fill(k *Option, o Option) {
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
