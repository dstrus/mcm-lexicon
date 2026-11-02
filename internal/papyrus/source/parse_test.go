package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lexicon/internal/papyrus/ir"
	"lexicon/internal/testcorpus"
)

const sample = `ScriptName My_MCM extends SKI_ConfigBase Conditional
{The documentation comment
spans lines}
Import Utility

GlobalVariable Property MyGlobal Auto
String Property Title = "Menu" AutoReadOnly
int Property Count
	int Function Get()
		return 3
	EndFunction
EndProperty
string[] choices
int toggle_B = -1 ; an option's ID
float scale = 0.5

;/ a block
   comment /;

Event OnConfigInit()
	Pages = new string[2]
	Pages[0] = "General"
	Pages[1] = "$MY_More"
	choices = new String[2]
	choices[0] = "Off" ; first
	choices[1] = "A \"quoted\" one"
	count += 1
EndEvent

Event OnPageReset(string page)
	If page == "General" && !Busy()
		toggle_B = AddToggleOption("Enabled", MyGlobal.GetValue() as bool)
		AddSliderOption("Scale", scale, \
			"{1}")
	ElseIf (page == Pages[1])
		Helper(page)
	Else
		parent.OnPageReset(page)
	EndIf
	int i = 0
	While i < 3
		AddTextOption("Slot " + i, "")
		i += 1
	EndWhile
EndEvent

Function Helper(string a, int b = 2)
	SetCursorFillMode(TOP_TO_BOTTOM)
EndFunction

bool Function Busy() native

Auto State Ready
	Event OnSelectST()
		Debug.Notification("hi")
	EndEvent
EndState

State OID_Thing
	Event OnHighlightST()
		SetInfoText("Help for the thing.")
	EndEvent
EndState
`

func TestParseSample(t *testing.T) {
	r := Parse(sample)
	if len(r.Errors) > 0 {
		t.Fatalf("errors: %v", r.Errors)
	}
	s := r.Script
	if s.Name != "My_MCM" || s.Parent != "SKI_ConfigBase" {
		t.Errorf("header: %q extends %q", s.Name, s.Parent)
	}
	var props []string
	for _, p := range s.Properties {
		props = append(props, p.Type+" "+p.Name)
	}
	if strings.Join(props, ", ") != "GlobalVariable MyGlobal, String Title, int Count" {
		t.Errorf("properties = %q", props)
	}
	if c, ok := s.Properties[1].Init.(*ir.Const); !ok || c.String != "Menu" {
		t.Errorf("Title's value = %#v", s.Properties[1].Init)
	}
	var vars []string
	for _, v := range s.Variables {
		vars = append(vars, v.Type+" "+v.Name)
	}
	if strings.Join(vars, ", ") != "string[] choices, int toggle_B, float scale" {
		t.Errorf("variables = %q", vars)
	}
	if c := s.Variables[1].Init.(*ir.Const); c.Number != -1 {
		t.Errorf("toggle_B = %v", c.Number)
	}

	empty := s.States[0]
	var fns []string
	for _, f := range empty.Functions {
		fns = append(fns, f.Name)
	}
	if strings.Join(fns, ", ") != "OnConfigInit, OnPageReset, Helper, Busy" {
		t.Errorf("functions = %q", fns)
	}
	init := empty.Function("onconfiginit")
	if len(init.Body) != 7 {
		t.Fatalf("OnConfigInit has %d statements", len(init.Body))
	}
	if a := init.Body[5].(*ir.Assign); a.Value.(*ir.Const).String != `A "quoted" one` {
		t.Errorf("escaped string = %q", a.Value.(*ir.Const).String)
	}
	if a := init.Body[6].(*ir.Assign); a.Value.(*ir.Binary).Op != "+" {
		t.Errorf("+= wasn't lowered: %#v", a.Value)
	}

	page := empty.Function("OnPageReset")
	if len(page.Params) != 1 || page.Params[0].Name != "page" || len(page.Body) != 3 {
		t.Fatalf("OnPageReset = %+v", page)
	}
	cond := page.Body[0].(*ir.If)
	if len(cond.Branches) != 2 || len(cond.Else) != 1 {
		t.Fatalf("if = %+v", cond)
	}
	and := cond.Branches[0].Cond.(*ir.Binary)
	if and.Op != "&&" || and.L.(*ir.Binary).Op != "==" {
		t.Errorf("condition = %#v", and)
	}
	set := cond.Branches[0].Body[0].(*ir.Assign)
	call := set.Value.(*ir.Call)
	if call.Method != "AddToggleOption" || call.Object != nil || len(call.Args) != 2 {
		t.Errorf("AddToggleOption = %#v", call)
	}
	if _, ok := call.Args[1].(*ir.Cast); !ok {
		t.Errorf("the cast = %#v", call.Args[1])
	}
	slider := cond.Branches[0].Body[1].(*ir.ExprStmt).X.(*ir.Call)
	if slider.Args[2].(*ir.Const).String != "{1}" {
		t.Errorf("a continued line's argument = %#v", slider.Args[2])
	}
	if p := cond.Else[0].(*ir.ExprStmt).X.(*ir.Call); !p.Parent || p.Method != "OnPageReset" {
		t.Errorf("parent call = %#v", p)
	}
	loop := page.Body[2].(*ir.While)
	if len(loop.Body) != 2 {
		t.Errorf("while body = %+v", loop.Body)
	}
	if h := empty.Function("Helper"); len(h.Params) != 2 || h.Params[1].Default.(*ir.Const).Number != 2 {
		t.Errorf("Helper's params = %+v", h.Params)
	}
	if b := empty.Function("Busy"); b.Body != nil {
		t.Error("a native function has a body")
	}

	ready := s.State("READY")
	if ready == nil || !ready.Auto || ready.Function("OnSelectST") == nil {
		t.Errorf("auto state = %+v", ready)
	}
	if hl := s.State("OID_Thing").Function("OnHighlightST"); hl.Body[0].(*ir.ExprStmt).X.(*ir.Call).Args[0].(*ir.Const).String != "Help for the thing." {
		t.Errorf("state event = %+v", hl)
	}
}

func TestParseRecoversAtTheNextLine(t *testing.T) {
	r := Parse(`ScriptName Broken extends SKI_ConfigBase
Event OnPageReset(string page)
	x = = 3
	AddToggleOption("Still read", true)
	If page ==
		AddToggleOption("Inside a broken if", false)
	EndIf
EndEvent
`)
	if len(r.Errors) != 2 {
		t.Errorf("errors = %v", r.Errors)
	}
	body := r.Script.States[0].Function("OnPageReset").Body
	found := 0
	var walk func([]ir.Stmt)
	walk = func(ss []ir.Stmt) {
		for _, s := range ss {
			switch s := s.(type) {
			case *ir.ExprStmt:
				if c, ok := s.X.(*ir.Call); ok && c.Method == "AddToggleOption" {
					found++
				}
			case *ir.If:
				for _, b := range s.Branches {
					walk(b.Body)
				}
			}
		}
	}
	walk(body)
	if found != 2 {
		t.Errorf("found %d of the 2 options after the bad lines", found)
	}
}

func TestCaseDoesntMatter(t *testing.T) {
	r := Parse("scriptname a EXTENDS b\nEVENT onInit()\n\tIF true\n\tENDIF\nENDEVENT\n")
	if r.Script.Parent != "b" || len(r.Errors) > 0 || r.Script.States[0].Function("OnInit") == nil {
		t.Errorf("%+v %v", r.Script, r.Errors)
	}
}

// Every script in the real lists parses with few errors.
func TestRealSnapshots(t *testing.T) {
	for _, snap := range testcorpus.Snapshots(t) {
		t.Run(snap.Name, func(t *testing.T) {
			files, lines, errs := 0, 0, 0
			filepath.WalkDir(snap.Dir, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".psc") {
					return nil
				}
				b, _ := os.ReadFile(path)
				r := Parse(string(b))
				files++
				lines += strings.Count(string(b), "\n")
				errs += len(r.Errors)
				for _, e := range r.Errors {
					t.Logf("%s: %v", filepath.Base(path), e)
				}
				if r.Script.Name == "" {
					t.Errorf("%s: no ScriptName read", path)
				}
				return nil
			})
			t.Logf("%d scripts, %d lines, %d unreadable lines", files, lines, errs)
			if errs*1000 > lines {
				t.Errorf("more than one line in a thousand couldn't be read")
			}
		})
	}
}
