package server

import (
	"lexicon/internal/catalog"
	"lexicon/internal/modlist"
)

// The server's own requests and replies; each domain declares its own.

// AppID identifies MCM Lexicon to /api/ping, so a second copy can find this one.
const AppID = "lexicon"

// TokenHeader carries the per-session token on every API request.
const TokenHeader = "X-Lexicon-Token"

type Ping struct {
	App string `json:"app"`
}

// Status is what the page needs to know before showing anything: which mod
// list (or catalog file) is open, and whether anything can be written.
type Status struct {
	Source      *modlist.Source `json:"source,omitempty"` // the list being read; absent until one is chosen
	CatalogFile string          `json:"catalogFile"`      // a catalog opened instead of a list; "" otherwise
	Scan        catalog.State   `json:"scan"`             // the catalog: scanning, when it was scanned, and any failure
	GameRunning bool            `json:"gameRunning"`
	Version     string          `json:"version"` // MCM Lexicon's own
	Theme       string          `json:"theme"`   // "system", "dark" or "light"
	// OpenRecording is the recording new steps go to, by name; "" for none.
	OpenRecording string `json:"openRecording"`
}

// OpenRequest chooses the open recording; "" closes it.
type OpenRequest struct {
	Name string `json:"name"`
}

// SourceRequest chooses what to read: a mod list, or a catalog file.
type SourceRequest struct {
	Path    string `json:"path,omitempty"`    // a Mod Organizer 2 instance, or the game's Data folder
	Profile string `json:"profile,omitempty"` // an MO2 profile; "" means the one MO2 has selected
	Catalog string `json:"catalog,omitempty"` // a catalog file to open instead of a list
}

type ThemeRequest struct {
	Theme string `json:"theme"` // "system", "dark" or "light"
}

func (s *Server) status() Status {
	source, file := s.Source()
	return Status{
		Source:        source,
		CatalogFile:   file,
		Scan:          s.catalog.State(),
		GameRunning:   s.Running != nil && s.Running(),
		Version:       s.Version,
		Theme:         s.Prefs.Get().ThemeOf(),
		OpenRecording: s.Prefs.Get().OpenRecording,
	}
}
