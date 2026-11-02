package mcm

import (
	"strconv"
	"strings"
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
)

// Value is a value as far as the analysis can tell.
type Value struct {
	k     kind
	s     string
	n     float64
	arr   *[]Value // arrays are shared by reference, as in Papyrus
	parts []part   // a pattern's parts
	ids   []int    // an option's ID, or a scoped condition's options
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
func (v Value) known() bool {
	return v.k != kUnknown && v.k != kPattern && v.k != kScoped && v.k != kParam
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
	case kArray, kOption:
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
		if s, isScope := scopeOf(a, b); isScope && op == "==" {
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
