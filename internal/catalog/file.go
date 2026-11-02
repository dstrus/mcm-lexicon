package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"lexicon/internal/apperr"
	"lexicon/internal/textfile"
)

// ReadFile opens a catalog file. One written by a newer MCM Lexicon is
// refused, since it may hold what this one would misread.
func ReadFile(path string) (*Catalog, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, apperr.New(apperr.Invalid, "Couldn't read %s: %v", path, err)
	}
	var c Catalog
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, apperr.New(apperr.Invalid, "%s isn't a catalog file: %v", path, err)
	}
	switch {
	case c.SchemaVersion == 0:
		return nil, apperr.New(apperr.Invalid, "%s isn't a catalog file MCM Lexicon wrote", path)
	case c.SchemaVersion > SchemaVersion:
		return nil, apperr.New(apperr.Invalid, "%s was written by a newer MCM Lexicon (catalog version %d; this one reads up to %d)", path, c.SchemaVersion, SchemaVersion)
	}
	if c.Menus == nil {
		c.Menus = []Menu{}
	}
	if c.Skipped == nil {
		c.Skipped = []Skipped{}
	}
	return &c, nil
}

// Marshal is the catalog as a file holds it: indented JSON.
func Marshal(c *Catalog) ([]byte, error) {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// WriteFile writes the catalog to path, replacing it in one step.
func WriteFile(path string, c *Catalog) error {
	if !strings.HasSuffix(strings.ToLower(path), ".json") {
		return fmt.Errorf("a catalog file's name ends in .json: %s", path)
	}
	b, err := Marshal(c)
	if err != nil {
		return err
	}
	return textfile.WriteAtomic(path, b, 0o644)
}
