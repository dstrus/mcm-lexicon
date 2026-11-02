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
	f.Write(record("TES4", tes4Flags, 0, sub("HEDR", make([]byte, 12))))
	f.Write(group("GLOB", 0, record("GLOB", 0, 0x800, sub("EDID", []byte("Ignored\x00")))))
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
