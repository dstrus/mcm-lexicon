// Package pex reads compiled Papyrus scripts (.pex) and lowers them into
// the shared intermediate form, so MCMs that ship without their source are
// read by the same analysis. The file layout follows UESP's "Compiled
// Script File Format" page; Skyrim's files are big-endian.
//
// Lowering rebuilds just enough structure: If and While from conditional
// jumps, and && and || from the short jumps the compiler emits for them.
// Every length is checked against what's there, so a bad file is an error,
// never a crash.
package pex

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
)

// File is a compiled script as stored.
type File struct {
	Source  string   // the source file it was compiled from
	Strings []string // the string table: every name and string literal it uses
	Objects []Object
	lines   map[string][]int // instruction line numbers, by object, state, and function
}

type Object struct {
	Name       string
	Parent     string
	AutoState  string
	Variables  []Variable
	Properties []Property
	States     []State
}

type Variable struct {
	Name  string
	Type  string
	Value Operand // its initial value
}

type Property struct {
	Name    string
	Type    string
	AutoVar string    // the variable an auto property keeps its value in
	Get     *Function // a property's read function, when it has one
}

type State struct {
	Name      string
	Functions []Function
}

type Function struct {
	Name         string
	ReturnType   string
	Native       bool
	Params       []Variable
	Locals       []Variable
	Instructions []Instruction
}

// Operand is an instruction's argument or a variable's initial value.
type Operand struct {
	Kind   OperandKind
	String string // an identifier's name, or a string's value
	Number float64
}

type OperandKind uint8

const (
	OperandNone   OperandKind = 0
	OperandIdent  OperandKind = 1
	OperandString OperandKind = 2
	OperandInt    OperandKind = 3
	OperandFloat  OperandKind = 4
	OperandBool   OperandKind = 5
)

type Instruction struct {
	Op   Opcode
	Args []Operand // the fixed arguments, then a call's own
	Line int       // from debug info; 0 when there is none
}

// Opcode is a Skyrim Papyrus instruction.
type Opcode uint8

const (
	OpNop Opcode = iota
	OpIAdd
	OpFAdd
	OpISub
	OpFSub
	OpIMul
	OpFMul
	OpIDiv
	OpFDiv
	OpIMod
	OpNot
	OpINeg
	OpFNeg
	OpAssign
	OpCast
	OpCmpEq
	OpCmpLt
	OpCmpLe
	OpCmpGt
	OpCmpGe
	OpJmp
	OpJmpT
	OpJmpF
	OpCallMethod
	OpCallParent
	OpCallStatic
	OpReturn
	OpStrCat
	OpPropGet
	OpPropSet
	OpArrayCreate
	OpArrayLength
	OpArrayGetElement
	OpArraySetElement
	OpArrayFindElement
	OpArrayRFindElement
	opCount
)

// fixed is each opcode's count of fixed arguments; calls then carry a count
// and their own arguments.
var fixed = [opCount]int{0, 3, 3, 3, 3, 3, 3, 3, 3, 3, 2, 2, 2, 2, 2, 3, 3, 3, 3, 3, 1, 2, 2, 3, 2, 3, 1, 3, 3, 3, 2, 2, 3, 3, 4, 4}

func (op Opcode) varargs() bool {
	return op == OpCallMethod || op == OpCallParent || op == OpCallStatic
}

// ReadFile reads the compiled script at path.
func ReadFile(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Read(b)
}

// Read reads a compiled script.
func Read(b []byte) (f *File, err error) {
	r := &reader{b: b}
	if len(b) < 4 {
		return nil, errors.New("too short to be a compiled script")
	}
	switch binary.BigEndian.Uint32(b) {
	case 0xFA57C0DE:
		r.order = binary.BigEndian
	case 0xDEC057FA:
		r.order = binary.LittleEndian // Fallout 4's; read the same way
	default:
		return nil, errors.New("not a compiled Papyrus script")
	}
	r.u32()
	r.u8() // major version
	r.u8() // minor version
	r.u16()
	r.u64() // compiled at
	f = &File{lines: map[string][]int{}}
	f.Source = r.wstring()
	r.wstring() // user
	r.wstring() // machine

	r.strings = make([]string, r.u16())
	for i := range r.strings {
		r.strings[i] = r.wstring()
	}
	f.Strings = r.strings
	if r.u8() != 0 { // debug info
		r.u64()
		n := int(r.u16())
		for i := 0; i < n && r.err == nil; i++ {
			obj, state, fn := r.str(), r.str(), r.str()
			r.u8() // function type
			lines := make([]int, r.count(2))
			for j := range lines {
				lines[j] = int(r.u16())
			}
			f.lines[lineKey(obj, state, fn)] = lines
		}
	}
	for n := int(r.u16()); n > 0 && r.err == nil; n-- { // user flags
		r.u16()
		r.u8()
	}
	for n := int(r.u16()); n > 0 && r.err == nil; n-- {
		f.Objects = append(f.Objects, r.object())
	}
	if r.err != nil {
		return nil, r.err
	}
	for oi := range f.Objects {
		o := &f.Objects[oi]
		for si := range o.States {
			for fi := range o.States[si].Functions {
				fn := &o.States[si].Functions[fi]
				lines := f.lines[lineKey(o.Name, o.States[si].Name, fn.Name)]
				for ii := range fn.Instructions {
					if ii < len(lines) {
						fn.Instructions[ii].Line = lines[ii]
					}
				}
			}
		}
	}
	return f, nil
}

func lineKey(obj, state, fn string) string {
	return strings.ToLower(obj + "\x00" + state + "\x00" + fn)
}

func (r *reader) object() Object {
	o := Object{Name: r.str()}
	r.u32() // size
	o.Parent = r.str()
	r.str() // documentation
	r.u32() // user flags
	o.AutoState = r.str()
	for n := int(r.u16()); n > 0 && r.err == nil; n-- {
		v := Variable{Name: r.str(), Type: r.str()}
		r.u32()
		v.Value = r.operand()
		o.Variables = append(o.Variables, v)
	}
	for n := int(r.u16()); n > 0 && r.err == nil; n-- {
		p := Property{Name: r.str(), Type: r.str()}
		r.str() // documentation
		r.u32() // user flags
		flags := r.u8()
		if flags&4 != 0 {
			p.AutoVar = r.str()
		}
		if flags&5 == 1 {
			get := r.function("")
			p.Get = &get
		}
		if flags&6 == 2 {
			r.function("")
		}
		o.Properties = append(o.Properties, p)
	}
	for n := int(r.u16()); n > 0 && r.err == nil; n-- {
		st := State{Name: r.str()}
		for m := int(r.u16()); m > 0 && r.err == nil; m-- {
			st.Functions = append(st.Functions, r.function(r.str()))
		}
		o.States = append(o.States, st)
	}
	return o
}

func (r *reader) function(name string) Function {
	f := Function{Name: name, ReturnType: r.str()}
	r.str() // documentation
	r.u32() // user flags
	flags := r.u8()
	f.Native = flags&2 != 0
	for n := int(r.u16()); n > 0 && r.err == nil; n-- {
		f.Params = append(f.Params, Variable{Name: r.str(), Type: r.str()})
	}
	for n := int(r.u16()); n > 0 && r.err == nil; n-- {
		f.Locals = append(f.Locals, Variable{Name: r.str(), Type: r.str()})
	}
	n := int(r.u16())
	for i := 0; i < n && r.err == nil; i++ {
		op := Opcode(r.u8())
		if op >= opCount {
			r.fail(fmt.Sprintf("unknown instruction %d in %s", op, name))
			break
		}
		ins := Instruction{Op: op}
		for a := 0; a < fixed[op]; a++ {
			ins.Args = append(ins.Args, r.operand())
		}
		if op.varargs() {
			count := r.operand()
			if count.Kind != OperandInt || count.Number < 0 || count.Number > 255 {
				r.fail("a call's argument count isn't a small integer")
				break
			}
			for a := 0; a < int(count.Number); a++ {
				ins.Args = append(ins.Args, r.operand())
			}
		}
		f.Instructions = append(f.Instructions, ins)
	}
	return f
}

// reader reads fields in the file's byte order, remembering the first
// time it runs out rather than failing each read.
type reader struct {
	b       []byte
	at      int
	order   binary.ByteOrder
	strings []string
	err     error
}

func (r *reader) fail(msg string) {
	if r.err == nil {
		r.err = errors.New(msg)
	}
}

func (r *reader) take(n int) []byte {
	if r.err != nil || n < 0 || r.at+n > len(r.b) {
		r.fail(fmt.Sprintf("ends part-way through a field at byte %d", r.at))
		return make([]byte, max(n, 0))
	}
	out := r.b[r.at : r.at+n]
	r.at += n
	return out
}

func (r *reader) u8() uint8   { return r.take(1)[0] }
func (r *reader) u16() uint16 { return r.order.Uint16(r.take(2)) }
func (r *reader) u32() uint32 { return r.order.Uint32(r.take(4)) }
func (r *reader) u64() uint64 { return r.order.Uint64(r.take(8)) }

func (r *reader) wstring() string { return string(r.take(int(r.u16()))) }

// str is a string by its index in the string table.
func (r *reader) str() string {
	i := int(r.u16())
	if r.err != nil {
		return ""
	}
	if i >= len(r.strings) {
		r.fail(fmt.Sprintf("string %d isn't in the table of %d", i, len(r.strings)))
		return ""
	}
	return r.strings[i]
}

// count is an array's length, bounded by what's left.
func (r *reader) count(each int) int {
	n := int(r.u16())
	if r.err == nil && r.at+n*each > len(r.b) {
		r.fail("an array claims more than the file holds")
		return 0
	}
	return n
}

func (r *reader) operand() Operand {
	k := OperandKind(r.u8())
	switch k {
	case OperandNone:
		return Operand{Kind: k}
	case OperandIdent, OperandString:
		return Operand{Kind: k, String: r.str()}
	case OperandInt:
		return Operand{Kind: k, Number: float64(int32(r.u32()))}
	case OperandFloat:
		return Operand{Kind: k, Number: float64(math.Float32frombits(r.u32()))}
	case OperandBool:
		return Operand{Kind: k, Number: float64(r.u8())}
	}
	r.fail(fmt.Sprintf("unknown value type %d", k))
	return Operand{}
}
