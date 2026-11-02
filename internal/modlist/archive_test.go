package modlist

import (
	"strings"
	"testing"

	"lexicon/internal/testcorpus"
)

func archive(files map[string]string) string {
	return testcorpus.Archive(testcorpus.ArchiveOptions{Compressed: true}, files)
}

func indexed(t *testing.T, inst *testcorpus.Instance) *List {
	t.Helper()
	src, err := Check(inst.Dir, "")
	if err != nil {
		t.Fatal(err)
	}
	l, err := Read(src)
	if err != nil {
		t.Fatal(err)
	}
	l.Index(MCMScope)
	return l
}

func read(t *testing.T, l *List, rel string) (File, string) {
	t.Helper()
	f, ok := l.Open(rel)
	if !ok {
		t.Fatalf("%s isn't seen", rel)
	}
	b, err := f.Read()
	if err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
	return f, string(b)
}

func TestArchivesBeneathLooseFiles(t *testing.T) {
	inst := testcorpus.New(t, "MGO", "Default")
	// Packed is above Alpha in the mod list, but Alpha's loose copy wins.
	inst.Mod("Alpha", true, map[string]string{"Scripts/Shared.pex": "loose"})
	inst.Mod("Packed", true, map[string]string{
		"Packed.esp": "x",
		"Packed.bsa": archive(map[string]string{
			"scripts/shared.pex":                        "packed",
			"scripts/packed_mcm.pex":                    "packed mcm",
			"interface/translations/packed_english.txt": "$Packed\tPacked",
			"mcm/config/packed/config.json":             "{}",
			"mcm/settings/packed.ini":                   "settings stay loose",
			"textures/packed/a.dds":                     "not in scope",
		}),
		"Packed - Textures.bsa": archive(map[string]string{"scripts/textures_only.pex": "from the textures archive"}),
	})
	// Later's plugin loads after Packed's, so its archive wins over Packed's.
	inst.Mod("Later", true, map[string]string{"Later.esm": "x", "Later.bsa": archive(map[string]string{"scripts/packed_mcm.pex": "later"})})
	// An archive whose plugin is inactive, or named after no plugin, isn't loaded.
	inst.Mod("Off", true, map[string]string{"Off.esp": "x", "Off.bsa": archive(map[string]string{"scripts/off.pex": "off"})})
	inst.Mod("Stray", true, map[string]string{"Stray.bsa": archive(map[string]string{"scripts/stray.pex": "stray"})})
	inst.Plugins("*Packed.esp", "*Later.esm", "Off.esp")

	l := indexed(t, inst)
	if f, b := read(t, l, "Scripts/Shared.pex"); b != "loose" || f.Archive != "" || f.Mod != "Alpha" {
		t.Errorf("Shared.pex = %q from %+v", b, f)
	}
	if c := l.Copies("scripts/shared.pex"); len(c) != 2 || c[1].Archive != "Packed.bsa" {
		t.Errorf("copies of Shared.pex = %+v", c)
	}
	if f, b := read(t, l, "scripts/PACKED_MCM.pex"); b != "later" || f.Archive != "Later.bsa" || f.Mod != "Later" {
		t.Errorf("Packed_MCM.pex = %q from %+v", b, f)
	}
	if _, b := read(t, l, "Scripts/textures_only.pex"); b != "from the textures archive" {
		t.Errorf("textures_only.pex = %q", b)
	}
	for _, rel := range []string{"Scripts/off.pex", "Scripts/stray.pex", "textures/packed/a.dds", "MCM/Settings/packed.ini"} {
		if f, ok := l.Open(rel); ok {
			t.Errorf("%s is seen, from %+v", rel, f)
		}
	}
	for _, rel := range []string{"Interface/Translations/Packed_ENGLISH.txt", "MCM/Config/Packed/config.json"} {
		read(t, l, rel)
	}
	var loaded []string
	for _, a := range l.Archives() {
		loaded = append(loaded, a.Path)
	}
	if got := strings.Join(loaded, ", "); got != "Packed.bsa, Packed - Textures.bsa, Later.bsa" {
		t.Errorf("archives loaded: %s", got)
	}
}

func TestArchivesWithoutPluginsFileAndDamaged(t *testing.T) {
	inst := testcorpus.New(t, "MGO", "Default")
	inst.Mod("Low", true, map[string]string{"Low.esp": "x", "Low.bsa": archive(map[string]string{"scripts/a.pex": "low"})})
	inst.Mod("High", true, map[string]string{"High.esp": "x", "High.bsa": archive(map[string]string{"scripts/a.pex": "high"})})
	inst.Mod("Broken", true, map[string]string{"Broken.esp": "x", "Broken.bsa": "BSA\x00 cut short"})
	inst.NoPluginsFile()

	l := indexed(t, inst)
	// Without plugins.txt, plugins load in the order of the mods holding them.
	if _, b := read(t, l, "scripts/a.pex"); b != "high" {
		t.Errorf("a.pex = %q", b)
	}
	if p := l.ArchiveProblems(); len(p) != 1 || p[0].File.Path != "Broken.bsa" || p[0].File.Mod != "Broken" {
		t.Errorf("problems = %+v", p)
	}
}
