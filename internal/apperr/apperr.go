// Package apperr holds errors the person using MCM Lexicon should see, each
// with the kind of failure it is. Domain packages return them without
// knowing about HTTP; the server turns each kind into a status code.
package apperr

import (
	"errors"
	"fmt"
)

type Kind int

const (
	Invalid     Kind = iota + 1 // the request itself is wrong
	NotFound                    // what it names doesn't exist
	Conflict                    // the request is fine, but the current state refuses it, e.g. Skyrim is running
	Unavailable                 // MCM Lexicon isn't set up for it yet, e.g. no mod list chosen
)

type Error struct {
	Kind Kind
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

func New(kind Kind, format string, a ...any) error {
	return &Error{kind, fmt.Sprintf(format, a...)}
}

// KindOf reports the kind of err, or ok=false if it isn't meant for the user.
func KindOf(err error) (kind Kind, ok bool) {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind, true
	}
	return 0, false
}
