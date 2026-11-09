package source

import (
	"fmt"
	"strconv"
	"strings"

	"lexicon/internal/papyrus/ir"
)

// Result is a parsed script, and the lines that couldn't be read.
type Result struct {
	Script *ir.Script
	Errors []Error
}

type Error struct {
	Line int
	Msg  string
}

func (e Error) Error() string { return fmt.Sprintf("line %d: %s", e.Line, e.Msg) }

// Parse reads a script's source. It never fails outright: a line it can't
// read is skipped and reported in Errors, and a script without a ScriptName
// header comes back with no name.
func Parse(src string) Result {
	p := &parser{toks: lex(src)}
	s := &ir.Script{States: []*ir.State{{Name: ""}}}
	p.script = s
	for !p.at(tEOF) {
		p.top()
	}
	return Result{Script: s, Errors: p.errs}
}

type parser struct {
	toks   []token
	pos    int
	script *ir.Script
	errs   []Error
}

// bail is how a parse error unwinds to the start of the next line.
type bail struct{}

func (p *parser) peek() token         { return p.toks[p.pos] }
func (p *parser) peekAt(n int) token  { return p.toks[min(p.pos+n, len(p.toks)-1)] }
func (p *parser) at(k kind) bool      { return p.peek().kind == k }
func (p *parser) atOp(op string) bool { return p.peek().kind == tOp && p.peek().text == op }
func (p *parser) atWord(w string) bool {
	return p.peek().kind == tIdent && strings.EqualFold(p.peek().text, w)
}
func (p *parser) next() token {
	t := p.toks[p.pos]
	if t.kind != tEOF {
		p.pos++
	}
	return t
}

func (p *parser) fail(format string, a ...any) {
	p.errs = append(p.errs, Error{Line: p.peek().line, Msg: fmt.Sprintf(format, a...)})
	panic(bail{})
}

func (p *parser) expectOp(op string) {
	if !p.atOp(op) {
		p.fail("expected %q, found %q", op, p.peek().text)
	}
	p.next()
}

func (p *parser) ident() string {
	if !p.at(tIdent) {
		p.fail("expected a name, found %q", p.peek().text)
	}
	return p.next().text
}

// endLine consumes whatever is left of the line, ignoring flags such as
// Auto, Conditional, Hidden, Global, and Native.
func (p *parser) endLine() {
	for !p.at(tNewline) && !p.at(tEOF) {
		p.next()
	}
	if p.at(tNewline) {
		p.next()
	}
}

// recover runs fn, and if it fails, skips to the next line.
func (p *parser) recover(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(bail); !ok {
				panic(r)
			}
			p.endLine()
		}
	}()
	fn()
}

// lineFlags reports whether the rest of the current line holds word,
// without consuming it.
func (p *parser) lineHas(word string) bool {
	for i := p.pos; i < len(p.toks) && p.toks[i].kind != tNewline && p.toks[i].kind != tEOF; i++ {
		if p.toks[i].kind == tIdent && strings.EqualFold(p.toks[i].text, word) {
			return true
		}
	}
	return false
}

// top reads one top-level line: the header, an import, a property, a
// variable, a function or event, or a state.
func (p *parser) top() {
	p.recover(func() {
		switch {
		case p.at(tNewline):
			p.next()
		case p.atWord("ScriptName"):
			p.next()
			p.script.Name = p.ident()
			if p.atWord("extends") {
				p.next()
				p.script.Parent = p.ident()
			}
			p.endLine()
		case p.atWord("Import"):
			p.endLine()
		case p.atWord("Auto") && p.peekAt(1).kind == tIdent && strings.EqualFold(p.peekAt(1).text, "State"):
			p.next()
			p.state(true)
		case p.atWord("State"):
			p.state(false)
		case p.atWord("Function") || p.atWord("Event") || p.isTypedFunction():
			p.script.States[0].Functions = append(p.script.States[0].Functions, p.function())
		default:
			p.declaration()
		}
	})
}

// isTypedFunction reports whether the line is "Type Function …" or
// "Type[] Function …".
func (p *parser) isTypedFunction() bool {
	if !p.at(tIdent) {
		return false
	}
	n := 1
	if p.peekAt(1).kind == tOp && p.peekAt(1).text == "[" && p.peekAt(2).kind == tOp && p.peekAt(2).text == "]" {
		n = 3
	}
	t := p.peekAt(n)
	return t.kind == tIdent && strings.EqualFold(t.text, "Function")
}

// typeName reads a type: a name, with [] for an array.
func (p *parser) typeName() string {
	name := p.ident()
	if p.atOp("[") && p.peekAt(1).kind == tOp && p.peekAt(1).text == "]" {
		p.next()
		p.next()
		name += "[]"
	}
	return name
}

// declaration reads a property (Type Property Name [= value] [flags], or a
// full property with Get and Set functions) or a script variable.
func (p *parser) declaration() {
	typ := p.typeName()
	if p.atWord("Property") {
		p.next()
		v := ir.Variable{Name: p.ident(), Type: typ}
		if p.atOp("=") {
			p.next()
			v.Init = p.expr()
		}
		full := !p.lineHas("Auto") && !p.lineHas("AutoReadOnly")
		p.endLine()
		if full {
			// Get and Set functions until EndProperty. A Get that returns a
			// constant gives the property's value, as the compiled script
			// reads it; any other Get computes it.
			for !p.atWord("EndProperty") && !p.at(tEOF) {
				if p.atWord("Function") || p.isTypedFunction() {
					if f := p.function(); strings.EqualFold(f.Name, "Get") {
						if c := constantReturn(f); c != nil {
							v.Init = c
						} else {
							v.Computed, v.Get = true, f
						}
					}
					continue
				}
				p.next()
			}
			p.endLine()
		}
		p.script.Properties = append(p.script.Properties, v)
		return
	}
	v := ir.Variable{Name: p.ident(), Type: typ}
	if p.atOp("=") {
		p.next()
		v.Init = p.expr()
	}
	p.endLine()
	p.script.Variables = append(p.script.Variables, v)
}

// constantReturn is the constant a function's whole body returns, if
// that's all it does.
func constantReturn(f *ir.Function) ir.Expr {
	if len(f.Body) != 1 {
		return nil
	}
	if r, ok := f.Body[0].(*ir.Return); ok {
		if c, ok := r.Value.(*ir.Const); ok {
			return c
		}
	}
	return nil
}

func (p *parser) state(auto bool) {
	p.next() // State
	st := &ir.State{Name: p.ident(), Auto: auto}
	p.endLine()
	for !p.atWord("EndState") && !p.at(tEOF) {
		p.recover(func() {
			switch {
			case p.at(tNewline):
				p.next()
			case p.atWord("Function") || p.atWord("Event") || p.isTypedFunction():
				st.Functions = append(st.Functions, p.function())
			default:
				p.fail("expected a function or event in state %s", st.Name)
			}
		})
	}
	p.endLine()
	p.script.States = append(p.script.States, st)
}

// function reads a function or event with its body. A Native function
// has no body.
func (p *parser) function() *ir.Function {
	if !p.atWord("Function") && !p.atWord("Event") {
		p.typeName()
	}
	event := p.atWord("Event")
	p.next()
	f := &ir.Function{Name: p.ident(), Line: p.peek().line}
	p.expectOp("(")
	for !p.atOp(")") {
		prm := ir.Param{Type: p.typeName(), Name: p.ident()}
		if p.atOp("=") {
			p.next()
			prm.Default = p.expr()
		}
		f.Params = append(f.Params, prm)
		if !p.atOp(",") {
			break
		}
		p.next()
	}
	p.expectOp(")")
	native := p.lineHas("Native")
	p.endLine()
	if native {
		return f
	}
	end := "EndFunction"
	if event {
		end = "EndEvent"
	}
	f.Body = p.block(end)
	p.endLine()
	return f
}

// block reads statements until one of the words ends it, leaving that word
// for the caller.
func (p *parser) block(ends ...string) []ir.Stmt {
	var out []ir.Stmt
	for !p.at(tEOF) {
		for _, e := range ends {
			if p.atWord(e) {
				return out
			}
		}
		// A function's end word met where a statement was expected means
		// an If or While was left open; stop rather than swallow the file.
		if p.atWord("EndFunction") || p.atWord("EndEvent") || p.atWord("EndState") || p.atWord("Function") || p.atWord("Event") {
			return out
		}
		p.recover(func() {
			if s := p.statement(); s != nil {
				out = append(out, s)
			}
		})
	}
	return out
}

func (p *parser) statement() ir.Stmt {
	line := p.peek().line
	switch {
	case p.at(tNewline):
		p.next()
		return nil
	case p.atWord("If"):
		return p.ifStmt()
	case p.atWord("While"):
		p.next()
		w := &ir.While{Cond: p.expr(), Line: line}
		p.endLine()
		w.Body = p.block("EndWhile")
		p.endLine()
		return w
	case p.atWord("Return"):
		p.next()
		r := &ir.Return{Line: line}
		if !p.at(tNewline) && !p.at(tEOF) {
			r.Value = p.expr()
		}
		p.endLine()
		return r
	case p.isLocalDecl():
		p.typeName()
		name := p.ident()
		var s ir.Stmt
		if p.atOp("=") {
			p.next()
			s = &ir.Assign{Target: &ir.Ident{Name: name}, Value: p.expr(), Declare: true, Line: line}
		} else {
			s = &ir.Assign{Target: &ir.Ident{Name: name}, Value: &ir.Const{Kind: ir.ConstNone}, Declare: true, Line: line}
		}
		p.endLine()
		return s
	}
	x := p.expr()
	if p.at(tOp) {
		switch op := p.peek().text; op {
		case "=", "+=", "-=", "*=", "/=", "%=":
			p.next()
			v := p.expr()
			if op != "=" {
				v = &ir.Binary{Op: op[:1], L: x, R: v}
			}
			p.endLine()
			return &ir.Assign{Target: x, Value: v, Line: line}
		}
	}
	p.endLine()
	return &ir.ExprStmt{X: x, Line: line}
}

// isLocalDecl reports whether the line declares a local: "Type name" or
// "Type[] name".
func (p *parser) isLocalDecl() bool {
	if !p.at(tIdent) {
		return false
	}
	n := 1
	if p.peekAt(1).kind == tOp && p.peekAt(1).text == "[" && p.peekAt(2).kind == tOp && p.peekAt(2).text == "]" {
		n = 3
	}
	t := p.peekAt(n)
	if t.kind != tIdent {
		return false
	}
	// "x as int" isn't a declaration.
	return !strings.EqualFold(t.text, "as")
}

func (p *parser) ifStmt() ir.Stmt {
	line := p.peek().line
	s := &ir.If{Line: line}
	p.next() // If
	cond := p.expr()
	p.endLine()
	body := p.block("ElseIf", "Else", "EndIf")
	s.Branches = append(s.Branches, ir.Branch{Cond: cond, Body: body})
	for p.atWord("ElseIf") {
		p.next()
		var c ir.Expr = &ir.Dynamic{}
		p.recover(func() {
			c = p.expr()
			p.endLine()
		})
		s.Branches = append(s.Branches, ir.Branch{Cond: c, Body: p.block("ElseIf", "Else", "EndIf")})
	}
	if p.atWord("Else") {
		p.endLine()
		s.Else = p.block("EndIf")
	}
	if p.atWord("EndIf") {
		p.endLine()
	}
	return s
}

// Expressions, from the loosest binding to the tightest: ||, &&,
// comparisons, + and -, * / %, unary - and !, as, then postfix . [] ().

func (p *parser) expr() ir.Expr { return p.binary(0) }

var levels = [][]string{{"||"}, {"&&"}, {"==", "!=", "<", ">", "<=", ">="}, {"+", "-"}, {"*", "/", "%"}}

func (p *parser) binary(level int) ir.Expr {
	if level == len(levels) {
		return p.unary()
	}
	x := p.binary(level + 1)
	for p.at(tOp) && contains(levels[level], p.peek().text) {
		op := p.next().text
		x = &ir.Binary{Op: op, L: x, R: p.binary(level + 1)}
	}
	return x
}

func contains(ops []string, op string) bool {
	for _, o := range ops {
		if o == op {
			return true
		}
	}
	return false
}

func (p *parser) unary() ir.Expr {
	if p.atOp("-") || p.atOp("!") {
		op := p.next().text
		x := p.unary()
		if c, ok := x.(*ir.Const); ok && op == "-" && (c.Kind == ir.ConstInt || c.Kind == ir.ConstFloat) {
			return &ir.Const{Kind: c.Kind, Number: -c.Number}
		}
		return &ir.Unary{Op: op, X: x}
	}
	x := p.postfix()
	for p.atWord("as") {
		p.next()
		x = &ir.Cast{X: x, Type: p.typeName()}
	}
	return x
}

func (p *parser) postfix() ir.Expr {
	x := p.primary()
	for {
		switch {
		case p.atOp("."):
			p.next()
			name := p.ident()
			if p.atOp("(") {
				c := &ir.Call{Method: name, Line: p.peek().line}
				c.Args, c.Names = p.args()
				if id, ok := x.(*ir.Ident); ok && ir.Key(id.Name) == "parent" {
					c.Parent = true
				} else if !(ok && ir.Key(id.Name) == "self") {
					c.Object = x
				}
				x = c
			} else {
				x = &ir.Member{X: x, Name: name}
			}
		case p.atOp("["):
			p.next()
			i := p.expr()
			p.expectOp("]")
			x = &ir.Index{X: x, I: i}
		default:
			return x
		}
	}
}

// args reads a call's arguments. A named argument (name = value) keeps
// only its value.
// args reads a call's arguments, and the parameter each names, if any.
func (p *parser) args() ([]ir.Expr, []string) {
	p.expectOp("(")
	var out []ir.Expr
	var names []string
	named := false
	for !p.atOp(")") {
		name := ""
		if p.at(tIdent) && p.peekAt(1).kind == tOp && p.peekAt(1).text == "=" {
			name = p.peek().text
			named = true
			p.next()
			p.next()
		}
		out = append(out, p.expr())
		names = append(names, name)
		if !p.atOp(",") {
			break
		}
		p.next()
	}
	p.expectOp(")")
	if !named {
		names = nil
	}
	return out, names
}

func (p *parser) primary() ir.Expr {
	t := p.peek()
	switch t.kind {
	case tString:
		p.next()
		return &ir.Const{Kind: ir.ConstString, String: t.text}
	case tInt:
		p.next()
		n, err := strconv.ParseInt(t.text, 0, 64)
		if err != nil {
			return &ir.Dynamic{}
		}
		return &ir.Const{Kind: ir.ConstInt, Number: float64(int32(n))}
	case tFloat:
		p.next()
		f, err := strconv.ParseFloat(t.text, 64)
		if err != nil {
			return &ir.Dynamic{}
		}
		return &ir.Const{Kind: ir.ConstFloat, Number: f}
	case tIdent:
		switch strings.ToLower(t.text) {
		case "true":
			p.next()
			return &ir.Const{Kind: ir.ConstBool, Number: 1}
		case "false":
			p.next()
			return &ir.Const{Kind: ir.ConstBool, Number: 0}
		case "none":
			p.next()
			return &ir.Const{Kind: ir.ConstNone}
		case "new":
			p.next()
			typ := p.ident()
			p.expectOp("[")
			size := p.expr()
			p.expectOp("]")
			return &ir.NewArray{Type: typ, Size: size}
		}
		p.next()
		if p.atOp("(") {
			c := &ir.Call{Method: t.text, Line: t.line}
			c.Args, c.Names = p.args()
			return c
		}
		return &ir.Ident{Name: t.text}
	case tOp:
		if t.text == "(" {
			p.next()
			x := p.expr()
			p.expectOp(")")
			return x
		}
	}
	p.fail("unexpected %q", t.text)
	return nil
}
