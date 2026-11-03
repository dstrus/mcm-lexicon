package catalog

import (
	"regexp"
	"strings"
)

// SkyUI draws an MCM's text as Scaleform HTML, so mods color and style it
// with tags: <font color='#DD3333'>…</font>, <b>, <i>, <u>, a <br> for a
// line. The catalog keeps each label raw, as a recording matches it, and
// gives the text for reading as the game shows it, without the markup.
var (
	markupBreak = regexp.MustCompile(`(?i)<br\s*/?>`)
	markupTag   = regexp.MustCompile(`(?i)</?(font|b|i|u|p|span|textformat|li|ul|a|img)\b[^>]*>`)
	entities    = strings.NewReplacer("&lt;", "<", "&gt;", ">", "&quot;", `"`, "&apos;", "'", "&#39;", "'", "&nbsp;", " ", "&amp;", "&")
)

// plain is s as SkyUI shows it: its markup gone, its entities written out.
// Text without markup is returned as it is.
func plain(s string) string {
	if !strings.ContainsAny(s, "<&") {
		return s
	}
	t := markupBreak.ReplaceAllString(s, "\n")
	t = markupTag.ReplaceAllString(t, "")
	if !strings.Contains(t, "&") {
		return t
	}
	return entities.Replace(t)
}

// plainText gives every text for reading in c as SkyUI shows it.
func plainText(c *Catalog) {
	for mi := range c.Menus {
		m := &c.Menus[mi]
		m.NameText = plain(m.NameText)
		for pi := range m.Pages {
			p := &m.Pages[pi]
			p.Text = plain(p.Text)
			for fi := range p.Fixtures {
				p.Fixtures[fi].Text = plain(p.Fixtures[fi].Text)
			}
			for oi := range p.Options {
				o := &p.Options[oi]
				o.Text, o.ValueText, o.HelpText = plain(o.Text), plain(o.ValueText), plain(o.HelpText)
				for ci := range o.Choices {
					o.Choices[ci].Text = plain(o.Choices[ci].Text)
				}
			}
		}
	}
	for i := range c.Skipped {
		c.Skipped[i].NameText = plain(c.Skipped[i].NameText)
	}
}
