package catalog

import (
	"log"
	"net/http"
	"sync"
	"time"

	"lexicon/internal/apperr"
	"lexicon/internal/httpapi"
	"lexicon/internal/modlist"
)

// Service holds the catalog the app shows: one scanned from the list in the
// background, or one read from a catalog file. While a list rescans, the
// catalog it had stays in place until the new one is ready.
type Service struct {
	version  string
	cacheDir string // where scans keep their cache; "" for none

	mu       sync.Mutex
	cur      *Catalog
	list     *modlist.List // the list cur was scanned from, indexed; nil for a catalog file
	err      error         // the last scan's or read's failure
	source   *modlist.Source
	file     string // the catalog file open instead of a list
	scanning bool
	done     chan struct{} // closed when the current scan ends
	runs     int           // bumped per scan, so a stale one doesn't land
	scanned  sums          // what the profile's modlist.txt and plugins.txt held when the last scan began
	watching sync.Once     // the watch on those files, started with the first list
}

func NewService(version string) *Service {
	done := make(chan struct{})
	close(done)
	return &Service{version: version, done: done}
}

// UseCache keeps what scans learn of a list's scripts in dir, so a rescan
// reads only the scripts that changed; "" keeps nothing.
func (s *Service) UseCache(dir string) {
	s.mu.Lock()
	s.cacheDir = dir
	s.mu.Unlock()
}

// UseList makes src the list the catalog comes from, and scans it.
func (s *Service) UseList(src modlist.Source) {
	s.mu.Lock()
	s.source, s.file = &src, ""
	s.cur, s.err = nil, nil
	s.mu.Unlock()
	s.Rescan()
	s.watching.Do(func() { go s.watch() })
}

// UseFile opens a catalog file in place of a list.
func (s *Service) UseFile(path string) error {
	c, err := ReadFile(path)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stop()
	s.source, s.file, s.cur, s.err, s.list = nil, path, c, nil, nil
	return nil
}

// stop abandons a scan still running, so it won't land, and lets anyone
// waiting on it go. Callers hold s.mu.
func (s *Service) stop() {
	s.runs++
	if s.scanning {
		s.scanning = false
		close(s.done)
	}
}

// Clear forgets what was open.
func (s *Service) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stop()
	s.source, s.file, s.cur, s.err, s.list = nil, "", nil, nil, nil
}

// Rescan scans the list again in the background. A catalog file isn't
// scanned; it's read again.
func (s *Service) Rescan() error {
	s.mu.Lock()
	if s.file != "" {
		file := s.file
		s.mu.Unlock()
		return s.UseFile(file)
	}
	if s.source == nil {
		s.mu.Unlock()
		return apperr.New(apperr.Unavailable, "Choose a mod list first")
	}
	src := *s.source
	// What the files say now, before the scan reads them: a change while it
	// runs is then a change the next look sees.
	s.scanned = sumsOf(profileFiles(&src))
	s.runs++
	run := s.runs
	if !s.scanning {
		s.done = make(chan struct{})
	}
	s.scanning = true
	done := s.done
	s.mu.Unlock()

	go func() {
		started := time.Now()
		var c *Catalog
		l, err := modlist.Read(src)
		if err == nil {
			c = Scan(l, Options{Version: s.version, CacheDir: s.cacheDir})
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if run != s.runs {
			return // superseded; the newer run closes done
		}
		s.scanning = false
		if err != nil {
			s.err = err
		} else {
			s.cur, s.err, s.list = c, nil, l
			log.Printf("Scanned %s: %d menus, %d not read, in %s", src.Path, len(c.Menus), len(c.Skipped), time.Since(started).Round(time.Millisecond))
		}
		close(done)
	}()
	return nil
}

// Source is what the catalog comes from: the list (nil if none) or the
// catalog file open instead ("" if none).
func (s *Service) Source() (*modlist.Source, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.source, s.file
}

// List is the list the current catalog was scanned from, with its files
// indexed, or nil when a catalog file is open or nothing is.
func (s *Service) List(r *http.Request) (*modlist.List, error) {
	if _, err := s.Catalog(r); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.list, nil
}

// State is what the page shows about the catalog while it works.
type State struct {
	Scanning  bool      `json:"scanning"`
	ScannedAt time.Time `json:"scannedAt,omitzero"` // the current catalog's; absent until there is one
	Error     string    `json:"error"`              // the last scan's failure, if it failed
	Menus     int       `json:"menus"`
}

func (s *Service) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := State{Scanning: s.scanning}
	if s.cur != nil {
		st.ScannedAt, st.Menus = s.cur.List.ScannedAt, len(s.cur.Menus)
	}
	if s.err != nil {
		st.Error = s.err.Error()
	}
	return st
}

// Catalog is the current catalog, waiting for a first scan to finish when
// there isn't one yet.
func (s *Service) Catalog(r *http.Request) (*Catalog, error) {
	for {
		s.mu.Lock()
		cur, err, done, scanning, open := s.cur, s.err, s.done, s.scanning, s.source != nil || s.file != ""
		s.mu.Unlock()
		switch {
		case cur != nil:
			return cur, nil
		case !open:
			return nil, apperr.New(apperr.Unavailable, "Choose a mod list or a catalog file first")
		case !scanning && err != nil:
			return nil, err
		case !scanning:
			return nil, apperr.New(apperr.Unavailable, "The list hasn't been scanned")
		}
		select {
		case <-done:
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
	}
}

// The API's replies.

type MenusReply struct {
	List    ListInfo  `json:"list"`
	Menus   []Summary `json:"menus"`
	Skipped []Skipped `json:"skipped"`
}

type SearchReply struct {
	Query   string   `json:"query"`
	Results []Result `json:"results"`
	Total   int      `json:"total"` // every match; Results stops at SearchLimit
}

// SearchLimit is the most results a search returns: search is for finding,
// and a menu's own page is for browsing.
const SearchLimit = 300

func (s *Service) Routes(m *http.ServeMux) {
	m.Handle("GET /api/menus", httpapi.JSON(func(r *http.Request, _ httpapi.None) (MenusReply, error) {
		c, err := s.Catalog(r)
		if err != nil {
			return MenusReply{}, err
		}
		reply := MenusReply{List: c.List, Menus: make([]Summary, len(c.Menus)), Skipped: c.Skipped}
		for i, menu := range c.Menus {
			reply.Menus[i] = Summarize(menu)
		}
		return reply, nil
	}))
	m.Handle("GET /api/menus/{id}", httpapi.JSON(func(r *http.Request, _ httpapi.None) (Menu, error) {
		c, err := s.Catalog(r)
		if err != nil {
			return Menu{}, err
		}
		id := r.PathValue("id")
		for _, menu := range c.Menus {
			if menu.ID == id {
				return menu, nil
			}
		}
		return Menu{}, apperr.New(apperr.NotFound, "There's no menu %q in this catalog", id)
	}))
	m.Handle("GET /api/search", httpapi.JSON(func(r *http.Request, _ httpapi.None) (SearchReply, error) {
		c, err := s.Catalog(r)
		if err != nil {
			return SearchReply{}, err
		}
		q := r.URL.Query().Get("q")
		results, total := Search(c, q, SearchLimit)
		return SearchReply{Query: q, Results: results, Total: total}, nil
	}))
	m.Handle("GET /api/catalog", httpapi.JSON(func(r *http.Request, _ httpapi.None) (*Catalog, error) {
		return s.Catalog(r)
	}))
	m.Handle("POST /api/scan", httpapi.JSON(func(*http.Request, httpapi.None) (State, error) {
		if err := s.Rescan(); err != nil {
			return State{}, err
		}
		return s.State(), nil
	}))
}
