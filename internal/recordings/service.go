package recordings

import (
	"net/http"
	"strings"

	"lexicon/internal/apperr"
	"lexicon/internal/catalog"
	"lexicon/internal/httpapi"
)

// Summary is a recording without its steps, for the list of recordings.
type Summary struct {
	Name     string    `json:"name"`
	Title    string    `json:"title"`
	Version  string    `json:"version"`
	Author   string    `json:"author"`
	Autorun  bool      `json:"autorun"`
	Mod      string    `json:"mod"`
	Info     string    `json:"info"`
	Losing   []string  `json:"losing"`
	Steps    int       `json:"steps"`
	Files    int       `json:"files"`
	Menus    int       `json:"menus"` // the menus its steps name
	Counts   Counts    `json:"counts"`
	Problems []Problem `json:"problems"`
}

type ListReply struct {
	Recordings []Summary `json:"recordings"`
	// Readable is false when a catalog file is open: recordings live in a
	// list's files.
	Readable bool `json:"readable"`
}

// Detail is one recording with every step's check, and its files as
// written.
type Detail struct {
	Recording *Recording `json:"recording"`
	Checks    []Check    `json:"checks"` // one per step, in order
	Counts    Counts     `json:"counts"`
	Texts     []string   `json:"texts"` // each file's contents, as Files orders them
}

// Routes serves recordings from the list the catalog service last scanned.
func Routes(m *http.ServeMux, svc *catalog.Service) {
	read := func(r *http.Request) ([]*Recording, *catalog.Catalog, error) {
		c, err := svc.Catalog(r)
		if err != nil {
			return nil, nil, err
		}
		l, err := svc.List(r)
		if err != nil || l == nil {
			return nil, c, err
		}
		return Read(l), c, nil
	}
	m.Handle("GET /api/recordings", httpapi.JSON(func(r *http.Request, _ httpapi.None) (ListReply, error) {
		recs, c, err := read(r)
		if err != nil {
			return ListReply{}, err
		}
		reply := ListReply{Recordings: []Summary{}, Readable: recs != nil}
		for _, rec := range recs {
			reply.Recordings = append(reply.Recordings, summarize(rec, c))
		}
		return reply, nil
	}))
	m.Handle("GET /api/recordings/{name}", httpapi.JSON(func(r *http.Request, _ httpapi.None) (Detail, error) {
		recs, c, err := read(r)
		if err != nil {
			return Detail{}, err
		}
		name := r.PathValue("name")
		for _, rec := range recs {
			if strings.EqualFold(rec.Name, name) {
				checks := Validate(rec, c)
				d := Detail{Recording: rec, Checks: checks, Counts: Count(checks), Texts: []string{}}
				for i := range rec.Files {
					text, _ := rec.Text(i)
					d.Texts = append(d.Texts, text)
				}
				return d, nil
			}
		}
		if recs == nil {
			return Detail{}, apperr.New(apperr.Unavailable, "Recordings are read from a mod list, and a catalog file is open")
		}
		return Detail{}, apperr.New(apperr.NotFound, "There's no recording %q in this list", name)
	}))
}

func summarize(r *Recording, c *catalog.Catalog) Summary {
	checks := Validate(r, c)
	menus := map[string]bool{}
	for _, s := range r.Steps {
		menus[s.Mod] = true
	}
	return Summary{
		Name: r.Name, Title: r.Title, Version: r.Version, Author: r.Author, Autorun: r.Autorun,
		Mod: r.Mod, Info: r.Info, Losing: r.Losing, Steps: len(r.Steps), Files: len(r.Files), Menus: len(menus),
		Counts: Count(checks), Problems: r.Problems,
	}
}

// CompareRequest names the catalog file to compare the current catalog
// with, as the older of the two.
type CompareRequest struct {
	Path string `json:"path"`
}

// Break is a recorded step that matched in the older catalog and doesn't
// in the current one: what an update broke.
type Break struct {
	Recording string `json:"recording"`
	Step      int    `json:"step"` // its position in the recording, from 1
	Mod       string `json:"mod"`
	Page      string `json:"page"`
	Option    string `json:"option"`
	Kind      Kind   `json:"kind"`
	Check     Check  `json:"check"` // against the current catalog
}

type CompareReply struct {
	Diff   catalog.Diff `json:"diff"`
	Breaks []Break      `json:"breaks"`
}

// CompareRoutes serves comparing the current catalog with a catalog file.
func CompareRoutes(m *http.ServeMux, svc *catalog.Service) {
	m.Handle("POST /api/compare", httpapi.JSON(func(r *http.Request, req CompareRequest) (CompareReply, error) {
		cur, err := svc.Catalog(r)
		if err != nil {
			return CompareReply{}, err
		}
		old, err := catalog.ReadFile(strings.TrimSpace(req.Path))
		if err != nil {
			return CompareReply{}, err
		}
		reply := CompareReply{Diff: catalog.Compare(old, cur), Breaks: []Break{}}
		l, err := svc.List(r)
		if err != nil || l == nil {
			return reply, err
		}
		for _, rec := range Read(l) {
			before, after := Validate(rec, old), Validate(rec, cur)
			for i, s := range rec.Steps {
				if before[i].Status == StatusMatched && after[i].Status != StatusMatched {
					reply.Breaks = append(reply.Breaks, Break{Recording: rec.Name, Step: i + 1, Mod: s.Mod, Page: s.Page, Option: s.Option, Kind: s.Kind, Check: after[i]})
				}
			}
		}
		return reply, nil
	}))
}
