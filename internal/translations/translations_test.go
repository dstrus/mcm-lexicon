package translations

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"

	"lexicon/internal/testcorpus"
)

// utf16LE is text as the game's translation files hold it.
func utf16LE(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, 2+2*len(u))
	b[0], b[1] = 0xFF, 0xFE
	for i, c := range u {
		binary.LittleEndian.PutUint16(b[2+2*i:], c)
	}
	return b
}

func TestNameOf(t *testing.T) {
	for file, want := range map[string][2]string{
		"Interface/Translations/Experience_ENGLISH.txt": {"Experience", "ENGLISH"},
		`Interface\Translations\skyui_se_english.txt`:   {"skyui_se", "ENGLISH"},
		"Azurite Weathers - MCM_english.txt":            {"Azurite Weathers - MCM", "ENGLISH"},
		"Spell Organizer_german.TXT":                    {"Spell Organizer", "GERMAN"},
	} {
		base, lang, ok := NameOf(file)
		if !ok || base != want[0] || lang != want[1] {
			t.Errorf("NameOf(%q) = %q, %q, %v", file, base, lang, ok)
		}
	}
	for _, file := range []string{"readme.txt", "_ENGLISH.txt", "Experience_.txt", "Experience_ENGLISH.json"} {
		if _, _, ok := NameOf(file); ok {
			t.Errorf("NameOf(%q) accepted it", file)
		}
	}
}

func TestText(t *testing.T) {
	tr := New()
	tr.Add("Experience", utf16LE("$EXP_ShowMessages\tShow Messages\r\n$EXP_General\tGeneral\r\nnot a key\tignored\r\n$Shared\tFrom Experience\r\n"))
	tr.Add("skyui_se", utf16LE("$Group {}\tGroup {}\n$SKI_INFO1{}\tDefault: {}\n$Medium\tMedium\n$Shared\tFrom SkyUI\n"))
	for label, want := range map[string]string{
		"$EXP_ShowMessages":   "Show Messages",
		"$exp_showmessages":   "Show Messages", // keys aren't case-sensitive
		"Plain label":         "Plain label",
		"$Missing":            "$Missing",
		"$Group {3}":          "Group 3",
		"$SKI_INFO1{$Medium}": "Default: Medium",
		"$Shared":             "From Experience", // the first file read, with no preference
	} {
		if got := tr.Text(label); got != want {
			t.Errorf("Text(%q) = %q, want %q", label, got, want)
		}
	}
	if got := tr.Text("$Shared", "SKYUI_SE"); got != "From SkyUI" {
		t.Errorf("preferring skyui_se: %q", got)
	}
}

func TestDecode(t *testing.T) {
	be := []byte{0xFE, 0xFF, 0, '$', 0, 'A'}
	for name, b := range map[string][]byte{
		"utf-16 le": utf16LE("$A"),
		"utf-16 be": be,
		"utf-8 bom": append([]byte{0xEF, 0xBB, 0xBF}, "$A"...),
		"utf-8":     []byte("$A"),
	} {
		if got := Decode(b); got != "$A" {
			t.Errorf("%s: %q", name, got)
		}
	}
}

// Experience's translation file holds what PLAN.md §2.4 says it does.
func TestRealSnapshots(t *testing.T) {
	for _, snap := range testcorpus.Snapshots(t) {
		path := filepath.Join(snap.Dir, "mods", "Experience - MCM", "Interface", "Translations", "Experience_ENGLISH.txt")
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		tr := New()
		tr.Add("Experience", b)
		if got := tr.Text("$EXP_ShowMessages"); got != "Show Messages" {
			t.Errorf("%s: $EXP_ShowMessages = %q", snap.Name, got)
		}
	}
}
