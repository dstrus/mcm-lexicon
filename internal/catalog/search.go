package catalog

import "strings"

// Summary is a menu without its pages, for the list of menus.
type Summary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	NameText  string    `json:"nameText"`
	Kind      Kind      `json:"kind"`
	Mod       string    `json:"mod"`
	Plugin    string    `json:"plugin"`
	Active    bool      `json:"active"`
	Script    string    `json:"script"`
	Pages     int       `json:"pages"`     // named pages; 0 for a menu without pages
	Options   int       `json:"options"`   // recordable options, across pages
	Uncertain int       `json:"uncertain"` // options whose label isn't exact
	Certainty Certainty `json:"certainty"`
	Notes     []Note    `json:"notes"`
	Source    SourceRef `json:"source"`
}

func Summarize(m Menu) Summary {
	s := Summary{ID: m.ID, Name: m.Name, NameText: m.NameText, Kind: m.Kind, Mod: m.Mod, Plugin: m.Plugin, Active: m.Active, Script: m.Script, Certainty: m.Certainty, Notes: m.Notes, Source: m.Source}
	for _, p := range m.Pages {
		if p.Name != "" {
			s.Pages++
		}
		s.Options += len(p.Options)
		for _, o := range p.Options {
			if o.Certainty != CertaintyExact {
				s.Uncertain++
			}
		}
	}
	return s
}

// Result is an option found by Search, with where it is.
type Result struct {
	MenuID   string `json:"menuId"`
	MenuName string `json:"menuName"`
	MenuText string `json:"menuText"`
	Mod      string `json:"mod"`
	Page     int    `json:"page"` // its page's position in the menu
	PageName string `json:"pageName"`
	PageText string `json:"pageText"`
	Position int    `json:"position"` // its position on the page
	Option   Option `json:"option"`
}

// Search finds options whose label, text, page, menu, or mod holds every
// word of query, without regard to case, in catalog order. It returns at
// most limit of them, and how many there were in all.
func Search(c *Catalog, query string, limit int) ([]Result, int) {
	words := strings.Fields(strings.ToLower(query))
	results := []Result{}
	if len(words) == 0 {
		return results, 0
	}
	total := 0
	for _, m := range c.Menus {
		menu := strings.ToLower(m.Name + "\x00" + m.NameText + "\x00" + m.Mod)
		for pi, p := range m.Pages {
			page := strings.ToLower(p.Name + "\x00" + p.Text)
			for oi, o := range p.Options {
				hay := menu + "\x00" + page + "\x00" + strings.ToLower(o.Label+"\x00"+o.Text)
				if !all(hay, words) {
					continue
				}
				total++
				if len(results) < limit {
					results = append(results, Result{MenuID: m.ID, MenuName: m.Name, MenuText: m.NameText, Mod: m.Mod, Page: pi, PageName: p.Name, PageText: p.Text, Position: oi, Option: o})
				}
			}
		}
	}
	return results, total
}

func all(hay string, words []string) bool {
	for _, w := range words {
		if !strings.Contains(hay, w) {
			return false
		}
	}
	return true
}
