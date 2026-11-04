// Package prefs remembers MCM Lexicon's own settings between runs, such as
// the mod list chosen in the browser, so it doesn't have to be chosen again.
// The game's and the mods' settings are a different thing entirely.
package prefs

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Values struct {
	ListDir     string `json:"listDir,omitempty"`     // the mod list: an MO2 instance or a Data folder
	Profile     string `json:"profile,omitempty"`     // the MO2 profile read, when chosen
	CatalogFile string `json:"catalogFile,omitempty"` // a catalog file opened instead of a list
	Theme       string `json:"theme,omitempty"`       // see Themes; "" follows the system
	// The recording new steps go to, by name, for the list it belongs to.
	OpenRecording string `json:"openRecording,omitempty"`
	// The author new recordings are written with.
	Author string `json:"author,omitempty"`
	// Where new recordings are saved with a catalog file open: a folder's
	// path.
	SaveTarget string `json:"saveTarget,omitempty"`
	// The mod folder recordings are saved to in an MO2 instance: its name,
	// and whether NoDeletePrefix goes before it (nil means it does).
	SaveName string `json:"saveName,omitempty"`
	NoDelete *bool  `json:"noDelete,omitempty"`
	// The folders MCM Lexicon has saved recordings into, by list, so it
	// deletes only recordings in those, never a mod list's own.
	WroteTo map[string][]string `json:"wroteTo,omitempty"`
}

// keys are the names Values writes, so an omitted value is cleared from
// the file rather than kept from last time.
var keys = []string{"listDir", "profile", "catalogFile", "theme", "openRecording", "author", "saveTarget", "saveName", "noDelete", "wroteTo"}

// Themes are the themes MCM Lexicon has. "system" follows the operating
// system's dark or light setting.
var Themes = []string{"system", "dark", "light"}

// ThemeOf is v's theme, or "system" when none (or an unknown one) is saved.
func (v Values) ThemeOf() string {
	for _, t := range Themes {
		if v.Theme == t {
			return t
		}
	}
	return "system"
}

// DefaultSaveName is the save folder's name until the player names it.
const DefaultSaveName = "MCM Lexicon Recordings"

// NoDeletePrefix keeps a mod folder through a Wabbajack update, which
// removes the folders it didn't install unless their names say to keep them.
const NoDeletePrefix = "[NoDelete] "

// SaveNameOf is the save folder's name as the player gave it, without the
// prefix, or DefaultSaveName when none is saved.
func (v Values) SaveNameOf() string {
	if n := strings.TrimSpace(v.SaveName); n != "" {
		return n
	}
	return DefaultSaveName
}

// NoDeleteOf is whether the save folder's name starts with NoDeletePrefix:
// it does unless the player turned it off.
func (v Values) NoDeleteOf() bool {
	return v.NoDelete == nil || *v.NoDelete
}

// SaveMod is the mod folder recordings are saved to in an MO2 instance.
func (v Values) SaveMod() string {
	if v.NoDeleteOf() {
		return NoDeletePrefix + v.SaveNameOf()
	}
	return v.SaveNameOf()
}

// Prefs holds the remembered values and saves every change to its file.
type Prefs struct {
	mu   sync.Mutex
	path string // "" keeps them in memory only
	v    Values
}

// merged is the file's contents with the current values written over it,
// so keys MCM Lexicon doesn't know (from another version, say) are kept.
// Callers hold p.mu.
func (p *Prefs) merged() map[string]json.RawMessage {
	all := map[string]json.RawMessage{}
	if b, err := os.ReadFile(p.path); err == nil {
		json.Unmarshal(b, &all)
	}
	var mine map[string]json.RawMessage
	b, _ := json.Marshal(p.v)
	json.Unmarshal(b, &mine)
	for _, key := range keys {
		delete(all, key) // an omitted value was cleared
	}
	for k, v := range mine {
		all[k] = v
	}
	return all
}

// Path is where the preferences are kept, in the OS's usual place for
// program settings, or "" if it doesn't have one.
func Path() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "MCM Lexicon", "settings.json")
}

// Beside is the path of another of MCM Lexicon's files, kept next to the
// settings; "" when the settings are kept in memory only.
func (p *Prefs) Beside(name string) string {
	if p.path == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(p.path), name)
}

// Load reads the preferences at path. A missing or unreadable file means
// nothing is remembered yet.
func Load(path string) *Prefs {
	p := &Prefs{path: path}
	if path != "" {
		if b, err := os.ReadFile(path); err == nil {
			json.Unmarshal(b, &p.v)
		}
	}
	return p
}

func (p *Prefs) Get() Values {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.v
}

// Update changes the values and saves them. Failing to save is logged
// rather than returned: it only means the choice won't be remembered.
func (p *Prefs) Update(change func(*Values)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	change(&p.v)
	if p.path == "" {
		return
	}
	b, _ := json.MarshalIndent(p.merged(), "", "  ")
	err := os.MkdirAll(filepath.Dir(p.path), 0o755)
	if err == nil {
		err = os.WriteFile(p.path, b, 0o644)
	}
	if err != nil {
		log.Printf("Could not save settings for next time: %v", err)
	}
}
