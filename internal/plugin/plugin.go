// Package plugin reads Skyrim's plugin files (.esp, .esm, .esl) as far as
// MCM Lexicon needs: the header's flags and masters, each quest's attached
// scripts with their property values (VMAD), which is where many MCMs keep
// their display name (ModName) and pages (Pages), and the global variables
// (GLOB) with the values a new game starts them at, which many MCMs show.
// Everything else is skipped without being read, so a large plugin costs a
// few seeks.
package plugin

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Plugin is what MCM Lexicon reads from a plugin file.
type Plugin struct {
	Name      string   `json:"name"`
	Master    bool     `json:"master"`    // ESM-flagged, or an .esm
	Light     bool     `json:"light"`     // ESL-flagged, or an .esl
	Localized bool     `json:"localized"` // its names live in string files
	Masters   []string `json:"masters"`   // the plugins its form IDs' first byte counts, in order
	Quests    []Quest  `json:"quests"`    // quests with scripts attached
	Globals   []Global `json:"globals"`
}

// Global is a global variable as the plugin defines or overrides it.
type Global struct {
	FormID   uint32  `json:"formId"`
	EditorID string  `json:"editorId"`
	Value    float64 `json:"value"` // a short's or long's too, which the game keeps as a float
}

// Owner is the plugin a form ID in this plugin belongs to: the master its
// first byte counts, or this plugin itself past the masters.
func (p *Plugin) Owner(formID uint32) string {
	if i := int(formID >> 24); i < len(p.Masters) {
		return p.Masters[i]
	}
	return p.Name
}

// FormKey names a form across plugins: its owner, lowercased, and its ID
// within the owner, so a reference from one plugin finds the record in
// another.
func FormKey(owner string, formID uint32) string {
	return strings.ToLower(owner) + ":" + strconv.FormatUint(uint64(formID&0xFFFFFF), 16)
}

type Quest struct {
	FormID   uint32   `json:"formId"`
	EditorID string   `json:"editorId"`
	Scripts  []Script `json:"scripts"`
	Aliases  []Alias  `json:"aliases"` // its aliases with scripts attached
}

// Alias is one of a quest's aliases, by its ID in the quest, with the
// scripts attached to it (a ReferenceAlias's, say).
type Alias struct {
	ID      int      `json:"id"`
	Scripts []Script `json:"scripts"`
}

type Script struct {
	Name       string     `json:"name"`
	Properties []Property `json:"properties"`
}

// PropertyType is a script property's type, as VMAD numbers them.
type PropertyType uint8

const (
	PropertyNone        PropertyType = 0 // declared, but the plugin doesn't fill it
	PropertyObject      PropertyType = 1
	PropertyString      PropertyType = 2
	PropertyInt         PropertyType = 3
	PropertyFloat       PropertyType = 4
	PropertyBool        PropertyType = 5
	PropertyObjectArray PropertyType = 11
	PropertyStringArray PropertyType = 12
	PropertyIntArray    PropertyType = 13
	PropertyFloatArray  PropertyType = 14
	PropertyBoolArray   PropertyType = 15
)

// Property is a script property's value as the plugin fills it.
type Property struct {
	Name    string       `json:"name"`
	Type    PropertyType `json:"type"`
	String  string       `json:"string"`
	Strings []string     `json:"strings"`
	Number  float64      `json:"number"` // an int, a float, a bool as 0 or 1, or an object's form ID
	Numbers []float64    `json:"numbers"`
	Alias   int          `json:"alias"` // an object's alias in the quest Number names; -1 for none
}

// Property is the named property of the script, compared without regard
// to case as Papyrus does.
func (s Script) Property(name string) (Property, bool) {
	for _, p := range s.Properties {
		if strings.EqualFold(p.Name, name) {
			return p, true
		}
	}
	return Property{}, false
}

// Record and group flags.
const (
	flagMaster     = 0x00000001
	flagLocalized  = 0x00000080
	flagLight      = 0x00000200
	flagCompressed = 0x00040000
)

const headerSize = 24

// ReadFile reads the plugin at path.
func ReadFile(path string) (*Plugin, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	p, err := Read(f, info.Size())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	p.Name = filepath.Base(path)
	switch strings.ToLower(filepath.Ext(path)) {
	case ".esm":
		p.Master = true
	case ".esl":
		p.Master, p.Light = true, true
	}
	return p, nil
}

// Read reads a plugin of size bytes from r. A malformed file is an error,
// never a crash: every length is checked against what's there.
func Read(r io.ReaderAt, size int64) (*Plugin, error) {
	head := make([]byte, headerSize)
	if _, err := r.ReadAt(head, 0); err != nil {
		return nil, errors.New("too short to be a plugin")
	}
	if string(head[:4]) != "TES4" {
		return nil, errors.New("not a Skyrim plugin (no TES4 header)")
	}
	flags := binary.LittleEndian.Uint32(head[8:])
	p := &Plugin{
		Master:    flags&flagMaster != 0,
		Light:     flags&flagLight != 0,
		Localized: flags&flagLocalized != 0,
		Masters:   []string{},
		Quests:    []Quest{},
		Globals:   []Global{},
	}
	// The header's masters (MAST), which a form ID's first byte counts.
	headerData := int64(binary.LittleEndian.Uint32(head[4:]))
	if headerData > size-headerSize {
		return nil, errors.New("its header runs past the end of the file")
	}
	header := make([]byte, headerData)
	if _, err := r.ReadAt(header, headerSize); err != nil {
		return nil, err
	}
	p.Masters = masters(header)
	at := int64(headerSize) + headerData
	// The top-level groups, one per record type: only QUST and GLOB are read.
	for at+headerSize <= size {
		if _, err := r.ReadAt(head, at); err != nil {
			return nil, err
		}
		if string(head[:4]) != "GRUP" {
			return nil, fmt.Errorf("expected a group at offset %d", at)
		}
		groupSize := int64(binary.LittleEndian.Uint32(head[4:]))
		if groupSize < headerSize || at+groupSize > size {
			return nil, fmt.Errorf("a group at offset %d runs past the end of the file", at)
		}
		switch string(head[8:12]) {
		case "QUST":
			body := make([]byte, groupSize-headerSize)
			if _, err := r.ReadAt(body, at+headerSize); err != nil {
				return nil, err
			}
			quests, err := readQuests(body)
			if err != nil {
				return nil, err
			}
			p.Quests = quests
		case "GLOB":
			body := make([]byte, groupSize-headerSize)
			if _, err := r.ReadAt(body, at+headerSize); err != nil {
				return nil, err
			}
			globals, err := readGlobals(body)
			if err != nil {
				return nil, err
			}
			p.Globals = globals
		}
		at += groupSize
	}
	if at != size {
		return nil, fmt.Errorf("ends part-way through a group at offset %d", at)
	}
	return p, nil
}

// masters is the MAST subrecords of a plugin's header, in order.
func masters(b []byte) []string {
	out := []string{}
	for len(b) >= 6 {
		typ := string(b[:4])
		n := int(binary.LittleEndian.Uint16(b[4:]))
		b = b[6:]
		if n > len(b) {
			break
		}
		if typ == "MAST" {
			out = append(out, zstring(b[:n]))
		}
		b = b[n:]
	}
	return out
}

// readGlobals reads the GLOB records in a GLOB group's body: each one's
// editor ID and value (FLTV).
func readGlobals(b []byte) ([]Global, error) {
	out := []Global{}
	for len(b) >= headerSize {
		typ := string(b[:4])
		n := int(binary.LittleEndian.Uint32(b[4:]))
		if typ == "GRUP" {
			if n < headerSize || n > len(b) {
				return nil, errors.New("a nested group runs past its parent")
			}
			b = b[n:]
			continue
		}
		if headerSize+n > len(b) {
			return nil, fmt.Errorf("a %s record runs past its group", typ)
		}
		flags := binary.LittleEndian.Uint32(b[8:])
		formID := binary.LittleEndian.Uint32(b[12:])
		data := b[headerSize : headerSize+n]
		b = b[headerSize+n:]
		if typ != "GLOB" {
			continue
		}
		if flags&flagCompressed != 0 {
			var err error
			if data, err = decompress(data); err != nil {
				return nil, fmt.Errorf("global %08X: %w", formID, err)
			}
		}
		g := Global{FormID: formID}
		for len(data) >= 6 {
			sub := string(data[:4])
			m := int(binary.LittleEndian.Uint16(data[4:]))
			data = data[6:]
			if m > len(data) {
				return nil, fmt.Errorf("global %08X: its %s runs past the record", formID, sub)
			}
			switch sub {
			case "EDID":
				g.EditorID = zstring(data[:m])
			case "FLTV":
				if m >= 4 {
					g.Value = float64(math.Float32frombits(binary.LittleEndian.Uint32(data)))
				}
			}
			data = data[m:]
		}
		out = append(out, g)
	}
	return out, nil
}

// readQuests reads the QUST records in a QUST group's body, skipping the
// groups nested in it (a quest's dialogue and scenes).
func readQuests(b []byte) ([]Quest, error) {
	var out []Quest
	for len(b) >= headerSize {
		typ := string(b[:4])
		n := int(binary.LittleEndian.Uint32(b[4:]))
		if typ == "GRUP" {
			if n < headerSize || n > len(b) {
				return nil, errors.New("a nested group runs past its parent")
			}
			b = b[n:]
			continue
		}
		if headerSize+n > len(b) {
			return nil, fmt.Errorf("a %s record runs past its group", typ)
		}
		flags := binary.LittleEndian.Uint32(b[8:])
		formID := binary.LittleEndian.Uint32(b[12:])
		data := b[headerSize : headerSize+n]
		b = b[headerSize+n:]
		if typ != "QUST" {
			continue
		}
		if flags&flagCompressed != 0 {
			var err error
			if data, err = decompress(data); err != nil {
				return nil, fmt.Errorf("quest %08X: %w", formID, err)
			}
		}
		q, err := readQuest(formID, data)
		if err != nil {
			return nil, fmt.Errorf("quest %08X: %w", formID, err)
		}
		if len(q.Scripts) > 0 {
			out = append(out, q)
		}
	}
	return out, nil
}

// decompress reads a compressed record's data: its size once decompressed,
// then a zlib stream.
func decompress(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return nil, errors.New("compressed data too short")
	}
	want := binary.LittleEndian.Uint32(data)
	if want > 64<<20 {
		return nil, fmt.Errorf("claims %d bytes decompressed", want)
	}
	zr, err := zlib.NewReader(bytes.NewReader(data[4:]))
	if err != nil {
		return nil, fmt.Errorf("compressed data: %w", err)
	}
	defer zr.Close()
	out := make([]byte, want)
	if _, err := io.ReadFull(zr, out); err != nil {
		return nil, fmt.Errorf("compressed data: %w", err)
	}
	return out, nil
}

// readQuest reads a quest's subrecords: its editor ID and its scripts.
func readQuest(formID uint32, b []byte) (Quest, error) {
	q := Quest{FormID: formID, Scripts: []Script{}}
	var big uint32 // an XXXX subrecord gives the next one's size
	for len(b) >= 6 {
		typ := string(b[:4])
		n := int(binary.LittleEndian.Uint16(b[4:]))
		b = b[6:]
		if big != 0 {
			n, big = int(big), 0
		}
		if n > len(b) {
			return q, fmt.Errorf("its %s runs past the record", typ)
		}
		data := b[:n]
		b = b[n:]
		switch typ {
		case "XXXX":
			if n >= 4 {
				big = binary.LittleEndian.Uint32(data)
			}
		case "EDID":
			q.EditorID = zstring(data)
		case "VMAD":
			scripts, aliases, err := readVMAD(data)
			if err != nil {
				return q, fmt.Errorf("its scripts: %w", err)
			}
			q.Scripts, q.Aliases = scripts, aliases
		}
	}
	return q, nil
}

// readVMAD reads the scripts attached to a record and their properties. A
// quest's VMAD goes on to its fragments and aliases, which MCM Lexicon
// doesn't need, so reading stops after the scripts.
func readVMAD(b []byte) ([]Script, []Alias, error) {
	r := &reader{b: b}
	version := r.i16()
	format := r.i16()
	scripts := r.scripts(version, format)
	if r.err != nil {
		return scripts, nil, r.err
	}
	// A quest's VMAD goes on to its fragments, skipped, then its aliases
	// with their scripts. A tail that doesn't read leaves the aliases out:
	// the scripts above are what count.
	r.u8() // always 2
	fragments := int(r.u16())
	r.wstring() // the fragments' script file
	for i := 0; i < fragments && r.err == nil; i++ {
		r.u16()
		r.i16()
		r.u32()
		r.u8()
		r.wstring()
		r.wstring()
	}
	n := int(r.u16())
	var aliases []Alias
	for i := 0; i < n && r.err == nil; i++ {
		_, id := r.object(format)
		v, f := r.i16(), r.i16()
		a := Alias{ID: id, Scripts: r.scripts(v, f)}
		if len(a.Scripts) > 0 {
			aliases = append(aliases, a)
		}
	}
	if r.err != nil {
		return scripts, nil, nil
	}
	return scripts, aliases, nil
}

// scripts reads a list of scripts with their properties.
func (r *reader) scripts(version, format int16) []Script {
	count := int(r.u16())
	scripts := make([]Script, 0, count)
	for i := 0; i < count && r.err == nil; i++ {
		s := Script{Name: r.wstring(), Properties: []Property{}}
		if version >= 4 {
			r.u8() // status
		}
		props := int(r.u16())
		for j := 0; j < props && r.err == nil; j++ {
			p := Property{Name: r.wstring(), Type: PropertyType(r.u8()), Alias: -1}
			if version >= 4 {
				r.u8() // status
			}
			r.value(&p, format)
			s.Properties = append(s.Properties, p)
		}
		scripts = append(scripts, s)
	}
	return scripts
}

// object is an object value's form ID and alias (-1 for none): format 1
// puts the form ID first, format 2 last.
func (r *reader) object(format int16) (uint32, int) {
	b := r.take(8)
	if format == 1 {
		return binary.LittleEndian.Uint32(b), int(int16(binary.LittleEndian.Uint16(b[4:])))
	}
	return binary.LittleEndian.Uint32(b[4:]), int(int16(binary.LittleEndian.Uint16(b[2:])))
}

// reader reads VMAD's little-endian fields, remembering the first time it
// runs out rather than failing each read.
type reader struct {
	b   []byte
	err error
}

func (r *reader) take(n int) []byte {
	if r.err != nil {
		return make([]byte, n)
	}
	if n > len(r.b) {
		r.err = errors.New("ends part-way through a field")
		return make([]byte, n)
	}
	out := r.b[:n]
	r.b = r.b[n:]
	return out
}

func (r *reader) u8() uint8   { return r.take(1)[0] }
func (r *reader) u16() uint16 { return binary.LittleEndian.Uint16(r.take(2)) }
func (r *reader) i16() int16  { return int16(r.u16()) }
func (r *reader) u32() uint32 { return binary.LittleEndian.Uint32(r.take(4)) }

func (r *reader) wstring() string { return windows1252(r.take(int(r.u16()))) }

// count is an array's length, bounded by what's left, so a corrupt one
// can't ask for a huge allocation.
func (r *reader) count(each int) int {
	n := int(r.u32())
	if r.err == nil && n*each > len(r.b) {
		r.err = errors.New("an array claims more than the record holds")
		return 0
	}
	return n
}

// value reads a property's value by its type. An object is a form ID and
// an alias; format says in which order.
func (r *reader) value(p *Property, format int16) {
	object := func() float64 {
		id, _ := r.object(format)
		return float64(id)
	}
	switch p.Type {
	case PropertyNone:
	case PropertyObject:
		id, alias := r.object(format)
		p.Number, p.Alias = float64(id), alias
	case PropertyString:
		p.String = r.wstring()
	case PropertyInt:
		p.Number = float64(int32(r.u32()))
	case PropertyFloat:
		p.Number = float32ToFloat(math.Float32frombits(r.u32()))
	case PropertyBool:
		p.Number = float64(r.u8())
	case PropertyObjectArray:
		n := r.count(8)
		for i := 0; i < n; i++ {
			p.Numbers = append(p.Numbers, object())
		}
	case PropertyStringArray:
		n := r.count(2)
		p.Strings = []string{}
		for i := 0; i < n && r.err == nil; i++ {
			p.Strings = append(p.Strings, r.wstring())
		}
	case PropertyIntArray:
		n := r.count(4)
		for i := 0; i < n; i++ {
			p.Numbers = append(p.Numbers, float64(int32(r.u32())))
		}
	case PropertyFloatArray:
		n := r.count(4)
		for i := 0; i < n; i++ {
			p.Numbers = append(p.Numbers, float32ToFloat(math.Float32frombits(r.u32())))
		}
	case PropertyBoolArray:
		n := r.count(1)
		for i := 0; i < n; i++ {
			p.Numbers = append(p.Numbers, float64(r.u8()))
		}
	default:
		if r.err == nil {
			r.err = fmt.Errorf("property %s has type %d, which Skyrim doesn't use", p.Name, p.Type)
		}
	}
}

// float32ToFloat widens a stored 32-bit float to the number it was
// written as: 0.1, not 0.100000001490116.
func float32ToFloat(f float32) float64 {
	n, err := strconv.ParseFloat(strconv.FormatFloat(float64(f), 'g', -1, 32), 64)
	if err != nil {
		return float64(f)
	}
	return n
}

func zstring(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return windows1252(b)
}

// windows1252 decodes text as the Creation Kit writes it.
func windows1252(b []byte) string {
	var sb strings.Builder
	for _, c := range b {
		if c >= 0x80 && c < 0xA0 {
			sb.WriteRune(cp1252[c-0x80])
		} else {
			sb.WriteRune(rune(c))
		}
	}
	return sb.String()
}

var cp1252 = [32]rune{
	'€', '\u0081', '‚', 'ƒ', '„', '…', '†', '‡', 'ˆ', '‰', 'Š', '‹', 'Œ', '\u008d', 'Ž', '\u008f',
	'\u0090', '‘', '’', '“', '”', '•', '–', '—', '˜', '™', 'š', '›', 'œ', '\u009d', 'ž', 'Ÿ',
}
