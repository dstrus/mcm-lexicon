package modlist

import (
	"os"
	"path/filepath"
	"testing"

	"lexicon/internal/testcorpus"
)

func TestCheckReadsAnMO2Instance(t *testing.T) {
	// MO2 folder names often include brackets, which file patterns would misread.
	dir := testcorpus.MO2Instance(t, "[NoDelete] MGO 4", "Default", "Low Spec")
	src, err := Check("  "+dir+"  ", "")
	if err != nil {
		t.Fatal(err)
	}
	if src.Kind != MO2 || src.Path != dir || src.Game != "Skyrim VR" || src.Profile != "Default" {
		t.Errorf("Check = %+v", src)
	}
	if len(src.Profiles) != 2 || src.Profiles[0] != "Default" || src.Profiles[1] != "Low Spec" {
		t.Errorf("profiles = %q", src.Profiles)
	}
	if src, err := Check(dir, "low spec"); err != nil || src.Profile != "Low Spec" {
		t.Errorf("Check with a profile = %+v, %v", src, err)
	}
	if _, err := Check(dir, "High Spec"); err == nil {
		t.Error("a profile that isn't there was accepted")
	}
}

func TestCheckFallsBackWhenTheSelectedProfileIsGone(t *testing.T) {
	dir := testcorpus.MO2Instance(t, "mgo", "VR")
	os.WriteFile(filepath.Join(dir, "ModOrganizer.ini"), []byte("[General]\ngameName=Skyrim VR\nselected_profile=@ByteArray(Gone)\n"), 0o644)
	src, err := Check(dir, "")
	if err != nil || src.Profile != "VR" {
		t.Errorf("Check = %+v, %v", src, err)
	}
}

func TestCheckReadsADataFolder(t *testing.T) {
	game := filepath.Join(t.TempDir(), "Skyrim VR")
	data := filepath.Join(game, "Data")
	os.MkdirAll(data, 0o755)
	os.WriteFile(filepath.Join(data, "skyrim.esm"), nil, 0o644)
	for _, path := range []string{data, game} {
		src, err := Check(path, "")
		if err != nil || src.Kind != Data || src.Path != data || src.Profile != "" {
			t.Errorf("Check(%q) = %+v, %v", path, src, err)
		}
	}
	if _, err := Check(data, "Default"); err == nil {
		t.Error("a profile was accepted for a Data folder")
	}
}

func TestCheckRefusesOtherFolders(t *testing.T) {
	for _, path := range []string{"", t.TempDir(), filepath.Join(t.TempDir(), "missing")} {
		if _, err := Check(path, ""); err == nil {
			t.Errorf("Check(%q) accepted a folder that isn't a mod list", path)
		}
	}
	empty := filepath.Join(t.TempDir(), "mo2")
	os.MkdirAll(empty, 0o755)
	os.WriteFile(filepath.Join(empty, "ModOrganizer.ini"), []byte("[General]\n"), 0o644)
	if _, err := Check(empty, ""); err == nil {
		t.Error("an instance without profiles was accepted")
	}
}

func TestFindInLooksBesideTheProgram(t *testing.T) {
	mo2 := testcorpus.MO2Instance(t, "[NoDelete] MO2", "Default")
	if got := findIn([]string{t.TempDir(), mo2}); got != mo2 {
		t.Errorf("findIn = %q, want %q", got, mo2)
	}
	game := t.TempDir()
	os.MkdirAll(filepath.Join(game, "Data"), 0o755)
	os.WriteFile(filepath.Join(game, "Data", "Skyrim.esm"), nil, 0o644)
	if got := findIn([]string{game}); got != filepath.Join(game, "Data") {
		t.Errorf("findIn(game folder) = %q", got)
	}
	if got := findIn([]string{t.TempDir()}); got != "" {
		t.Errorf("findIn(nothing) = %q", got)
	}
}

func TestCheckCatalog(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "mgo.JSON")
	os.WriteFile(good, []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644)
	if got, err := CheckCatalog(good); err != nil || got != good {
		t.Errorf("CheckCatalog = %q, %v", got, err)
	}
	for _, path := range []string{"", dir, filepath.Join(dir, "notes.txt"), filepath.Join(dir, "missing.json")} {
		if _, err := CheckCatalog(path); err == nil {
			t.Errorf("CheckCatalog(%q) accepted it", path)
		}
	}
}
