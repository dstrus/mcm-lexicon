package mcmhelper

import (
	"testing"
)

const paged = "\ufeff" + `{
	"modName": "PhysicalCollisionVR",
	"displayName": "Physical Collision VR",
	"pages": [
		{"pageDisplayName": "Main", "content": [
			{"text": "General", "type": "header"},
			{"id": "bEnabled:General", "text": "Enable", "type": "toggle", "help": "Master switch.", "valueOptions": {"sourceType": "ModSettingBool"}},
			{"id": "fRadius:Hands", "text": "Radius", "type": "slider", "valueOptions": {"sourceType": "ModSettingFloat", "min": 1, "max": 10, "step": 0.5, "formatString": "{1} cm"}}
		]},
		{"pageDisplayName": "$PCVR_Hands", "content": [
			{"id": "iMode:Hands", "text": "Mode", "type": "enum", "valueOptions": {"sourceType": "ModSettingInt", "options": ["Off", "$PCVR_On"]}},
			{"text": "Do it", "type": "text", "action": {"type": "CallFunction", "function": "Go"}}
		]}
	]
}`

const flat = `{
	"modName": "PrismaUIAddItemMenu",
	"displayName": "Prisma UI AddItem Menu",
	"content": [
		{"id": "Toggle:General", "text": "Toggle Hotkey", "type": "keymap", "valueOptions": {"sourceType": "GlobalValue", "sourceForm": "PrismaUIAddItemMenu.esp|800", "defaultValue": 59}}
	]
}`

func TestParsePagedConfig(t *testing.T) {
	c, err := Parse([]byte(paged))
	if err != nil {
		t.Fatal(err)
	}
	if c.ModName != "PhysicalCollisionVR" || c.DisplayName != "Physical Collision VR" || len(c.Pages) != 2 {
		t.Fatalf("config = %+v", c)
	}
	if c.Pages[0].Name != "Main" || c.Pages[1].Name != "$PCVR_Hands" || len(c.Pages[0].Controls) != 3 {
		t.Errorf("pages = %+v", c.Pages)
	}
	r := c.Pages[0].Controls[2]
	if r.Key != "fRadius" || r.Section != "Hands" || *r.Min != 1 || *r.Max != 10 || *r.Step != 0.5 || r.Format != "{1} cm" || !r.ModSetting() {
		t.Errorf("slider = %+v", r)
	}
	if e := c.Pages[1].Controls[0]; e.Type != "enum" || len(e.Options) != 2 || e.Options[1] != "$PCVR_On" {
		t.Errorf("enum = %+v", e)
	}
	if x := c.Pages[1].Controls[1]; x.Action != "CallFunction" || x.ID != "" {
		t.Errorf("text = %+v", x)
	}
}

func TestParseFlatConfig(t *testing.T) {
	c, err := Parse([]byte(flat))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Pages) != 1 || c.Pages[0].Name != "" || len(c.Pages[0].Controls) != 1 {
		t.Fatalf("pages = %+v", c.Pages)
	}
	k := c.Pages[0].Controls[0]
	if k.Type != "keymap" || k.Default != 59.0 || k.ModSetting() {
		t.Errorf("keymap = %+v", k)
	}
}

func TestParseRefusesWhatIsntAConfig(t *testing.T) {
	for _, b := range []string{"", "{", "[]", `{"pages": []}`} {
		if _, err := Parse([]byte(b)); err == nil {
			t.Errorf("Parse(%q) accepted it", b)
		}
	}
}

func TestParseINI(t *testing.T) {
	ini := ParseINI([]byte("\ufeff; a comment\n[General]\nbEnabled = 1\n# another\n[Hands]\r\nfRadius=4.0\r\nbroken line\n"))
	if v, ok := ini.Get("general", "BENABLED"); !ok || v != "1" {
		t.Errorf("bEnabled = %q, %v", v, ok)
	}
	if v, _ := ini.Get("Hands", "fRadius"); v != "4.0" {
		t.Errorf("fRadius = %q", v)
	}
	if _, ok := ini.Get("Hands", "broken line"); ok {
		t.Error("a line without = was read")
	}
}
