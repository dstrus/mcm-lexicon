package modlist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lexicon/internal/testcorpus"
)

func TestOrderOverridesAndDisabledMods(t *testing.T) {
	inst := testcorpus.New(t, "[NoDelete] MGO 4", "Default")
	inst.Mod("Alpha", true, map[string]string{"Scripts/Shared.pex": "alpha", "MCM/Config/Alpha/config.json": "{}", "Alpha.esp": "x"})
	inst.Separator("Overhauls")
	inst.Mod("Beta's Mod", true, map[string]string{"scripts/SHARED.PEX": "beta", "Source/Scripts/Beta.psc": "ScriptName Beta"})
	inst.Mod("Gamma (off)", false, map[string]string{"Scripts/Gamma.pex": "gamma", "Scripts/Shared.pex": "gamma"})
	inst.Unmanaged("DLC: Dawnguard")
	inst.Overwrite(map[string]string{"MCM/Settings/Alpha.ini": "a=1", "McmRecorder/Mine.json": "{}"})

	src, err := Check(inst.Dir, "")
	if err != nil {
		t.Fatal(err)
	}
	l, err := Read(src)
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, m := range l.Mods {
		order = append(order, m.Name)
	}
	if want := "Alpha, Beta's Mod, Gamma (off), DLC: Dawnguard, overwrite"; strings.Join(order, ", ") != want {
		t.Errorf("load order = %q, want %q", order, want)
	}
	// MO2 numbers from the bottom, counting the separator.
	if a, _ := l.Mod("alpha"); a.Priority != 0 || !a.Enabled {
		t.Errorf("Alpha = %+v", a)
	}
	if b, _ := l.Mod("Beta's Mod"); b.Priority != 2 {
		t.Errorf("Beta's priority = %d, want 2", b.Priority)
	}
	if g, _ := l.Mod("Gamma (off)"); g.Enabled || g.Dir == "" {
		t.Errorf("Gamma = %+v", g)
	}
	if d, _ := l.Mod("DLC: Dawnguard"); !d.Unmanaged || !d.Enabled || d.Dir != "" {
		t.Errorf("Dawnguard = %+v", d)
	}
	if o, _ := l.Mod("overwrite"); o.Layer != LayerOverwrite || o.Priority != 5 {
		t.Errorf("overwrite = %+v", o)
	}

	l.Index(MCMScope)
	// The higher mod wins, whatever the case; the disabled one is never seen.
	f, ok := l.Open(`SCRIPTS\shared.pex`)
	if !ok || f.Mod != "Beta's Mod" || f.Path != "scripts/SHARED.PEX" || f.Size != 4 {
		t.Errorf("Open(Shared.pex) = %+v, %v", f, ok)
	}
	copies := l.Copies("Scripts/Shared.pex")
	if len(copies) != 2 || copies[0].Mod != "Beta's Mod" || copies[1].Mod != "Alpha" {
		t.Errorf("Copies = %+v", copies)
	}
	if _, ok := l.Open("Scripts/Gamma.pex"); ok {
		t.Error("a disabled mod's file was indexed")
	}
	if f, ok := l.Open("mcm/settings/alpha.ini"); !ok || f.Mod != "overwrite" {
		t.Errorf("overwrite doesn't win: %+v, %v", f, ok)
	}
	if f, ok := l.Open("Alpha.esp"); !ok || f.Mod != "Alpha" {
		t.Errorf("top-level plugin: %+v, %v", f, ok)
	}
	var paths []string
	for _, f := range l.Files("MCM") {
		paths = append(paths, f.Path)
	}
	if want := "MCM/Config/Alpha/config.json, MCM/Settings/Alpha.ini"; strings.Join(paths, ", ") != want {
		t.Errorf("Files(MCM) = %q", paths)
	}
	if n := len(l.Files("")); n != 6 {
		t.Errorf("indexed %d files, want 6", n)
	}
}

func TestPlugins(t *testing.T) {
	inst := testcorpus.New(t, "mgo", "Default")
	inst.Plugins("*Alpha.esp", "Beta.esp")
	src, _ := Check(inst.Dir, "")
	l, err := Read(src)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]bool{"alpha.esp": true, "Beta.esp": false, "Skyrim.esm": true, "Unlisted.esp": false} {
		if got := l.PluginActive(name); got != want {
			t.Errorf("PluginActive(%q) = %v, want %v", name, got, want)
		}
	}
	if p := l.Plugins(); len(p) != 2 || p[0].Name != "Alpha.esp" || !p[0].Active || p[1].Active {
		t.Errorf("Plugins = %+v", p)
	}

	inst.NoPluginsFile()
	l, _ = Read(src)
	if !l.PluginActive("Anything.esp") || l.Plugins() != nil {
		t.Error("without plugins.txt every plugin should count as active")
	}
}

func TestGameDataSitsBelowEveryMod(t *testing.T) {
	game := filepath.Join(t.TempDir(), "Skyrim VR")
	os.MkdirAll(filepath.Join(game, "Data", "Scripts"), 0o755)
	os.WriteFile(filepath.Join(game, "Data", "Skyrim.esm"), nil, 0o644)
	os.WriteFile(filepath.Join(game, "Data", "Scripts", "SKI_ConfigBase.pex"), []byte("game"), 0o644)
	inst := testcorpus.New(t, "mgo", "Default")
	inst.GamePath(game)
	inst.Mod("SkyUI", true, map[string]string{"Scripts/SKI_ConfigBase.pex": "skyui"})
	src, _ := Check(inst.Dir, "")
	l, err := Read(src)
	if err != nil {
		t.Fatal(err)
	}
	if l.DataDir != filepath.Join(game, "Data") || l.Mods[0].Layer != LayerData {
		t.Errorf("DataDir = %q, first mod %+v", l.DataDir, l.Mods[0])
	}
	l.Index(MCMScope)
	if f, _ := l.Open("Scripts/SKI_ConfigBase.pex"); f.Mod != "SkyUI" {
		t.Errorf("the mod should win over Data: %+v", f)
	}
	if f, _ := l.Open("Skyrim.esm"); f.Mod != "Data" {
		t.Errorf("Skyrim.esm should come from Data: %+v", f)
	}
}

func TestMissingGamePathIsNoted(t *testing.T) {
	inst := testcorpus.New(t, "mgo", "Default")
	inst.GamePath(`F:\Games\Skyrim VR`)
	src, _ := Check(inst.Dir, "")
	l, err := Read(src)
	if err != nil {
		t.Fatal(err)
	}
	if l.DataDir != "" || l.DataPath == "" || len(l.Mods) != 1 {
		t.Errorf("DataDir %q, DataPath %q, mods %+v", l.DataDir, l.DataPath, l.Mods)
	}
}

func TestDataFolderIsOneLayer(t *testing.T) {
	data := filepath.Join(t.TempDir(), "Data")
	os.MkdirAll(filepath.Join(data, "MCM", "Config", "X"), 0o755)
	os.WriteFile(filepath.Join(data, "Skyrim.esm"), nil, 0o644)
	os.WriteFile(filepath.Join(data, "MCM", "Config", "X", "config.json"), []byte("{}"), 0o644)
	src, _ := Check(data, "")
	l, err := Read(src)
	if err != nil {
		t.Fatal(err)
	}
	l.Index(MCMScope)
	if f, ok := l.Open("mcm/config/x/config.json"); !ok || f.Mod != "Data" {
		t.Errorf("Open = %+v, %v", f, ok)
	}
	if !l.PluginActive("Anything.esp") {
		t.Error("a Data folder has no plugins.txt here, so every plugin counts as active")
	}
}

func TestResolveMatchesCaseAsWindowsDoes(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "Source", "scripts"), 0o755)
	os.WriteFile(filepath.Join(root, "Source", "scripts", "A.psc"), nil, 0o644)
	abs, spelled, ok := Resolve(root, `source\SCRIPTS/a.psc`)
	if !ok || abs != filepath.Join(root, "Source", "scripts", "A.psc") || spelled != "Source/scripts/A.psc" {
		t.Errorf("Resolve = %q, %q, %v", abs, spelled, ok)
	}
	if _, _, ok := Resolve(root, "Source/missing"); ok {
		t.Error("resolved a path that isn't there")
	}
}

// The real lists, when their snapshots are here (PLAN.md §10).
func TestRealSnapshots(t *testing.T) {
	for _, snap := range testcorpus.Snapshots(t) {
		t.Run(snap.Name, func(t *testing.T) {
			src, err := Check(snap.Dir, "")
			if err != nil {
				t.Fatal(err)
			}
			l, err := Read(src)
			if err != nil {
				t.Fatal(err)
			}
			if src.Game != "Skyrim VR" || src.Profile == "" {
				t.Errorf("source = %+v", src)
			}
			// Every mod folder in the snapshot is listed, and listed enabled,
			// except the ones the PowerShell script copied files from without
			// listing (those with only an MCM Helper settings.ini).
			entries, _ := os.ReadDir(filepath.Join(snap.Dir, "mods"))
			listed := 0
			for _, e := range entries {
				m, ok := l.Mod(e.Name())
				switch {
				case ok && m.Enabled && m.Dir != "":
					listed++
				case !ok:
					t.Logf("mod folder %q isn't in modlist.txt", e.Name())
				default:
					t.Errorf("mod %q: %+v", e.Name(), m)
				}
			}
			if listed < len(entries)-5 || listed == 0 {
				t.Errorf("%d of %d mod folders listed", listed, len(entries))
			}
			if l.DataDir != "" {
				t.Errorf("a snapshot carries no game folder, but DataDir = %q", l.DataDir)
			}
			if l.Plugins() == nil {
				t.Error("the snapshot's plugins.txt wasn't read")
			}
			l.Index(MCMScope)
			if n := len(l.Files("McmRecorder")); n == 0 {
				t.Error("no recordings found")
			}
			for _, f := range l.Files("") {
				if !IsPlugin(f.Path) {
					continue
				}
				if !l.PluginActive(filepath.Base(f.Path)) {
					t.Logf("inactive plugin in the list: %s (%s)", f.Path, f.Mod)
				}
			}
			if !strings.Contains(strings.ToLower(snap.Name), "mgo") {
				return
			}
			// Facts about MGO 4 RC 4.5 (PLAN.md §2.3).
			if f, ok := l.Open("Scripts/Dirty_Quest_MCM.pex"); !ok || f.Mod != "Dirt and Blood - Dynamic Visual Effects" {
				t.Errorf("Dirt & Blood's script: %+v, %v", f, ok)
			}
			if f, ok := l.Open("source/scripts/dirty_quest_mcm.psc"); !ok || f.Path != "Source/scripts/Dirty_Quest_MCM.psc" {
				t.Errorf("Dirt & Blood's source: %+v, %v", f, ok)
			}
			if f, ok := l.Open("Experience.esl"); !ok || f.Mod != "Experience - MCM" || !l.PluginActive("Experience.esl") {
				t.Errorf("Experience's plugin: %+v, %v", f, ok)
			}
			if f, ok := l.Open("MCM/Settings/DismemberingFramework.ini"); !ok || f.Mod != "overwrite" {
				t.Errorf("overwrite's settings: %+v, %v", f, ok)
			}
			if f, ok := l.Open(`McmRecorder\DirtAndBlood\0001_Dirt & Blood.json`); !ok || f.Mod != "[NoDelete] Davey's MCM Recordings" {
				t.Errorf("the recording in a bracketed folder: %+v, %v", f, ok)
			}
		})
	}
}
