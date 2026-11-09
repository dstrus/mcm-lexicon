package mcm

import (
	"strconv"
	"strings"
	"unicode"
)

// kind is what the analysis knows of a value.
type kind int

const (
	kUnknown kind = iota // anything: computed at runtime from what the game holds
	kNone
	kString
	kInt
	kFloat
	kBool
	kArray
	kPattern // a string with known parts and runtime-built gaps
	kOption  // the ID an Add…Option call returned
	kParam   // the option an option event was called for
	kScoped  // a condition true for some options only: option == X
	kOthers  // a condition true for all options but some: option != X
	kGlobal  // a global variable the plugin fills, holding n; NaN when what it holds isn't known
	kRemote  // a form another script is attached to: rem says what its properties hold, s names the script it's taken as
)

// Value is a value as far as the analysis can tell.
type Value struct {
	k     kind
	s     string
	n     float64
	arr   *[]Value // arrays are shared by reference, as in Papyrus
	parts []part   // a pattern's parts
	ids   []int    // an option's ID, or a scoped condition's options
	// hint, on an unknown value, is what it is on a new game: a global
	// variable's value from the plugin, and what's worked out from it. It
	// says what an option is shown with, never which way a condition
	// goes: a menu shows other options once a setting changes, and the
	// catalog lists every option a menu can show.
	hint *Value
	// pooled is a value a script variable was left holding after an If
	// that may not have run the arm that set it (see ifArms): kept so an
	// option ID stays linked, and a label as it was, but not what the
	// option is shown with, which may be what the variable held before.
	pooled bool
	rem    Remote // a kRemote's scripts and what their properties hold
}

// hinted is an unknown value that's v on a new game.
func hinted(v Value) Value {
	if !v.known() {
		return unknown
	}
	return Value{k: kUnknown, hint: &v}
}

// onNewGame is what v is on a new game: v itself when it's known, else its
// hint, if it has one.
func onNewGame(v Value) (Value, bool) {
	if v.pooled {
		return Value{}, false
	}
	if v.known() {
		return v, true
	}
	if v.k == kUnknown && v.hint != nil {
		return *v.hint, true
	}
	return Value{}, false
}

type part struct {
	text string
	gap  bool
}

var unknown = Value{k: kUnknown}

func str(s string) Value      { return Value{k: kString, s: s} }
func num(n float64) Value     { return Value{k: kFloat, n: n} }
func integer(n float64) Value { return Value{k: kInt, n: float64(int64(n))} }
func boolean(b bool) Value    { return Value{k: kBool, n: b2f(b)} }
func option(id int) Value     { return Value{k: kOption, ids: []int{id}} }
func scoped(ids []int) Value  { return Value{k: kScoped, ids: ids} }
func others(ids []int) Value  { return Value{k: kOthers, ids: ids} }
func global(n float64) Value  { return Value{k: kGlobal, n: n} }
func (v Value) known() bool {
	return v.k != kUnknown && v.k != kPattern && v.k != kScoped && v.k != kOthers && v.k != kParam
}
func (v Value) numeric() bool  { return v.k == kInt || v.k == kFloat || v.k == kBool }
func (v Value) isString() bool { return v.k == kString }

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func newArray(n int) Value {
	if n < 0 || n > 4096 {
		return unknown
	}
	a := make([]Value, n)
	for i := range a {
		a[i] = Value{k: kNone}
	}
	return Value{k: kArray, arr: &a}
}

// truth is a value's truth if it's known: ok=false when it depends on the
// game.
func (v Value) truth() (bool, bool) {
	switch v.k {
	case kNone:
		return false, true
	case kBool, kInt, kFloat:
		return v.n != 0, true
	case kString:
		return v.s != "", true
	case kArray, kOption, kGlobal, kRemote:
		return true, true
	}
	return false, false
}

// text is a value as a string, as Papyrus converts it: an int as its
// digits, a float with six decimal places, a bool as True or False. A
// pattern or an unknown has no text.
func (v Value) text() (string, bool) {
	switch v.k {
	case kString:
		return v.s, true
	case kInt:
		return strconv.FormatInt(int64(v.n), 10), true
	case kFloat:
		return strconv.FormatFloat(v.n, 'f', 6, 64), true
	case kBool:
		if v.n != 0 {
			return "True", true
		}
		return "False", true
	case kNone:
		return "None", true
	}
	return "", false
}

// asParts is a value as a pattern's parts.
func (v Value) asParts() []part {
	if v.k == kPattern {
		return v.parts
	}
	if t, ok := v.text(); ok {
		return []part{{text: t}}
	}
	return []part{{gap: true}}
}

// textual reports whether v is text: a string, or a pattern.
func textual(v Value) bool { return v.k == kString || v.k == kPattern }

// shared is what two texts have in common, as a pattern: the start and the
// end they share, with a gap for where they differ ("Left Hand …" and
// "Right Hand …" share "… Hand …"). Texts that share nothing are unknown.
func shared(a, b Value) Value {
	// A translation key ($FF_On) is translated only whole, so a piece of
	// one ($FF_) shared with another means nothing.
	if a.k == kString && strings.HasPrefix(a.s, "$") || b.k == kString && strings.HasPrefix(b.s, "$") {
		return unknown
	}
	const gap = '\x00'
	runes := func(v Value) []rune {
		var out []rune
		for _, p := range v.asParts() {
			if p.gap {
				out = append(out, gap)
			} else {
				out = append(out, []rune(p.text)...)
			}
		}
		return out
	}
	x, y := runes(a), runes(b)
	pre := 0
	for pre < len(x) && pre < len(y) && x[pre] == y[pre] {
		pre++
	}
	suf := 0
	for suf < len(x)-pre && suf < len(y)-pre && x[len(x)-1-suf] == y[len(y)-1-suf] {
		suf++
	}
	// Whole words only: a shared start or end that cuts a word in two
	// ("Lef|t" and "Righ|t") gives the part of the word back to the gap.
	letter := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
	for pre > 0 && pre < len(x) && pre < len(y) && letter(x[pre-1]) && (letter(x[pre]) || letter(y[pre])) {
		pre--
	}
	for suf > 0 && suf < len(x) && suf < len(y) && letter(x[len(x)-suf]) && (letter(x[len(x)-1-suf]) || letter(y[len(y)-1-suf])) {
		suf--
	}
	whole := append(append(append([]rune{}, x[:pre]...), gap), x[len(x)-suf:]...)
	var parts []part
	for _, r := range whole {
		isGap := r == gap
		if n := len(parts); n > 0 && parts[n-1].gap == isGap {
			if !isGap {
				parts[n-1].text += string(r)
			}
			continue
		}
		if isGap {
			parts = append(parts, part{gap: true})
		} else {
			parts = append(parts, part{text: string(r)})
		}
	}
	for _, p := range parts {
		if !p.gap && strings.TrimSpace(p.text) != "" {
			return Value{k: kPattern, parts: parts}
		}
	}
	return unknown
}

// concat is a + b with at least one side a string.
func concat(a, b Value) Value {
	ta, oka := a.text()
	tb, okb := b.text()
	if oka && okb {
		return str(ta + tb)
	}
	var parts []part
	// Adjacent texts join, and adjacent gaps are one gap.
	for _, p := range append(append([]part{}, a.asParts()...), b.asParts()...) {
		if n := len(parts); n > 0 && parts[n-1].gap == p.gap {
			parts[n-1].text += p.text
			continue
		}
		parts = append(parts, p)
	}
	return Value{k: kPattern, parts: parts}
}

// patternSelector is the selector MCM Recorder can match a pattern with:
// its longest known part between asterisks, which means "contains". It's
// "" when nothing of the label is known.
func patternSelector(parts []part) string {
	best := ""
	for _, p := range parts {
		if !p.gap && len(strings.TrimSpace(p.text)) > len(strings.TrimSpace(best)) {
			best = p.text
		}
	}
	if strings.TrimSpace(best) == "" {
		return ""
	}
	return "*" + best + "*"
}

// patternText is a pattern for reading, with … for each gap.
func patternText(parts []part) string {
	var b strings.Builder
	for _, p := range parts {
		if p.gap {
			b.WriteString("…")
		} else {
			b.WriteString(p.text)
		}
	}
	return b.String()
}

// binary is the result of a binary operator on two values.
func binary(op string, a, b Value) Value {
	if a.pooled || b.pooled {
		a.pooled, b.pooled = false, false
		r := binary(op, a, b)
		r.pooled = true
		return r
	}
	// Text joined with a new game's value stays a pattern, with a gap where
	// the value goes: a label or a value read at runtime, never the new
	// game's text.
	if op == "+" && (textual(a) || textual(b)) {
		a.hint, b.hint = nil, nil
	}
	// Worked out from a new game's value: still only a hint.
	if (a.k == kUnknown && a.hint != nil) || (b.k == kUnknown && b.hint != nil) {
		ha, okA := onNewGame(a)
		hb, okB := onNewGame(b)
		if !okA || !okB {
			return unknown
		}
		return hinted(binary(op, ha, hb))
	}
	switch op {
	case "&&":
		ta, oka := a.truth()
		tb, okb := b.truth()
		switch {
		case oka && !ta, okb && !tb:
			return boolean(false)
		case oka && okb:
			return boolean(true)
		case a.k == kScoped && (okb && tb || b.k == kUnknown):
			return a // option == X && something: still scoped to X
		case b.k == kScoped && (oka && ta || a.k == kUnknown):
			return b
		}
		return unknown
	case "||":
		ta, oka := a.truth()
		tb, okb := b.truth()
		switch {
		case oka && ta, okb && tb:
			return boolean(true)
		case oka && okb:
			return boolean(false)
		case a.k == kScoped && b.k == kScoped:
			return scoped(append(append([]int{}, a.ids...), b.ids...))
		case a.k == kScoped && oka:
			return a
		case b.k == kScoped && okb:
			return b
		}
		return unknown
	case "==", "!=":
		eq, ok := equal(a, b)
		if s, isScope := scopeOf(a, b); isScope {
			if op == "!=" {
				return others(s.ids)
			}
			return s
		}
		if !ok {
			return unknown
		}
		return boolean(eq == (op == "=="))
	case "<", ">", "<=", ">=":
		if !a.numeric() || !b.numeric() {
			return unknown
		}
		switch op {
		case "<":
			return boolean(a.n < b.n)
		case ">":
			return boolean(a.n > b.n)
		case "<=":
			return boolean(a.n <= b.n)
		}
		return boolean(a.n >= b.n)
	case "+":
		if a.k == kString || b.k == kString || a.k == kPattern || b.k == kPattern {
			return concat(a, b)
		}
		fallthrough
	case "-", "*", "/", "%":
		if !a.numeric() || !b.numeric() {
			return unknown
		}
		x, y := a.n, b.n
		var r float64
		switch op {
		case "+":
			r = x + y
		case "-":
			r = x - y
		case "*":
			r = x * y
		case "/":
			if y == 0 {
				return unknown
			}
			r = x / y
		case "%":
			if int64(y) == 0 {
				return unknown
			}
			return integer(float64(int64(x) % int64(y)))
		}
		if a.k == kInt && b.k == kInt {
			if op == "/" {
				return integer(float64(int64(x) / int64(y)))
			}
			return integer(r)
		}
		return num(r)
	}
	return unknown
}

// unary is the result of ! or - on a value.
func unary(op string, x Value) Value {
	if x.pooled {
		x.pooled = false
		r := unary(op, x)
		r.pooled = true
		return r
	}
	if x.k == kUnknown && x.hint != nil {
		return hinted(unary(op, *x.hint))
	}
	if op == "!" {
		if t, ok := x.truth(); ok {
			return boolean(!t)
		}
		switch x.k {
		case kScoped:
			return others(x.ids)
		case kOthers:
			return scoped(x.ids)
		}
		return unknown
	}
	if x.numeric() {
		if x.k == kInt {
			return integer(-x.n)
		}
		return num(-x.n)
	}
	return unknown
}

// scopeOf is option == X, when one side is the option an event was called
// for and the other an option ID: a condition true for that option only.
func scopeOf(a, b Value) (Value, bool) {
	if a.k == kParam && b.k == kOption {
		return scoped(b.ids), true
	}
	if b.k == kParam && a.k == kOption {
		return scoped(a.ids), true
	}
	return Value{}, false
}

// equal compares two known values as Papyrus does: strings without regard
// to case.
func equal(a, b Value) (bool, bool) {
	if !a.known() || !b.known() {
		return false, false
	}
	if a.k == kString && b.k == kString {
		return strings.EqualFold(a.s, b.s), true
	}
	if a.numeric() && b.numeric() {
		return a.n == b.n, true
	}
	if a.k == kNone || b.k == kNone {
		return a.k == b.k, true
	}
	if a.k == kOption && b.k == kOption {
		return a.ids[0] == b.ids[0], true
	}
	return false, false
}

// cast is v as typ.
func cast(v Value, typ string) Value {
	if v.pooled {
		v.pooled = false
		r := cast(v, typ)
		r.pooled = true
		return r
	}
	if v.k == kUnknown && v.hint != nil {
		return hinted(cast(*v.hint, typ))
	}
	switch strings.ToLower(typ) {
	case "string":
		if t, ok := v.text(); ok {
			return str(t)
		}
		if v.k == kPattern {
			return v
		}
	case "int":
		if v.numeric() {
			return integer(v.n)
		}
		if v.k == kString {
			if n, err := strconv.ParseFloat(strings.TrimSpace(v.s), 64); err == nil {
				return integer(n)
			}
		}
	case "float":
		if v.numeric() {
			return num(v.n)
		}
	case "bool":
		if t, ok := v.truth(); ok {
			return boolean(t)
		}
		if v.k == kScoped || v.k == kOthers {
			return v // the compiler casts each test of option == X to Bool
		}
	default:
		// To an object or array type, None stays None (the compiler casts
		// the None it compares with), and an array stays itself. A form
		// another script is attached to is taken as the script cast to.
		if v.k == kNone || v.k == kArray {
			return v
		}
		if v.k == kRemote {
			v.s = strings.ToLower(typ)
			return v
		}
	}
	return unknown
}

// copyValue is a deep copy, so a page walk starts from the state after
// initialization each time.
func copyValue(v Value) Value {
	if v.k == kArray && v.arr != nil {
		a := make([]Value, len(*v.arr))
		for i, e := range *v.arr {
			a[i] = copyValue(e)
		}
		v.arr = &a
	}
	return v
}

// remoteProperty is the named property of the script a kRemote is taken
// as, or, when the form has no script by that name, of the only script on
// it that has the property.
func (v Value) remoteProperty(name string) (RemoteValue, bool) {
	key := strings.ToLower(name)
	if table, ok := v.rem[v.s]; ok {
		rv, ok := table[key]
		return rv, ok
	}
	var found RemoteValue
	n := 0
	for _, table := range v.rem {
		if rv, ok := table[key]; ok {
			found = rv
			n++
		}
	}
	return found, n == 1
}
