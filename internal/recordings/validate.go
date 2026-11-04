package recordings

import (
	"strconv"
	"strings"

	"lexicon/internal/catalog"
)

// Status is what checking a step found.
type Status string

const (
	StatusMatched    Status = "matched"     // the option is there, and allows the value
	StatusNotAllowed Status = "not-allowed" // the option is there, but not with this value
	StatusNotFound   Status = "not-found"   // the menu is there, but not this page or option
	StatusModMissing Status = "mod-missing" // no menu by this name, or only one the game doesn't load
	StatusUnchecked  Status = "unchecked"   // the menu's options aren't all known, so a miss proves nothing
)

// Why is what a check's status is about.
type Why string

const (
	WhyPage     Why = "page"     // no page by this name
	WhyOption   Why = "option"   // no option of this type and label on the page
	WhyIndex    Why = "index"    // fewer options with this label than the index says
	WhyInactive Why = "inactive" // the menu's plugin isn't active
	WhyRange    Why = "range"    // a slider value outside its range
	WhyChoice   Why = "choice"   // a menu entry the menu doesn't have
	WhyValue    Why = "value"    // a right-side click: no text option on the page shows this value
	WhyDisabled Why = "disabled" // the menu grays the option out; MCM Recorder plays it anyway, calling the mod's handler
)

// Check is a step's check.
type Check struct {
	Status     Status          `json:"status"`
	Why        Why             `json:"why"`
	MenuID     string          `json:"menuId"` // the menu it names, when there is one
	MenuText   string          `json:"menuText"`
	Page       int             `json:"page"` // the page's position in the menu, when found
	PageText   string          `json:"pageText"`
	Position   int             `json:"position"` // the option's position on its page, when found
	Option     *catalog.Option `json:"option"`   // the option it found
	Suggestion *Suggestion     `json:"suggestion"`
}

// Suggestion is what the catalog has that a step that misses probably
// means: a label written differently, or the option on another page.
type Suggestion struct {
	Page     string `json:"page"` // raw
	PageText string `json:"pageText"`
	Label    string `json:"label"` // raw
	Text     string `json:"text"`
}

// optionType is the option type a step's kind plays.
var optionType = map[Kind]catalog.OptionType{
	KindToggle: catalog.OptionTypeToggle, KindSlider: catalog.OptionTypeSlider, KindChoose: catalog.OptionTypeMenu,
	KindShortcut: catalog.OptionTypeKeymap, KindColor: catalog.OptionTypeColor, KindText: catalog.OptionTypeInput, KindClick: catalog.OptionTypeText,
}

// Validate checks every step of a recording against a catalog, as MCM
// Recorder would find its option: the menu by its name, the page by its
// name, then the option by type and selector (exact, or containing the
// part between asterisks), the index-th when the step gives one. Papyrus
// compares strings without regard to case, so these do too; spaces count.
func Validate(r *Recording, c *catalog.Catalog) []Check {
	out := make([]Check, len(r.Steps))
	for i, s := range r.Steps {
		out[i] = check(s, c)
	}
	return out
}

func check(s Step, c *catalog.Catalog) Check {
	var menus []*catalog.Menu
	for i := range c.Menus {
		if strings.EqualFold(c.Menus[i].Name, s.Mod) {
			menus = append(menus, &c.Menus[i])
		}
	}
	if len(menus) == 0 {
		return Check{Status: StatusModMissing}
	}
	m := menus[0]
	for _, x := range menus {
		if x.Active {
			m = x
			break
		}
	}
	ch := Check{MenuID: m.ID, MenuText: m.NameText}
	if !m.Active {
		ch.Status, ch.Why = StatusModMissing, WhyInactive
		return ch
	}
	uncertain := m.Certainty != catalog.CertaintyExact || guessed(m)
	page := -1
	for i, p := range m.Pages {
		if strings.EqualFold(p.Name, s.Page) {
			page = i
			break
		}
	}
	typ := optionType[s.Kind]
	right := s.Kind == KindClick && strings.EqualFold(s.Side, "right")
	if page < 0 {
		ch.Status, ch.Why = StatusNotFound, WhyPage
		if uncertain {
			ch.Status = StatusUnchecked
		}
		ch.Suggestion = suggest(m, -1, typ, s.Option)
		return ch
	}
	p := m.Pages[page]
	ch.Page, ch.PageText = page, p.Text
	if right {
		return checkRight(ch, s, p, uncertain)
	}
	seen := 0
	for i := range p.Options {
		o := &p.Options[i]
		if o.Type != typ || !matches(s, o.Label) {
			continue
		}
		seen++
		if s.Index > 0 && seen != s.Index {
			continue
		}
		ch.Position, ch.Option = i, o
		ch.Status, ch.Why = allowed(s, o)
		return ch
	}
	ch.Status, ch.Why = StatusNotFound, WhyOption
	if seen > 0 {
		ch.Why = WhyIndex
	}
	if uncertain {
		ch.Status = StatusUnchecked
	}
	ch.Suggestion = suggest(m, page, typ, s.Option)
	return ch
}

// checkRight checks a click on an option's right side, which MCM Recorder
// matches by the value shown there rather than the label: a text option on
// the page whose value matches (exact, or containing a *part*), the
// index-th when the step gives one. A value the menu shows at some point
// matches, even one it changes; a miss proves the step won't replay only
// when every text option's value on the page is known and fixed.
func checkRight(ch Check, s Step, p catalog.Page, uncertain bool) Check {
	seen := 0
	doubt := uncertain
	for i := range p.Options {
		o := &p.Options[i]
		if o.Type != catalog.OptionTypeText {
			continue
		}
		if o.ValueCertainty != catalog.CertaintyExact || o.ValueChanges {
			doubt = true
		}
		if o.ValueCertainty != catalog.CertaintyExact || !matches(s, o.Value) {
			continue
		}
		seen++
		if s.Index > 0 && seen != s.Index {
			continue
		}
		ch.Position, ch.Option = i, o
		ch.Status = StatusMatched
		return ch
	}
	ch.Status, ch.Why = StatusNotFound, WhyValue
	if seen > 0 {
		ch.Why = WhyIndex
	}
	if doubt {
		ch.Status = StatusUnchecked
	}
	return ch
}

// guessed reports whether a menu's pages were guessed, so a page missing
// from them may still be there.
func guessed(m *catalog.Menu) bool {
	for _, n := range m.Notes {
		if n.Kind == catalog.NoteKindPagesGuessed {
			return true
		}
	}
	return false
}

// matches is MCM Recorder's selector match: exact, or with a *part*
// selector, a label that holds part.
func matches(s Step, label string) bool {
	if s.Wildcard {
		return strings.Contains(strings.ToLower(label), strings.ToLower(s.Option[1:len(s.Option)-1]))
	}
	return strings.EqualFold(label, s.Option)
}

// allowed checks a step's value against what its option allows, where
// the catalog knows it. A step on an option the menu grays out matches,
// with that said: MCM Recorder doesn't look at an option's flags, so it
// plays the step, calling the mod's handler as a click would, though a
// player couldn't select it.
func allowed(s Step, o *catalog.Option) (Status, Why) {
	st, why := allowedValue(s, o)
	if st == StatusMatched && o.Disabled == catalog.DisabledAlways {
		return StatusMatched, WhyDisabled
	}
	return st, why
}

func allowedValue(s Step, o *catalog.Option) (Status, Why) {
	switch s.Kind {
	case KindSlider:
		if o.Slider == nil {
			break
		}
		v, err := strconv.ParseFloat(s.Value, 64)
		if err != nil || v < o.Slider.Min-1e-6 || v > o.Slider.Max+1e-6 {
			return StatusNotAllowed, WhyRange
		}
	case KindChoose:
		if len(o.Choices) == 0 {
			break
		}
		for _, c := range o.Choices {
			if strings.EqualFold(c.Raw, s.Value) {
				return StatusMatched, ""
			}
		}
		return StatusNotAllowed, WhyChoice
	}
	return StatusMatched, ""
}

// suggest finds what a step that misses most likely means: on its page
// (or every page, when the page is missing), the option of its type whose
// label is closest to the selector; or the same label on another page.
func suggest(m *catalog.Menu, page int, typ catalog.OptionType, selector string) *Suggestion {
	best, score := (*Suggestion)(nil), 0.0
	for pi, p := range m.Pages {
		for _, o := range p.Options {
			if o.Type != typ || o.Label == "" {
				continue
			}
			sc := similarity(selector, o.Label)
			if page >= 0 && pi != page {
				if !strings.EqualFold(o.Label, selector) {
					continue // another page counts only for the same label
				}
				sc = 0.95
			}
			if sc > score {
				best, score = &Suggestion{Page: p.Name, PageText: p.Text, Label: o.Label, Text: o.Text}, sc
			}
		}
	}
	if score < 0.6 {
		return nil
	}
	return best
}

// similarity is how alike two labels are, from 0 to 1: 1 when they differ
// only in case or in spaces at their ends, otherwise by edit distance.
func similarity(a, b string) float64 {
	if strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b)) {
		return 1
	}
	x, y := []rune(strings.ToLower(a)), []rune(strings.ToLower(b))
	longest := max(len(x), len(y))
	if longest == 0 {
		return 1
	}
	return 1 - float64(distance(x, y))/float64(longest)
}

func distance(a, b []rune) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

// Counts is how a recording's steps checked.
type Counts struct {
	Matched    int `json:"matched"`
	NotAllowed int `json:"notAllowed"`
	NotFound   int `json:"notFound"`
	ModMissing int `json:"modMissing"`
	Unchecked  int `json:"unchecked"`
}

func Count(checks []Check) Counts {
	var c Counts
	for _, ch := range checks {
		switch ch.Status {
		case StatusMatched:
			c.Matched++
		case StatusNotAllowed:
			c.NotAllowed++
		case StatusNotFound:
			c.NotFound++
		case StatusModMissing:
			c.ModMissing++
		case StatusUnchecked:
			c.Unchecked++
		}
	}
	return c
}
