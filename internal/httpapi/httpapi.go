// Package httpapi turns typed Go functions into JSON handlers, so each
// domain's routes declare their request and response types instead of
// passing maps around. Checking the host and token is the server's job,
// not this package's.
package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"lexicon/internal/apperr"
)

// None is the request type of a handler that takes no body.
type None struct{}

// maxBody bounds request bodies; the largest legitimate ones are prompt and
// bio texts.
const maxBody = 16 << 20

// JSON wraps fn as a handler. A request body, if there is one, is decoded
// into Req; fn's result is sent as JSON, and its error as {"error": "…"} with
// a status matching the error's kind.
func JSON[Req, Resp any](fn func(r *http.Request, req Req) (Resp, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Req
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			var tooBig *http.MaxBytesError
			if !errors.As(err, &tooBig) {
				err = apperr.New(apperr.Invalid, "Invalid request: %v", err)
			}
			SendError(w, err)
			return
		}
		resp, err := fn(r, req)
		if err != nil {
			SendError(w, err)
			return
		}
		Send(w, http.StatusOK, resp)
	})
}

func Send(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) // escapes <, > and & by default
}

// Error is the body of every failed API request.
type Error struct {
	Error string `json:"error"`
}

var statusOf = map[apperr.Kind]int{
	apperr.Invalid:     http.StatusBadRequest,
	apperr.NotFound:    http.StatusNotFound,
	apperr.Conflict:    http.StatusConflict,
	apperr.Unavailable: http.StatusServiceUnavailable,
}

// SendError reports err. Errors that aren't meant for the user are logged
// and still shown, since the user is the only one there to read them.
func SendError(w http.ResponseWriter, err error) {
	if kind, ok := apperr.KindOf(err); ok {
		Send(w, statusOf[kind], Error{err.Error()})
		return
	}
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		Send(w, http.StatusRequestEntityTooLarge, Error{"Request too large"})
		return
	}
	log.Printf("Error: %v", err)
	Send(w, http.StatusInternalServerError, Error{err.Error()})
}
