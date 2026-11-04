package mcm

import "testing"

// Two texts combine into what they share, as a pattern.
func TestShared(t *testing.T) {
	hand := concat(str("Left Hand "), unknown)
	for _, c := range []struct {
		a, b Value
		want string // patternText, or "" for unknown
	}{
		{hand, concat(str("Right Hand "), unknown), "… Hand …"},
		{str("Left Hand Only"), str("Right Hand Only"), "… Hand Only"},
		{str("Volume: 10"), str("Volume: 20"), "Volume: …"},
		{str("-"), concat(concat(unknown, str(" and hold Button")), unknown), ""},
		{str("on"), str("off"), ""},
		{str("A"), str("B"), ""},
		{str("$FF_On"), str("$FF_Off"), ""},
	} {
		got := shared(c.a, c.b)
		text := ""
		if got.k == kPattern {
			text = patternText(got.parts)
		} else if got.k != kUnknown {
			t.Errorf("%v / %v: kind %d", c.a, c.b, got.k)
		}
		if text != c.want {
			t.Errorf("shared(%q, %q) = %q, want %q", patternText(c.a.asParts()), patternText(c.b.asParts()), text, c.want)
		}
	}
}
