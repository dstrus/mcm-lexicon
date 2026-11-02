package mcm

import (
	"fmt"
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
