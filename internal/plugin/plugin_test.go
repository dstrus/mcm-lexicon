package plugin

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lexicon/internal/testcorpus"
)

// The encoder: just enough of the format to build plugins by hand.

func le(v any) []byte {
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, v)
	return b.Bytes()
}

func wstr(s string) []byte { return append(le(uint16(len(s))), s...) }

func sub(typ string, data []byte) []byte {
	return append(append([]byte(typ), le(uint16(len(data)))...), data...)
}

func record(typ string, flags, formID uint32, data []byte) []byte {
	var b bytes.Buffer
	b.WriteString(typ)
	b.Write(le(uint32(len(data))))
	b.Write(le(flags))
	b.Write(le(formID))
	b.Write(make([]byte, 8))
	b.Write(data)
	return b.Bytes()
}

func group(label string, groupType int32, body []byte) []byte {
	var b bytes.Buffer
	b.WriteString("GRUP")
	b.Write(le(uint32(headerSize + len(body))))
	b.WriteString(label)
	b.Write(le(groupType))
	b.Write(make([]byte, 8))
	b.Write(body)
	return b.Bytes()
}

func compress(data []byte) []byte {
	var z bytes.Buffer
	w := zlib.NewWriter(&z)
	w.Write(data)
	w.Close()
	return append(le(uint32(len(data))), z.Bytes()...)
}

// vmad is a VMAD subrecord (version 5, object format 2) with one script.
func vmad(script string, props ...[]byte) []byte {
	var b bytes.Buffer
	b.Write(le(int16(5)))
	b.Write(le(int16(2)))
	b.Write(le(uint16(1)))
	b.Write(wstr(script))
	b.WriteByte(0) // status
	b.Write(le(uint16(len(props))))
	for _, p := range props {
		b.Write(p)
	}
	b.Write([]byte{0x02, 0x00, 0xFF}) // the start of quest fragments, which aren't read
	return sub("VMAD", b.Bytes())
}

func prop(name string, typ PropertyType, value []byte) []byte {
	return append(append(append(wstr(name), byte(typ)), 1), value...)
}

func strings_(ss ...string) []byte {
	b := le(uint32(len(ss)))
	for _, s := range ss {
		b = append(b, wstr(s)...)
	}
	return b
}

func build(tes4Flags uint32) []byte {
	var f bytes.Buffer
	header := append(sub("HEDR", make([]byte, 12)), sub("MAST", []byte("Skyrim.esm\x00"))...)
	header = append(append(header, sub("DATA", make([]byte, 8))...), sub("MAST", []byte("Base.esp\x00"))...)
	f.Write(record("TES4", tes4Flags, 0, header))
	glob := func(edid, kind string, v float32) []byte {
		return append(append(sub("EDID", []byte(edid+"\x00")), sub("FNAM", []byte(kind))...), sub("FLTV", le(v))...)
	}
	f.Write(group("GLOB", 0, bytes.Join([][]byte{
		record("GLOB", 0, 0x02000800, glob("Dirty_Toggle", "s", 1)),
		record("GLOB", flagCompressed, 0x02000801, compress(glob("Dirty_Rate", "f", 2.5))),
		record("GLOB", 0, 0x01000D62, glob("BaseOverride", "l", 7)), // a master's, overridden
	}, nil)))
	quest := append(sub("EDID", []byte("DirtyMCMQuest\x00")), vmad("Dirty_Quest_MCM",
		prop("ModName", PropertyString, wstr("Dirt & Blood")),
		prop("Pages", PropertyStringArray, strings_("Settings", "Caf\xe9 \x93Quoted\x94")),
		prop("PlayerRef", PropertyObject, append(le(uint32(0)), le(uint32(0x14))...)),
		prop("Version", PropertyInt, le(int32(-3))),
		prop("Scale", PropertyFloat, le(float32(1.5))),
		prop("Enabled", PropertyBool, []byte{1}),
		prop("Unfilled", PropertyNone, nil),
	)...)
	compressed := append(sub("EDID", []byte("Squeezed\x00")), vmad("Squeezed_MCM", prop("ModName", PropertyString, wstr("Squeezed")))...)
	plain := sub("EDID", []byte("NoScripts\x00"))
	body := bytes.Join([][]byte{
		record("QUST", 0, 0x01000801, quest),
		group("\x01\x08\x00\x01", 10, record("DIAL", 0, 0x900, nil)), // a quest's children
		record("QUST", flagCompressed, 0x01000802, compress(compressed)),
		record("QUST", 0, 0x01000803, plain),
	}, nil)
	f.Write(group("QUST", 0, body))
	f.Write(group("NPC_", 0, nil))
	return f.Bytes()
}

func TestRead(t *testing.T) {
	b := build(flagMaster | flagLight)
	p, err := Read(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	if !p.Master || !p.Light || p.Localized {
		t.Errorf("flags: %+v", p)
	}
	if len(p.Quests) != 2 {
		t.Fatalf("quests = %+v", p.Quests)
	}
	q := p.Quests[0]
	if q.EditorID != "DirtyMCMQuest" || q.FormID != 0x01000801 || len(q.Scripts) != 1 || q.Scripts[0].Name != "Dirty_Quest_MCM" {
		t.Fatalf("quest = %+v", q)
	}
	s := q.Scripts[0]
	if v, ok := s.Property("modname"); !ok || v.String != "Dirt & Blood" {
		t.Errorf("ModName = %+v", v)
	}
	if v, _ := s.Property("Pages"); len(v.Strings) != 2 || v.Strings[1] != "Café “Quoted”" {
		t.Errorf("Pages = %q", v.Strings)
	}
	for name, want := range map[string]float64{"PlayerRef": 0x14, "Version": -3, "Scale": 1.5, "Enabled": 1} {
		if v, _ := s.Property(name); v.Number != want {
			t.Errorf("%s = %v, want %v", name, v.Number, want)
		}
	}
	if v, ok := s.Property("Unfilled"); !ok || v.Type != PropertyNone {
		t.Errorf("an unfilled property: %+v, %v", v, ok)
	}
	if c := p.Quests[1]; c.EditorID != "Squeezed" || c.Scripts[0].Name != "Squeezed_MCM" {
		t.Errorf("the compressed quest = %+v", c)
	}
	if strings.Join(p.Masters, ",") != "Skyrim.esm,Base.esp" {
		t.Errorf("masters = %q", p.Masters)
	}
	if len(p.Globals) != 3 || p.Globals[0].EditorID != "Dirty_Toggle" || p.Globals[0].Value != 1 || p.Globals[1].Value != 2.5 || p.Globals[2].Value != 7 {
		t.Fatalf("globals = %+v", p.Globals)
	}
	// A form ID's first byte counts the masters, then the plugin itself.
	p.Name = "Dirty.esp"
	for id, owner := range map[uint32]string{0x00000014: "Skyrim.esm", 0x01000D62: "Base.esp", 0x02000800: "Dirty.esp"} {
		if got := p.Owner(id); got != owner {
			t.Errorf("owner of %08X = %s, want %s", id, got, owner)
		}
	}
	if FormKey("Base.esp", 0x01000D62) != FormKey("BASE.ESP", 0x00000D62) {
		t.Error("a form's key depends on the byte that counts its owner")
	}
}

func TestReadFileNamesAndExtensions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Experience.esl")
	os.WriteFile(path, build(0), 0o644)
	p, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Experience.esl" || !p.Light || !p.Master {
		t.Errorf("an .esl: %+v", p)
	}
}

func TestMalformedPluginsAreErrorsNotCrashes(t *testing.T) {
	good := build(0)
	cases := map[string][]byte{
		"empty":       nil,
		"not tes4":    append([]byte("TES3"), good[4:]...),
		"short group": good[:len(good)-10],
	}
	// Cut the file at every length: never a panic, and an error or a
	// result for each.
	for n := 0; n < len(good); n += 7 {
		cases["cut"] = good[:n]
		for name, b := range cases {
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("%s (%d bytes): panic %v", name, len(b), r)
					}
				}()
				Read(bytes.NewReader(b), int64(len(b)))
			}()
		}
	}
	for name, b := range map[string][]byte{"empty": nil, "not tes4": cases["not tes4"], "short group": cases["short group"]} {
		if _, err := Read(bytes.NewReader(b), int64(len(b))); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

// The plugins PLAN.md §2.3 names, in the real lists.
func TestRealSnapshots(t *testing.T) {
	want := map[string]string{ // plugin file → the ModName its MCM quest holds
		"Dirt and Blood - Dynamic Visuals.esp": "Dirt & Blood",
		"BeSeated_3.0.0.esp":                   "Be Seated",
		"SofiaFollower.esp":                    "Sofia Follower",
	}
	for _, snap := range testcorpus.Snapshots(t) {
		t.Run(snap.Name, func(t *testing.T) {
			read, failed := 0, 0
			filepath.WalkDir(filepath.Join(snap.Dir, "mods"), func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				switch strings.ToLower(filepath.Ext(path)) {
				case ".esp", ".esm", ".esl":
				default:
					return nil
				}
				p, err := ReadFile(path)
				if err != nil {
					failed++
					t.Errorf("%v", err)
					return nil
				}
				read++
				if name, ok := want[p.Name]; ok {
					found := false
					for _, q := range p.Quests {
						for _, s := range q.Scripts {
							if v, ok := s.Property("ModName"); ok && v.String == name {
								found = true
							}
						}
					}
					if !found {
						t.Errorf("%s: no quest script with ModName %q in %+v", p.Name, name, p.Quests)
					}
				}
				return nil
			})
			t.Logf("read %d plugins, %d failed", read, failed)
		})
	}
}

// A quest's aliases come after its scripts and fragments: Seasonal
// Weathers' main script is on its quest's alias 0, which the MCM's SFMain
// property names.
func TestQuestAliases(t *testing.T) {
	object := func(alias int16, id uint32) []byte {
		return append(append(le(uint16(0)), le(alias)...), le(id)...)
	}
	var v bytes.Buffer
	v.Write(le(int16(5)))
	v.Write(le(int16(2)))
	v.Write(le(uint16(1)))
	v.Write(wstr("Seasons_MCM"))
	v.WriteByte(0)
	v.Write(le(uint16(2)))
	v.Write(prop("SFMain", PropertyObject, object(0, 0x800)))
	v.Write(prop("PlayerRef", PropertyObject, object(-1, 0x14)))
	v.WriteByte(2)              // fragments: unknown
	v.Write(le(uint16(1)))      // one fragment
	v.Write(wstr("QF_Seasons")) // its file
	v.Write(le(uint16(10)))
	v.Write(le(int16(0)))
	v.Write(le(int32(0)))
	v.WriteByte(0)
	v.Write(wstr("QF_Seasons"))
	v.Write(wstr("Fragment_0"))
	v.Write(le(uint16(1))) // one alias
	v.Write(object(0, 0x800))
	v.Write(le(int16(5)))
	v.Write(le(int16(2)))
	v.Write(le(uint16(1)))
	v.Write(wstr("Seasons_Main"))
	v.WriteByte(0)
	v.Write(le(uint16(1)))
	v.Write(prop("Speed", PropertyFloat, le(float32(2.5))))
	quest := append(sub("EDID", []byte("SeasonsQuest\x00")), sub("VMAD", v.Bytes())...)
	var f bytes.Buffer
	f.Write(record("TES4", 0, 0, sub("HEDR", make([]byte, 12))))
	f.Write(group("QUST", 0, record("QUST", 0, 0x800, quest)))
	p, err := Read(bytes.NewReader(f.Bytes()), int64(f.Len()))
	if err != nil {
		t.Fatal(err)
	}
	q := p.Quests[0]
	if len(q.Aliases) != 1 || q.Aliases[0].ID != 0 || q.Aliases[0].Scripts[0].Name != "Seasons_Main" {
		t.Fatalf("aliases = %+v", q.Aliases)
	}
	if v, _ := q.Aliases[0].Scripts[0].Property("Speed"); v.Number != 2.5 {
		t.Errorf("the alias script's Speed = %v", v.Number)
	}
	s := q.Scripts[0]
	if v, _ := s.Property("SFMain"); v.Alias != 0 || v.Number != 0x800 {
		t.Errorf("SFMain = %+v", v)
	}
	if v, _ := s.Property("PlayerRef"); v.Alias != -1 {
		t.Errorf("PlayerRef's alias = %d, want -1", v.Alias)
	}
}
