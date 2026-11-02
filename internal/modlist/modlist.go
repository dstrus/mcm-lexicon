// Package modlist finds and reads a mod list: a Mod Organizer 2 instance,
// or the game's Data folder (Vortex and manual installs). This is the part
// that says where the list is and which profile is in use; the mods, their
// order, and the files the game sees come with scanning.
package modlist

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"lexicon/internal/apperr"
)

// Kind is how a mod list is organized.
type Kind string

const (
	MO2  Kind = "mo2"  // a Mod Organizer 2 instance: ModOrganizer.ini, mods/, profiles/, overwrite/
	Data Kind = "data" // the game's Data folder, with every mod's files in it
)

// Source is a mod list as MCM Lexicon was pointed at it.
type Source struct {
	Kind     Kind     `json:"kind"`
	Path     string   `json:"path"`     // the MO2 instance, or the Data folder
	Game     string   `json:"game"`     // MO2's game name, such as "Skyrim VR"; "" when unknown
	Profile  string   `json:"profile"`  // the profile in use (MO2 only)
	Profiles []string `json:"profiles"` // every profile the instance has, in name order (MO2 only)
}

// Check returns the mod list at path, or an error saying why there isn't
// one. For an MO2 instance, profile names the profile to read, or "" for
// the one MO2 has selected. The game folder itself is accepted in place of
// its Data folder.
func Check(path, profile string) (Source, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Source{}, apperr.New(apperr.Invalid, "Enter the path of your mod list")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return Source{}, apperr.New(apperr.Invalid, "Invalid folder: %v", err)
	}
	if !IsDir(abs) {
		return Source{}, apperr.New(apperr.Invalid, "Folder not found: %s", abs)
	}
	if ini := Find1(abs, "ModOrganizer.ini"); ini != "" {
		return mo2(abs, ini, profile)
	}
	if profile != "" {
		return Source{}, apperr.New(apperr.Invalid, "%s isn't a Mod Organizer 2 instance, so it has no profiles", abs)
	}
	if Find1(abs, "Skyrim.esm") != "" {
		return Source{Kind: Data, Path: abs}, nil
	}
	if data := filepath.Join(abs, "Data"); Find1(data, "Skyrim.esm") != "" {
		return Source{Kind: Data, Path: data}, nil
	}
	return Source{}, apperr.New(apperr.Invalid, "%s isn't a Mod Organizer 2 instance (no ModOrganizer.ini) or a Skyrim Data folder (no Skyrim.esm)", abs)
}

// mo2 reads what an instance's ModOrganizer.ini and profiles folder say.
func mo2(dir, ini, profile string) (Source, error) {
	src := Source{Kind: MO2, Path: dir}
	settings := readINI(ini)
	src.Game = settings["gameName"]
	selected := settings["selected_profile"]
	entries, _ := os.ReadDir(filepath.Join(dir, "profiles"))
	for _, e := range entries {
		if e.IsDir() {
			src.Profiles = append(src.Profiles, e.Name())
		}
	}
	sort.Strings(src.Profiles)
	if len(src.Profiles) == 0 {
		return Source{}, apperr.New(apperr.Invalid, "%s has no profiles folder, so it isn't a Mod Organizer 2 instance MCM Lexicon can read", dir)
	}
	want := profile
	if want == "" {
		want = selected
	}
	for _, p := range src.Profiles {
		if strings.EqualFold(p, want) {
			src.Profile = p
			return src, nil
		}
	}
	if profile != "" {
		return Source{}, apperr.New(apperr.Invalid, "There's no profile %q in %s", profile, dir)
	}
	// MO2 names a profile that isn't there, or none: take the first.
	src.Profile = src.Profiles[0]
	return src, nil
}

// readINI reads the [General] section of ModOrganizer.ini: the keys that
// name the game and the selected profile. Values MO2 writes as
// @ByteArray(…) are unwrapped.
func readINI(path string) map[string]string {
	out := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	section := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(line[1 : len(line)-1])
			continue
		}
		if section != "general" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "@ByteArray(") && strings.HasSuffix(value, ")") {
			value = value[len("@ByteArray(") : len(value)-1]
		}
		out[strings.TrimSpace(key)] = value
	}
	return out
}

// Find looks for a mod list in the places MCM Lexicon is likely to be
// started from: next to the program, or the current directory. It returns
// "" if there isn't one.
func Find() string {
	var roots []string
	if exe, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Dir(exe))
	}
	if wd, err := os.Getwd(); err == nil {
		roots = append(roots, wd)
	}
	return findIn(roots)
}

func findIn(roots []string) string {
	for _, root := range roots {
		if Find1(root, "ModOrganizer.ini") != "" || Find1(root, "Skyrim.esm") != "" {
			return root
		}
		if data := filepath.Join(root, "Data"); Find1(data, "Skyrim.esm") != "" {
			return data
		}
	}
	return ""
}

// CheckCatalog returns the absolute path of the catalog file at path, or an
// error saying why it can't be opened.
func CheckCatalog(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", apperr.New(apperr.Invalid, "Enter the path of a catalog file")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", apperr.New(apperr.Invalid, "Invalid path: %v", err)
	}
	if !IsFile(abs) {
		return "", apperr.New(apperr.Invalid, "File not found: %s", abs)
	}
	if !strings.EqualFold(filepath.Ext(abs), ".json") {
		return "", apperr.New(apperr.Invalid, "%s isn't a catalog file; a catalog is a .json file MCM Lexicon wrote", abs)
	}
	return abs, nil
}

// Find1 is the path of the file named name in dir, compared without regard
// to case as Windows does, or "" if there isn't one. Unlike filepath.Glob
// it reads dir literally, so MO2 folder names such as "[NoDelete] …" aren't
// taken as patterns.
func Find1(dir, name string) string {
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(e.Name(), name) {
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
}

func IsDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func IsFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}
