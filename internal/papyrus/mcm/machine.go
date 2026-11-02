package mcm

import (
	"lexicon/internal/papyrus/ir"
)

// step is one choice on the way to a statement: which arm of which If.
type step struct {
	at  *ir.If
	arm int
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
	depth   int
	budget  int // statements left before the run gives up

	// call handles calls to SKI_ConfigBase's functions; ok=false for any
	// other function.
	call func(name string, args []Value, line int) (Value, bool)
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
	_, ret := m.exec(f.Body)
	m.frames = m.frames[:len(m.frames)-1]
	m.depth--
	return ret
}

// exec runs statements; returned is true when a Return was certainly met.
func (m *machine) exec(body []ir.Stmt) (returned bool, value Value) {
	for _, s := range body {
		if m.budget--; m.budget <= 0 {
			return true, unknown
		}
		switch s := s.(type) {
		case *ir.Assign:
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
		case *ir.While:
			m.loop(s)
		}
	}
	return false, unknown
}

// ifStmt runs the arm a known condition picks, or every arm a condition
// that depends on the game might pick, each with its place in the path.
func (m *machine) ifStmt(s *ir.If) (bool, Value) {
	certain := false
	for i, b := range s.Branches {
		c := m.eval(b.Cond)
		t, known := c.truth()
		if known && !t {
			continue
		}
		m.path = append(m.path, step{s, i})
		if c.k == kScoped {
			m.scope = append(m.scope, c.ids)
		}
		r, v := m.exec(b.Body)
		if c.k == kScoped {
			m.scope = m.scope[:len(m.scope)-1]
		}
		m.path = m.path[:len(m.path)-1]
		if known {
			certain = true
			if r {
				return true, v
			}
			break
		}
	}
	if !certain && s.Else != nil {
		m.path = append(m.path, step{s, len(s.Branches)})
		m.exec(s.Else)
		m.path = m.path[:len(m.path)-1]
	}
	return false, unknown
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
			m.exec(s.Body)
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
				return (*arr.arr)[n]
			}
		}
		return unknown
	case *ir.Member:
		x := m.eval(e.X)
		if x.k == kArray && x.arr != nil && ir.Key(e.Name) == "length" {
			return integer(float64(len(*x.arr)))
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
		x := m.eval(e.X)
		if e.Op == "!" {
			if t, ok := x.truth(); ok {
				return boolean(!t)
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
	if c.Object != nil || c.Parent {
		return unknown // another object's function, or SKI_ConfigBase's own
	}
	if m.call != nil {
		if v, ok := m.call(c.Method, args, c.Line); ok {
			return v
		}
	}
	return m.invoke(m.function("", c.Method), args)
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
