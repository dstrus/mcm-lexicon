package recordings

import (
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"lexicon/internal/modlist"
	"lexicon/internal/testcorpus"
)

func TestPlanWritesWhatMCMRecorderWrites(t *testing.T) {
	d := Draft{Name: "Mine/New", Title: "My Settings", Author: "Me", Autorun: true, Steps: []Step{
		{Mod: "Be Seated", Option: "Enable / Disable Sitting Prompts", Kind: KindToggle, Value: "OFF"},
		{Mod: "Dirt & Blood", Page: "Settings", Option: "Dirt on Player", Kind: KindChoose, Value: "Deactivated"},
		{Mod: "Dirt & Blood", Page: "Settings", Option: "Restart <Player> Effects", Kind: KindClick, Side: "left"},
		{Mod: "dirt & blood", Page: "Settings", Option: "*Bandits*", Kind: KindToggle, Value: "on", Index: 2},
		{Mod: "Sofia", Page: "General", Option: "Sofia Health: ", Kind: KindSlider, Value: "25"},
		{Mod: "Sofia", Page: "General", Option: "Dodge", Kind: KindShortcut, Value: "48"},
		{Mod: "Mod: Two", Option: "Tint", Kind: KindColor, Value: "0xFF8000"},
		{Mod: "Mod: Two", Option: "Name", Kind: KindText, Value: "Lydia"},
	}}
	files, err := Plan(&d)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range files {
		names = append(names, f.Path)
	}
	want := "McmRecorder/Mine_New.json McmRecorder/Mine_New/0001_Be Seated.json McmRecorder/Mine_New/0002_Dirt & Blood.json McmRecorder/Mine_New/0003_Sofia.json McmRecorder/Mine_New/0004_Mod_ Two.json"
	if strings.Join(names, " ") != want {
		t.Errorf("files:\n%s\nwant:\n%s", strings.Join(names, " "), want)
	}
	if files[0].Text != "{\n  \"author\": \"Me\",\n  \"autorun\": \"true\",\n  \"name\": \"My Settings\",\n  \"version\": \"1.0.0\"\n}" {
		t.Errorf("info:\n%s", files[0].Text)
	}
	wantDirt := `[
  {
    "choose": "Deactivated",
    "mod": "Dirt & Blood",
    "option": "Dirt on Player",
    "page": "Settings"
  },
  {
    "click": "Restart <Player> Effects",
    "mod": "Dirt & Blood",
    "page": "Settings",
    "side": "left"
  },
  {
    "index": 2,
    "mod": "dirt & blood",
    "option": "*Bandits*",
    "page": "Settings",
    "toggle": "on"
  }
]`
	if files[2].Text != wantDirt {
		t.Errorf("Dirt & Blood:\n%s", files[2].Text)
	}
	for _, want := range []string{`"slider": 25.0`, `"option": "Sofia Health: "`, `"shortcut": 48`} {
		if !strings.Contains(files[3].Text, want) {
			t.Errorf("Sofia lacks %s:\n%s", want, files[3].Text)
		}
	}
	for _, want := range []string{`"color": 16744448`, `"text": "Lydia"`} {
		if !strings.Contains(files[4].Text, want) {
			t.Errorf("Mod: Two lacks %s:\n%s", want, files[4].Text)
		}
	}
	if got := []int{d.Steps[0].File, d.Steps[1].File, d.Steps[3].File, d.Steps[7].File}; !reflect.DeepEqual(got, []int{0, 1, 1, 3}) {
		t.Errorf("steps' files = %v", got)
	}
	for _, bad := range []Step{
		{Mod: "M", Option: "T", Kind: KindToggle, Value: "maybe"},
		{Mod: "M", Option: "S", Kind: KindSlider, Value: "lots"},
		{Mod: "M", Option: "K", Kind: KindShortcut, Value: "F1"},
		{Mod: "", Option: "T", Kind: KindToggle, Value: "on"},
		{Mod: "M", Option: "T"},
	} {
		if _, err := Plan(&Draft{Name: "X", Steps: []Step{bad}}); err == nil {
			t.Errorf("%+v was written", bad)
		}
	}
	if _, err := Plan(&Draft{Name: " / "}); err != nil && !strings.Contains(err.Error(), "name") {
		t.Error(err)
	}
}

// Every real recording reads back the same after it's written.
func TestRealRecordingsRoundTrip(t *testing.T) {
	for _, snap := range testcorpus.Snapshots(t) {
		t.Run(snap.Name, func(t *testing.T) {
			src, err := modlist.Check(snap.Dir, "")
			if err != nil {
				t.Fatal(err)
			}
			l, err := modlist.Read(src)
			if err != nil {
				t.Fatal(err)
			}
			l.Index(modlist.MCMScope)
			recs := Read(l)
			if len(recs) == 0 {
				t.Skip("no recordings")
			}
			for _, r := range recs {
				dir := t.TempDir()
				if _, err := Save(dir, DraftOf(r, ""), filepath.Join(dir, "backups"), time.Now()); err != nil {
					t.Errorf("%s: %v", r.Name, err)
					continue
				}
				inst := testcorpus.New(t, "again", "Default")
				inst.Mod("Written", true, nil)
				os.Rename(filepath.Join(dir, "McmRecorder"), filepath.Join(inst.Dir, "mods", "Written", "McmRecorder"))
				back := Read(list(t, inst))
				if len(back) != 1 {
					t.Fatalf("%s: read back %d recordings", r.Name, len(back))
				}
				b := back[0]
				if b.Name != r.Name || b.Author != r.Author || b.Autorun != r.Autorun || len(b.Problems) != 0 {
					t.Errorf("%s: info read back as %+v", r.Name, b)
				}
				if len(b.Steps) != len(r.Steps) {
					t.Fatalf("%s: %d steps read back, not %d", r.Name, len(b.Steps), len(r.Steps))
				}
				for i := range r.Steps {
					a, c := r.Steps[i], b.Steps[i]
					a.File, c.File = 0, 0
					if a.Kind == KindToggle {
						a.Value = strings.ToLower(a.Value)
					}
					if a != c {
						t.Errorf("%s step %d: %+v read back as %+v", r.Name, i+1, a, c)
					}
				}
			}
			t.Logf("%d recordings written and read back", len(recs))
		})
	}
}

func TestSaveBacksUpAndSwaps(t *testing.T) {
	dir := t.TempDir()
	backups := filepath.Join(t.TempDir(), "lexicon-backups")
	first := Draft{Name: "Mine", Steps: []Step{{Mod: "A", Option: "x", Kind: KindToggle, Value: "on"}, {Mod: "B", Option: "y", Kind: KindToggle, Value: "on"}}}
	at := time.Date(2026, 11, 2, 23, 4, 0, 0, time.Local)
	s, err := Save(dir, first, backups, at)
	if err != nil {
		t.Fatal(err)
	}
	if s.Backup != "" || len(s.Written) != 3 {
		t.Errorf("first save = %+v", s)
	}
	second := Draft{Name: "Mine", Steps: []Step{{Mod: "C", Option: "z", Kind: KindToggle, Value: "off"}}}
	s, err = Save(dir, second, backups, at.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(backups, "Mine", "20261102-230500.000"); s.Backup != want {
		t.Errorf("backup = %q, want %q", s.Backup, want)
	}
	if _, err := os.Stat(filepath.Join(s.Backup, "Mine", "0002_B.json")); err != nil {
		t.Errorf("the backup lacks the old step file: %v", err)
	}
	disk := Disk(dir, "Mine")
	if len(disk) != 2 || disk["McmRecorder/Mine/0001_C.json"] == "" {
		t.Errorf("on disk: %v", keys(disk))
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "McmRecorder"))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			t.Errorf("left %s behind", e.Name())
		}
	}
	b, err := Delete(dir, "Mine", backups, at.Add(2*time.Minute))
	if err != nil || b == "" {
		t.Fatalf("delete = %q, %v", b, err)
	}
	if len(Disk(dir, "Mine")) != 0 {
		t.Error("the recording is still there")
	}
	if _, err := Delete(dir, "Mine", backups, at.Add(3*time.Minute)); err == nil {
		t.Error("deleting a recording that isn't there didn't fail")
	}
}

func keys(m map[string]string) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// Each backup says what made it and where its files came from, and a
// backup put back gives exactly its files again, with names and spacing
// as they were, after backing up what it replaces.
func TestBackupsAndPut(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "My Recordings")
	backups := filepath.Join(t.TempDir(), "lexicon-backups")
	at := time.Date(2026, 11, 2, 23, 4, 0, 0, time.Local)
	// A recording as another tool wrote it: two files numbered 0001, its
	// own spacing, which Save wouldn't write.
	original := map[string]string{
		"McmRecorder/Mine.json":        `{"name": "Mine",   "autorun": "True"}`,
		"McmRecorder/Mine/0001_A.json": `[{"Mod": "A", "option": "x", "toggle": "on"}]`,
		"McmRecorder/Mine/0001_B.json": `[{"Mod": "B", "option": "y", "toggle": "off"}]`,
	}
	for p, text := range original {
		full := filepath.Join(dir, filepath.FromSlash(p))
		os.MkdirAll(filepath.Dir(full), 0o755)
		os.WriteFile(full, []byte(text), 0o644)
	}
	if _, err := Save(dir, Draft{Name: "Mine", Steps: []Step{{Mod: "C", Option: "z", Kind: KindToggle, Value: "off"}}}, backups, at); err != nil {
		t.Fatal(err)
	}
	list := Backups(backups, "Mine")
	if len(list) != 1 || list[0].Reason != BackupSave || list[0].Mod != "My Recordings" || list[0].Files != 3 || list[0].Steps != 2 || !list[0].At.Equal(at) {
		t.Fatalf("after a save: %+v", list)
	}
	files := backupFiles(list[0].dir, "Mine")
	s, err := Put(dir, "Mine", files, backups, at.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if disk := Disk(dir, "Mine"); !maps.Equal(disk, original) {
		t.Errorf("put back: %v", disk)
	}
	if s.Backup == "" || len(s.Written) != 3 {
		t.Errorf("put = %+v", s)
	}
	list = Backups(backups, "Mine")
	if len(list) != 2 || list[0].Reason != BackupRestore || list[1].Reason != BackupSave {
		t.Errorf("newest first, restore then save: %+v", list)
	}
	// Deleted, then put back from the backup the delete made.
	if _, err := Delete(dir, "Mine", backups, at.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	list = Backups(backups, "Mine")
	if list[0].Reason != BackupDelete {
		t.Fatalf("after a delete: %+v", list[0])
	}
	if _, err := Put(dir, "Mine", backupFiles(list[0].dir, "Mine"), backups, at.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if disk := Disk(dir, "Mine"); !maps.Equal(disk, original) {
		t.Errorf("put back after a delete: %v", disk)
	}
	// A backup without the info file leaves none.
	delete(files, "McmRecorder/Mine.json")
	if _, err := Put(dir, "Mine", files, backups, at.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, ok := Disk(dir, "Mine")["McmRecorder/Mine.json"]; ok {
		t.Error("the info file stayed")
	}
}
