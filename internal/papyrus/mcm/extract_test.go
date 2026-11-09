package mcm

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"lexicon/internal/papyrus/ir"
	"lexicon/internal/papyrus/source"
)

func parse(t *testing.T, src string) *ir.Script {
	t.Helper()
	r := source.Parse(src)
	if len(r.Errors) > 0 {
		t.Fatalf("parse: %v", r.Errors)
	}
	return r.Script
}

func page(t *testing.T, m Menu, name string) Page {
	t.Helper()
	for _, p := range m.Pages {
		if strings.EqualFold(p.Name, name) {
			return p
		}
	}
	var names []string
	for _, p := range m.Pages {
		names = append(names, fmt.Sprintf("%q", p.Name))
	}
	t.Fatalf("no page %q in %s", name, strings.Join(names, ", "))
	return Page{}
}

func opt(t *testing.T, p Page, label string) Option {
	t.Helper()
	for _, o := range p.Options {
		if o.Label == label {
			return o
		}
	}
	t.Fatalf("no option %q on page %q: %+v", label, p.Name, p.Options)
	return Option{}
}

// The patterns of Dirt & Blood: option IDs in variables, menu entries
// filled element by element, mixed-case names, the same option in both
// arms of an If, and ModName only in the plugin.
const dirty = `Scriptname Dirty_Quest_MCM extends SKI_ConfigBase
GlobalVariable Property Dirty_ProfessionsDirt Auto
int dirtsetting_M
int professionsdirt_b
int WaterRequired_b
int DirtSettingIndex = 1
string[] DirtSettingList

Event OnConfigInit()
	Pages = new string[1]
	Pages[0] = "Settings"
EndEvent

event OnInit()
	parent.OnInit()
	DirtSettingList = new string[4]
	DirtSettingList[0] = "Rarely - 2 to 4 days"
	DirtSettingList[1] = "Default - 1 to 2 days"
	DirtSettingList[2] = "Often - 12 to 24 hours"
	DirtSettingList[3] = "Deactivated"
endEvent

Event OnPageReset(string page)
	If (page == "")
		LoadCustomContent("MCM/MCM_DNB.dds", 0.0, 0.0)
	Endif
	If (page == "Settings")
		AddHeaderOption("Frequency Settings")
		dirtsetting_M = AddMenuOption("Dirt on Player", dirtsettingList[dirtsettingIndex])
		professionsdirt_B = AddToggleOption("Dirt on NPCs Based on Profession", Dirty_ProfessionsDirt.GetValue() as bool)
		if ! game.isPluginInstalled("WadeInWater.esp")
			WaterRequired_B = AddToggleOption("Water Required For Bathing", False, OPTION_FLAG_DISABLED)
		else
			WaterRequired_B = AddToggleOption("Water Required For Bathing", true)
		endif
	Endif
endEvent

event OnOptionMenuOpen(int option)
	if (option == dirtsetting_M)
		SetMenuDialogOptions(dirtsettingList)
		SetMenuDialogDefaultIndex(1)
	endIf
endEvent

event OnOptionHighlight(int option)
	if (option == professionsdirt_B)
		SetInfoText("NPCs with certain occupations have dirt.")
	elseif option == WaterRequired_b
		SetInfoText("Requires Wade in Water.")
	endif
endevent
`

func TestDirtAndBlood(t *testing.T) {
	name := "Dirt & Blood"
	m := Extract([]*ir.Script{parse(t, dirty)}, []Property{{Name: "ModName", String: &name}})
	if m.ModName != "Dirt & Blood" || m.PagesGuessed {
		t.Errorf("menu = %+v", m)
	}
	if len(m.Pages) != 1 {
		t.Fatalf("pages = %+v (the splash page has no options, so it's left out)", m.Pages)
	}
	p := page(t, m, "Settings")
	if len(p.Options) != 3 {
		t.Fatalf("options = %+v (the option in both arms of an If is one option)", p.Options)
	}
	d := opt(t, p, "Dirt on Player")
	if d.Type != "menu" || d.Certainty != Exact || strings.Join(d.Choices, "|") != "Rarely - 2 to 4 days|Default - 1 to 2 days|Often - 12 to 24 hours|Deactivated" || d.Default == nil || *d.Default != 1 {
		t.Errorf("Dirt on Player = %+v", d)
	}
	if o := opt(t, p, "Dirt on NPCs Based on Profession"); o.Type != "toggle" || o.Help != "NPCs with certain occupations have dirt." {
		t.Errorf("profession toggle = %+v", o)
	}
	if o := opt(t, p, "Water Required For Bathing"); o.Help != "Requires Wade in Water." {
		t.Errorf("merged toggle = %+v", o)
	}
}

// The patterns of EVG Conditional Idles: ModName and Pages set in a
// function called from OnConfigInit, pages by constant, ST options, and
// help in each state.
const evg = `Scriptname EVGCI1p5_MCM extends SKI_ConfigBase
GlobalVariable Property EVGCI14_NPCCold Auto
Event OnConfigInit()
	Utility.Wait(1)
	RegisterForSingleUpdate(1)
EndEvent
Event OnUpdate()
	InstallMCM()
EndEvent
Function InstallMCM()
	ModName = "EVG Conditional Idles"
	Pages = New String[2]
	Pages[0] = "Toggle - Player"
	Pages[1] = "Toggle - NPCs"
EndFunction
Event OnPageReset(string page)
	SetCursorFillMode(TOP_TO_BOTTOM)
	If Page == ""
		LoadCustomContent("x.dds")
	ElseIf Page == "Toggle - Player"
		AddToggleOptionST("OID_EVGCI14_Cold", "Cold", 1)
	ElseIf Page == "Toggle - NPCs"
		AddToggleOptionST("OID_EVGCI14_NPCCold", "Cold", EVGCI14_NPCCold.GetValueInt())
		AddToggleOptionST("OID_EVGCI14_NPCInjured", "Injured", 0)
	Endif
EndEvent
State OID_EVGCI14_NPCCold
	event OnHighlightST()
		SetInfoText("Toggle COLD for NPCs.")
	EndEvent
EndState
State OID_EVGCI14_Cold
	event OnHighlightST()
		SetInfoText("Toggle COLD for the player.")
	EndEvent
EndState
`

func TestEVG(t *testing.T) {
	m := Extract([]*ir.Script{parse(t, evg)}, nil)
	if m.ModName != "EVG Conditional Idles" || len(m.Pages) != 2 {
		t.Fatalf("menu = %+v", m)
	}
	npcs := page(t, m, "Toggle - NPCs")
	cold := opt(t, npcs, "Cold")
	if cold.State != "OID_EVGCI14_NPCCold" || cold.Type != "toggle" || cold.Help != "Toggle COLD for NPCs." {
		t.Errorf("NPC Cold = %+v", cold)
	}
	if inj := opt(t, npcs, "Injured"); inj.Help != "" {
		t.Errorf("Injured has no state events, but help %q", inj.Help)
	}
	if c := opt(t, page(t, m, "Toggle - Player"), "Cold"); c.Help != "Toggle COLD for the player." {
		t.Errorf("player Cold = %+v", c)
	}
}

// The patterns of Experience: $KEY pages through Pages[i], options in
// helper functions per page, option IDs in array slots reused on each
// page, dialogs scoped by CurrentPage, and an OnVersionUpdate that calls
// OnConfigInit.
const experience = `ScriptName Experience_MCM Extends SKI_ConfigBase
int function GetVersion()
    return 5
endFunction
Int[] pagePool
string[] meterModes
Event OnConfigInit()
	Pages = new string[2]
	Pages[0] = "$EXP_General"
	Pages[1] = "$EXP_Leveling"
	pagePool = new Int[40]
	meterModes = new string[2]
	meterModes[0] = "$EXP_Hidden"
	meterModes[1] = "$EXP_Dynamic"
EndEvent
Event OnVersionUpdate(int a_version)
	OnConfigInit()
EndEvent
Event OnPageReset(String a_page)
	if (a_page == Pages[0])
		PageReset_General()
	elseIf (a_page == Pages[1])
		PageReset_Leveling()
	endIf
EndEvent
Function PageReset_General()
	pagePool[0] = AddSliderOption("$EXP_MaximumLevel", 81, "{0}")
	pagePool[4] = AddToggleOption("$EXP_ShowMessages", true)
	pagePool[6] = AddMenuOption("$EXP_MeterMode", meterModes[0])
	AddTextOption("$EXP_ScriptVersion", GetVersion(), OPTION_FLAG_DISABLED)
endFunction
Function PageReset_Leveling()
	pagePool[0] = AddSliderOption("$EXP_Quest_Main", 75, "{0}")
endFunction
function OnOptionSliderOpen(Int a_option)
	if (CurrentPage == Pages[0])
		if (a_option == pagePool[0])
			SetSliderDialogDefaultValue(255.0)
			SetSliderDialogRange(0.0, 255.0)
			SetSliderDialogInterval(5.0)
			return
		endIf
	elseIf (CurrentPage == Pages[1])
		if (a_option == pagePool[0])
			SetSliderDialogDefaultValue(75.0)
			SetSliderDialogRange(0.0, 200.0)
			SetSliderDialogInterval(5.0)
			return
		endIf
	endIf
endFunction
event OnOptionMenuOpen(int a_option)
	if (a_option == pagePool[6])
		SetMenuDialogOptions(meterModes)
	endIf
endEvent
`

func TestExperience(t *testing.T) {
	m := Extract([]*ir.Script{parse(t, experience)}, nil)
	general := page(t, m, "$EXP_General")
	max := opt(t, general, "$EXP_MaximumLevel")
	if max.Range == nil || *max.Range != (Range{Min: 0, Max: 255, Step: 5}) || *max.Default != 255 || max.Format != "{0}" {
		t.Errorf("Maximum level = %+v %+v", max, max.Range)
	}
	if o := opt(t, general, "$EXP_ShowMessages"); o.Type != "toggle" || o.Certainty != Exact {
		t.Errorf("Show messages = %+v", o)
	}
	if o := opt(t, general, "$EXP_MeterMode"); strings.Join(o.Choices, ",") != "$EXP_Hidden,$EXP_Dynamic" {
		t.Errorf("Meter mode = %+v", o)
	}
	opt(t, general, "$EXP_ScriptVersion")
	quest := opt(t, page(t, m, "$EXP_Leveling"), "$EXP_Quest_Main")
	if quest.Range == nil || quest.Range.Max != 200 || *quest.Default != 75 {
		t.Errorf("the same slot on another page took this page's dialog: %+v %+v", quest, quest.Range)
	}
}

func TestPatternsAndLoops(t *testing.T) {
	m := Extract([]*ir.Script{parse(t, `ScriptName Loops extends SKI_ConfigBase
Actor[] followers
Event OnPageReset(string page)
	int i = 0
	While i < 3
		AddToggleOption("Slot " + (i + 1), true)
		i += 1
	EndWhile
	int j = 0
	While j < followers.Length
		AddToggleOption(followers[j].GetDisplayName() + " follows", true)
		AddTextOption(followers[j].GetDisplayName(), "")
		j += 1
	EndWhile
EndEvent
`)}, nil)
	if len(m.Pages) != 1 || m.Pages[0].Name != "" {
		t.Fatalf("a menu without pages: %+v", m.Pages)
	}
	p := m.Pages[0]
	var got []string
	for _, o := range p.Options {
		got = append(got, fmt.Sprintf("%s=%s(%s)", o.Certainty, o.Label, o.Text))
	}
	want := "exact=Slot 1(Slot 1), exact=Slot 2(Slot 2), exact=Slot 3(Slot 3), pattern=* follows*(… follows), unknown=()"
	if strings.Join(got, ", ") != want {
		t.Errorf("options:\n%s\nwant:\n%s", strings.Join(got, ", "), want)
	}
}

func TestPagesGuessedFromComparisons(t *testing.T) {
	m := Extract([]*ir.Script{parse(t, `ScriptName Guess extends SKI_ConfigBase
Event OnConfigInit()
	Pages = SomeLib.BuildPages()
EndEvent
Event OnPageReset(string a_page)
	if a_page == "Alpha"
		AddToggleOption("A", true)
	elseif "Beta" == a_page
		AddToggleOption("B", true)
	endif
EndEvent
`)}, nil)
	if !m.PagesGuessed || len(m.Pages) != 2 || m.Pages[0].Name != "Alpha" || m.Pages[1].Options[0].Label != "B" {
		t.Errorf("menu = %+v", m)
	}
}

func TestParentChainAndRecursionAreBounded(t *testing.T) {
	base := parse(t, `ScriptName Framework_Base extends SKI_ConfigBase
Event OnPageReset(string page)
	AddToggleOption("From the framework", true)
	Recurse(1)
EndEvent
Function Recurse(int n)
	Recurse(n + 1)
EndFunction
`)
	child := parse(t, `ScriptName Framework_Child extends Framework_Base
Event OnConfigInit()
	ModName = "Child"
EndEvent
`)
	m := Extract([]*ir.Script{child, base}, nil)
	if m.ModName != "Child" || len(m.Pages) != 1 || m.Pages[0].Options[0].Label != "From the framework" {
		t.Errorf("menu = %+v", m)
	}
}

// What the menu's opening sets holds for every page (Accuracy), a flag an
// If and its Else set differently depends on the game, a counter bumped
// only in some arms keeps counting (STFU), and a property worked out by its
// Get function is unknown (MCM Recorder), while Pages built two ways keeps
// the longer list.
const opening = `Scriptname Opening_MCM extends SKI_ConfigBase
bool Status
int Count
bool Property IsVR
	bool Function Get()
		return Game.GetModByName("SkyrimVR.esm") != 255
	EndFunction
EndProperty
int Property Version
	int Function Get()
		return 3
	EndFunction
EndProperty

Event OnConfigOpen()
	If Framework.GetVersion() > 2
		Status = True
	Else
		Status = False
	EndIf
	If IsVR
		Pages = new string[2]
		Pages[0] = "Main"
		Pages[1] = "VR"
	Else
		Pages = new string[1]
		Pages[0] = "Main"
	EndIf
	If Framework.HasA()
		Count += 1
	EndIf
	If Framework.HasB()
		Count += 1
	EndIf
EndEvent

Event OnPageReset(string page)
	If !Status
		AddTextOption("Framework missing", "")
		Return
	EndIf
	AddToggleOption("Enabled", true)
	If Count == 2
		AddToggleOption("Both", true)
	EndIf
	If Version == 3
		AddToggleOption("Version three", true)
	EndIf
	If page == "VR"
		AddToggleOption("Gestures", true)
	EndIf
EndEvent
`

func TestOpeningAndUncertainBranches(t *testing.T) {
	m := Extract([]*ir.Script{parse(t, opening)}, nil)
	if m.PagesGuessed {
		t.Error("pages were guessed")
	}
	main := page(t, m, "Main")
	for _, label := range []string{"Framework missing", "Enabled", "Both", "Version three"} {
		opt(t, main, label)
	}
	opt(t, page(t, m, "VR"), "Gestures")
}

func TestJoin(t *testing.T) {
	list := func(s ...string) Value {
		v := newArray(len(s))
		for i, x := range s {
			(*v.arr)[i] = str(x)
		}
		return v
	}
	ptr := func(v Value) *Value { return &v }
	for name, c := range map[string]struct {
		a, b Value
		want string
	}{
		"same":         {str("a"), str("a"), `"a"`},
		"differ":       {boolean(true), boolean(false), "unknown"},
		"options":      {option(1), option(2), "options [1 2]"},
		"prefix":       {list("Main"), list("Main", "VR"), "[Main VR]"},
		"not a prefix": {list("Main"), list("Other", "VR"), "unknown"},
		"kinds":        {str("1"), integer(1), "unknown"},
	} {
		got := join(c.a, ptr(c.b))
		var s string
		switch got.k {
		case kUnknown:
			s = "unknown"
		case kString:
			s = fmt.Sprintf("%q", got.s)
		case kOption:
			s = fmt.Sprintf("options %v", got.ids)
		case kArray:
			var parts []string
			for _, v := range *got.arr {
				parts = append(parts, v.s)
			}
			s = "[" + strings.Join(parts, " ") + "]"
		}
		if s != c.want {
			t.Errorf("%s: join = %s, want %s", name, s, c.want)
		}
	}
}

// A text option's value, which a right-side click matches: fixed, changed
// by SetTextOptionValue, or built at runtime.
const textValues = `Scriptname Values_MCM extends SKI_ConfigBase
int statusOID
Event OnPageReset(string page)
	AddTextOption("", "Reset everything")
	statusOID = AddTextOption("Status", "Enabled")
	AddTextOption("Level", "Level " + Game.GetPlayer().GetLevel())
EndEvent
Event OnOptionSelect(int option)
	if option == statusOID
		SetTextOptionValue(statusOID, "Disabled")
	endif
EndEvent
`

func TestTextOptionValues(t *testing.T) {
	m := Extract([]*ir.Script{parse(t, textValues)}, nil)
	p := page(t, m, "")
	reset, status, level := opt(t, p, ""), opt(t, p, "Status"), opt(t, p, "Level")
	if reset.Value != "Reset everything" || reset.ValueCertainty != Exact || reset.ValueChanges {
		t.Errorf("reset = %q %s %v", reset.Value, reset.ValueCertainty, reset.ValueChanges)
	}
	if status.Value != "Enabled" || !status.ValueChanges {
		t.Errorf("status = %q, changes %v", status.Value, status.ValueChanges)
	}
	if level.ValueCertainty != Pattern || level.Value != "*Level *" || level.ValueText != "Level …" {
		t.Errorf("level = %q %q %s", level.Value, level.ValueText, level.ValueCertainty)
	}
}

// SkyUI's cursor: left to right by default, top to bottom on request, moved
// outright by position, with headers and gaps taking slots too.
const placed = `Scriptname Placed_MCM extends SKI_ConfigBase
Event OnPageReset(string page)
	if page == "Grid"
		AddHeaderOption("Left")
		AddHeaderOption("Right")
		AddToggleOption("A", true)
		AddToggleOption("B", true)
	elseif page == "Columns"
		SetCursorFillMode(TOP_TO_BOTTOM)
		AddToggleOption("C", true)
		AddEmptyOption()
		AddToggleOption("D", true)
		SetCursorPosition(1)
		AddToggleOption("E", true)
	elseif page == "Lost"
		SetCursorPosition(Game.GetPlayer().GetLevel())
		AddToggleOption("F", true)
	endif
EndEvent
`

func TestCursorPositions(t *testing.T) {
	m := Extract([]*ir.Script{parse(t, placed)}, nil)
	grid := page(t, m, "Grid")
	if !grid.Placed || len(grid.Fixtures) != 2 || grid.Fixtures[0].Position != 0 || grid.Fixtures[1].Position != 1 || opt(t, grid, "A").Position != 2 || opt(t, grid, "B").Position != 3 {
		t.Errorf("grid = %+v", grid)
	}
	cols := page(t, m, "Columns")
	if !cols.Placed || opt(t, cols, "C").Position != 0 || cols.Fixtures[0].Position != 2 || opt(t, cols, "D").Position != 4 || opt(t, cols, "E").Position != 1 {
		t.Errorf("columns = %+v", cols)
	}
	lost := page(t, m, "Lost")
	if lost.Placed || opt(t, lost, "F").Position != -1 {
		t.Errorf("lost = %+v", lost)
	}
}

// A call that names its arguments binds them by name, as the compiler does.
const namedArgs = `Scriptname Named_MCM extends SKI_ConfigBase
Function Fill(bool topDown = false, bool leftRight = false)
	if topDown
		SetCursorFillMode(TOP_TO_BOTTOM)
	else
		SetCursorFillMode(LEFT_TO_RIGHT)
	endif
EndFunction
Event OnPageReset(string page)
	Fill(leftRight = true)
	AddToggleOption("A", true)
	AddToggleOption("B", true)
	Fill(true)
	AddToggleOption("C", true)
	AddToggleOption("D", true)
	self.SetCursorFillMode(self.LEFT_TO_RIGHT)
	AddToggleOption("E", true)
	AddToggleOption("F", true)
EndEvent
`

func TestNamedArgumentsAndSelf(t *testing.T) {
	m := Extract([]*ir.Script{parse(t, namedArgs)}, nil)
	p := page(t, m, "")
	want := map[string]int{"A": 0, "B": 1, "C": 2, "D": 4, "E": 6, "F": 7}
	for label, pos := range want {
		if got := opt(t, p, label).Position; got != pos {
			t.Errorf("%s at %d, want %d", label, got, pos)
		}
	}
}

// An option added with OPTION_FLAG_DISABLED can't be selected; one whose
// flags the game decides, or that SetOptionFlags changes, sometimes can't.
const flagged = `Scriptname Flagged_MCM extends SKI_ConfigBase
GlobalVariable property Ready auto
int changing
Event OnPageReset(string page)
	AddTextOption("Version", "1.2", OPTION_FLAG_DISABLED)
	AddToggleOption("Plain", true)
	AddSliderOption("Height", 1.0, "{1}", OPTION_FLAG_DISABLED)
	AddToggleOption("Decided", true, Utility.RandomInt(0, 1))
	if Ready.GetValue() as bool
		AddToggleOption("Armed", true)
	else
		AddToggleOption("Armed", false, OPTION_FLAG_DISABLED)
	endif
	changing = AddToggleOption("Changing", true, OPTION_FLAG_DISABLED)
	AddMenuOption("Passed", "x", OPTION_FLAG_NONE)
EndEvent
Event OnOptionSelect(int option)
	SetOptionFlags(changing, OPTION_FLAG_NONE)
EndEvent`

func TestOptionFlags(t *testing.T) {
	m := Extract([]*ir.Script{parse(t, flagged)}, nil)
	p := m.Pages[0]
	for label, want := range map[string]string{"Version": "always", "Plain": "", "Height": "always", "Decided": "sometimes", "Armed": "sometimes", "Changing": "sometimes", "Passed": ""} {
		if got := opt(t, p, label).Disabled; got != want {
			t.Errorf("%s: disabled %q, want %q", label, got, want)
		}
	}
}

// A text option whose click handler shows it literal values in turn is a
// stepper: the values are read from the handler's If chain, whether the
// handler finds the option by its ID or some other way.
const cycling = `Scriptname Cycling_MCM extends SKI_ConfigBase
GlobalVariable property ModeGlobal auto
int modeOID
int handOID
int resetOID
string Function NameOf(int option)
	Return "hand" + option
EndFunction
Event OnPageReset(string page)
	modeOID = AddTextOption("Mode", "Low")
	handOID = AddTextOption("Permit Hands", "Both Hands")
	resetOID = AddTextOption("Reset Position", "")
	AddTextOption("Plain", "Fixed")
EndEvent
Event OnOptionSelect(int option)
	if option == modeOID
		int mode = ModeGlobal.GetValueInt() + 1
		if mode > 2
			mode = 0
		endif
		if mode == 0
			SetTextOptionValue(option, "Low")
		elseif mode == 1
			SetTextOptionValue(option, "Medium")
		else
			SetTextOptionValue(option, "High")
		endif
	else
		string name = NameOf(option)
		if name == "hand"
			int v = Utility.RandomInt(0, 2)
			if v == 0
				SetTextOptionValue(option, "Both Hands")
			elseif v == 1
				SetTextOptionValue(option, "Left Hand Only")
			else
				SetTextOptionValue(option, "Right Hand Only")
			endif
		elseif name == "reset"
			SetTextOptionValue(option, "Will Reset")
		endif
	endif
EndEvent`

func TestClickCycles(t *testing.T) {
	m := Extract([]*ir.Script{parse(t, cycling)}, nil)
	p := m.Pages[0]
	for label, want := range map[string][]string{
		"Mode":           {"Low", "Medium", "High"},
		"Permit Hands":   {"Both Hands", "Left Hand Only", "Right Hand Only"},
		"Reset Position": nil,
		"Plain":          nil,
	} {
		if got := opt(t, p, label).Cycle; !slices.Equal(got, want) {
			t.Errorf("%s: cycle %q, want %q", label, got, want)
		}
	}
}

// A helper that adds an option in one of an If's arms, called for several
// slots, adds an option per call: the arms of one call exclude each other,
// not the arms of two calls.
const helperArms = `Scriptname Helper_MCM extends SKI_ConfigBase
GlobalVariable property Setting auto
int Function AddHand(string text)
	if Setting.GetValueInt() == 0
		Return AddTextOption(text, "Both Hands")
	elseif Setting.GetValueInt() == 1
		Return AddTextOption(text, "Left Hand Only")
	else
		Return AddTextOption(text, "Right Hand Only")
	endif
EndFunction
Event OnPageReset(string page)
	SetCursorFillMode(TOP_TO_BOTTOM)
	AddHeaderOption("Left Hip")
	AddHand("Permit Hands")
	AddHeaderOption("Left Shoulder")
	AddHand("Permit Hands")
	SetCursorPosition(1)
	AddHeaderOption("Right Hip")
	AddHand("Permit Hands")
EndEvent`

func TestHelperCallsAreSeparate(t *testing.T) {
	m := Extract([]*ir.Script{parse(t, helperArms)}, nil)
	var positions []int
	for _, o := range m.Pages[0].Options {
		if o.Label == "Permit Hands" {
			positions = append(positions, o.Position)
		}
	}
	if !slices.Equal(positions, []int{2, 6, 3}) {
		t.Errorf("Permit Hands at %v, want one per call at 2, 6, 3", positions)
	}
}

// A helper that returns an option from one of an If's arms, or adds one
// after it when no arm returned, adds one option a call, in one slot.
const earlyReturn = `Scriptname Early_MCM extends SKI_ConfigBase
GlobalVariable property Setting auto
int Function AddHand(string text)
	int v = Setting.GetValueInt()
	if v == 0
		Return AddTextOption(text, "Both Hands")
	elseif v == 1
		Return AddTextOption(text, "Left Hand Only")
	endif
	Return AddTextOption(text, "Right Hand Only")
EndFunction
Event OnPageReset(string page)
	SetCursorFillMode(TOP_TO_BOTTOM)
	AddHeaderOption("Left Hip")
	AddToggleOption("Visible", true)
	AddHand("Permit Hands")
	AddToggleOption("Allow Daggers", true)
	AddHeaderOption("Left Shoulder")
	AddToggleOption("Visible", true)
	AddHand("Permit Hands")
	SetCursorPosition(1)
	AddHeaderOption("Right Hip")
	AddHand("Permit Hands")
EndEvent`

func TestEarlyReturnIsOneOption(t *testing.T) {
	m := Extract([]*ir.Script{parse(t, earlyReturn)}, nil)
	got := map[string][]int{}
	for _, o := range m.Pages[0].Options {
		got[o.Label] = append(got[o.Label], o.Position)
	}
	if !slices.Equal(got["Permit Hands"], []int{4, 12, 3}) || !slices.Equal(got["Visible"], []int{2, 10}) || !slices.Equal(got["Allow Daggers"], []int{6}) {
		t.Errorf("positions: %v", got)
	}
	for _, o := range m.Pages[0].Options {
		if o.Label == "Permit Hands" && !slices.Equal(o.Cycle, nil) && len(o.Cycle) != 3 {
			t.Errorf("cycle: %v", o.Cycle)
		}
	}
}

// A function that returns from inside an If's arms returns one of those
// values or the one it falls through to: unknown when they differ, so a
// condition on it doesn't drop the arms that depend on it.
const armReturns = `Scriptname Returns_MCM extends SKI_ConfigBase
int colorOID
int otherOID
string Function NameOf(int option)
	if option == colorOID
		Return "color"
	elseif option == otherOID
		Return "other"
	endif
	Return ""
EndFunction
bool Function Enabled()
	if Game.GetPlayer().GetLevel() > 10
		Return true
	endif
	Return false
EndFunction
string Function Always()
	Return "same"
EndFunction
Event OnPageReset(string page)
	colorOID = AddTextOption("Color", "Red")
	otherOID = AddToggleOption("Other", true)
	if !Enabled()
		AddTextOption("Note", "Off")
	endif
	if Always() == "same"
		AddToggleOption("Known", true)
	else
		AddToggleOption("Never", true)
	endif
EndEvent
Event OnOptionSelect(int option)
	string name = NameOf(option)
	if name == "color"
		int v = Utility.RandomInt(0, 2)
		if v == 0
			SetTextOptionValue(option, "Red")
		elseif v == 1
			SetTextOptionValue(option, "Green")
		else
			SetTextOptionValue(option, "Blue")
		endif
	endif
EndEvent`

func TestFunctionReturnsFromArms(t *testing.T) {
	m := Extract([]*ir.Script{parse(t, armReturns)}, nil)
	p := m.Pages[0]
	if got := opt(t, p, "Color").Cycle; !slices.Equal(got, []string{"Red", "Green", "Blue"}) {
		t.Errorf("cycle behind a name the handler works out: %q", got)
	}
	labels := map[string]bool{}
	for _, o := range p.Options {
		labels[o.Label] = true
	}
	if !labels["Note"] || !labels["Known"] || labels["Never"] {
		t.Errorf("options: %v", labels)
	}
}

// A text option a click does nothing to is inert: no handler responds to
// it, in the empty state, in the state its page enters (Simply Balanced),
// or in its own option state. A guard ("If option != X / Return") scopes
// what follows to X, for help text as for clicks.
const inertSrc = `Scriptname Inert_MCM extends SKI_ConfigBase
string CurrentFile = ""
int saveOID
int loadOID
int unusedOID
Event OnConfigInit()
	Pages = new string[2]
	Pages[0] = "Main"
	Pages[1] = "Files"
EndEvent
Event OnPageReset(string page)
	if page == "Files"
		GoToState("FilesPage")
		loadOID = AddTextOption("Load", "")
		unusedOID = AddTextOption("Unused", "")
	else
		GoToState("")
		saveOID = AddTextOption("Save", "")
		AddTextOption("Version", "1.0")
		AddTextOptionST("ResetST", "Reset", "")
		AddTextOptionST("AboutST", "About", "")
	endif
EndEvent
Event OnOptionSelect(int option)
	if option == saveOID
		if CurrentFile != ""
			Debug.Notification("Saved")
		endif
	endif
EndEvent
State ResetST
	Event OnSelectST()
		CurrentFile = ""
	EndEvent
EndState
State AboutST
	Event OnHighlightST()
		SetInfoText("About this mod")
	EndEvent
EndState
State FilesPage
	Event OnOptionSelect(int option)
		CheckLoad("select", option)
	EndEvent
	Event OnOptionHighlight(int option)
		CheckLoad("highlight", option)
	EndEvent
EndState
Function CheckLoad(string what, int option)
	if option != loadOID
		return
	endif
	if what == "select"
		Debug.Notification("Loaded")
	else
		SetInfoText("Load the chosen file")
	endif
EndFunction`

func TestInertTextOptions(t *testing.T) {
	m := Extract([]*ir.Script{parse(t, inertSrc)}, nil)
	main, files := page(t, m, "Main"), page(t, m, "Files")
	for _, c := range []struct {
		p     Page
		label string
		want  bool
	}{
		{main, "Save", false}, {main, "Version", true}, {main, "Reset", false}, {main, "About", true},
		{files, "Load", false}, {files, "Unused", true},
	} {
		if got := opt(t, c.p, c.label).Inert; got != c.want {
			t.Errorf("%s: inert %v, want %v", c.label, got, c.want)
		}
	}
	if got := opt(t, files, "Load").Help; got != "Load the chosen file" {
		t.Errorf("Load: help %q", got)
	}
	if got := opt(t, files, "Unused").Help; got != "" {
		t.Errorf("Unused: help %q, want none", got)
	}
}

// What a new game shows: a global's value from the plugin, changed by the
// script's initialization, a variable's declared value, a literal, and
// nothing where the value depends on the game or differs between arms.
func TestStartValues(t *testing.T) {
	const src = `Scriptname Start_MCM extends SKI_ConfigBase
GlobalVariable Property Start_Enabled Auto
GlobalVariable Property Start_Rate Auto
GlobalVariable Property Start_Count Auto
GlobalVariable Property Start_Unfilled Auto
Actor Property PlayerRef Auto
bool Shown = true
int Hotkey = 42

Event OnConfigInit()
	Start_Rate.SetValue(3.5)
	Start_Count.Mod(2)
EndEvent

Event OnPageReset(string page)
	AddToggleOption("Enabled", Start_Enabled.GetValue() as bool)
	AddSliderOption("Rate", Start_Rate.GetValue(), "{1}")
	AddSliderOption("Count", Start_Count.GetValueInt())
	AddSliderOption("Value", Start_Count.Value)
	AddToggleOption("Shown", Shown)
	AddKeyMapOption("Hotkey", Hotkey)
	AddMenuOption("Style", "Medium")
	AddTextOption("Version", Start_Count.GetValueInt() + "." + Start_Rate.GetValueInt())
	AddSliderOption("Unfilled", Start_Unfilled.GetValue())
	AddSliderOption("Level", PlayerRef.GetLevel())
	; Shown only while the setting is off: still an option the menu has.
	If Start_Enabled.GetValue() == 0
		AddToggleOption("Only when off", False)
	EndIf
	If PlayerRef.IsInCombat()
		AddToggleOption("Arms", True)
	Else
		AddToggleOption("Arms", False)
	EndIf
EndEvent
`
	one, rate, count := 1.0, 0.0, 5.0
	m := Extract([]*ir.Script{parse(t, src)}, []Property{
		{Name: "Start_Enabled", Global: &one},
		{Name: "start_rate", Global: &rate},
		{Name: "Start_Count", Global: &count},
	})
	p := page(t, m, "")
	for label, want := range map[string]float64{"Enabled": 1, "Rate": 3.5, "Count": 7, "Value": 7, "Shown": 1, "Hotkey": 42} {
		if o := opt(t, p, label); o.StartNumber == nil || *o.StartNumber != want {
			t.Errorf("%s starts at %v, want %v", label, o.StartNumber, want)
		}
	}
	if o := opt(t, p, "Style"); o.StartText == nil || *o.StartText != "Medium" {
		t.Errorf("Style starts at %v", o.StartText)
	}
	if o := opt(t, p, "Only when off"); o.StartNumber == nil || *o.StartNumber != 0 {
		t.Errorf("an option a condition on a global hides on a new game: %+v", o)
	}
	if o := opt(t, p, "Version"); o.Value != "*.*" || o.ValueCertainty != Pattern {
		t.Errorf("text joined with a global's value = %q (%s), want the pattern *.*", o.Value, o.ValueCertainty)
	}
	for _, label := range []string{"Unfilled", "Level", "Arms"} {
		if o := opt(t, p, label); o.StartNumber != nil || o.StartText != nil {
			t.Errorf("%s starts at %v %v, want unknown", label, o.StartNumber, o.StartText)
		}
	}
}

// VRIK's way: a helper per kind of option, which turns a setting its SKSE
// plugin reads from an INI file into the value shown, through an If.
func TestINISettings(t *testing.T) {
	const src = `Scriptname Ini_MCM extends SKI_ConfigBase
GlobalVariable Property Ini_Flag Auto

Int Function AddSettingToggle(String asText, String asSettingName)
	Int iValue = 0
	If (IniPlugin.GetSetting(asSettingName) > 0)
		iValue = 1
	EndIf
	Return AddToggleOption(asText, iValue)
EndFunction

Int Function AddSettingSlider(String asText, String asSettingName)
	Return AddSliderOption(asText, IniPlugin.GetSetting(asSettingName), "{1}")
EndFunction

Event OnPageReset(string page)
	AddSettingToggle("On", "enabled")
	AddSettingToggle("Off", "disabled")
	AddSettingToggle("Unset", "notInTheFile")
	AddSettingSlider("Scale", "vrScale")
	Int flag = 0
	If Ini_Flag.GetValue() > 0
		flag = 1
	EndIf
	AddToggleOption("Flag", flag)
	; Shown only once a setting is on: still an option the menu has.
	If IniPlugin.GetSetting("showMore") > 0
		AddToggleOption("More", True)
	EndIf
EndEvent
`
	one := 1.0
	m := ExtractWith([]*ir.Script{parse(t, src)}, []Property{{Name: "Ini_Flag", Global: &one}}, Settings{
		"iniplugin.getsetting": {"enabled": 1, "disabled": 0, "vrscale": 79.5, "showmore": 0},
	})
	p := page(t, m, "")
	for label, want := range map[string]float64{"On": 1, "Off": 0, "Scale": 79.5, "Flag": 1, "More": 1} {
		if o := opt(t, p, label); o.StartNumber == nil || *o.StartNumber != want {
			t.Errorf("%s starts at %v, want %v", label, o.StartNumber, want)
		}
	}
	if o := opt(t, p, "Unset"); o.StartNumber != nil {
		t.Errorf("a setting the file doesn't set starts at %v", *o.StartNumber)
	}
}

// A script variable an If may not set holds either value after it: the
// option shown with it has no start value, though an option ID kept that
// way still links. An option added in a loop the game counts repeats.
func TestUncertainValuesAndGameCountedLoops(t *testing.T) {
	const src = `Scriptname Pool_MCM extends SKI_ConfigBase
Actor Property PlayerRef Auto
Int Picked = 1
Int Kept = 2

Event OnPageReset(string page)
	If PlayerRef.IsInCombat()
		Picked = 3
	EndIf
	AddSliderOption("Picked", Picked)
	AddSliderOption("Kept", Kept)
	Int i = 0
	While i < PlayerRef.GetNumItems()
		AddToggleOption("Item", True)
		i += 1
	EndWhile
EndEvent
`
	p := page(t, Extract([]*ir.Script{parse(t, src)}, nil), "")
	if o := opt(t, p, "Picked"); o.StartNumber != nil {
		t.Errorf("a variable an If may not set starts at %v", *o.StartNumber)
	}
	if o := opt(t, p, "Kept"); o.StartNumber == nil || *o.StartNumber != 2 {
		t.Errorf("Kept starts at %v, want 2", o.StartNumber)
	}
	if o := opt(t, p, "Item"); !o.Repeats {
		t.Error("an option added in a loop the game counts doesn't repeat")
	}
}

// Another script's properties, read through a property pointing to the
// quest it's on: what its OnInit leaves them holding on a new game, from
// its declarations and the plugin's values.
func TestRemoteProperties(t *testing.T) {
	const main = `Scriptname Remote_Main extends Quest
Bool Property ResetPosition Auto
Float Property Speed = 1.5 Auto
Int Property Filled Auto
Int Property Unsure Auto
String Property Style Auto
GlobalVariable Property Remote_Count Auto

Event OnInit()
	Startup()
EndEvent

Function Startup()
	ResetPosition = True
	Style = "Medium"
	If Game.GetPlayer().IsInCombat()
		Unsure = 3
	EndIf
EndFunction
`
	five := 5.0
	twelve := 12.0
	remote := Initialized([]*ir.Script{parse(t, main)}, []Property{{Name: "Filled", Number: &twelve}, {Name: "Remote_Count", Global: &five}}, nil)
	want := map[string]float64{"resetposition": 1, "speed": 1.5, "filled": 12}
	for name, n := range want {
		if v, ok := remote[name]; !ok || v.Number == nil || *v.Number != n {
			t.Errorf("%s = %+v, want %v", name, v, n)
		}
	}
	if v := remote["style"]; v.String == nil || *v.String != "Medium" {
		t.Errorf("style = %+v", v)
	}
	if v := remote["remote_count"]; v.Global == nil || *v.Global != 5 {
		t.Errorf("remote_count = %+v", v)
	}
	if v, ok := remote["unsure"]; ok {
		t.Errorf("a property OnInit may or may not set = %+v", v)
	}

	const mcmSrc = `Scriptname Remote_MCM extends SKI_ConfigBase
Remote_Main Property Main Auto
Quest Property MainQuest Auto

Event OnPageReset(string page)
	AddToggleOption("Reset Position", Main.ResetPosition)
	AddSliderOption("Speed", Main.Speed, "{1}")
	AddSliderOption("Count", Main.Remote_Count.GetValue())
	AddMenuOption("Style", Main.Style)
	AddSliderOption("Cast", (MainQuest as Remote_Main).Filled)
	AddSliderOption("Unsure", Main.Unsure)
	If Main.ResetPosition
		AddToggleOption("Only when off", False)
	EndIf
EndEvent
`
	m := Extract([]*ir.Script{parse(t, mcmSrc)}, []Property{{Name: "Main", Remote: Remote{"remote_main": remote}}, {Name: "MainQuest", Remote: Remote{"remote_main": remote}}})
	p := page(t, m, "")
	for label, n := range map[string]float64{"Reset Position": 1, "Speed": 1.5, "Count": 5, "Cast": 12} {
		if o := opt(t, p, label); o.StartNumber == nil || *o.StartNumber != n {
			t.Errorf("%s starts at %v, want %v", label, o.StartNumber, n)
		}
	}
	if o := opt(t, p, "Style"); o.StartText == nil || *o.StartText != "Medium" {
		t.Errorf("Style starts at %v", o.StartText)
	}
	if o := opt(t, p, "Unsure"); o.StartNumber != nil {
		t.Errorf("Unsure starts at %v", *o.StartNumber)
	}
	opt(t, p, "Only when off") // a condition on another script's property hides no option
}

// A classic MCM that keeps its values with MCM Helper: a global set from
// MCM.GetModSettingBool(mod, "key:Section") during setup shows what MCM
// Helper's settings hold.
func TestMCMHelperSettingsInAClassicMCM(t *testing.T) {
	const src = `Scriptname Helper_MCM extends SKI_ConfigBase
GlobalVariable Property Helper_Latest Auto
String Property PluginName = "HelperMod" Auto

Event OnConfigInit()
	Helper_Latest.SetValue((MCM.GetModSettingBool(PluginName, "bLatest:Features") as Int) as Float)
EndEvent

Event OnPageReset(string page)
	AddToggleOption("Latest", Helper_Latest.GetValue() as Bool)
	AddSliderOption("Chance", MCM.GetModSettingFloat(PluginName, "fChance:Features"))
	AddSliderOption("Missing", MCM.GetModSettingFloat(PluginName, "fMissing:Features"))
EndEvent
`
	zero := 0.0
	helper := map[string]float64{"helpermod|blatest:features": 1, "helpermod|fchance:features": 0.25}
	m := ExtractWith([]*ir.Script{parse(t, src)}, []Property{{Name: "Helper_Latest", Global: &zero}}, Settings{
		"mcm.getmodsettingbool": helper, "mcm.getmodsettingfloat": helper,
	})
	p := page(t, m, "")
	for label, want := range map[string]float64{"Latest": 1, "Chance": 0.25} {
		if o := opt(t, p, label); o.StartNumber == nil || *o.StartNumber != want {
			t.Errorf("%s starts at %v, want %v", label, o.StartNumber, want)
		}
	}
	if o := opt(t, p, "Missing"); o.StartNumber != nil {
		t.Errorf("a setting MCM Helper doesn't have starts at %v", *o.StartNumber)
	}
}
