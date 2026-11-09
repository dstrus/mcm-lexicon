// Package snapshot copies everything MCM Lexicon reads from a mod list into
// a trimmed Mod Organizer 2 instance, for developing and testing against
// real lists on any machine (PLAN.md §10). A snapshot holds other authors'
// files: it's for local use, never for the repository.
package snapshot

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"lexicon/internal/apperr"
	"lexicon/internal/bsa"
	"lexicon/internal/catalog"
	"lexicon/internal/modlist"
)

// SchemaVersion of snapshot.json; readers refuse a newer one.
const SchemaVersion = 1

type Options struct {
	Languages []string // translation files to include, as their names spell them: ENGLISH, GERMAN…
	Tool      string   // what made the snapshot, for SNAPSHOT.md
	Now       time.Time
}

// Inventory is snapshot.json: what the snapshot holds, for tests to read.
type Inventory struct {
	SchemaVersion int       `json:"schemaVersion"`
	Made          time.Time `json:"made"`
	Tool          string    `json:"tool"`
	List          ListInfo  `json:"list"`
	Mods          []ModInfo `json:"mods"`  // included, in priority order, highest first
	Files         []Entry   `json:"files"` // under mo2/, in the order they were copied
	Steps         []Step    `json:"steps"` // every recorded step, as acceptance cases
	Gaps          []string  `json:"gaps"`
}

type ListInfo struct {
	Name         string   `json:"name"` // the instance folder's name
	Path         string   `json:"path"`
	Profile      string   `json:"profile"`
	Game         string   `json:"game"`
	Languages    []string `json:"languages"`
	EnabledMods  int      `json:"enabledMods"`
	IncludedMods int      `json:"includedMods"`
}

type ModInfo struct {
	Name     string   `json:"name"`
	Priority int      `json:"priority"`
	Kinds    []string `json:"kinds"` // mcmhelper:N, settings:N, source:N, compiled:N, recordings, translations
}

type Entry struct {
	Path   string `json:"path"` // under mo2/, with forward slashes
	Mod    string `json:"mod"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// Step is one action of a recording, read leniently (keys in any case).
type Step struct {
	Recording string `json:"recording"`
	Mod       string `json:"mod"`
	Page      string `json:"page,omitempty"`
	Option    string `json:"option"`
	Index     int    `json:"index,omitempty"`
	Key       string `json:"key"`   // toggle, slider, choose, shortcut, color, text, click
	Value     any    `json:"value"` // as the file holds it: a string, or a number as its text
}

// Summary is what Write reports.
type Summary struct {
	Out   string
	Mods  int
	Files int
	Steps int
	Bytes int64
}

// Gaps a snapshot leaves, stated in SNAPSHOT.md.
var gaps = []string{
	"MCM scripts were found by following each compiled script's Extends chain to SKI_ConfigBase or MCM_ConfigBase, frameworks between included. A chain through a script that isn't in the list is lost.",
	"Plugins are copied whole, from mods with MCM content only. A mod whose MCM quest lives in a plugin from a different mod (a patch) needs that plugin added by hand.",
	"Loose files in the game's real Data folder (outside MO2) aren't included.",
	"Archives are included only when the game loads them (named after an active plugin) and they hold an MCM script or MCM Helper config; the INI files' archive lists aren't read.",
}

// Write makes a snapshot of list at out: a zip when out ends in .zip,
// otherwise a folder, which must not exist yet.
func Write(list *modlist.List, out string, o Options) (Summary, error) {
	if list.Source.Kind != modlist.MO2 {
		return Summary{}, apperr.New(apperr.Invalid, "Snapshots are made from Mod Organizer 2 instances; %s is a Data folder", list.Source.Path)
	}
	if len(o.Languages) == 0 {
		o.Languages = []string{"ENGLISH"}
	}
	if o.Now.IsZero() {
		o.Now = time.Now()
	}
	abs, err := filepath.Abs(out)
	if err != nil {
		return Summary{}, err
	}
	toZip := strings.EqualFold(filepath.Ext(abs), ".zip")
	if !toZip {
		if _, err := os.Stat(abs); err == nil {
			return Summary{}, apperr.New(apperr.Invalid, "%s already exists; choose a new folder for the snapshot", abs)
		}
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return Summary{}, err
	}
	// Staged beside the output, so a folder is a rename away.
	stage, err := os.MkdirTemp(filepath.Dir(abs), ".snapshot-*")
	if err != nil {
		return Summary{}, err
	}
	defer os.RemoveAll(stage)

	w := &writer{list: list, o: o, stage: stage, mo2: filepath.Join(stage, "mo2"), mcm: catalog.Scripts(list)}
	if err := w.run(); err != nil {
		return Summary{}, err
	}
	sum := Summary{Out: abs, Mods: len(w.inv.Mods), Files: len(w.inv.Files), Steps: len(w.inv.Steps)}
	for _, f := range w.inv.Files {
		sum.Bytes += f.Bytes
	}
	if toZip {
		return sum, zipDir(stage, abs)
	}
	return sum, os.Rename(stage, abs)
}

type writer struct {
	mcm    map[string]bool // the scripts that make MCMs, by lowercase name
	list   *modlist.List
	o      Options
	stage  string
	mo2    string
	inv    Inventory
	kinds  map[string][]string // by mod
	copied map[string]bool     // lowercase paths under mo2/
	packed []modlist.File      // the archives copied
}

func (w *writer) run() error {
	l := w.list
	w.kinds = map[string][]string{}
	w.copied = map[string]bool{}
	w.inv = Inventory{SchemaVersion: SchemaVersion, Made: w.o.Now, Tool: w.o.Tool, Gaps: gaps}
	w.inv.List = ListInfo{Name: filepath.Base(l.Source.Path), Path: l.Source.Path, Profile: l.Source.Profile, Game: l.Source.Game, Languages: w.o.Languages}
	langs := regexp.MustCompile(`(?i)_(` + strings.Join(escapeAll(w.o.Languages), "|") + `)\.txt$`)

	// Mods are visited from the highest priority down, as modlist.txt lists
	// them and as SNAPSHOT.md numbers them.
	var enabled []modlist.Mod
	for i := len(l.Mods) - 1; i >= 0; i-- {
		m := l.Mods[i]
		if m.Enabled && m.Layer == modlist.LayerMod && m.Dir != "" && !m.Unmanaged {
			enabled = append(enabled, m)
		}
	}
	w.inv.List.EnabledMods = len(enabled)
	for _, m := range enabled {
		if err := w.mod(m, langs); err != nil {
			return err
		}
	}
	// An archive loads only beside its plugin, which may be another mod's
	// (Sofia's SkyUI replacer holds only the archive).
	if err := w.archivePlugins(); err != nil {
		return err
	}
	// Translations often live in a mod without scripts (a separate "MCM"
	// or translation mod), so pick up translation files from every enabled
	// mod whose name matches an included plugin.
	plugins := map[string]bool{}
	for _, f := range w.inv.Files {
		if modlist.IsPlugin(f.Path) {
			base := filepath.Base(f.Path)
			plugins[strings.ToLower(strings.TrimSuffix(base, filepath.Ext(base)))] = true
		}
	}
	for _, m := range enabled {
		if _, included := w.kinds[m.Name]; included {
			continue
		}
		dir, rel, ok := modlist.Resolve(m.Dir, "Interface/Translations")
		if !ok {
			continue
		}
		for _, f := range listFiles(dir) {
			if !langs.MatchString(f) || !plugins[strings.ToLower(langs.ReplaceAllString(f, ""))] {
				continue
			}
			if err := w.copy(filepath.Join(dir, f), "mods/"+m.Name+"/"+rel+"/"+f, m.Name); err != nil {
				return err
			}
			w.addKind(m.Name, "translations")
		}
	}
	// Every INI file SKSE plugins keep settings in, from every mod, small
	// as they are: MCM Lexicon reads some (VRIK's), and the rest show which
	// it could. Never the plugins themselves, and a mod with nothing else
	// included brings none of its plugins along.
	for _, m := range enabled {
		dir, rel, ok := modlist.Resolve(m.Dir, "SKSE/Plugins")
		if !ok {
			continue
		}
		inis := 0
		for _, f := range listFiles(dir) {
			if !strings.EqualFold(filepath.Ext(f), ".ini") {
				continue
			}
			if err := w.copy(filepath.Join(dir, f), "mods/"+m.Name+"/"+rel+"/"+f, m.Name); err != nil {
				return err
			}
			inis++
		}
		if inis > 0 {
			w.addKind(m.Name, fmt.Sprintf("plugin-settings:%d", inis))
		}
	}
	// overwrite: MCM Helper's saved settings, any recordings made through
	// MO2, and SKSE plugins' INI files the game wrote there.
	if ow, ok := l.Mod("overwrite"); ok {
		for _, sub := range []string{"MCM/Settings", "McmRecorder"} {
			dir, rel, ok := modlist.Resolve(ow.Dir, sub)
			if !ok {
				continue
			}
			if err := w.copyTree(dir, "overwrite/"+rel, "overwrite"); err != nil {
				return err
			}
		}
		if dir, rel, ok := modlist.Resolve(ow.Dir, "SKSE/Plugins"); ok {
			for _, f := range listFiles(dir) {
				if strings.EqualFold(filepath.Ext(f), ".ini") {
					if err := w.copy(filepath.Join(dir, f), "overwrite/"+rel+"/"+f, "overwrite"); err != nil {
						return err
					}
				}
			}
		}
	}
	for _, m := range enabled {
		if kinds, ok := w.kinds[m.Name]; ok {
			w.inv.Mods = append(w.inv.Mods, ModInfo{Name: m.Name, Priority: m.Priority, Kinds: kinds})
		}
	}
	w.inv.List.IncludedMods = len(w.inv.Mods)
	if err := w.profile(); err != nil {
		return err
	}
	if err := w.steps(); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(w.stage, "SNAPSHOT.md"), []byte(w.markdown()), 0o644); err != nil {
		return err
	}
	b, err := json.MarshalIndent(w.inv, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(w.stage, "snapshot.json"), append(b, '\n'), 0o644)
}

// mod copies what one mod holds of MCM content, if anything.
func (w *writer) mod(m modlist.Mod, langs *regexp.Regexp) error {
	prefix := "mods/" + m.Name + "/"
	var source, compiled int
	if scripts, rel, ok := modlist.Resolve(m.Dir, "Scripts"); ok {
		for _, f := range listFiles(scripts) {
			base := strings.TrimSuffix(f, filepath.Ext(f))
			if !strings.EqualFold(filepath.Ext(f), ".pex") || !w.mcm[strings.ToLower(base)] {
				continue
			}
			if err := w.copy(filepath.Join(scripts, f), prefix+rel+"/"+f, m.Name); err != nil {
				return err
			}
			if psc, found := findFile(m.Dir, base+".psc"); found {
				inside, _ := filepath.Rel(m.Dir, psc)
				if err := w.copy(psc, prefix+filepath.ToSlash(inside), m.Name); err != nil {
					return err
				}
				source++
			} else {
				compiled++
			}
		}
	}
	// Archives this mod brings that the game loads, holding MCM scripts or
	// MCM Helper configs: copied whole, with the loose sources of the
	// scripts inside.
	for _, af := range w.list.Archives() {
		if af.Mod != m.Name {
			continue
		}
		a, err := bsa.Open(af.Abs)
		if err != nil {
			continue // the scan lists it as unreadable
		}
		packed := 0
		for _, e := range a.Files {
			dir, name := path.Split(e.Path)
			base := strings.TrimSuffix(name, path.Ext(name))
			switch {
			case dir == "scripts/" && path.Ext(name) == ".pex" && w.mcm[base]:
				packed++
				if psc, found := findFile(m.Dir, base+".psc"); found {
					inside, _ := filepath.Rel(m.Dir, psc)
					if err := w.copy(psc, prefix+filepath.ToSlash(inside), m.Name); err != nil {
						return err
					}
					source++
				}
			case strings.HasPrefix(e.Path, "mcm/config/") && name == "config.json":
				packed++
			}
		}
		if packed > 0 {
			if err := w.copy(af.Abs, prefix+af.Path, m.Name); err != nil {
				return err
			}
			w.addKind(m.Name, fmt.Sprintf("archive:%d", packed))
			w.packed = append(w.packed, af)
		}
	}
	if source > 0 {
		w.addKind(m.Name, fmt.Sprintf("source:%d", source))
	}
	if compiled > 0 {
		w.addKind(m.Name, fmt.Sprintf("compiled:%d", compiled))
	}
	for _, sub := range []string{"MCM", "McmRecorder"} {
		dir, rel, ok := modlist.Resolve(m.Dir, sub)
		if !ok || !modlist.IsDir(dir) {
			continue
		}
		files := allFiles(dir)
		if len(files) == 0 {
			continue
		}
		for _, f := range files {
			if err := w.copy(f, prefix+rel+"/"+filepath.ToSlash(mustRel(dir, f)), m.Name); err != nil {
				return err
			}
		}
		if sub == "MCM" {
			// MCM Helper's configs; a mod with only settings files (its
			// own defaults for a menu defined elsewhere) counts too.
			configs, settings := 0, 0
			for _, f := range files {
				switch strings.ToLower(filepath.Base(f)) {
				case "config.json":
					configs++
				case "settings.ini":
					settings++
				}
			}
			if configs > 0 {
				w.addKind(m.Name, fmt.Sprintf("mcmhelper:%d", configs))
			} else if settings > 0 {
				w.addKind(m.Name, fmt.Sprintf("settings:%d", settings))
			} else {
				w.addKind(m.Name, "mcm")
			}
		} else {
			w.addKind(m.Name, "recordings")
		}
	}
	if _, included := w.kinds[m.Name]; !included {
		return nil
	}
	for _, f := range listFiles(m.Dir) {
		if modlist.IsPlugin(f) {
			if err := w.copy(filepath.Join(m.Dir, f), prefix+f, m.Name); err != nil {
				return err
			}
		}
	}
	if tr, rel, ok := modlist.Resolve(m.Dir, "Interface/Translations"); ok {
		for _, f := range listFiles(tr) {
			if langs.MatchString(f) {
				if err := w.copy(filepath.Join(tr, f), prefix+rel+"/"+f, m.Name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// archivePlugins copies the plugin each copied archive is named after, as
// the game sees it, when it isn't copied already.
func (w *writer) archivePlugins() error {
	for _, af := range w.packed {
		base := strings.TrimSuffix(af.Path, filepath.Ext(af.Path))
		if i := strings.LastIndex(strings.ToLower(base), " - textures"); i >= 0 && i == len(base)-len(" - textures") {
			base = base[:i]
		}
		for _, ext := range []string{".esp", ".esm", ".esl"} {
			p, ok := w.list.Open(base + ext)
			if !ok || !w.list.PluginActive(p.Path) {
				continue
			}
			dest := "mods/" + p.Mod + "/" + p.Path
			if !w.copied[strings.ToLower(dest)] {
				if err := w.copy(p.Abs, dest, p.Mod); err != nil {
					return err
				}
				w.addKind(p.Mod, "plugin")
			}
			break
		}
	}
	return nil
}

// profile writes the trimmed instance files: included mods keep their real
// order, plugins.txt keeps its real marks for the included plugins, and
// ModOrganizer.ini names the game and the profile.
func (w *writer) profile() error {
	l := w.list
	profile := filepath.Join(l.Source.Path, "profiles", l.Source.Profile)
	included := map[string]bool{}
	for _, m := range w.inv.Mods {
		included[m.Name] = true
	}
	lines, err := readLines(filepath.Join(profile, "modlist.txt"))
	if err != nil {
		return err
	}
	var kept []string
	for _, line := range lines {
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "*") || (len(line) > 1 && included[line[1:]]) {
			kept = append(kept, line)
		}
	}
	pdir := filepath.Join(w.mo2, "profiles", l.Source.Profile)
	if err := os.MkdirAll(pdir, 0o755); err != nil {
		return err
	}
	if err := writeLines(filepath.Join(pdir, "modlist.txt"), kept); err != nil {
		return err
	}
	if l.Plugins() != nil {
		names := map[string]bool{}
		for _, f := range w.inv.Files {
			if modlist.IsPlugin(f.Path) {
				names[strings.ToLower(filepath.Base(f.Path))] = true
			}
		}
		lines, err := readLines(filepath.Join(profile, "plugins.txt"))
		if err != nil {
			return err
		}
		var kept []string
		for _, line := range lines {
			if strings.HasPrefix(line, "#") || names[strings.ToLower(strings.TrimPrefix(line, "*"))] {
				kept = append(kept, line)
			}
		}
		if err := writeLines(filepath.Join(pdir, "plugins.txt"), kept); err != nil {
			return err
		}
	}
	ini := "[General]\n"
	if l.Source.Game != "" {
		ini += "gameName=" + l.Source.Game + "\n"
	}
	ini += "selected_profile=@ByteArray(" + l.Source.Profile + ")\n"
	return os.WriteFile(filepath.Join(w.mo2, "ModOrganizer.ini"), []byte(ini), 0o644)
}

var valueKeys = []string{"toggle", "slider", "choose", "shortcut", "color", "text", "click"}

// steps reads every recording's step files for SNAPSHOT.md's acceptance
// list, as MCM Recorder would: keys in any case.
func (w *writer) steps() error {
	for _, f := range w.inv.Files {
		parts := strings.Split(f.Path, "/")
		n := len(parts)
		if n < 3 || !strings.EqualFold(parts[n-3], "McmRecorder") || !strings.EqualFold(filepath.Ext(parts[n-1]), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(w.mo2, filepath.FromSlash(f.Path)))
		if err != nil {
			return err
		}
		// Numbers stay as the file spells them (25.0, not 25), since that
		// spelling is what MCM Recorder writes and what a test compares.
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.UseNumber()
		var actions []map[string]any
		if dec.Decode(&actions) != nil {
			continue // not a step file
		}
		for _, a := range actions {
			props := map[string]any{}
			for k, v := range a {
				props[strings.ToLower(k)] = v
			}
			s := Step{Recording: parts[n-2]}
			s.Mod, _ = props["mod"].(string)
			s.Page, _ = props["page"].(string)
			s.Option, _ = props["option"].(string)
			if idx, ok := props["index"].(json.Number); ok {
				if n, err := idx.Int64(); err == nil {
					s.Index = int(n)
				}
			}
			for _, k := range valueKeys {
				if v, ok := props[k]; ok {
					s.Key, s.Value = k, v
					break
				}
			}
			w.inv.Steps = append(w.inv.Steps, s)
		}
	}
	return nil
}

func (w *writer) markdown() string {
	inv := w.inv
	var b strings.Builder
	fmt.Fprintf(&b, "# MCM Lexicon snapshot: %s\n\n", inv.List.Name)
	fmt.Fprintf(&b, "Made %s from `%s`, profile `%s`, with %s. Languages: %s.\n\n", inv.Made.Format("2006-01-02 15:04"), inv.List.Path, inv.List.Profile, inv.Tool, strings.Join(inv.List.Languages, ", "))
	var bytes int64
	for _, f := range inv.Files {
		bytes += f.Bytes
	}
	fmt.Fprintf(&b, "`mo2/` is a trimmed MO2 instance: %d of %d enabled mods, %d files, %.1f MB. It holds other authors' files: local testing only, never commit or share it.\n\n", inv.List.IncludedMods, inv.List.EnabledMods, len(inv.Files), float64(bytes)/(1024*1024))
	b.WriteString("## Mods included\n\n")
	b.WriteString("Kinds: `mcmhelper:N` config.json files, `settings:N` MCM Helper settings.ini files without a config, `source:N` MCM scripts with .psc, `compiled:N` MCM scripts with only .pex, `archive:N` MCM scripts and configs packed in an archive the game loads (copied whole), `plugin` the plugin such an archive is named after, from a mod with nothing else included, `recordings`, `translations` (only translation files for an included plugin), `plugin-settings:N` INI files SKSE plugins keep settings in, from any mod (its plugins aren't brought along).\n\n")
	b.WriteString("| Priority | Mod | Kinds |\n|---|---|---|\n")
	// Numbered from the top of modlist.txt, counting enabled mods: the
	// highest priority is 1.
	for _, m := range inv.Mods {
		fmt.Fprintf(&b, "| %d | %s | %s |\n", rank(w.list, m.Name), cell(m.Name), strings.Join(m.Kinds, ", "))
	}
	b.WriteString("\n## Recorded steps (acceptance cases)\n\n")
	if len(inv.Steps) == 0 {
		b.WriteString("No MCM Recorder recordings in this list.\n")
	} else {
		b.WriteString("Each step should resolve to an `exact` catalog option, or report \"mod not found\" when its MCM isn't in this list. Labels are exact, spaces included (shown in quotes).\n\n")
		b.WriteString("| Recording | Mod | Page | Option | Index | Value |\n|---|---|---|---|---|---|\n")
		for _, s := range inv.Steps {
			index := ""
			if s.Index > 0 {
				index = fmt.Sprint(s.Index)
			}
			value := ""
			if s.Key != "" {
				value = s.Key + " " + fmt.Sprint(s.Value)
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n", quoted(s.Recording), quoted(s.Mod), quoted(s.Page), quoted(s.Option), quoted(index), quoted(value))
		}
	}
	b.WriteString("\n## Known gaps\n\n")
	for _, g := range inv.Gaps {
		b.WriteString("- " + g + "\n")
	}
	return b.String()
}

// rank is a mod's place counting enabled mods from the top of modlist.txt,
// starting at 1.
func rank(l *modlist.List, name string) int {
	n := 0
	for i := len(l.Mods) - 1; i >= 0; i-- {
		m := l.Mods[i]
		if m.Enabled && m.Layer == modlist.LayerMod && !m.Unmanaged {
			n++
			if m.Name == name {
				return n
			}
		}
	}
	return n
}

func (w *writer) addKind(mod, kind string) {
	for _, k := range w.kinds[mod] {
		if k == kind {
			return
		}
	}
	w.kinds[mod] = append(w.kinds[mod], kind)
}

// copy puts the file at from under mo2/ at rel, recording it.
func (w *writer) copy(from, rel, mod string) error {
	if w.copied[strings.ToLower(rel)] {
		return nil
	}
	dest := filepath.Join(w.mo2, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(out, h), in)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	w.copied[strings.ToLower(rel)] = true
	w.inv.Files = append(w.inv.Files, Entry{Path: rel, Mod: mod, Bytes: n, SHA256: hex.EncodeToString(h.Sum(nil))})
	return nil
}

func (w *writer) copyTree(dir, rel, mod string) error {
	for _, f := range allFiles(dir) {
		if err := w.copy(f, rel+"/"+filepath.ToSlash(mustRel(dir, f)), mod); err != nil {
			return err
		}
	}
	return nil
}

// listFiles is the names of the files directly in dir, in name order.
func listFiles(dir string) []string {
	entries, _ := os.ReadDir(dir)
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// allFiles is every file under dir, in walk order.
func allFiles(dir string) []string {
	var out []string
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && !modlist.Clutter(d.Name()) {
			out = append(out, p)
		}
		return nil
	})
	return out
}

// findFile is the first file named name (in any case) under dir, in walk order.
func findFile(dir, name string) (string, bool) {
	found := ""
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.EqualFold(d.Name(), name) {
			found = p
			return fs.SkipAll
		}
		return nil
	})
	return found, found != ""
}

func mustRel(base, p string) string {
	rel, _ := filepath.Rel(base, p)
	return rel
}

func escapeAll(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = regexp.QuoteMeta(s)
	}
	return out
}

func cell(s string) string { return strings.ReplaceAll(s, "|", `\|`) }

// quoted is a value in quotes, so its spaces show; an empty cell stays empty.
func quoted(s string) string {
	if s == "" {
		return ""
	}
	return cell(`"` + s + `"`)
}

func readLines(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := strings.TrimPrefix(strings.ReplaceAll(string(b), "\r\n", "\n"), "\ufeff")
	return strings.Split(strings.TrimRight(text, "\n"), "\n"), nil
}

func writeLines(path string, lines []string) error {
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

// zipDir writes dir's contents to the zip at out, with forward slashes.
func zipDir(dir, out string) error {
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		h, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		h.Name = filepath.ToSlash(mustRel(dir, p))
		h.Method = zip.Deflate
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(w, in)
		return err
	})
	if cerr := zw.Close(); err == nil {
		err = cerr
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}
