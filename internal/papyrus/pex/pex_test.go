package pex

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"lexicon/internal/papyrus/ir"
	"lexicon/internal/papyrus/mcm"
	"lexicon/internal/papyrus/source"
	"lexicon/internal/testcorpus"
)

// The encoder: just enough of the format to write the instruction subset
// MCMs use, as Skyrim's compiler would.

type enc struct {
	strs  []string
	index map[string]uint16
}

func (e *enc) s(v string) uint16 {
	if e.index == nil {
		e.index = map[string]uint16{}
	}
	if i, ok := e.index[v]; ok {
		return i
	}
	e.index[v] = uint16(len(e.strs))
	e.strs = append(e.strs, v)
	return e.index[v]
}

type w struct{ bytes.Buffer }

func (b *w) u8(v uint8)   { b.WriteByte(v) }
func (b *w) u16(v uint16) { binary.Write(b, binary.BigEndian, v) }
func (b *w) u32(v uint32) { binary.Write(b, binary.BigEndian, v) }
func (b *w) ws(v string)  { b.u16(uint16(len(v))); b.WriteString(v) }

// Operands, as instructions take them.
type op struct {
	kind OperandKind
	str  string
	n    float64
}

func id(name string) op { return op{kind: OperandIdent, str: name} }
func lit(v string) op   { return op{kind: OperandString, str: v} }
func num(n int) op      { return op{kind: OperandInt, n: float64(n)} }
func flt(n float64) op  { return op{kind: OperandFloat, n: n} }
func (e *enc) operand(b *w, o op) {
	b.u8(uint8(o.kind))
	switch o.kind {
	case OperandIdent, OperandString:
		b.u16(e.s(o.str))
	case OperandInt:
		b.u32(uint32(int32(o.n)))
	case OperandFloat:
		b.u32(mathFloat(o.n))
	case OperandBool:
		b.u8(uint8(o.n))
	}
}

type ins struct {
	op   Opcode
	args []op
	call []op // a call's own arguments
}

type fn struct {
	name   string
	params []string // name:type
	locals []string
	code   []ins
}

func (e *enc) function(b *w, f fn) {
	b.u16(e.s("None"))
	b.u16(e.s(""))
	b.u32(0)
	b.u8(0)
	pair := func(list []string) {
		b.u16(uint16(len(list)))
		for _, p := range list {
			name, typ, _ := strings.Cut(p, ":")
			b.u16(e.s(name))
			b.u16(e.s(typ))
		}
	}
	pair(f.params)
	pair(f.locals)
	b.u16(uint16(len(f.code)))
	for _, in := range f.code {
		b.u8(uint8(in.op))
		for _, a := range in.args {
			e.operand(b, a)
		}
		if in.op.varargs() {
			e.operand(b, num(len(in.call)))
			for _, a := range in.call {
				e.operand(b, a)
			}
		}
	}
}

// build writes a script named name extending parent, with the empty
// state's functions, an auto property, and a variable.
func build(name, parent string, fns ...fn) []byte {
	e := &enc{}
	var obj w
	obj.u16(e.s(parent))
	obj.u16(e.s(""))
	obj.u32(0)
	obj.u16(e.s(""))
	obj.u16(2) // variables
	obj.u16(e.s("::Title_var"))
	obj.u16(e.s("String"))
	obj.u32(0)
	e.operand(&obj, lit("From the property"))
	obj.u16(e.s("toggle_B"))
	obj.u16(e.s("Int"))
	obj.u32(0)
	e.operand(&obj, num(0))
	obj.u16(1) // properties
	obj.u16(e.s("Title"))
	obj.u16(e.s("String"))
	obj.u16(e.s(""))
	obj.u32(0)
	obj.u8(7) // read, write, auto
	obj.u16(e.s("::Title_var"))
	obj.u16(1) // states
	obj.u16(e.s(""))
	obj.u16(uint16(len(fns)))
	for _, f := range fns {
		obj.u16(e.s(f.name))
		e.function(&obj, f)
	}
	nameIndex := e.s(name)

	var out w
	out.u32(0xFA57C0DE)
	out.u8(3)
	out.u8(2)
	out.u16(1)
	out.Write(make([]byte, 8))
	out.ws(name + ".psc")
	out.ws("user")
	out.ws("machine")
	out.u16(uint16(len(e.strs)))
	for _, s := range e.strs {
		out.ws(s)
	}
	out.u8(0)  // no debug info
	out.u16(0) // user flags
	out.u16(1) // objects
	out.u16(nameIndex)
	out.u32(uint32(obj.Len() + 4))
	out.Write(obj.Bytes())
	return out.Bytes()
}

// The menu, compiled: an if/elseif/else on the page, a && condition, a
// while loop with a counter, ST options, and a dialog linked by option ID.
func menu() []byte {
	none := id("::NoneVar")
	return build("Built_MCM", "SKI_ConfigBase",
		fn{name: "OnConfigInit", locals: []string{"::temp0:String[]"}, code: []ins{
			{op: OpArrayCreate, args: []op{id("::temp0"), num(2)}},
			{op: OpPropSet, args: []op{id("Pages"), id("self"), id("::temp0")}},
			{op: OpArraySetElement, args: []op{id("::temp0"), num(0), lit("General")}},
			{op: OpArraySetElement, args: []op{id("::temp0"), num(1), lit("Extra")}},
			{op: OpPropSet, args: []op{id("ModName"), id("self"), id("::Title_var")}},
		}},
		fn{name: "OnPageReset", params: []string{"page:String"}, locals: []string{"::temp1:Bool", "::temp2:Int", "i:Int", "::temp3:String"}, code: []ins{
			/* 0 */ {op: OpCmpEq, args: []op{id("::temp1"), id("page"), lit("General")}},
			/* 1 */ {op: OpJmpF, args: []op{id("::temp1"), num(2)}}, // && : skip to 3
			/* 2 */ {op: OpCallMethod, args: []op{id("IsReady"), id("self"), id("::temp1")}},
			/* 3 */ {op: OpJmpF, args: []op{id("::temp1"), num(5)}}, // to 8
			/* 4 */ {op: OpCallMethod, args: []op{id("AddToggleOption"), id("self"), id("::temp2")}, call: []op{lit("Shiny"), num(1)}},
			/* 5 */ {op: OpAssign, args: []op{id("toggle_B"), id("::temp2")}},
			/* 6 */ {op: OpCallMethod, args: []op{id("AddSliderOption"), id("self"), none}, call: []op{lit("Level"), num(3), lit("{0}")}},
			/* 7 */ {op: OpJmp, args: []op{num(10)}}, // to 17
			/* 8 */ {op: OpCmpEq, args: []op{id("::temp1"), id("page"), lit("Extra")}},
			/* 9 */ {op: OpJmpF, args: []op{id("::temp1"), num(8)}}, // to 17
			/* 10 */ {op: OpAssign, args: []op{id("i"), num(0)}},
			/* 11 */ {op: OpCmpLt, args: []op{id("::temp1"), id("i"), num(2)}},
			/* 12 */ {op: OpJmpF, args: []op{id("::temp1"), num(5)}}, // to 17
			/* 13 */ {op: OpStrCat, args: []op{id("::temp3"), lit("Slot "), id("i")}},
			/* 14 */ {op: OpCallMethod, args: []op{id("AddToggleOptionST"), id("self"), none}, call: []op{lit("SlotState"), id("::temp3"), num(0)}},
			/* 15 */ {op: OpIAdd, args: []op{id("i"), id("i"), num(1)}},
			/* 16 */ {op: OpJmp, args: []op{num(-5)}}, // to 11
		}},
		fn{name: "OnOptionSliderOpen", params: []string{"option:Int"}, locals: []string{"::temp4:Bool"}, code: []ins{
			{op: OpCmpEq, args: []op{id("::temp4"), id("option"), id("toggle_B")}},
			{op: OpJmpF, args: []op{id("::temp4"), num(2)}},
			{op: OpCallMethod, args: []op{id("SetInfoText"), id("self"), none}, call: []op{lit("The shiny one.")}},
		}},
		fn{name: "OnOptionHighlight", params: []string{"option:Int"}, locals: []string{"::temp4:Bool"}, code: []ins{
			{op: OpCmpEq, args: []op{id("::temp4"), id("option"), id("toggle_B")}},
			{op: OpJmpF, args: []op{id("::temp4"), num(3)}},
			{op: OpCallMethod, args: []op{id("SetInfoText"), id("self"), none}, call: []op{lit("Makes it shiny.")}},
			{op: OpJmp, args: []op{num(2)}},
			{op: OpCallMethod, args: []op{id("SetSliderDialogRange"), id("self"), none}, call: []op{flt(0), flt(10)}},
		}},
		fn{name: "IsReady", code: []ins{{op: OpReturn, args: []op{{kind: OperandBool, n: 1}}}}},
	)
}

func TestReadAndLower(t *testing.T) {
	f, err := Read(menu())
	if err != nil {
		t.Fatal(err)
	}
	if f.Source != "Built_MCM.psc" || len(f.Objects) != 1 || f.Objects[0].Parent != "SKI_ConfigBase" {
		t.Fatalf("file = %+v", f)
	}
	s := f.Script()
	if s.Name != "Built_MCM" || s.Parent != "SKI_ConfigBase" || len(s.Properties) != 1 || len(s.Variables) != 1 || s.Variables[0].Name != "toggle_B" {
		t.Errorf("script = %+v", s)
	}
	m := mcm.Extract([]*ir.Script{s}, nil)
	if m.ModName != "From the property" || len(m.Pages) != 2 {
		t.Fatalf("menu = %+v", m)
	}
	gen := m.Pages[0]
	if gen.Name != "General" || len(gen.Options) != 2 || gen.Options[0].Label != "Shiny" || gen.Options[1].Format != "{0}" {
		t.Errorf("General = %+v", gen)
	}
	if gen.Options[0].Help != "Makes it shiny." {
		t.Errorf("the dialog wasn't linked: %+v", gen.Options[0])
	}
	extra := m.Pages[1]
	var labels []string
	for _, o := range extra.Options {
		labels = append(labels, o.Label+"/"+o.State)
	}
	if strings.Join(labels, ",") != "Slot 0/SlotState,Slot 1/SlotState" {
		t.Errorf("the loop's options = %q", labels)
	}
}

// Find and RFind put the array first and their result second, unlike the
// other array instructions: the result is what's unknown afterward, not
// the array, and not the value looked for, which the compiler often keeps
// in the same temporary.
func TestFindWritesItsResult(t *testing.T) {
	f, err := Read(build("Find_MCM", "SKI_ConfigBase",
		fn{name: "OnPageReset", params: []string{"page:String"}, locals: []string{"::temp0:Int", "list:Int[]"}, code: []ins{
			{op: OpArrayCreate, args: []op{id("list"), num(3)}},
			{op: OpAssign, args: []op{id("::temp0"), num(7)}},
			{op: OpArrayFindElement, args: []op{id("list"), id("::temp0"), id("::temp0"), num(0)}},
			{op: OpCallMethod, args: []op{id("AddSliderOption"), id("self"), id("::NoneVar")}, call: []op{lit("Found"), id("::temp0"), lit("{0}")}},
			{op: OpArrayLength, args: []op{id("::temp0"), id("list")}},
			{op: OpCallMethod, args: []op{id("AddSliderOption"), id("self"), id("::NoneVar")}, call: []op{lit("Length"), id("::temp0"), lit("{0}")}},
		}},
	))
	if err != nil {
		t.Fatal(err)
	}
	m := mcm.Extract([]*ir.Script{f.Script()}, nil)
	opts := m.Pages[0].Options
	if len(opts) != 2 {
		t.Fatalf("options = %+v", opts)
	}
	if opts[0].StartNumber != nil {
		t.Errorf("Find's result is taken as the value it looked for: %v", *opts[0].StartNumber)
	}
	if opts[1].StartNumber == nil || *opts[1].StartNumber != 3 {
		t.Errorf("the array was lost to Find's result: its length is %v", opts[1].StartNumber)
	}
}

func TestMalformedFilesAreErrorsNotCrashes(t *testing.T) {
	good := menu()
	for n := 0; n < len(good); n += 3 {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("cut at %d: panic %v", n, r)
				}
			}()
			if f, err := Read(good[:n]); err == nil && n < len(good)-1 {
				f.Script()
			}
		}()
	}
	if _, err := Read([]byte("not a script")); err == nil {
		t.Error("a file without the magic number was read")
	}
}

func mathFloat(f float64) uint32 {
	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, float32(f))
	return binary.BigEndian.Uint32(b.Bytes())
}

// Every compiled script in the real lists reads; and where a mod ships the
// source too, the two front ends find the same options. This is the
// strongest check there is on both, and it grows with every list.
func TestRealSnapshots(t *testing.T) {
	for _, snap := range testcorpus.Snapshots(t) {
		t.Run(snap.Name, func(t *testing.T) {
			sources := map[string]string{}
			var compiled []string
			filepath.WalkDir(snap.Dir, func(p string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				base := strings.ToLower(strings.TrimSuffix(filepath.Base(p), filepath.Ext(p)))
				switch strings.ToLower(filepath.Ext(p)) {
				case ".psc":
					sources[modOf(snap.Dir, p)+"/"+base] = p
				case ".pex":
					compiled = append(compiled, p)
				}
				return nil
			})
			read, compared, same, stale := 0, 0, 0, 0
			for _, p := range compiled {
				f, err := ReadFile(p)
				if err != nil {
					t.Errorf("%s: %v", filepath.Base(p), err)
					continue
				}
				read++
				s := f.Script()
				if !strings.EqualFold(s.Parent, "SKI_ConfigBase") {
					continue
				}
				// The source from the same mod: two mods can ship scripts of
				// one name.
				src, ok := sources[modOf(snap.Dir, p)+"/"+strings.ToLower(strings.TrimSuffix(filepath.Base(p), filepath.Ext(p)))]
				if !ok {
					continue
				}
				b, _ := os.ReadFile(src)
				if reason := f.OtherVersion(string(b)); reason != "" {
					t.Logf("%s: not compared, the source is another version (%s)", filepath.Base(p), reason)
					stale++
					continue
				}
				r := source.Parse(string(b))
				a, c := mcm.Extract([]*ir.Script{r.Script}, nil), mcm.Extract([]*ir.Script{s}, nil)
				compared++
				da, dc := describe(a), describe(c)
				switch {
				case da == dc:
					same++
				case sortedLines(da) == sortedLines(dc):
					// The same options, in another order: the arms of an If
					// laid out the other way round by the compiler.
					t.Logf("%s: the same options in another order", filepath.Base(p))
					same++
				default:
					t.Errorf("%s: source and compiled differ:\n%s", filepath.Base(p), diff(da, dc))
				}
			}
			t.Logf("%d compiled scripts read; %d menus compared with their source, %d the same; %d sources of another version", read, compared, same, stale)
		})
	}
}

// describe is a menu's pages and options, one per line, as both front ends
// should agree on them.
func describe(m mcm.Menu) string {
	var b strings.Builder
	b.WriteString("modname " + m.ModName + "\n")
	for _, p := range m.Pages {
		b.WriteString("page " + p.Name + "\n")
		for _, o := range p.Options {
			// Labels without regard to case: the compiler's string table
			// keeps one spelling of strings that differ only in case.
			b.WriteString("  " + o.Type + " " + string(o.Certainty) + " " + strings.ToLower(o.Label) + " [" + strings.ToLower(o.State) + "]")
			if o.Range != nil {
				b.WriteString(" range")
			}
			if len(o.Choices) > 0 {
				b.WriteString(" choices " + strings.Join(o.Choices, "|"))
			}
			b.WriteString(" @" + strconv.Itoa(o.Position))
			if o.Disabled != "" {
				b.WriteString(" disabled=" + o.Disabled)
			}
			if o.Inert {
				b.WriteString(" inert")
			}
			if len(o.Cycle) > 0 {
				b.WriteString(" cycle=" + strings.Join(o.Cycle, "|"))
			}
			b.WriteString("\n")
		}
		for _, f := range p.Fixtures {
			b.WriteString("  " + f.Kind + " " + strings.ToLower(f.Label) + " @" + strconv.Itoa(f.Position) + "\n")
		}
	}
	return b.String()
}

// modOf is the mod folder a file in an instance sits in.
func modOf(instance, p string) string {
	rel, _ := filepath.Rel(instance, p)
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) > 1 {
		return strings.ToLower(parts[0] + "/" + parts[1])
	}
	return ""
}

func sortedLines(s string) string {
	lines := strings.Split(s, "\n")
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func diff(a, b string) string {
	la, lb := strings.Split(a, "\n"), strings.Split(b, "\n")
	var out []string
	for i := 0; i < len(la) || i < len(lb); i++ {
		var x, y string
		if i < len(la) {
			x = la[i]
		}
		if i < len(lb) {
			y = lb[i]
		}
		if x != y {
			out = append(out, "  source:   "+x, "  compiled: "+y)
			if len(out) > 12 {
				break
			}
		}
	}
	return strings.Join(out, "\n")
}
