// Package translations reads the game's translation files,
// Interface/Translations/<name>_<LANGUAGE>.txt, which turn the $KEY labels
// MCMs are written with into the text a player reads. The files are UTF-16
// LE with a byte order mark, one "$KEY<TAB>Text" per line.
package translations

import (
	"bytes"
	"encoding/binary"
	"path"
	"regexp"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// Table is every translation the game would load for one language, kept
// per file so a menu's own file can win when two files share a key.
type Table struct {
	files map[string]map[string]string // by lowercase base name (Experience for Experience_ENGLISH.txt), then by lowercase key
	order []string                     // base names in the order they were added
}

func New() *Table { return &Table{files: map[string]map[string]string{}} }

// NameOf splits a translation file's name into its base name and language,
// or reports ok=false if it isn't one.
func NameOf(file string) (base, language string, ok bool) {
	name := path.Base(strings.ReplaceAll(file, `\`, "/"))
	if !strings.EqualFold(path.Ext(name), ".txt") {
		return "", "", false
	}
	name = name[:len(name)-4]
	i := strings.LastIndex(name, "_")
	if i <= 0 || i == len(name)-1 {
		return "", "", false
	}
	return name[:i], strings.ToUpper(name[i+1:]), true
}

// Add reads one translation file's contents under its base name. A later
// file with the same base name replaces the earlier one, as the game's
// load order would. A \n in a value is a line break, as it is in a
// Papyrus string; authors write it for the info text's lines.
func (t *Table) Add(base string, data []byte) {
	key := strings.ToLower(base)
	if _, seen := t.files[key]; !seen {
		t.order = append(t.order, key)
	}
	entries := map[string]string{}
	for _, line := range strings.Split(Decode(data), "\n") {
		line = strings.TrimRight(line, "\r")
		k, v, ok := strings.Cut(line, "\t")
		if !ok || !strings.HasPrefix(k, "$") {
			continue
		}
		entries[strings.ToLower(k)] = strings.ReplaceAll(v, `\n`, "\n")
	}
	t.files[key] = entries
}

// Len is the number of files read.
func (t *Table) Len() int { return len(t.files) }

// Decode turns a translation file's bytes into text: UTF-16 LE or BE by its
// byte order mark, UTF-8 (with or without one) otherwise.
func Decode(b []byte) string {
	switch {
	case bytes.HasPrefix(b, []byte{0xFF, 0xFE}):
		return utf16Text(b[2:], binary.LittleEndian)
	case bytes.HasPrefix(b, []byte{0xFE, 0xFF}):
		return utf16Text(b[2:], binary.BigEndian)
	case bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}):
		return string(b[3:])
	case len(b) >= 2 && b[1] == 0 && !utf8.Valid(b):
		return utf16Text(b, binary.LittleEndian) // UTF-16 LE without its mark
	}
	return string(b)
}

func utf16Text(b []byte, order binary.ByteOrder) string {
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = order.Uint16(b[2*i:])
	}
	return string(utf16.Decode(u))
}

var argument = regexp.MustCompile(`\{[^{}]*\}`)

// Text is what label reads as: its translation when it's a $KEY the table
// holds, or label itself. prefer names the files (by base name) to look in
// first, such as the menu's own plugin; any file serves after them.
//
// A key may carry arguments, as SkyUI's do: "$Group {3}" is the entry for
// "$Group {}" with 3 in place of its {}. An argument that's itself a $KEY
// is translated too.
func (t *Table) Text(label string, prefer ...string) string {
	if !strings.HasPrefix(label, "$") {
		return label
	}
	if text, ok := t.lookup(label, prefer); ok {
		return text
	}
	args := argument.FindAllString(label, -1)
	if len(args) == 0 {
		return label
	}
	text, ok := t.lookup(argument.ReplaceAllString(label, "{}"), prefer)
	if !ok {
		return label
	}
	for _, a := range args {
		a = a[1 : len(a)-1]
		if strings.HasPrefix(a, "$") {
			a = t.Text(a, prefer...)
		}
		if i := strings.Index(text, "{}"); i >= 0 {
			text = text[:i] + a + text[i+2:]
		}
	}
	return text
}

func (t *Table) lookup(key string, prefer []string) (string, bool) {
	k := strings.ToLower(key)
	for _, p := range prefer {
		if entries, ok := t.files[strings.ToLower(p)]; ok {
			if v, ok := entries[k]; ok {
				return v, true
			}
		}
	}
	for _, base := range t.order {
		if v, ok := t.files[base][k]; ok {
			return v, true
		}
	}
	return "", false
}
