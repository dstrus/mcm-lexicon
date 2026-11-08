// Package server is MCM Lexicon's web server. It serves the app's pages, and
// mounts each domain's API routes behind the checks every request needs:
// addressed to this machine, and carrying the token from the page.
package server

import (
	"bytes"
	"crypto/subtle"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"lexicon/internal/apperr"
	"lexicon/internal/catalog"
	"lexicon/internal/httpapi"
	"lexicon/internal/modlist"
	"lexicon/internal/prefs"
	"lexicon/internal/recordings"
	"lexicon/internal/snapshot"
)

// Placeholders in index.html (see app.html): the session's token, and the
// saved theme, so the page never loads in the wrong one first.
var (
	tokenPlaceholder = []byte("__LEXICON_TOKEN__")
	themePlaceholder = []byte("__LEXICON_THEME__")
)

type Options struct {
	Source  *modlist.Source // the list to read, or nil until one is chosen
	Catalog string          // a catalog file opened instead of a list, or ""
	Prefs   *prefs.Prefs
	Running func() bool // whether Skyrim is running
	Token   string
	Port    int
	Files   fs.FS // the built frontend, with index.html at its root
	Version string
	// CacheDir is where scans keep their cache (catalog.DefaultCacheDir);
	// "" keeps nothing.
	CacheDir string
	// DevPages, during development, is the Vite dev server's address: page
	// requests go there, since the frontend built into this program may be
	// older than the source. "" serves the built frontend.
	DevPages string
}

type Server struct {
	Options
	index      []byte // index.html, with the token filled in
	api        *http.ServeMux
	catalog    *catalog.Service
	recordings *recordings.Service

	mu          sync.Mutex
	source      *modlist.Source
	catalogFile string
}

func New(o Options) (*Server, error) {
	index, err := fs.ReadFile(o.Files, "index.html")
	if err != nil {
		return nil, fmt.Errorf("the frontend isn't built: %w", err)
	}
	s := &Server{Options: o, index: bytes.ReplaceAll(index, tokenPlaceholder, []byte(o.Token)), api: http.NewServeMux(), catalog: catalog.NewService(o.Version)}
	s.catalog.UseCache(o.CacheDir)
	switch {
	case o.Source != nil:
		s.source = o.Source
		s.catalog.UseList(*o.Source)
	case o.Catalog != "":
		// A catalog file remembered from last time, if it still reads.
		if err := s.catalog.UseFile(o.Catalog); err != nil {
			log.Printf("Not opening the catalog from last time: %v", err)
		} else {
			s.catalogFile = o.Catalog
		}
	}
	s.routes()
	return s, nil
}

func (s *Server) routes() {
	m := s.api
	m.Handle("GET /api/status", httpapi.JSON(func(*http.Request, httpapi.None) (Status, error) {
		return s.status(), nil
	}))
	m.Handle("PUT /api/theme", httpapi.JSON(func(_ *http.Request, req ThemeRequest) (Status, error) {
		if !slices.Contains(prefs.Themes, req.Theme) {
			return Status{}, apperr.New(apperr.Invalid, "Unknown theme %q", req.Theme)
		}
		s.Prefs.Update(func(v *prefs.Values) { v.Theme = req.Theme })
		return s.status(), nil
	}))
	m.Handle("PUT /api/settings/save-folder", httpapi.JSON(func(_ *http.Request, req SaveFolder) (Status, error) {
		// The prefix is a switch of its own, so a name typed with it loses it.
		name := strings.TrimSpace(req.Name)
		if len(name) >= len(prefs.NoDeletePrefix)-1 && strings.EqualFold(name[:len(prefs.NoDeletePrefix)-1], strings.TrimSpace(prefs.NoDeletePrefix)) {
			name = strings.TrimSpace(name[len(prefs.NoDeletePrefix)-1:])
		}
		if name == prefs.DefaultSaveName {
			name = ""
		}
		v := prefs.Values{SaveName: name, NoDelete: &req.NoDelete}
		if folder := v.SaveMod(); !recordings.ModName(folder) || strings.EqualFold(folder, "overwrite") {
			return Status{}, apperr.New(apperr.Invalid, "%q can't be a mod folder's name", folder)
		}
		s.Prefs.Update(func(p *prefs.Values) {
			p.SaveName = name
			if req.NoDelete {
				p.NoDelete = nil // the default
			} else {
				p.NoDelete = &req.NoDelete
			}
		})
		return s.status(), nil
	}))
	m.Handle("GET /api/folders", httpapi.JSON(func(r *http.Request, _ httpapi.None) (modlist.Listing, error) {
		looking := modlist.LookingList
		switch l := modlist.Looking(r.URL.Query().Get("for")); l {
		case modlist.LookingCatalog, modlist.LookingFolder:
			looking = l
		}
		return modlist.Browse(r.URL.Query().Get("path"), looking)
	}))
	// A copy of what was read, to a file the player names: the catalog, to
	// open on another machine or compare against later, and a snapshot, the
	// list's files MCM Lexicon reads, for testing. Neither changes the list.
	m.Handle("POST /api/export/catalog", httpapi.JSON(func(r *http.Request, req ExportRequest) (ExportReply, error) {
		if src, _ := s.catalog.Source(); src == nil {
			return ExportReply{}, apperr.New(apperr.Unavailable, "A catalog file is open, and it's a file already; open a mod list to save its catalog")
		}
		c, err := s.catalog.Catalog(r)
		if err != nil {
			return ExportReply{}, err
		}
		p, err := exportPath(req, ".json")
		if err != nil {
			return ExportReply{}, err
		}
		if err := catalog.WriteFile(p, c); err != nil {
			return ExportReply{}, err
		}
		return ExportReply{Path: p, Menus: len(c.Menus)}, nil
	}))
	m.Handle("POST /api/export/snapshot", httpapi.JSON(func(r *http.Request, req ExportRequest) (ExportReply, error) {
		l, err := s.catalog.List(r)
		if err != nil {
			return ExportReply{}, err
		}
		if l == nil {
			return ExportReply{}, apperr.New(apperr.Unavailable, "A catalog file is open; a snapshot is made from a mod list")
		}
		p, err := exportPath(req, ".zip")
		if err != nil {
			return ExportReply{}, err
		}
		sum, err := snapshot.Write(l, p, snapshot.Options{Tool: "MCM Lexicon " + s.Version})
		if err != nil {
			return ExportReply{}, err
		}
		// The zip's own size, which is what the player has to find room
		// for, rather than its files' before compression.
		reply := ExportReply{Path: sum.Out, Mods: sum.Mods, Files: sum.Files, Steps: sum.Steps, Bytes: sum.Bytes}
		if info, err := os.Stat(sum.Out); err == nil {
			reply.Bytes = info.Size()
		}
		return reply, nil
	}))
	// What's at a folder, without choosing it: for the list form's profiles.
	m.Handle("GET /api/source", httpapi.JSON(func(r *http.Request, _ httpapi.None) (modlist.Source, error) {
		return modlist.Check(r.URL.Query().Get("path"), "")
	}))
	m.Handle("PUT /api/source", httpapi.JSON(func(_ *http.Request, req SourceRequest) (Status, error) {
		switch {
		case req.Catalog != "" && req.Path != "":
			return Status{}, apperr.New(apperr.Invalid, "Choose a mod list or a catalog file, not both")
		case req.Catalog != "":
			if err := s.OpenCatalog(req.Catalog); err != nil {
				return Status{}, err
			}
		default:
			if _, err := s.SetSource(req.Path, req.Profile); err != nil {
				return Status{}, err
			}
		}
		return s.status(), nil
	}))

	s.catalog.Routes(m)
	s.recordings = recordings.NewService(s.catalog, s.Prefs, s.Running)
	s.recordings.Routes(m)
	m.Handle("PUT /api/editing", httpapi.JSON(func(_ *http.Request, req EditingRequest) (Status, error) {
		if err := s.recordings.SetEditing(req.Editing, req.Discard); err != nil {
			return Status{}, err
		}
		return s.status(), nil
	}))
	recordings.CompareRoutes(m, s.catalog)
	m.Handle("PUT /api/recordings/open", httpapi.JSON(func(_ *http.Request, req OpenRequest) (Status, error) {
		s.Prefs.Update(func(v *prefs.Values) { v.OpenRecording = strings.TrimSpace(req.Name) })
		return s.status(), nil
	}))

	m.Handle("/api/", httpapi.JSON(func(r *http.Request, _ httpapi.None) (httpapi.None, error) {
		return httpapi.None{}, apperr.New(apperr.NotFound, "Unknown endpoint: %s %s", r.Method, r.URL.Path)
	}))
}

// SetSource makes the mod list at path (with profile, for an MO2 instance)
// the one being read, and remembers it.
func (s *Server) SetSource(path, profile string) (modlist.Source, error) {
	src, err := modlist.Check(path, profile)
	if err != nil {
		return modlist.Source{}, err
	}
	s.mu.Lock()
	s.source = &src
	s.catalogFile = ""
	s.mu.Unlock()
	s.catalog.UseList(src)
	s.recordings.Clear()
	log.Printf("Mod list: %s", src.Path)
	s.Prefs.Update(func(v *prefs.Values) {
		if v.ListDir != src.Path {
			v.OpenRecording = "" // another list's recordings
		}
		v.ListDir = src.Path
		v.Profile = src.Profile
		v.CatalogFile = ""
	})
	return src, nil
}

// OpenCatalog opens a catalog file instead of a list, and remembers it.
func (s *Server) OpenCatalog(path string) error {
	file, err := modlist.CheckCatalog(path)
	if err != nil {
		return err
	}
	if err := s.catalog.UseFile(file); err != nil {
		return err
	}
	s.mu.Lock()
	s.source = nil
	s.catalogFile = file
	s.mu.Unlock()
	s.recordings.Clear()
	log.Printf("Catalog: %s", file)
	s.Prefs.Update(func(v *prefs.Values) {
		v.ListDir = ""
		v.Profile = ""
		v.CatalogFile = file
	})
	return nil
}

// Source is the list being read (nil if none) and the catalog file opened
// instead ("" if none).
func (s *Server) Source() (*modlist.Source, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.source, s.catalogFile
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only answer requests addressed to this machine by name (blocks DNS-rebinding tricks).
	if r.Host != fmt.Sprintf("localhost:%d", s.Port) && r.Host != fmt.Sprintf("127.0.0.1:%d", s.Port) {
		httpapi.Send(w, http.StatusForbidden, httpapi.Error{Error: "Forbidden"})
		return
	}
	switch {
	case r.URL.Path == "/api/ping":
		// Needs no token: it only says that this is MCM Lexicon.
		httpapi.Send(w, http.StatusOK, Ping{App: AppID})
	case strings.HasPrefix(r.URL.Path, "/api/"):
		s.guard(w, r)
	case s.DevPages != "":
		http.Redirect(w, r, strings.TrimRight(s.DevPages, "/")+r.URL.RequestURI(), http.StatusTemporaryRedirect)
	default:
		s.page(w, r)
	}
}

// guard requires the per-session token embedded in the page, so other
// websites open in the same browser can't read or change anything.
func (s *Server) guard(w http.ResponseWriter, r *http.Request) {
	if subtle.ConstantTimeCompare([]byte(r.Header.Get(TokenHeader)), []byte(s.Token)) != 1 {
		httpapi.Send(w, http.StatusForbidden, httpapi.Error{Error: "Missing or invalid MCM Lexicon token. Reload the page."})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	s.api.ServeHTTP(w, r)
}

// page serves the frontend's files. Any other path is a view inside the app
// (every view has its own address), so it gets index.html.
func (s *Server) page(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if name != "" && name != "index.html" {
		if info, err := fs.Stat(s.Files, name); err == nil && !info.IsDir() {
			if strings.HasPrefix(name, "_app/immutable/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.ServeFileFS(w, r, s.Files, name)
			return
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store") // it carries this session's token
	w.Write(bytes.Replace(s.index, themePlaceholder, []byte(s.Prefs.Get().ThemeOf()), 1))
}

// exportPath is where an export goes: the named file in the chosen folder,
// with the extension it needs. A name with a folder in it is refused, so
// the file lands where the player looked, and a file already there is
// replaced only when the request says so.
func exportPath(req ExportRequest, ext string) (string, error) {
	folder, name := strings.TrimSpace(req.Folder), strings.TrimSpace(req.Name)
	if folder == "" || !modlist.IsDir(folder) {
		return "", apperr.New(apperr.NotFound, "Folder not found: %s", folder)
	}
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\:*?"<>|`) {
		return "", apperr.New(apperr.Invalid, "Give the file a name, without a folder or any of / \\ : * ? \" < > |")
	}
	if !strings.EqualFold(filepath.Ext(name), ext) {
		name += ext
	}
	p := filepath.Join(folder, name)
	if info, err := os.Stat(p); err == nil {
		if info.IsDir() {
			return "", apperr.New(apperr.Invalid, "%s is a folder", p)
		}
		if !req.Replace {
			return "", apperr.New(apperr.Conflict, "%s is already there", p)
		}
	}
	return p, nil
}
