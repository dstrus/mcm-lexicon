package pex

import (
	"strconv"
	"strings"

	"lexicon/internal/papyrus/source"
)

// OtherVersion says why src isn't the source the script was compiled from,
// or "" when it is, as far as their strings tell: a string literal of the
// source the compiled script lacks, or a phrase of the compiled script (a
// one-line string with a space, so not a name) the source lacks. Mods
// often ship a source from another version than the compiled script the
// game runs.
func (f *File) OtherVersion(src string) string {
	have := map[string]bool{}
	for _, s := range f.Strings {
		have[strings.ToLower(s)] = true
	}
	lits := map[string]bool{}
	var all []string
	for _, s := range source.Literals(src) {
		lits[strings.ToLower(s)] = true
		all = append(all, strings.ToLower(s))
		if len(s) >= 4 && !have[strings.ToLower(s)] {
			return "its source has " + strconv.Quote(s)
		}
	}
	lower := strings.ToLower(src) + "\x00" + strings.Join(all, "\x00")
	for _, s := range f.Strings {
		if !strings.Contains(s, " ") || lits[strings.ToLower(s)] || strings.Contains(lower, strings.ToLower(s)) {
			continue
		}
		// Documentation the compiler keeps: { } comments, which span lines,
		// and what it writes for SKI_ConfigBase's own functions.
		if strings.Contains(s, "\n") || strings.HasPrefix(s, "Function that ") || strings.HasPrefix(s, "Called when ") {
			continue
		}
		return "its compiled script has " + strconv.Quote(s)
	}
	return ""
}
