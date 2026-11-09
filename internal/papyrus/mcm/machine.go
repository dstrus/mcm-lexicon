package mcm

import (
	"math"
	"strings"

	"lexicon/internal/papyrus/ir"
)

// step is one choice on the way to a statement: which arm of which If.
// step is one way a run went: an arm of an If, or a call into one of the
// script's own functions (at nil), so that the same If in a function run
// from two call sites is two Ifs, as each call runs it afresh.
type step struct {
	at   *ir.If
	arm  int
	call *ir.Call
}

// machine runs a script's functions over Values. Where a condition depends
// on the game, it runs every arm that might be taken; where a value does,
// it's unknown. It's an approximation that leans toward finding every
// option a menu can show.
type machine struct {
	chain   []*ir.Script // the script, then its ancestors with source
	globals map[string]*Value
	frames  []map[string]*Value
	path    []step
	scope   [][]int // the options a scoped condition limits the current statements to
	state   string  // the script's state, as GoToState with a known name last left it
	depth   int
	budget  int // statements left before the run gives up

	// call handles calls to SKI_ConfigBase's functions; ok=false for any
	// other function.
	call func(site *ir.Call, args []Value) (Value, bool)

	// arms, when set, hears of each If the run takes more than one way
	// through: the page's cursor follows each arm from where the If began.
	arms armHooks
	rets [][]Value // per call running, what its arms that returned returned
	// effect, when set, hears of each thing a run does that might show: a
	// call it doesn't follow into (but a log trace), or a change to a script
	// variable. A compiled script keeps every call's result in a temporary,
	// so a call counts wherever it is, which errs toward something happening.
	effect   func()
	guard    []int // the last If's, when it returns for every option but these: "If option != X / Return"
	skipped  bool  // the last If's: whether no arm may have run
	returned bool  // the last If's: whether an arm returned, so what follows runs only when it didn't

	// settings is what SKSE plugins' functions return for a setting from
	// an INI file the plugin keeps, by "script.function", then the
	// setting's name, lowercased.
	settings Settings
	// looping counts the loops running whose count the game decides, which
	// the run goes through once: an option one adds is added once per time
	// round.
	looping int
	// newGame, while a page runs a second time to see what a new game
	// shows, counts what's only a hint (a global's value, a plugin's INI
	// setting) as known, so conditions on it go the way they would.
	newGame bool
}

// armHooks hear of an If's arms as the machine runs them: enter as it
// starts, arm before each arm it runs, done after each (returned when the
// arm returns from its function), and exit as it ends, with skipped true
// when no arm may run at all; and call and ret around each call into one
// of the script's own functions.
type armHooks interface {
	enter()
	arm()
	done(returned bool)
	exit(skipped bool)
	call()
	ret()
}

const maxDepth = 10

func newMachine(chain []*ir.Script) *machine {
	return &machine{chain: chain, globals: map[string]*Value{}}
}

// function finds a function the script or an ancestor defines, in the
// named state first, then the empty state.
func (m *machine) function(state, name string) *ir.Function {
	for _, s := range m.chain {
		if state != "" {
			if f := s.State(state).Function(name); f != nil {
				return f
			}
		}
	}
	for _, s := range m.chain {
		if f := s.States[0].Function(name); f != nil {
			return f
		}
	}
	return nil
}

func (m *machine) lookup(name string) *Value {
	k := ir.Key(name)
	if n := len(m.frames); n > 0 {
		if v, ok := m.frames[n-1][k]; ok {
			return v
		}
	}
	if v, ok := m.globals[k]; ok {
		return v
	}
	return nil
}

// isLocal reports whether name is a local of the running function.
func (m *machine) isLocal(name string) bool {
	if n := len(m.frames); n > 0 {
		_, ok := m.frames[n-1][ir.Key(name)]
		return ok
	}
	return false
}

// localTarget reports whether an assignment's target is a local of the
// running function (or an element of one): no effect outside it.
func (m *machine) localTarget(target ir.Expr) bool {
	switch t := target.(type) {
	case *ir.Ident:
		return m.isLocal(t.Name) || strings.HasPrefix(t.Name, "::")
	case *ir.Index:
		return m.localTarget(t.X)
	}
	return false
}

// traceOnly reports whether a call only writes to the Papyrus log, which
// a player never sees.
func traceOnly(c *ir.Call) bool {
	if id, ok := c.Object.(*ir.Ident); ok && strings.EqualFold(id.Name, "Debug") {
		return strings.EqualFold(c.Method, "Trace") || strings.EqualFold(c.Method, "TraceStack")
	}
	return false
}

// set assigns to a local or a script variable of that name. A name that's
// neither is a property the script inherits (SKI_ConfigBase's ModName and
// Pages), so it's a script variable.
func (m *machine) set(name string, v Value) {
	if p := m.lookup(name); p != nil {
		*p = v
		return
	}
	m.globals[ir.Key(name)] = &v
}

// declare makes a local of the running function.
func (m *machine) declare(name string, v Value) {
	if n := len(m.frames); n > 0 {
		m.frames[n-1][ir.Key(name)] = &v
		return
	}
	m.globals[ir.Key(name)] = &v
}

// invoke runs a function with the arguments given, returning its value.
func (m *machine) invoke(f *ir.Function, args []Value) Value {
	if f == nil || m.depth >= maxDepth || m.budget <= 0 {
		return unknown
	}
	frame := map[string]*Value{}
	for i, p := range f.Params {
		v := unknown
		if i < len(args) {
			v = args[i]
		} else if p.Default != nil {
			v = m.eval(p.Default)
		}
		frame[ir.Key(p.Name)] = &v
	}
	m.depth++
	m.frames = append(m.frames, frame)
	m.rets = append(m.rets, nil)
	_, ret := m.exec(f.Body)
	// The function returns whatever one of its ways through returns: the
	// value its arms that returned agree on with the last, or unknown.
	for _, r := range m.rets[len(m.rets)-1] {
		ret = join(ret, &r)
	}
	m.rets = m.rets[:len(m.rets)-1]
	m.frames = m.frames[:len(m.frames)-1]
	m.depth--
	return ret
}

// fillDefaults evaluates, for the first last+1 parameters, the defaults of
// those not given, and returns that prefix.
func fillDefaults(f *ir.Function, out []Value, given []bool, last int) []Value {
	for i := 0; i <= last; i++ {
		if !given[i] {
			out[i] = unknown
			if f.Params[i].Default != nil {
				out[i] = evalConst(f.Params[i].Default)
			}
		}
	}
	return out[:last+1]
}

// evalConst is a parameter default's value: a constant, or unknown.
func evalConst(e ir.Expr) Value {
	m := &machine{globals: map[string]*Value{}}
	return m.eval(e)
}

// exec runs statements; returned is true when a Return was certainly met.
func (m *machine) exec(body []ir.Stmt) (returned bool, value Value) {
	// After an If whose arm returned, the rest of the block runs only when
	// that arm didn't: its fall-through, which the path records as an arm
	// of its own until the block ends.
	defer func(n int) { m.path = m.path[:n] }(len(m.path))
	// Past a guard, the rest of the block is about the options it let by.
	defer func(n int) { m.scope = m.scope[:n] }(len(m.scope))
	for _, s := range body {
		if m.budget--; m.budget <= 0 {
			return true, unknown
		}
		switch s := s.(type) {
		case *ir.Assign:
			if m.effect != nil && !s.Declare && !m.localTarget(s.Target) {
				m.effect()
			}
			if id, ok := s.Target.(*ir.Ident); ok && s.Declare {
				m.declare(id.Name, m.eval(s.Value))
			} else {
				m.assign(s.Target, m.eval(s.Value))
			}
		case *ir.ExprStmt:
			m.eval(s.X)
		case *ir.Return:
			v := unknown
			if s.Value != nil {
				v = m.eval(s.Value)
			}
			return true, v
		case *ir.If:
			if r, v := m.ifStmt(s); r {
				return true, v
			}
			if m.returned {
				m.path = append(m.path, step{at: s, arm: fallThrough})
			}
			if m.guard != nil {
				m.scope, m.guard = append(m.scope, m.guard), nil
			}
		case *ir.While:
			m.loop(s)
		}
	}
	return false, unknown
}

// ifStmt runs the arm a known condition picks, or every arm a condition
// that depends on the game might pick, each with its place in the path.
// Arms that might run, run one after another, so a counter one arm bumps
// keeps counting in the next and every option is found. But where some arm
// is certain to run (an Else, or a condition known to hold) and the arms
// leave a variable differently, it depends on the game, so it's unknown
// afterward (option IDs are pooled): Accuracy's "If found / Status = True /
// Else / Status = False".
func (m *machine) ifStmt(s *ir.If) (bool, Value) {
	if m.arms == nil {
		return m.ifArms(s)
	}
	m.arms.enter()
	r, v := m.ifArms(s)
	m.arms.exit(m.skipped)
	return r, v
}

// fallThrough is the arm a path takes past an If whose arms returned.
const fallThrough = -1

func (m *machine) ifArms(s *ir.If) (bool, Value) {
	m.skipped, m.guard = false, nil
	anyReturned := false
	defer func() { m.returned = anyReturned }()
	names := ifAssigned(s)
	// What the variables hold if no arm runs: a local with a value (a
	// counter being wrapped, "If n > 2 / n = 0 / EndIf") is joined with
	// what the arms set, so it's uncertain after. A script variable is
	// left as the arms set it, so an option ID assigned in an arm that
	// may not run stays pooled.
	before := m.values(names)
	var outcomes [][]*Value // the variables' values after each arm that carries on
	skipped := false        // whether no arm may run
	run := func(arm int, body []ir.Stmt, scope *Value) (bool, Value) {
		m.path = append(m.path, step{at: s, arm: arm})
		if scope != nil {
			m.scope = append(m.scope, scope.ids)
		}
		if m.arms != nil {
			m.arms.arm()
		}
		r, v := m.exec(body)
		if m.arms != nil {
			m.arms.done(r)
		}
		if r {
			anyReturned = true
			if n := len(m.rets); n > 0 {
				m.rets[n-1] = append(m.rets[n-1], v)
			}
		}
		if scope != nil {
			m.scope = m.scope[:len(m.scope)-1]
		}
		m.path = m.path[:len(m.path)-1]
		if !r {
			outcomes = append(outcomes, m.values(names))
		}
		return r, v
	}
	certain := false
	for i, b := range s.Branches {
		c := m.eval(b.Cond)
		t, known := c.truth()
		if known && !t {
			continue
		}
		var scope *Value
		if c.k == kScoped {
			scope = &c
		}
		r, v := run(i, b.Body, scope)
		if r && c.k == kOthers && len(s.Branches) == 1 && s.Else == nil {
			defer func() { m.guard = c.ids }()
		}
		if known {
			certain = true
			if r && len(outcomes) == 0 {
				return true, v // the only way through returns
			}
			break
		}
	}
	if !certain {
		if s.Else != nil {
			run(len(s.Branches), s.Else, nil)
		}
		if s.Else == nil || !exhaustive(s) {
			skipped = true
		}
	}
	m.skipped = skipped && !certain
	if len(outcomes) < 2 && !skipped {
		return false, unknown
	}
	for i, name := range names {
		if len(outcomes) == 0 || outcomes[0][i] == nil {
			continue
		}
		merged := *outcomes[0][i]
		for _, o := range outcomes[1:] {
			merged = join(merged, o[i])
		}
		// A local that may keep its earlier value: a counter being wrapped.
		// A script variable is left pooled: a compiled script declares every
		// one with a zero, which isn't a value it was given.
		if skipped && before[i] != nil && m.isLocal(name) {
			merged = join(merged, before[i])
		}
		// A script variable an arm that may not run set, to something else
		// than it held: pooled, so what an option shows isn't taken from it.
		if skipped && !m.isLocal(name) && merged.k != kOption && (before[i] == nil || !same(merged, *before[i])) {
			merged.pooled = true
		}
		m.set(name, merged)
	}
	return false, unknown
}

// exhaustive reports whether an If's arms cover every way through: it
// ends in an Else that's more than the rest of an ElseIf chain. A compiled
// script's ElseIf is an Else holding the condition's temporaries and an
// If, so this reads it as its source does.
func exhaustive(s *ir.If) bool {
	for s.Else != nil {
		inner := continuation(s.Else)
		if inner == nil {
			return true
		}
		s = inner
	}
	return false
}

// continuation is the If an Else holds when it's only the next step of an
// ElseIf chain: temporaries, then that If.
func continuation(body []ir.Stmt) *ir.If {
	if len(body) == 0 {
		return nil
	}
	for _, st := range body[:len(body)-1] {
		a, ok := st.(*ir.Assign)
		if !ok {
			return nil
		}
		if id, ok := a.Target.(*ir.Ident); !ok || !strings.HasPrefix(id.Name, "::") {
			return nil
		}
	}
	inner, _ := body[len(body)-1].(*ir.If)
	return inner
}

// ifAssigned is the plain variables an If's arms assign to, once each.
func ifAssigned(s *ir.If) []string {
	var body []ir.Stmt
	for _, b := range s.Branches {
		body = append(body, b.Body...)
	}
	body = append(body, s.Else...)
	seen := map[string]bool{}
	var out []string
	for _, n := range assigned(body) {
		if k := ir.Key(n); !seen[k] {
			seen[k] = true
			out = append(out, n)
		}
	}
	return out
}

// values is the variables' values now: copies, or nil for one not set.
func (m *machine) values(names []string) []*Value {
	out := make([]*Value, len(names))
	for i, n := range names {
		if p := m.lookup(n); p != nil {
			c := *p
			out[i] = &c
		}
	}
	return out
}

// join is what a variable holds after two ways through: the value both
// agree on, every option ID either holds, the longer of two lists where
// one starts the other, or unknown.
func join(a Value, b *Value) Value {
	if b == nil {
		return unknown
	}
	switch {
	case a.k == kOption && b.k == kOption:
		ids := append([]int{}, a.ids...)
		for _, id := range b.ids {
			if !containsInt(ids, id) {
				ids = append(ids, id)
			}
		}
		return Value{k: kOption, ids: ids}
	case same(a, *b):
		return a
	case textual(a) && textual(*b):
		return shared(a, *b)
	case a.k == kArray && b.k == kArray && a.arr != nil && b.arr != nil:
		// One list that starts the other, such as Pages with a VR-only
		// page last: the longer, so every page is found.
		short, long := *a.arr, *b.arr
		if len(short) > len(long) {
			short, long = long, short
		}
		for i := range short {
			if !same(short[i], long[i]) {
				return unknown
			}
		}
		if len(long) == len(*a.arr) {
			return a
		}
		return *b
	}
	return unknown
}

// same reports whether two values are known to be equal: the same scalar,
// or arrays holding the same values (two arrays built alike, as a script
// that rebuilds Pages on every version update does).
func same(a, b Value) bool {
	if a.k != b.k {
		return false
	}
	switch a.k {
	case kNone:
		return true
	case kString, kInt, kFloat, kBool, kGlobal:
		return a.s == b.s && a.n == b.n // a global whose value isn't known (NaN) is never the same
	case kArray:
		if a.arr == b.arr {
			return true
		}
		if a.arr == nil || b.arr == nil || len(*a.arr) != len(*b.arr) {
			return false
		}
		for i := range *a.arr {
			if !same((*a.arr)[i], (*b.arr)[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func containsInt(xs []int, x int) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// loop runs a loop while its condition is known to hold, up to a bound;
// when the condition depends on the game, the body runs once with what
// it assigns treated as unknown, so labels built from a counter read as
// patterns.
func (m *machine) loop(s *ir.While) {
	for i := 0; i < 128; i++ {
		t, known := m.eval(s.Cond).truth()
		if known && !t {
			return
		}
		if !known {
			for _, name := range assigned(s.Body) {
				m.set(name, unknown)
			}
			m.looping++
			m.exec(s.Body)
			m.looping--
			return
		}
		if r, _ := m.exec(s.Body); r {
			return
		}
	}
}

// assigned is the names a body assigns to.
func assigned(body []ir.Stmt) []string {
	var out []string
	var walk func([]ir.Stmt)
	walk = func(ss []ir.Stmt) {
		for _, s := range ss {
			switch s := s.(type) {
			case *ir.Assign:
				if id, ok := s.Target.(*ir.Ident); ok {
					out = append(out, id.Name)
				}
			case *ir.If:
				for _, b := range s.Branches {
					walk(b.Body)
				}
				walk(s.Else)
			case *ir.While:
				walk(s.Body)
			}
		}
	}
	walk(body)
	return out
}

func (m *machine) assign(target ir.Expr, v Value) {
	switch t := target.(type) {
	case *ir.Ident:
		m.set(t.Name, v)
	case *ir.Index:
		arr := m.eval(t.X)
		i := m.eval(t.I)
		if arr.k != kArray || arr.arr == nil || !i.numeric() {
			return
		}
		n := int(i.n)
		if n < 0 || n > 4096 {
			return
		}
		for len(*arr.arr) <= n {
			*arr.arr = append(*arr.arr, Value{k: kNone})
		}
		(*arr.arr)[n] = v
	}
}

func (m *machine) eval(e ir.Expr) Value {
	switch e := e.(type) {
	case *ir.Const:
		switch e.Kind {
		case ir.ConstString:
			return str(e.String)
		case ir.ConstInt:
			return integer(e.Number)
		case ir.ConstFloat:
			return num(e.Number)
		case ir.ConstBool:
			return boolean(e.Number != 0)
		}
		return Value{k: kNone}
	case *ir.Ident:
		if p := m.lookup(e.Name); p != nil {
			return *p
		}
		return unknown
	case *ir.Index:
		arr := m.eval(e.X)
		i := m.eval(e.I)
		if arr.k == kArray && arr.arr != nil && i.numeric() {
			if n := int(i.n); n >= 0 && n < len(*arr.arr) {
				v := (*arr.arr)[n]
				v.pooled = v.pooled || i.pooled || arr.pooled
				return v
			}
		}
		return unknown
	case *ir.Member:
		// self.X is the script's own X.
		if id, ok := e.X.(*ir.Ident); ok && ir.Key(id.Name) == "self" {
			return m.eval(&ir.Ident{Name: e.Name})
		}
		x := m.eval(e.X)
		if x.k == kArray && x.arr != nil && ir.Key(e.Name) == "length" {
			return integer(float64(len(*x.arr)))
		}
		// Another script's property, on the form a property points to: a
		// new game's value, so a hint, as a global's is.
		if x.k == kRemote {
			if rv, ok := x.remoteProperty(e.Name); ok {
				switch {
				case rv.Number != nil:
					return m.hint(num(*rv.Number))
				case rv.String != nil:
					return m.hint(str(*rv.String))
				case rv.Global != nil:
					return global(*rv.Global)
				}
			}
			return unknown
		}
		// A global's Value property, which GetValue reads.
		if x.k == kGlobal && ir.Key(e.Name) == "value" && !math.IsNaN(x.n) {
			return m.hint(num(x.n))
		}
		return unknown
	case *ir.Call:
		return m.callExpr(e)
	case *ir.Binary:
		a := m.eval(e.L)
		// Short circuits, so a known left side spares the right's effects.
		if t, ok := a.truth(); ok && ((e.Op == "&&" && !t) || (e.Op == "||" && t)) {
			return boolean(t)
		}
		return binary(e.Op, a, m.eval(e.R))
	case *ir.Unary:
		return unary(e.Op, m.eval(e.X))
	case *ir.Cast:
		return cast(m.eval(e.X), e.Type)
	case *ir.NewArray:
		if size := m.eval(e.Size); size.numeric() {
			return newArray(int(size.n))
		}
		return unknown
	}
	return unknown
}

func (m *machine) callExpr(c *ir.Call) Value {
	args := make([]Value, len(c.Args))
	for i, a := range c.Args {
		args[i] = m.eval(a)
	}
	if v, ok := m.globalCall(c, args); ok {
		return v
	}
	if c.Object != nil || c.Parent {
		m.affect(c)
		return unknown // another object's function, or SKI_ConfigBase's own
	}
	if strings.EqualFold(c.Method, "GoToState") && len(args) > 0 && args[0].k == kString {
		m.state = ir.Key(args[0].s)
	}
	if m.call != nil {
		if v, ok := m.call(c, args); ok {
			m.affect(c)
			return v
		}
	}
	f := m.function(m.state, c.Method)
	if f == nil || m.depth >= maxDepth {
		m.affect(c) // a native function, or one too deep to follow
	}
	if c.Names != nil && f != nil {
		args = byName(f, args, c.Names)
	}
	m.path = append(m.path, step{call: c})
	if m.arms != nil {
		m.arms.call()
	}
	v := m.invoke(f, args)
	if m.arms != nil {
		m.arms.ret()
	}
	m.path = m.path[:len(m.path)-1]
	return v
}

// globalCall is a call on a global variable the plugin fills, named by the
// script's own property or variable: GetValue and GetValueInt read what it
// holds, and SetValue, SetValueInt, and Mod change it for the rest of the
// run, which counts as an effect, as any call that changes the game does.
// ok is false for a call on anything else.
func (m *machine) globalCall(c *ir.Call, args []Value) (Value, bool) {
	if c.Parent {
		return Value{}, false
	}
	switch c.Object.(type) {
	case *ir.Ident, *ir.Member:
	default:
		return Value{}, false // a call's result, say: not evaluated twice
	}
	if v, ok := m.settingCall(c, args); ok {
		return v, true
	}
	g := m.eval(c.Object)
	if g.k != kGlobal {
		return Value{}, false
	}
	held := !math.IsNaN(g.n)
	arg := math.NaN()
	if len(args) > 0 {
		if v, ok := onNewGame(args[0]); ok && v.numeric() {
			arg = v.n
		}
	}
	// What it holds is a new game's value: a hint, never a known value,
	// so no condition turns on it.
	switch ir.Key(c.Method) {
	case "getvalue":
		if held {
			return m.hint(num(g.n)), true
		}
		return unknown, true
	case "getvalueint":
		if held {
			return m.hint(integer(g.n)), true
		}
		return unknown, true
	case "setvalue":
		m.setGlobal(c.Object, arg)
	case "setvalueint":
		m.setGlobal(c.Object, math.Trunc(arg))
	case "mod":
		m.setGlobal(c.Object, g.n+arg) // NaN unless both are known
	default:
		return Value{}, false
	}
	m.affect(c)
	return Value{k: kNone}, true
}

// hint is a value that's what it is on a new game: known while the page
// runs as a new game, else only a hint.
func (m *machine) hint(v Value) Value {
	if m.newGame {
		return v
	}
	return hinted(v)
}

// settingCall is a call to an SKSE plugin's function that returns a
// setting from an INI file the plugin keeps (VRIK.VrikGetSetting("x")):
// the setting's value there, as a hint, or unknown when the file doesn't
// set it. It still counts as a call the run doesn't follow into.
func (m *machine) settingCall(c *ir.Call, args []Value) (Value, bool) {
	id, ok := c.Object.(*ir.Ident)
	if !ok || m.settings == nil || m.lookup(id.Name) != nil {
		return Value{}, false // a variable's function, not a script's
	}
	table, ok := m.settings[ir.Key(id.Name)+"."+ir.Key(c.Method)]
	if !ok {
		return Value{}, false
	}
	m.affect(c)
	if len(args) == 0 || args[0].k != kString {
		return unknown, true
	}
	if n, ok := table[strings.ToLower(args[0].s)]; ok {
		return m.hint(num(n)), true
	}
	return unknown, true
}

// setGlobal changes what the global the script names holds.
func (m *machine) setGlobal(target ir.Expr, n float64) {
	if mem, ok := target.(*ir.Member); ok {
		if id, ok := mem.X.(*ir.Ident); ok && ir.Key(id.Name) == "self" {
			target = &ir.Ident{Name: mem.Name}
		}
	}
	if id, ok := target.(*ir.Ident); ok {
		m.set(id.Name, global(n))
	}
}

// affect tells effect of a call the run doesn't follow into.
func (m *machine) affect(c *ir.Call) {
	if m.effect != nil && !traceOnly(c) {
		m.effect()
	}
}

// byName puts a call's arguments in the function's parameter order: the
// positional ones first, in order, then each named one at its parameter.
// A parameter given neither way is left for its default.
func byName(f *ir.Function, args []Value, names []string) []Value {
	out := make([]Value, len(f.Params))
	given := make([]bool, len(f.Params))
	next := 0
	for i, a := range args {
		name := ""
		if i < len(names) {
			name = names[i]
		}
		at := -1
		if name != "" {
			for j, p := range f.Params {
				if ir.Key(p.Name) == ir.Key(name) {
					at = j
					break
				}
			}
		} else if next < len(f.Params) {
			at = next
			next++
		}
		if at >= 0 {
			out[at], given[at] = a, true
		}
	}
	// Trailing parameters not given fall to their defaults in invoke, which
	// takes a shorter list; one not given before a given one has its default
	// evaluated here, so the list keeps its shape.
	last := -1
	for i, g := range given {
		if g {
			last = i
		}
	}
	return fillDefaults(f, out, given, last)
}

// snapshot copies the script variables, so each page walk can start from
// the state initialization left.
func (m *machine) snapshot() map[string]Value {
	out := make(map[string]Value, len(m.globals))
	for k, v := range m.globals {
		out[k] = copyValue(*v)
	}
	return out
}

func (m *machine) restore(s map[string]Value) {
	m.globals = make(map[string]*Value, len(s))
	for k, v := range s {
		c := copyValue(v)
		m.globals[k] = &c
	}
	m.frames, m.path, m.scope, m.depth = nil, nil, nil, 0
}
