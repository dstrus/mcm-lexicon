// Package ir is the small intermediate form both of MCM Lexicon's Papyrus
// front ends lower into: source (.psc) and compiled (.pex) scripts become
// the same scripts, states, functions, statements, and expressions, so one
// analysis reads MCMs from either. It isn't a compiler's IR: it keeps just
// enough structure to follow assignments, calls, and conditions.
//
// Papyrus is case-insensitive, so every name here is compared with Key.
package ir

import "strings"

// Key is a name as Papyrus compares it.
func Key(name string) string { return strings.ToLower(name) }

type Script struct {
	Name       string
	Parent     string // the script it extends; "" for none
	Properties []Variable
	Variables  []Variable
	States     []*State // the empty state, named "", first
}

// Variable is a script variable or property, with its initial value when
// the declaration gives one.
type Variable struct {
	Name string
	Type string // as declared, such as "string[]"
	Init Expr   // nil when not given
	// Computed is a property whose Get function works its value out each
	// time it's read (MCM Recorder's IsSkyrimVR): the game decides it.
	Computed bool
	// Get is a computed property's Get function, when it's at hand: what
	// another script reads of the property, run once the script's
	// initialization has (some menus' settings read globals this way).
	Get *Function
}

type State struct {
	Name      string // "" for the empty state
	Auto      bool
	Functions []*Function
}

// Function is a function or an event.
type Function struct {
	Name   string
	Params []Param
	Body   []Stmt
	Line   int
}

type Param struct {
	Name    string
	Type    string
	Default Expr
}

// State is the state named name, if the script has one.
func (s *Script) State(name string) *State {
	for _, st := range s.States {
		if Key(st.Name) == Key(name) {
			return st
		}
	}
	return nil
}

// Function is the function named name in the state, if it has one.
func (st *State) Function(name string) *Function {
	if st == nil {
		return nil
	}
	for _, f := range st.Functions {
		if Key(f.Name) == Key(name) {
			return f
		}
	}
	return nil
}

// Stmt is a statement: Assign, ExprStmt, If, While, or Return.
type Stmt interface{ stmt() }

// Assign is target = value; a compound assignment (+=) is lowered to
// target = target + value, and a local declaration to an assignment with
// Declare set (its value None when it has none).
type Assign struct {
	Target  Expr
	Value   Expr
	Declare bool // declares a local
	Line    int
}

// ExprStmt is an expression evaluated for its effect: a call.
type ExprStmt struct {
	X    Expr
	Line int
}

// If is a chain of conditions and bodies, with an optional else.
type If struct {
	Branches []Branch
	Else     []Stmt
	Line     int
}

type Branch struct {
	Cond Expr
	Body []Stmt
}

type While struct {
	Cond Expr
	Body []Stmt
	Line int
}

type Return struct {
	Value Expr // nil for none
	Line  int
}

func (*Assign) stmt()   {}
func (*ExprStmt) stmt() {}
func (*If) stmt()       {}
func (*While) stmt()    {}
func (*Return) stmt()   {}

// Expr is an expression.
type Expr interface{ expr() }

// ConstKind is a constant's type.
type ConstKind int

const (
	ConstNone ConstKind = iota
	ConstString
	ConstInt
	ConstFloat
	ConstBool
)

type Const struct {
	Kind   ConstKind
	String string
	Number float64 // an int, a float, or a bool as 0 or 1
}

// Ident is a name: a variable, property, parameter, self, or parent.
type Ident struct{ Name string }

// Index is X[I].
type Index struct{ X, I Expr }

// Member is X.Name, a property of another object.
type Member struct {
	X    Expr
	Name string
}

// Call is Object.Method(Args), or Method(Args) on self when Object is nil.
// Parent is set for parent.Method(…).
type Call struct {
	Object Expr
	Parent bool
	Method string
	Args   []Expr
	// Names, when a source call names its arguments (f(b = 1)), is each
	// argument's parameter, "" for a positional one; nil when none is named.
	Names []string
	Line  int
}

type Binary struct {
	Op   string // ||, &&, ==, !=, <, >, <=, >=, +, -, *, /, %
	L, R Expr
}

type Unary struct {
	Op string // - or !
	X  Expr
}

// Cast is X as Type.
type Cast struct {
	X    Expr
	Type string
}

// NewArray is new Type[Size].
type NewArray struct {
	Type string
	Size Expr
}

// Dynamic is anything the front end couldn't express: its value is
// unknown to the analysis.
type Dynamic struct{}

func (*Const) expr()    {}
func (*Ident) expr()    {}
func (*Index) expr()    {}
func (*Member) expr()   {}
func (*Call) expr()     {}
func (*Binary) expr()   {}
func (*Unary) expr()    {}
func (*Cast) expr()     {}
func (*NewArray) expr() {}
func (*Dynamic) expr()  {}
