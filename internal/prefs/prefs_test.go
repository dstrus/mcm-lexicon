package prefs

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestChangesAreRemembered(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MCM Lexicon", "settings.json")
	Load(path).Update(func(v *Values) { v.ListDir = "/games/mgo"; v.Profile = "VR" })
	if got := Load(path).Get(); got.ListDir != "/games/mgo" || got.Profile != "VR" {
		t.Errorf("remembered = %+v", got)
	}
}

func TestUnreadableFileRemembersNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte("not json"), 0o644)
	if got := Load(path).Get(); !reflect.DeepEqual(got, Values{}) {
		t.Errorf("values = %+v, want none", got)
	}
}

func TestWroteToIsKeptPerList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	Load(path).Update(func(v *Values) { v.WroteTo = map[string][]string{"/games/mgo": {"MCM Lexicon Recordings"}} })
	if got := Load(path).Get().WroteTo["/games/mgo"]; len(got) != 1 || got[0] != "MCM Lexicon Recordings" {
		t.Errorf("wroteTo = %q", got)
	}
}

func TestThemeOf(t *testing.T) {
	for saved, want := range map[string]string{"": "system", "dark": "dark", "light": "light", "mgo": "system"} {
		if got := (Values{Theme: saved}).ThemeOf(); got != want {
			t.Errorf("ThemeOf(%q) = %q, want %q", saved, got, want)
		}
	}
}

func TestUnknownKeysAreKeptAndClearedOnesDropped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte(`{"listDir": "/old", "catalogFile": "/c.json", "showAll": true}`), 0o644)
	Load(path).Update(func(v *Values) { v.Theme = "dark"; v.CatalogFile = "" })
	b, _ := os.ReadFile(path)
	for _, want := range []string{`"showAll": true`, `"theme": "dark"`, `"listDir": "/old"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("settings.json lacks %s: %s", want, b)
		}
	}
	if strings.Contains(string(b), "catalogFile") {
		t.Errorf("a cleared value was kept: %s", b)
	}
}
