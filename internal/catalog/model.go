// Package catalog is what MCM Lexicon knows about a list's menus: every MCM,
// its pages and options, and what can be recorded for each. It scans a list
// into a catalog, reads and writes catalog files, searches, and serves it.
package catalog

import "time"

// SchemaVersion of a catalog file; a reader refuses a newer one.
const SchemaVersion = 1

// Catalog is every menu of a list, as one file can hold it.
type Catalog struct {
	SchemaVersion int       `json:"schemaVersion"`
	List          ListInfo  `json:"list"`
	Menus         []Menu    `json:"menus"`   // by their text, in reading order
	Skipped       []Skipped `json:"skipped"` // what couldn't be read, and why
}

// ListInfo says where a catalog came from.
type ListInfo struct {
	Name      string    `json:"name"` // the instance or Data folder's name
	Path      string    `json:"path"`
	Kind      string    `json:"kind"` // "mo2" or "data"
	Profile   string    `json:"profile"`
	Game      string    `json:"game"`
	Language  string    `json:"language"` // the translations read, such as ENGLISH
	ScannedAt time.Time `json:"scannedAt"`
	Version   string    `json:"version"` // MCM Lexicon's, at the scan
}

// Kind is how a menu is defined.
type Kind string

const (
	KindMCMHelper Kind = "mcmhelper" // MCM Helper's config.json
	KindSource    Kind = "source"    // a Papyrus script, read from its source
	KindCompiled  Kind = "compiled"  // a Papyrus script, read from its compiled form
	KindFramework Kind = "framework" // a framework's menu, read in part
)

// Certainty is how sure MCM Lexicon is of an option's label.
type Certainty string

const (
	CertaintyExact   Certainty = "exact"   // the label as the game shows it
	CertaintyPattern Certainty = "pattern" // built at runtime; a *wildcard* can match it
	CertaintyUnknown Certainty = "unknown" // can't be known without the game
)

type Menu struct {
	ID        string    `json:"id"`       // stable across scans and list versions
	Name      string    `json:"name"`     // what a recording calls it ("mod"), raw: may be a $KEY
	NameText  string    `json:"nameText"` // translated, for people
	Kind      Kind      `json:"kind"`
	Mod       string    `json:"mod"`    // the mod whose file defines it
	Plugin    string    `json:"plugin"` // the plugin it belongs to, when known
	Active    bool      `json:"active"` // whether the game loads that plugin
	Script    string    `json:"script"` // the Papyrus script, or MCM Helper's config id
	Pages     []Page    `json:"pages"`  // in display order; one with Name "" for a menu without pages
	Certainty Certainty `json:"certainty"`
	Notes     []Note    `json:"notes"`
	Source    SourceRef `json:"source"`
}

type Page struct {
	Name    string   `json:"name"` // raw; "" for the default page
	Text    string   `json:"text"`
	Options []Option `json:"options"`
	// Fixtures are what the page shows besides options: headers and gaps.
	Fixtures []Fixture `json:"fixtures"`
	// Placed is true when every option and fixture has a position, so the
	// page can be drawn as SkyUI draws it: 64 rows of two columns, even
	// positions on the left and odd on the right.
	Placed bool `json:"placed"`
}

// Fixture is a header or an empty slot on a page.
type Fixture struct {
	Kind      FixtureKind `json:"kind"`
	Label     string      `json:"label"` // a header's text, raw
	Text      string      `json:"text"`  // translated
	Certainty Certainty   `json:"certainty"`
	Position  int         `json:"position"`
}

type FixtureKind string

const (
	FixtureKindHeader FixtureKind = "header"
	FixtureKindEmpty  FixtureKind = "empty"
)

// OptionType is an option's type as a recording names it.
type OptionType string

const (
	OptionTypeToggle OptionType = "toggle"
	OptionTypeSlider OptionType = "slider"
	OptionTypeMenu   OptionType = "menu"
	OptionTypeKeymap OptionType = "keymap"
	OptionTypeColor  OptionType = "color"
	OptionTypeInput  OptionType = "input"
	OptionTypeText   OptionType = "text"
)

type Option struct {
	Type      OptionType `json:"type"`
	Control   string     `json:"control"`  // the control as its source names it (MCM Helper's stepper, enum…), when it differs
	Label     string     `json:"label"`    // the selector a recording matches, raw
	Text      string     `json:"text"`     // translated
	Index     int        `json:"index"`    // 1-based among options of the same type and label on the page; 0 when unique
	State     string     `json:"state"`    // an ST option's state name
	Setting   string     `json:"setting"`  // MCM Helper's setting id, "key:Section"
	Slider    *Range     `json:"slider"`   // a slider's range, when known
	Choices   []Choice   `json:"choices"`  // a menu's or stepper's entries, when known
	Default   *Value     `json:"default"`  // when the source states one
	Saved     *Value     `json:"saved"`    // the player's own value, when MCM Helper has saved one
	Help      string     `json:"help"`     // the info text the menu shows, raw
	HelpText  string     `json:"helpText"` // translated
	Certainty Certainty  `json:"certainty"`
	// Position is SkyUI's cursor position when the option is added: where
	// the game draws it; -1 when the script decides it at runtime.
	Position int `json:"position"`
	// A text option's value, shown on its right, which a click on that side
	// matches: raw (exact, or a *pattern*), translated, and how certain.
	// ValueChanges is true when the menu shows other values at times.
	Value          string    `json:"value"`
	ValueText      string    `json:"valueText"`
	ValueCertainty Certainty `json:"valueCertainty"`
	ValueChanges   bool      `json:"valueChanges"`
	// ValueIndex is Index for a right-side click: 1-based among the page's
	// text options showing the same value; 0 when it's the only one.
	ValueIndex int       `json:"valueIndex"`
	Notes      []Note    `json:"notes"`
	Source     SourceRef `json:"source"`
}

type Range struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Step   float64 `json:"step"`
	Format string  `json:"format"` // as the source writes it, such as "{1} cm"
}

type Choice struct {
	Raw  string `json:"raw"` // what a recording's "choose" holds
	Text string `json:"text"`
}

// ValueKind is which field of a Value holds it.
type ValueKind string

const (
	ValueKindBool   ValueKind = "bool"
	ValueKindNumber ValueKind = "number"
	ValueKindString ValueKind = "string"
)

// Value is an option's value: on or off, a number (a slider's value, a key's
// scan code, a color, or a menu's choice by position), or text.
type Value struct {
	Kind   ValueKind `json:"kind"`
	Bool   bool      `json:"bool"`
	Number float64   `json:"number"`
	String string    `json:"string"`
}

// SourceRef is where something is defined: a file as the game sees it, the
// mod it comes from, the archive it's packed in if it is, and a line when
// known.
type SourceRef struct {
	Path    string `json:"path"`
	Mod     string `json:"mod"`
	Archive string `json:"archive,omitempty"`
	Line    int    `json:"line"`
}

// NoteKind says what a note is about; the page words it.
type NoteKind string

const (
	NoteKindNoPlugin       NoteKind = "no-plugin"        // the menu's mod has no plugin, so whether the game loads it isn't known
	NoteKindStepper        NoteKind = "stepper"          // a stepper cycles on each click; a recording can only click it
	NoteKindOverridden     NoteKind = "overridden"       // another mod's copy of the defining file loses to this one
	NoteKindPluginGuessed  NoteKind = "plugin-guessed"   // no quest with the menu's script was found; Plugin is the mod's likeliest
	NoteKindNameFromScript NoteKind = "name-from-script" // neither the script nor its quest sets ModName, so the menu goes by its script's name
	NoteKindPagesGuessed   NoteKind = "pages-guessed"    // Pages couldn't be read; the pages are the names OnPageReset compares against
	NoteKindFramework      NoteKind = "framework"        // the menu is built on another script (Detail), which was read too
	NoteKindSourceStale    NoteKind = "source-stale"     // the shipped source is another version (Detail says how it differs), so the compiled script was read
	NoteKindRepeats        NoteKind = "repeats"          // one call adds this option again and again in a loop the game decides
)

type Note struct {
	Kind   NoteKind `json:"kind"`
	Detail string   `json:"detail"` // a name or path the note refers to
}

// SkipReason says why something wasn't read.
type SkipReason string

const (
	SkipReasonNoQuest    SkipReason = "no-quest"   // a script built on SKI_ConfigBase that no quest runs, so no menu the game shows
	SkipReasonEmpty      SkipReason = "empty"      // a script on a quest with no name and no options: SkyUI lists nothing for it
	SkipReasonUnreadable SkipReason = "unreadable" // the file couldn't be read; Detail says why
)

type Skipped struct {
	Path    string     `json:"path"`
	Mod     string     `json:"mod"`
	Archive string     `json:"archive,omitempty"` // the archive it's packed in, if it is
	Reason  SkipReason `json:"reason"`
	Detail  string     `json:"detail"`
	// For a Papyrus menu, what its quest in a plugin says, when one was
	// found: the display name and pages its script's properties hold.
	Name     string   `json:"name"`     // raw: may be a $KEY
	NameText string   `json:"nameText"` // translated
	Pages    []string `json:"pages"`
	Plugin   string   `json:"plugin"`
	Quest    string   `json:"quest"` // the quest's editor ID
	Active   bool     `json:"active"`
}
