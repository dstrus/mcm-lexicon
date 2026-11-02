package textfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAtomicReplacesTheFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.yaml")
	os.WriteFile(p, []byte("old"), 0o600)
	if err := WriteAtomic(p, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	info, _ := os.Stat(p)
	if string(b) != "new" || info.Mode().Perm() != 0o600 {
		t.Errorf("file = %q, mode %v", b, info.Mode().Perm())
	}
	entries, _ := os.ReadDir(filepath.Dir(p))
	if len(entries) != 1 {
		t.Errorf("a temporary file was left behind: %v", entries)
	}
}

func TestBackupsWithinOneMillisecondAreKept(t *testing.T) {
	base := filepath.Join(t.TempDir(), "[NoDelete] x", "a.yaml")
	var names []string
	for i := 0; i < 5; i++ {
		name, err := Backup(base, []byte(fmt.Sprint(i)), 0o600)
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	for i, n := range names {
		if b, _ := os.ReadFile(n); string(b) != fmt.Sprint(i) {
			t.Errorf("backup %d (%s) holds %q", i, filepath.Base(n), b)
		}
		if !strings.HasPrefix(filepath.Base(n), "a.yaml.") || !strings.HasSuffix(n, ".bak") {
			t.Errorf("backup named %s", n)
		}
	}
	if info, _ := os.Stat(names[0]); info.Mode().Perm() != 0o600 {
		t.Errorf("backup permissions = %v", info.Mode().Perm())
	}
}

func TestMatchLineEndings(t *testing.T) {
	if got := MatchLineEndings("a\nb\r\nc", "x\r\ny"); got != "a\r\nb\r\nc" {
		t.Errorf("CRLF file: %q", got)
	}
	if got := MatchLineEndings("a\r\nb", "x\ny"); got != "a\nb" {
		t.Errorf("LF file: %q", got)
	}
	if got := MatchLineEndings("a\nb", ""); got != "a\nb" {
		t.Errorf("empty file: %q", got)
	}
}

func TestRemoveTakesEmptyFoldersWithIt(t *testing.T) {
	root := t.TempDir()
	lone := filepath.Join(root, "voice_effects", "ghost.yaml")
	kept := filepath.Join(root, "triggers", "a.yaml")
	for _, p := range []string{lone, kept, filepath.Join(root, "triggers", "b.yaml")} {
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte("x"), 0o644)
	}
	if err := Remove(lone, root); err != nil {
		t.Fatal(err)
	}
	if err := Remove(kept, root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(lone)); !os.IsNotExist(err) {
		t.Error("the emptied folder is still there")
	}
	if _, err := os.Stat(filepath.Join(root, "triggers", "b.yaml")); err != nil {
		t.Error("a folder that still holds a file was removed")
	}
	if _, err := os.Stat(root); err != nil {
		t.Error("the root was removed")
	}
}
