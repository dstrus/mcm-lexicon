package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"lexicon/internal/catalog"
	"lexicon/internal/prefs"
	"lexicon/internal/testcorpus"
)

const token = "test-token"

type harness struct {
	t       *testing.T
	list    string // an MO2 instance
	running bool
	prefs   *prefs.Prefs
	srv     *Server
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{t: t, list: testcorpus.MO2Instance(t, "[NoDelete] My List", "Default", "VR")}
	h.prefs = prefs.Load(filepath.Join(t.TempDir(), "settings.json"))
	files := fstest.MapFS{
		"index.html":            {Data: []byte(`<html data-theme="__LEXICON_THEME__"><meta name="lexicon-token" content="__LEXICON_TOKEN__">`)},
		"_app/immutable/app.js": {Data: []byte(`console.log("hi")`)},
		"favicon.svg":           {Data: []byte(`<svg/>`)},
	}
	srv, err := New(Options{Prefs: h.prefs, Running: func() bool { return h.running }, Token: token, Port: 8766, Files: files, Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	h.srv = srv
	return h
}

// request sends a request as the page would, from 127.0.0.1 with the token.
func (h *harness) request(method, path, body string) *httptest.ResponseRecorder {
	h.t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Host = "127.0.0.1:8766"
	r.Header.Set(TokenHeader, token)
	w := httptest.NewRecorder()
	h.srv.ServeHTTP(w, r)
	return w
}

// call makes an API request and decodes the reply into out, failing unless
// the status matches. out is a pointer to a Status, cleared first so an
// absent field reads as absent rather than as the last reply's.
func (h *harness) call(wantStatus int, method, path, body string, out *Status) {
	h.t.Helper()
	w := h.request(method, path, body)
	if w.Code != wantStatus {
		h.t.Fatalf("%s %s: status %d, want %d: %s", method, path, w.Code, wantStatus, w.Body)
	}
	if out != nil {
		*out = Status{}
		if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
			h.t.Fatalf("%s %s: %v", method, path, err)
		}
	}
}

func TestRequestsNeedLocalHostAndToken(t *testing.T) {
	h := newHarness(t)
	send := func(host, tok string) int {
		r := httptest.NewRequest("GET", "/api/status", nil)
		r.Host = host
		if tok != "" {
			r.Header.Set(TokenHeader, tok)
		}
		w := httptest.NewRecorder()
		h.srv.ServeHTTP(w, r)
		return w.Code
	}
	for _, c := range []struct {
		host, token string
		want        int
	}{
		{"evil.example:8766", token, 403},
		{"127.0.0.1:9999", token, 403},
		{"127.0.0.1:8766", "", 403},
		{"127.0.0.1:8766", "wrong", 403},
		{"127.0.0.1:8766", token, 200},
		{"localhost:8766", token, 200},
	} {
		if got := send(c.host, c.token); got != c.want {
			t.Errorf("host %s, token %q: status %d, want %d", c.host, c.token, got, c.want)
		}
	}

	// A second copy of MCM Lexicon asks without a token.
	r := httptest.NewRequest("GET", "/api/ping", nil)
	r.Host = "127.0.0.1:8766"
	w := httptest.NewRecorder()
	h.srv.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), AppID) {
		t.Errorf("ping: %d %s", w.Code, w.Body)
	}
}

func TestDeepLinksServeThePage(t *testing.T) {
	h := newHarness(t)
	for _, path := range []string{"/", "/index.html", "/menus/source-mod-script?item=3", "/_app/missing.js"} {
		w := h.request("GET", path, "")
		if !strings.Contains(w.Body.String(), `content="`+token+`"`) {
			t.Errorf("GET %s didn't serve the page with its token: %q", path, w.Body)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("GET %s: the page may be cached", path)
		}
	}
	if w := h.request("POST", "/menus/1", ""); w.Code != 404 {
		t.Errorf("POST to an app path: status %d, want 404", w.Code)
	}
	if w := h.request("GET", "/api/nothing", ""); w.Code != 404 || strings.Contains(w.Body.String(), token) {
		t.Errorf("unknown API path served %d %q", w.Code, w.Body)
	}
}

func TestFrontendFiles(t *testing.T) {
	h := newHarness(t)
	w := h.request("GET", "/_app/immutable/app.js", "")
	if w.Code != 200 || w.Body.String() != `console.log("hi")` || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("app.js: %d %q, cache %q", w.Code, w.Body, w.Header().Get("Cache-Control"))
	}
	if w := h.request("GET", "/favicon.svg", ""); w.Code != 200 || w.Header().Get("Cache-Control") != "" {
		t.Errorf("favicon: %d, cache %q", w.Code, w.Header().Get("Cache-Control"))
	}
}

func TestStatusAndSource(t *testing.T) {
	h := newHarness(t)
	var st Status
	h.call(200, "GET", "/api/status", "", &st)
	if st.Source != nil || st.CatalogFile != "" || st.Version != "test" || st.GameRunning {
		t.Errorf("status before choosing = %+v", st)
	}

	h.call(400, "PUT", "/api/source", `{"path": "`+filepath.ToSlash(t.TempDir())+`"}`, nil)
	h.call(400, "PUT", "/api/source", `{"folder": "typo"}`, nil)
	h.call(400, "PUT", "/api/source", `{"path": "x", "catalog": "y"}`, nil)
	body, _ := json.Marshal(SourceRequest{Path: h.list, Profile: "VR"})
	h.call(200, "PUT", "/api/source", string(body), &st)
	if st.Source == nil || st.Source.Path != h.list || st.Source.Profile != "VR" || len(st.Source.Profiles) != 2 {
		t.Errorf("source = %+v", st.Source)
	}
	if v := h.prefs.Get(); v.ListDir != h.list || v.Profile != "VR" {
		t.Errorf("remembered = %+v", v)
	}

	// A catalog file takes the list's place, and the other way round.
	file := filepath.Join(t.TempDir(), "mgo.json")
	if err := catalog.WriteFile(file, &catalog.Catalog{SchemaVersion: catalog.SchemaVersion, List: catalog.ListInfo{Name: "mgo"}}); err != nil {
		t.Fatal(err)
	}
	notCatalog := filepath.Join(t.TempDir(), "notes.json")
	os.WriteFile(notCatalog, []byte("{}"), 0o644)
	h.call(400, "PUT", "/api/source", `{"catalog": "`+filepath.ToSlash(filepath.Join(t.TempDir(), "missing.json"))+`"}`, nil)
	h.call(400, "PUT", "/api/source", `{"catalog": "`+filepath.ToSlash(notCatalog)+`"}`, nil)
	body, _ = json.Marshal(SourceRequest{Catalog: file})
	h.call(200, "PUT", "/api/source", string(body), &st)
	if st.Source != nil || st.CatalogFile != file {
		t.Errorf("after opening a catalog: %+v", st)
	}
	if v := h.prefs.Get(); v.ListDir != "" || v.CatalogFile != file {
		t.Errorf("remembered = %+v", v)
	}
	var menus catalog.MenusReply
	w := h.request("GET", "/api/menus", "")
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &menus) != nil || menus.List.Name != "mgo" {
		t.Errorf("menus from the catalog file: %d %s", w.Code, w.Body)
	}
	body, _ = json.Marshal(SourceRequest{Path: h.list})
	h.call(200, "PUT", "/api/source", string(body), &st)
	if st.Source == nil || st.Source.Profile != "Default" || st.CatalogFile != "" {
		t.Errorf("after choosing the list again: %+v", st)
	}

	h.running = true
	h.call(200, "GET", "/api/status", "", &st)
	if !st.GameRunning {
		t.Error("status doesn't say Skyrim is running")
	}
}

func TestThemeIsRememberedAndServed(t *testing.T) {
	h := newHarness(t)
	if page := h.request("GET", "/", "").Body.String(); !strings.Contains(page, `data-theme="system"`) {
		t.Errorf("page before choosing a theme: %q", page)
	}
	var st Status
	h.call(200, "PUT", "/api/theme", `{"theme": "light"}`, &st)
	if st.Theme != "light" || h.prefs.Get().Theme != "light" {
		t.Errorf("theme = %q, remembered %q", st.Theme, h.prefs.Get().Theme)
	}
	if page := h.request("GET", "/some/view", "").Body.String(); !strings.Contains(page, `data-theme="light"`) {
		t.Errorf("page after choosing light: %q", page)
	}
	h.call(400, "PUT", "/api/theme", `{"theme": "mgo"}`, nil)
}

func TestListenSkipsBusyPorts(t *testing.T) {
	other := busyPort(t, http.NotFoundHandler()) // some other web server
	closed := busyPort(t, nil)                   // something that isn't HTTP at all
	want := freePort(t)
	ln, running, err := Listen([]int{other, closed, want})
	if err != nil || running != "" {
		t.Fatalf("Listen = %v, %q, %v", ln, running, err)
	}
	defer ln.Close()
	if got := ln.Addr().(*net.TCPAddr).Port; got != want {
		t.Errorf("listening on %d, want %d", got, want)
	}
}

func TestListenFallsBackToAnyPort(t *testing.T) {
	busy := []int{busyPort(t, nil), busyPort(t, nil)}
	ln, running, err := Listen(busy)
	if err != nil || running != "" {
		t.Fatalf("Listen = %v, %q, %v", ln, running, err)
	}
	defer ln.Close()
	if got := ln.Addr().(*net.TCPAddr).Port; got == busy[0] || got == busy[1] {
		t.Errorf("listening on busy port %d", got)
	}
}

func TestListenFindsRunningCopy(t *testing.T) {
	h := newHarness(t)
	// The running copy's host check needs its real port, so serve through a
	// handler that rewrites the Host header the test client sends.
	port := busyPort(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Host = "127.0.0.1:8766"
		h.srv.ServeHTTP(w, r)
	}))
	ln, running, err := Listen([]int{busyPort(t, nil), port})
	if ln != nil {
		ln.Close()
		t.Fatal("started a second copy")
	}
	if want := fmt.Sprintf("http://127.0.0.1:%d/", port); err != nil || running != want {
		t.Errorf("Listen = %q, %v; want %q", running, err, want)
	}
}

// busyPort occupies a port until the test ends, serving handler on it, or
// accepting connections and saying nothing if handler is nil.
func busyPort(t *testing.T, handler http.Handler) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if handler != nil {
		srv := &http.Server{Handler: handler}
		go srv.Serve(ln)
		t.Cleanup(func() { srv.Close() })
	} else {
		t.Cleanup(func() { ln.Close() })
	}
	return ln.Addr().(*net.TCPAddr).Port
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

// Under make dev, pages come from Vite, at the same address; the API stays.
func TestDevPagesGoToVite(t *testing.T) {
	h := newHarness(t)
	h.srv.DevPages = "http://localhost:5174/"
	w := h.request("GET", "/recordings/Fresh/info?item=2", "")
	if w.Code != 307 || w.Header().Get("Location") != "http://localhost:5174/recordings/Fresh/info?item=2" {
		t.Errorf("page: %d to %q", w.Code, w.Header().Get("Location"))
	}
	if w := h.request("GET", "/api/status", ""); w.Code != 200 {
		t.Errorf("the API under dev: %d", w.Code)
	}
}

func TestExports(t *testing.T) {
	h := newHarness(t)
	body, _ := json.Marshal(SourceRequest{Path: h.list})
	h.call(200, "PUT", "/api/source", string(body), nil)
	out := t.TempDir()
	export := func(kind string, req ExportRequest, want int) ExportReply {
		t.Helper()
		b, _ := json.Marshal(req)
		w := h.request("POST", "/api/export/"+kind, string(b))
		if w.Code != want {
			t.Fatalf("%s %+v: status %d, want %d: %s", kind, req, w.Code, want, w.Body)
		}
		var r ExportReply
		json.Unmarshal(w.Body.Bytes(), &r)
		return r
	}

	// The extension is added, and the file is a catalog the app can open.
	r := export("catalog", ExportRequest{Folder: out, Name: "my list"}, 200)
	if want := filepath.Join(out, "my list.json"); r.Path != want {
		t.Errorf("path = %s, want %s", r.Path, want)
	}
	if _, err := catalog.ReadFile(r.Path); err != nil {
		t.Errorf("the saved catalog doesn't open: %v", err)
	}
	// One already there is replaced only when asked.
	export("catalog", ExportRequest{Folder: out, Name: "my list.json"}, 409)
	export("catalog", ExportRequest{Folder: out, Name: "my list.json", Replace: true}, 200)
	// The name can't lead elsewhere, and the folder has to be there.
	export("catalog", ExportRequest{Folder: out, Name: "../away"}, 400)
	export("catalog", ExportRequest{Folder: out, Name: ""}, 400)
	export("catalog", ExportRequest{Folder: filepath.Join(out, "missing"), Name: "x"}, 404)

	r = export("snapshot", ExportRequest{Folder: out, Name: "snap"}, 200)
	if want := filepath.Join(out, "snap.zip"); r.Path != want {
		t.Errorf("snapshot = %+v", r)
	}
	if _, err := os.Stat(r.Path); err != nil {
		t.Errorf("no snapshot written: %v", err)
	}
	export("snapshot", ExportRequest{Folder: out, Name: "snap"}, 409)

	// With a catalog file open, there's no list to save or snapshot.
	body, _ = json.Marshal(SourceRequest{Catalog: filepath.Join(out, "my list.json")})
	h.call(200, "PUT", "/api/source", string(body), nil)
	export("catalog", ExportRequest{Folder: out, Name: "again"}, 503)
	export("snapshot", ExportRequest{Folder: out, Name: "again"}, 503)
}
