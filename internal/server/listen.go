package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// Listen binds to localhost on the first free port in ports, or any free
// port if they're all taken. If one of them is a copy of MCM Lexicon that's
// already running, it returns that copy's address instead, so it's reused
// rather than having two copies editing the same files.
func Listen(ports []int) (ln net.Listener, running string, err error) {
	for _, p := range ports {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err == nil {
			return ln, "", nil
		}
		if isLexicon(p) {
			return nil, fmt.Sprintf("http://127.0.0.1:%d/", p), nil
		}
	}
	ln, err = net.Listen("tcp", "127.0.0.1:0")
	return ln, "", err
}

// isLexicon reports whether the server on port answers like MCM Lexicon.
func isLexicon(port int) bool {
	// A running copy answers at once; anything slower isn't one.
	client := http.Client{Timeout: 300 * time.Millisecond}
	res, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/api/ping", port))
	if err != nil {
		return false
	}
	defer res.Body.Close()
	var reply Ping
	return res.StatusCode == http.StatusOK && json.NewDecoder(io.LimitReader(res.Body, 1024)).Decode(&reply) == nil && reply.App == AppID
}
