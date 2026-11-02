package pex

import (
	"fmt"
	"strings"

	"lexicon/internal/papyrus/ir"
)

// Script lowers the file's first object into the intermediate form.
func (f *File) Script() *ir.Script {
	if len(f.Objects) == 0 {
		return &ir.Script{States: []*ir.State{{}}}
	}
	o := f.Objects[0]
	s := &ir.Script{Name: o.Name, Parent: o.Parent}
	// An auto property keeps its value in a hidden variable; code in the
	// script reads and writes that variable, so it's named for the property.
	names := map[string]string{}
	values := map[string]Operand{}
	for _, v := range o.Variables {
		values[ir.Key(v.Name)] = v.Value
	}
	for _, p := range o.Properties {
		prop := ir.Variable{Name: p.Name, Type: p.Type}
		if p.AutoVar != "" {
			names[ir.Key(p.AutoVar)] = p.Name
			if v := values[ir.Key(p.AutoVar)]; v.Kind != OperandNone {
				prop.Init = operand(v, names)
			}
		} else if p.Get != nil && len(p.Get.Instructions) == 1 && p.Get.Instructions[0].Op == OpReturn {
			// An AutoReadOnly property's value is its read function's
			// constant.
			if v := p.Get.Instructions[0].Args[0]; v.Kind > OperandIdent {
				prop.Init = operand(v, names)
			}
		}
		s.Properties = append(s.Properties, prop)
	}
	for _, v := range o.Variables {
		if _, isProp := names[ir.Key(v.Name)]; isProp {
			continue
		}
		var init ir.Expr
		if v.Value.Kind != OperandNone {
			init = operand(v.Value, names)
		}
		s.Variables = append(s.Variables, ir.Variable{Name: v.Name, Type: v.Type, Init: init})
	}
	types := map[string]string{}
	for _, v := range o.Variables {
		types[ir.Key(v.Name)] = v.Type
	}
	s.States = []*ir.State{{Name: ""}}
	for _, st := range o.States {
		target := s.States[0]
		if st.Name != "" {
			target = &ir.State{Name: st.Name, Auto: strings.EqualFold(st.Name, o.AutoState)}
			s.States = append(s.States, target)
		}
		for _, fn := range st.Functions {
			target.Functions = append(target.Functions, lowerFunction(fn, names, types))
		}
	}
	return s
}

func lowerFunction(fn Function, names, objTypes map[string]string) *ir.Function {
	out := &ir.Function{Name: fn.Name}
	for _, p := range fn.Params {
		out.Params = append(out.Params, ir.Param{Name: p.Name, Type: p.Type})
	}
	if len(fn.Instructions) > 0 {
		out.Line = fn.Instructions[0].Line
	}
	if fn.Native {
		return out
	}
	types := map[string]string{}
	for k, t := range objTypes {
		types[k] = t
	}
	for _, v := range append(append([]Variable{}, fn.Params...), fn.Locals...) {
		types[ir.Key(v.Name)] = v.Type
	}
	l := &lowerer{code: fn.Instructions, names: names, types: types}
	// Locals start as None in a new frame, as declared locals do.
	for _, v := range fn.Locals {
		out.Body = append(out.Body, &ir.Assign{Target: &ir.Ident{Name: v.Name}, Value: &ir.Const{Kind: ir.ConstNone}, Declare: true, Line: out.Line})
	}
	out.Body = append(out.Body, l.block(0, len(fn.Instructions), 0)...)
	return out
}

type lowerer struct {
	code  []Instruction
	names map[string]string // auto properties' variables, to the properties' names
	types map[string]string // locals' and parameters' types, for casts
}

// target is where a jump at i goes.
func (l *lowerer) target(i int) int {
	ins := l.code[i]
	arg := ins.Args[len(ins.Args)-1]
	return i + int(arg.Number)
}

// block lowers instructions [lo, hi) into statements, rebuilding While
// from a backward jump and If from a forward conditional jump.
func (l *lowerer) block(lo, hi, depth int) []ir.Stmt {
	var out []ir.Stmt
	// Each elseif nests one deeper, and real scripts have chains of
	// hundreds; the bound only stops a malformed file's runaway.
	if depth > 4096 {
		return []ir.Stmt{&ir.ExprStmt{X: &ir.Dynamic{}}}
	}
	for i := lo; i < hi; {
		// A loop: some later jmp in this block comes back here.
		if end := l.loopEnd(i, hi); end >= 0 {
			if w, next, ok := l.loop(i, end, depth); ok {
				out = append(out, w)
				i = next
				continue
			}
		}
		ins := l.code[i]
		switch ins.Op {
		case OpJmpF, OpJmpT:
			t := l.target(i)
			if t <= i || t > hi {
				i++ // a jump out of this block (a loop's exit): nothing to rebuild
				continue
			}
			if s, ok := l.shortCircuit(i, t); ok {
				out = append(out, s...)
				i = t
				continue
			}
			cond := operand(ins.Args[0], l.names)
			if ins.Op == OpJmpT {
				cond = &ir.Unary{Op: "!", X: cond}
			}
			s := &ir.If{Line: ins.Line}
			thenEnd, next := t, t
			var elseBody []ir.Stmt
			// The then-block ends with a jmp past an else-block. In an elseif
			// chain the jmp goes to the end of the whole chain, past this
			// block, so the else is the rest of the block.
			if t-1 > i && l.code[t-1].Op == OpJmp {
				if e := l.target(t - 1); e > t {
					thenEnd, next = t-1, min(e, hi)
					elseBody = l.block(t, next, depth+1)
				}
			}
			s.Branches = []ir.Branch{{Cond: cond, Body: l.block(i+1, thenEnd, depth+1)}}
			if next > t {
				s.Else = elseBody
				if s.Else == nil {
					s.Else = []ir.Stmt{}
				}
			}
			out = append(out, s)
			i = next
		case OpJmp:
			i++ // the end of a then-block or loop, handled by its owner
		default:
			if s := l.stmt(ins); s != nil {
				out = append(out, s)
			}
			i++
		}
	}
	return out
}

// loopEnd is the index of a jmp in (i, hi) that jumps back to i, or -1.
func (l *lowerer) loopEnd(i, hi int) int {
	for j := hi - 1; j > i; j-- {
		if l.code[j].Op == OpJmp && l.target(j) == i {
			return j
		}
	}
	return -1
}

// loop rebuilds a While whose condition starts at top and whose closing
// jmp is at end: the condition's instructions, a jmpf past end, the body.
func (l *lowerer) loop(top, end, depth int) (ir.Stmt, int, bool) {
	for k := top; k < end; k++ {
		ins := l.code[k]
		if ins.Op == OpJmpF && l.target(k) == end+1 {
			// The condition's own instructions run each time round, after
			// the body, so the next test reads their new result.
			pre := l.block(top, k, depth+1)
			body := append(l.block(k+1, end, depth+1), pre...)
			w := &ir.While{Cond: operand(ins.Args[0], l.names), Body: body, Line: ins.Line}
			// Run the condition's instructions once before the loop too, so
			// its first test sees them.
			return &ir.If{Branches: []ir.Branch{{Cond: &ir.Const{Kind: ir.ConstBool, Number: 1}, Body: append(append([]ir.Stmt{}, pre...), w)}}, Line: ins.Line}, end + 1, true
		}
		if ins.Op == OpJmp || ins.Op == OpJmpT || (ins.Op == OpJmpF && l.target(k) != end+1) {
			break
		}
	}
	return nil, 0, false
}

// shortCircuit rebuilds a && b or a || b: the compiler writes a into a
// temporary, jumps over b's instructions when a decides the answer, and
// otherwise computes b into the same temporary.
func (l *lowerer) shortCircuit(i, t int) ([]ir.Stmt, bool) {
	cond := l.code[i].Args[0]
	if cond.Kind != OperandIdent || !strings.HasPrefix(strings.ToLower(cond.String), "::temp") || t-1 <= i {
		return nil, false
	}
	for k := i + 1; k < t; k++ {
		switch l.code[k].Op {
		case OpJmp, OpJmpT, OpJmpF, OpReturn:
			return nil, false
		}
	}
	last := l.code[t-1]
	dest, ok := destination(last)
	if !ok || !strings.EqualFold(dest.String, cond.String) {
		return nil, false
	}
	a, isAssign := l.stmt(last).(*ir.Assign)
	if !isAssign {
		return nil, false
	}
	// b's instructions may reuse a's temporary, so a's value is kept
	// apart first.
	left := fmt.Sprintf("::left%d", i)
	out := []ir.Stmt{&ir.Assign{Target: &ir.Ident{Name: left}, Value: operand(cond, l.names), Declare: true, Line: l.code[i].Line}}
	for k := i + 1; k < t-1; k++ {
		if s := l.stmt(l.code[k]); s != nil {
			out = append(out, s)
		}
	}
	op := "&&"
	if l.code[i].Op == OpJmpT {
		op = "||"
	}
	out = append(out, &ir.Assign{Target: a.Target, Value: &ir.Binary{Op: op, L: &ir.Ident{Name: left}, R: a.Value}, Line: last.Line})
	return out, true
}

// destination is the variable an instruction writes, if it writes one.
func destination(ins Instruction) (Operand, bool) {
	switch ins.Op {
	case OpIAdd, OpFAdd, OpISub, OpFSub, OpIMul, OpFMul, OpIDiv, OpFDiv, OpIMod, OpNot, OpINeg, OpFNeg, OpAssign, OpCast,
		OpCmpEq, OpCmpLt, OpCmpLe, OpCmpGt, OpCmpGe, OpStrCat, OpArrayCreate, OpArrayLength, OpArrayGetElement,
		OpArrayFindElement, OpArrayRFindElement:
		return ins.Args[0], true
	case OpCallMethod, OpCallStatic, OpPropGet:
		return ins.Args[2], true
	case OpCallParent:
		return ins.Args[1], true
	}
	return Operand{}, false
}

var binaryOps = map[Opcode]string{
	OpIAdd: "+", OpFAdd: "+", OpISub: "-", OpFSub: "-", OpIMul: "*", OpFMul: "*", OpIDiv: "/", OpFDiv: "/", OpIMod: "%",
	OpCmpEq: "==", OpCmpLt: "<", OpCmpLe: "<=", OpCmpGt: ">", OpCmpGe: ">=", OpStrCat: "+",
}

// stmt lowers one instruction that isn't a jump.
func (l *lowerer) stmt(ins Instruction) ir.Stmt {
	a := ins.Args
	op := func(i int) ir.Expr { return operand(a[i], l.names) }
	set := func(dest Operand, v ir.Expr) ir.Stmt {
		if isNone(dest) {
			return &ir.ExprStmt{X: v, Line: ins.Line}
		}
		return &ir.Assign{Target: operand(dest, l.names), Value: v, Line: ins.Line}
	}
	if sym, ok := binaryOps[ins.Op]; ok {
		return set(a[0], &ir.Binary{Op: sym, L: op(1), R: op(2)})
	}
	switch ins.Op {
	case OpNot:
		return set(a[0], &ir.Unary{Op: "!", X: op(1)})
	case OpINeg, OpFNeg:
		return set(a[0], &ir.Unary{Op: "-", X: op(1)})
	case OpAssign:
		return set(a[0], op(1))
	case OpCast:
		return set(a[0], &ir.Cast{X: op(1), Type: l.types[ir.Key(a[0].String)]})
	case OpCallMethod:
		c := &ir.Call{Method: a[0].String, Args: l.args(a[3:]), Line: ins.Line}
		if !(a[1].Kind == OperandIdent && ir.Key(a[1].String) == "self") {
			c.Object = op(1)
		}
		return set(a[2], c)
	case OpCallParent:
		return set(a[1], &ir.Call{Parent: true, Method: a[0].String, Args: l.args(a[2:]), Line: ins.Line})
	case OpCallStatic:
		return set(a[2], &ir.Call{Object: &ir.Ident{Name: a[0].String}, Method: a[1].String, Args: l.args(a[3:]), Line: ins.Line})
	case OpReturn:
		if a[0].Kind == OperandNone {
			return &ir.Return{Line: ins.Line}
		}
		return &ir.Return{Value: op(0), Line: ins.Line}
	case OpPropGet:
		return set(a[2], l.prop(a[0].String, a[1]))
	case OpPropSet:
		return &ir.Assign{Target: l.prop(a[0].String, a[1]), Value: op(2), Line: ins.Line}
	case OpArrayCreate:
		return set(a[0], &ir.NewArray{Type: strings.TrimSuffix(l.types[ir.Key(a[0].String)], "[]"), Size: op(1)})
	case OpArrayLength:
		return set(a[0], &ir.Member{X: op(1), Name: "Length"})
	case OpArrayGetElement:
		return set(a[0], &ir.Index{X: op(1), I: op(2)})
	case OpArraySetElement:
		return &ir.Assign{Target: &ir.Index{X: op(0), I: op(1)}, Value: op(2), Line: ins.Line}
	case OpArrayFindElement, OpArrayRFindElement:
		return set(a[0], &ir.Dynamic{})
	}
	return nil
}

func (l *lowerer) args(ops []Operand) []ir.Expr {
	out := make([]ir.Expr, len(ops))
	for i, o := range ops {
		out[i] = operand(o, l.names)
	}
	return out
}

// prop is a property of an object: the script's own (an inherited one,
// such as SKI_ConfigBase's ModName and Pages) as a name, another's as a
// member.
func (l *lowerer) prop(name string, obj Operand) ir.Expr {
	if obj.Kind == OperandIdent && ir.Key(obj.String) == "self" {
		return &ir.Ident{Name: name}
	}
	return &ir.Member{X: operand(obj, l.names), Name: name}
}

func isNone(o Operand) bool {
	return o.Kind == OperandNone || (o.Kind == OperandIdent && ir.Key(o.String) == "::nonevar")
}

func operand(o Operand, names map[string]string) ir.Expr {
	switch o.Kind {
	case OperandIdent:
		if p, ok := names[ir.Key(o.String)]; ok {
			return &ir.Ident{Name: p}
		}
		// An inherited auto property's variable (SKI_ConfigBase's ::Pages_var)
		// is read and written directly too.
		if k := o.String; strings.HasPrefix(k, "::") && strings.HasSuffix(strings.ToLower(k), "_var") && len(k) > 6 {
			return &ir.Ident{Name: k[2 : len(k)-4]}
		}
		return &ir.Ident{Name: o.String}
	case OperandString:
		return &ir.Const{Kind: ir.ConstString, String: o.String}
	case OperandInt:
		return &ir.Const{Kind: ir.ConstInt, Number: o.Number}
	case OperandFloat:
		return &ir.Const{Kind: ir.ConstFloat, Number: o.Number}
	case OperandBool:
		return &ir.Const{Kind: ir.ConstBool, Number: o.Number}
	}
	return &ir.Const{Kind: ir.ConstNone}
}
