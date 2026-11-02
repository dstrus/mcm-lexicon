package bsa_test

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"lexicon/internal/bsa"
	"lexicon/internal/testcorpus"
)

func read(t *testing.T, b string) *bsa.Archive {
	t.Helper()
	a, err := bsa.Read(strings.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func contents(t *testing.T, a *bsa.Archive, b string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, f := range a.Files {
		got, err := a.ReadFrom(strings.NewReader(b), f)
		if err != nil {
			t.Fatalf("%s: %v", f.Path, err)
		}
		out[f.Path] = string(got)
	}
	return out
}

func TestBothVersionsEveryWayOfStoring(t *testing.T) {
	files := map[string]string{
		"Scripts/Alpha_MCM.pex":                               testcorpus.Compiled("Alpha_MCM", "SKI_ConfigBase"),
		"scripts/source/alpha_mcm.psc":                        "Scriptname Alpha_MCM extends SKI_ConfigBase\n",
		"Interface/Translations/Alpha_ENGLISH.txt":            "\ufeff$Alpha\tAlpha\n",
		"meshes/alpha/" + strings.Repeat("long", 20) + ".nif": strings.Repeat("x", 70000),
	}
	for _, version := range []int{104, 105} {
		for _, compressed := range []bool{false, true} {
			for _, embed := range []bool{false, true} {
				t.Run(fmt.Sprintf("v%d compressed=%v embed=%v", version, compressed, embed), func(t *testing.T) {
					b := testcorpus.Archive(testcorpus.ArchiveOptions{
						Version: version, Compressed: compressed, EmbedNames: embed,
						Flip: map[string]bool{"scripts/alpha_mcm.pex": true},
					}, files)
					a := read(t, b)
					if int(a.Version) != version || len(a.Files) != len(files) {
						t.Fatalf("version %d with %d files", a.Version, len(a.Files))
					}
					got := contents(t, a, b)
					for p, want := range files {
						if got[strings.ToLower(p)] != want {
							t.Errorf("%s: read %d bytes, want %d", p, len(got[strings.ToLower(p)]), len(want))
						}
					}
					for _, f := range a.Files {
						if wantCompressed := compressed != (f.Path == "scripts/alpha_mcm.pex"); f.Compressed != wantCompressed {
							t.Errorf("%s: compressed %v", f.Path, f.Compressed)
						}
					}
				})
			}
		}
	}
}

// Frames written by the lz4 tool, with and without block checksums, content
// size and a content checksum, and with matches that overlap what they copy.
const (
	lz4WithChecksums = "BCJNGHxAUHgAAAAAAACS7gAAAPE4U2NyaXB0bmFtZSBTYW1wbGVfTUNNIGV4dGVuZHMgU0tJX0NvbmZpZ0Jhc2UgOyBsaW5lIDAgb2YgYSByZXBldGl0aXZlIHNGAB8KTQAhHzFNADkfMk0AOR8zTQA5HzRNADkfNU0AOR82TQA5DxsC/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////19QcmlwdAp9sZveAAAAAHe0HHY="
	lz4Plain         = "BCJNGGRAp+4AAADxOFNjcmlwdG5hbWUgU2FtcGxlX01DTSBleHRlbmRzIFNLSV9Db25maWdCYXNlIDsgbGluZSAwIG9mIGEgcmVwZXRpdGl2ZSBzRgAfCk0AIR8xTQA5HzJNADkfM00AOR80TQA5HzVNADkfNk0AOA8bAv////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////9gUHJpcHQKAAAAAHe0HHY="
)

func lz4Input() string {
	var b strings.Builder
	for i := range 400 {
		fmt.Fprintf(&b, "Scriptname Sample_MCM extends SKI_ConfigBase ; line %d of a repetitive script\n", i%7)
	}
	return b.String()
}

func TestLZ4FromTheTool(t *testing.T) {
	want := lz4Input()
	for name, frame := range map[string]string{"checksums": lz4WithChecksums, "plain": lz4Plain} {
		raw, err := base64.StdEncoding.DecodeString(frame)
		if err != nil {
			t.Fatal(err)
		}
		got, err := bsa.LZ4Frame(raw, len(want))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if string(got) != want {
			t.Errorf("%s: unpacked %d bytes that differ from the %d written", name, len(got), len(want))
		}
		if _, err := bsa.LZ4Frame(raw, len(want)-1); err == nil {
			t.Errorf("%s: unpacking past the file's size wasn't refused", name)
		}
		for cut := 1; cut < len(raw); cut += 13 {
			bsa.LZ4Frame(raw[:cut], len(want)) // must not panic
		}
	}
}

func TestRefusals(t *testing.T) {
	good := testcorpus.Archive(testcorpus.ArchiveOptions{}, map[string]string{"scripts/a.pex": "x"})
	for name, b := range map[string]string{
		"empty":       "",
		"not BSA":     "BTDX" + good[4:],
		"Oblivion's":  good[:4] + "\x67\x00\x00\x00" + good[8:],
		"no names":    good[:12] + "\x00\x00\x00\x00" + good[16:],
		"huge counts": good[:20] + "\xff\xff\xff\x7f" + good[24:],
	} {
		if _, err := bsa.Read(strings.NewReader(b), int64(len(b))); err == nil {
			t.Errorf("%s: read without an error", name)
		}
	}
	for cut := 0; cut < len(good); cut++ {
		b := good[:cut]
		if a, err := bsa.Read(strings.NewReader(b), int64(len(b))); err == nil {
			for _, f := range a.Files {
				a.ReadFrom(strings.NewReader(b), f) // must not panic
			}
		}
	}
}
