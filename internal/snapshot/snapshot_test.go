package snapshot

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"lexicon/internal/modlist"
	"lexicon/internal/testcorpus"
)

// A compiled script that extends SKI_ConfigBase names it in its string table.
const mcmPex = "\xde\xc0\x57\xfa...Alpha_MCM\x00SKI_ConfigBase\x00OnPageReset..."

func build(t *testing.T) *testcorpus.Instance {
	t.Helper()
	inst := testcorpus.New(t, "[NoDelete] MGO", "VR")
	inst.Mod("Nothing", true, map[string]string{"meshes/a.nif": "x", "Nothing.esp": "p", "Interface/Translations/Nothing_ENGLISH.txt": "t"})
	inst.Mod("Alpha", true, map[string]string{
		"Scripts/Alpha_MCM.pex":                    mcmPex,
		"Scripts/Other.pex":                        "no menu here",
		"Scripts/SKI_ConfigBase.pex":               "SKI_ConfigBase itself",
		"Source/Scripts/Alpha_MCM.psc":             "ScriptName Alpha_MCM extends SKI_ConfigBase",
		"Alpha.esp":                                "plugin",
		"Interface/Translations/Alpha_ENGLISH.txt": "en",
		"Interface/Translations/Alpha_GERMAN.txt":  "de",
	})
	inst.Separator("Gameplay")
	inst.Mod("Beta Helper", true, map[string]string{
		"Scripts/Beta_MCM.pex":         "...MCM_ConfigBase...",
		"MCM/Config/Beta/config.json":  `{"displayName": "Beta"}`,
		"MCM/Config/Beta/settings.ini": "[Main]",
		"MCM/Settings/Beta.ini":        "[Main]\nx=1",
		"Beta.esp":                     "plugin",
		"BetaPatch.esl":                "plugin",
	})
	inst.Mod("Settings Patch", true, map[string]string{"MCM/Config/Beta/settings.ini": "[Main]\nx=2"})
	inst.Mod("Alpha - Translations", true, map[string]string{
		"Interface/Translations/Alpha_FRENCH.txt":  "fr",
		"Interface/Translations/alpha_english.txt": "en too",
	})
	inst.Mod("Recs", true, map[string]string{
		"McmRecorder/My.json":            `{"name": "My", "version": "1.0.0", "author": "me", "autorun": "false"}`,
		"McmRecorder/My/0001_Alpha.json": `[{"Mod": "Alpha", "page": "Settings", "option": "Thing", "toggle": "on"}, {"mod": "Alpha", "option": "Slide ", "index": 2, "slider": 25.0}]`,
	})
	inst.Mod("Off", false, map[string]string{"MCM/Config/Off/config.json": "{}", "Off.esp": "p"})
	inst.Unmanaged("DLC: Dawnguard")
	inst.Overwrite(map[string]string{"MCM/Settings/Alpha.ini": "a=1", "McmRecorder/Late.json": "{}", "SKSE/Plugins/x.log": "no"})
	inst.Plugins("*Alpha.esp", "Beta.esp", "*BetaPatch.esl", "*Nothing.esp", "*Off.esp")
	return inst
}

func read(t *testing.T, dir string) *modlist.List {
	t.Helper()
	src, err := modlist.Check(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	l, err := modlist.Read(src)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

// inventory reads a snapshot's snapshot.json, with numbers as their text.
func inventory(t *testing.T, dir string) Inventory {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var inv Inventory
	if err := dec.Decode(&inv); err != nil {
		t.Fatal(err)
	}
	return inv
}

func paths(inv Inventory) []string {
	var out []string
	for _, f := range inv.Files {
		out = append(out, f.Path)
	}
	sort.Strings(out)
	return out
}

func TestWriteFolder(t *testing.T) {
	inst := build(t)
	out := filepath.Join(t.TempDir(), "snap")
	sum, err := Write(read(t, inst.Dir), out, Options{Tool: "test", Now: time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	inv := inventory(t, out)
	want := []string{
		"mods/Alpha - Translations/Interface/Translations/alpha_english.txt",
		"mods/Alpha/Alpha.esp",
		"mods/Alpha/Interface/Translations/Alpha_ENGLISH.txt",
		"mods/Alpha/Scripts/Alpha_MCM.pex",
		"mods/Alpha/Source/Scripts/Alpha_MCM.psc",
		"mods/Beta Helper/Beta.esp",
		"mods/Beta Helper/BetaPatch.esl",
		"mods/Beta Helper/MCM/Config/Beta/config.json",
		"mods/Beta Helper/MCM/Config/Beta/settings.ini",
		"mods/Beta Helper/MCM/Settings/Beta.ini",
		"mods/Beta Helper/Scripts/Beta_MCM.pex",
		"mods/Recs/McmRecorder/My.json",
		"mods/Recs/McmRecorder/My/0001_Alpha.json",
		"mods/Settings Patch/MCM/Config/Beta/settings.ini",
		"overwrite/MCM/Settings/Alpha.ini",
		"overwrite/McmRecorder/Late.json",
	}
	if got := paths(inv); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("files:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for _, f := range inv.Files {
		data, err := os.ReadFile(filepath.Join(out, "mo2", filepath.FromSlash(f.Path)))
		if err != nil {
			t.Errorf("%s wasn't written: %v", f.Path, err)
			continue
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != f.SHA256 || int64(len(data)) != f.Bytes {
			t.Errorf("%s: hash or size is off", f.Path)
		}
	}
	if sum.Files != len(want) || sum.Mods != 5 || sum.Steps != 2 || sum.Out != out {
		t.Errorf("summary = %+v", sum)
	}

	// Mods in priority order, highest first, with their kinds.
	var mods []string
	for _, m := range inv.Mods {
		mods = append(mods, m.Name+" "+strings.Join(m.Kinds, ","))
	}
	if want := "Recs recordings; Alpha - Translations translations; Settings Patch settings:1; Beta Helper compiled:1,mcmhelper:1; Alpha source:1"; strings.Join(mods, "; ") != want {
		t.Errorf("mods = %q", mods)
	}
	if inv.List.EnabledMods != 6 || inv.List.IncludedMods != 5 || inv.List.Profile != "VR" || inv.List.Game != "Skyrim VR" || inv.List.Name != "[NoDelete] MGO" {
		t.Errorf("list = %+v", inv.List)
	}
	// Steps, with keys in any case, and the value by its kind.
	if len(inv.Steps) != 2 || inv.Steps[0].Mod != "Alpha" || inv.Steps[0].Page != "Settings" || inv.Steps[0].Key != "toggle" || inv.Steps[0].Value != "on" {
		t.Errorf("step 1 = %+v", inv.Steps[0])
	}
	if s := inv.Steps[1]; s.Recording != "My" || s.Option != "Slide " || s.Index != 2 || s.Key != "slider" || fmt.Sprint(s.Value) != "25.0" {
		t.Errorf("step 2 = %+v", s)
	}

	// The trimmed instance opens like any other, with the real order and marks.
	l := read(t, filepath.Join(out, "mo2"))
	var order []string
	for _, m := range l.Mods {
		order = append(order, m.Name)
	}
	if want := "Alpha, Beta Helper, Settings Patch, Alpha - Translations, Recs, DLC: Dawnguard, overwrite"; strings.Join(order, ", ") != want {
		t.Errorf("snapshot's order = %q", order)
	}
	if l.PluginActive("Beta.esp") || !l.PluginActive("Alpha.esp") || l.PluginActive("Nothing.esp") {
		t.Error("plugins.txt wasn't trimmed with its marks kept")
	}
	ini, _ := os.ReadFile(filepath.Join(out, "mo2", "ModOrganizer.ini"))
	if string(ini) != "[General]\ngameName=Skyrim VR\nselected_profile=@ByteArray(VR)\n" {
		t.Errorf("ModOrganizer.ini = %q", ini)
	}
	md, _ := os.ReadFile(filepath.Join(out, "SNAPSHOT.md"))
	for _, want := range []string{"# MCM Lexicon snapshot: [NoDelete] MGO", "5 of 6 enabled mods, 16 files", "| 1 | Recs | recordings |", "| 3 | Settings Patch | settings:1 |", "| 5 | Alpha | source:1 |", `| "My" | "Alpha" | "Settings" | "Thing" |  | "toggle on" |`, `| "My" | "Alpha" |  | "Slide " | "2" | "slider 25.0" |`, "## Known gaps"} {
		if !strings.Contains(string(md), want) {
			t.Errorf("SNAPSHOT.md lacks %q:\n%s", want, md)
		}
	}
	if _, err := Write(l, out, Options{}); err == nil {
		t.Error("writing over an existing folder was allowed")
	}
}

func TestWriteZipUsesForwardSlashes(t *testing.T) {
	inst := build(t)
	out := filepath.Join(t.TempDir(), "snap.zip")
	sum, err := Write(read(t, inst.Dir), out, Options{})
	if err != nil {
		t.Fatal(err)
	}
	r, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	names := map[string]bool{}
	for _, f := range r.File {
		if strings.Contains(f.Name, `\`) {
			t.Errorf("backslash in %q", f.Name)
		}
		names[f.Name] = true
	}
	// The files, the two profile files, the ini, and the two descriptions.
	if len(r.File) != sum.Files+5 || !names["SNAPSHOT.md"] || !names["snapshot.json"] || !names["mo2/profiles/VR/modlist.txt"] || !names["mo2/mods/Recs/McmRecorder/My/0001_Alpha.json"] {
		t.Errorf("zip holds %d files: %v", len(r.File), names)
	}
}

func TestDataFolderIsRefused(t *testing.T) {
	data := filepath.Join(t.TempDir(), "Data")
	os.MkdirAll(data, 0o755)
	os.WriteFile(filepath.Join(data, "Skyrim.esm"), nil, 0o644)
	if _, err := Write(read(t, data), filepath.Join(t.TempDir(), "x"), Options{}); err == nil {
		t.Error("a Data folder was snapshotted")
	}
}

// A snapshot of a snapshot holds the same files: the Go port picks what the
// script picked.
func TestRealSnapshotsRoundTrip(t *testing.T) {
	for _, snap := range testcorpus.Snapshots(t) {
		t.Run(snap.Name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "again")
			if _, err := Write(read(t, snap.Dir), out, Options{}); err != nil {
				t.Fatal(err)
			}
			have := map[string]bool{}
			filepath.WalkDir(filepath.Join(snap.Dir), func(p string, d os.DirEntry, err error) error {
				if err == nil && !d.IsDir() {
					rel, _ := filepath.Rel(snap.Dir, p)
					have[filepath.ToSlash(rel)] = true
				}
				return nil
			})
			inv := inventory(t, out)
			got := map[string]bool{}
			for _, f := range inv.Files {
				got[f.Path] = true
				if !have[f.Path] {
					t.Errorf("copied %s, which the script didn't", f.Path)
				}
			}
			original := read(t, snap.Dir)
			for p := range have {
				if strings.HasPrefix(p, "profiles/") || p == "ModOrganizer.ini" {
					continue
				}
				// The script copied files from some mods it then left out of
				// modlist.txt; those don't exist as far as the list says.
				if parts := strings.Split(p, "/"); parts[0] == "mods" {
					if _, listed := original.Mod(parts[1]); !listed {
						continue
					}
				}
				if !got[p] {
					t.Errorf("missed %s, which the script copied", p)
				}
			}
			// plugins.txt comes out the same, line for line. modlist.txt keeps
			// every line the script kept, in order, and may add mods the
			// script copied files from but forgot to list (those with only an
			// MCM Helper settings.ini).
			profile := func(dir, f string) []string {
				b, _ := os.ReadFile(filepath.Join(dir, "profiles", inv.List.Profile, f))
				return strings.Split(strings.TrimRight(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n"), "\n")
			}
			if a, b := profile(snap.Dir, "plugins.txt"), profile(filepath.Join(out, "mo2"), "plugins.txt"); strings.Join(a, "\n") != strings.Join(b, "\n") {
				t.Errorf("plugins.txt differs:\n%s\nwant:\n%s", strings.Join(b, "\n"), strings.Join(a, "\n"))
			}
			theirs, ours := profile(snap.Dir, "modlist.txt"), profile(filepath.Join(out, "mo2"), "modlist.txt")
			at := 0
			for _, line := range theirs {
				for at < len(ours) && ours[at] != line {
					if m, ok := read(t, filepath.Join(out, "mo2")).Mod(ours[at][1:]); !ok || m.Dir == "" {
						t.Errorf("modlist.txt lists %q, which has no folder", ours[at])
					}
					at++
				}
				if at == len(ours) {
					t.Errorf("modlist.txt lost %q", line)
					break
				}
				at++
			}
			l := read(t, filepath.Join(out, "mo2"))
			entries, _ := os.ReadDir(filepath.Join(out, "mo2", "mods"))
			for _, e := range entries {
				if m, ok := l.Mod(e.Name()); !ok || !m.Enabled {
					t.Errorf("mod folder %q isn't listed", e.Name())
				}
			}
			if len(inv.Steps) == 0 {
				t.Error("no recorded steps found")
			}
		})
	}
}
