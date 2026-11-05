package modlist

import (
	"os"
	"path/filepath"
	"testing"
)

// The picker marks an MO2 instance and a Data folder as mod lists, and a
// game folder as leading to its Data folder.
func TestBrowse(t *testing.T) {
	root := t.TempDir()
	write := func(rel string) {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, nil, 0o644)
	}
	write("MO2/ModOrganizer.ini")
	write("Game/data/Skyrim.esm")
	write("Plain/readme.txt")
	write("notes.txt")
	os.MkdirAll(filepath.Join(root, ".hidden"), 0o755)

	l, err := Browse(root, LookingList)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]Folder{}
	for _, f := range l.Folders {
		got[f.Name] = f
	}
	if len(l.Folders) != 3 || len(l.Files) != 1 || l.Files[0].Name != "notes.txt" || l.Files[0].Mark != MarkNone {
		t.Fatalf("listing: %+v", l)
	}
	if f := got["MO2"]; f.Mark != MarkHere || f.Kind != MO2 {
		t.Errorf("MO2: %+v", f)
	}
	if f := got["Game"]; f.Mark != MarkInside || f.Kind != Data || filepath.Base(f.Inside) != "data" {
		t.Errorf("Game: %+v", f)
	}
	if f := got["Plain"]; f.Mark != MarkNone {
		t.Errorf("Plain: %+v", f)
	}
	inside, err := Browse(got["Game"].Inside, LookingList)
	if err != nil || inside.Mark != MarkHere || inside.Kind != Data || inside.Parent == "" {
		t.Errorf("inside: %+v, %v", inside, err)
	}
	if _, err := Browse(filepath.Join(root, "missing"), LookingList); err == nil {
		t.Error("a missing folder listed")
	}
}

// Looking for a catalog file, the files that are one are marked, and
// folders aren't, since none is a catalog.
func TestBrowseCatalogs(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "mgo.json"), []byte("{\n  \"schemaVersion\": 1,\n  \"list\": {}\n}"), 0o644)
	os.WriteFile(filepath.Join(root, "other.json"), []byte(`{"name": "MGO"}`), 0o644)
	os.WriteFile(filepath.Join(root, "notes.txt"), []byte(`{"schemaVersion": 1}`), 0o644)
	os.MkdirAll(filepath.Join(root, "MO2"), 0o755)
	os.WriteFile(filepath.Join(root, "MO2", "ModOrganizer.ini"), nil, 0o644)
	l, err := Browse(root, LookingCatalog)
	if err != nil {
		t.Fatal(err)
	}
	marks := map[string]Mark{}
	for _, f := range l.Files {
		marks[f.Name] = f.Mark
	}
	if marks["mgo.json"] != MarkHere || marks["other.json"] != MarkNone || marks["notes.txt"] != MarkNone {
		t.Errorf("files: %v", marks)
	}
	if len(l.Folders) != 1 || l.Folders[0].Mark != MarkNone || l.Mark != MarkNone {
		t.Errorf("folders: %+v, own %q", l.Folders, l.Mark)
	}
}
