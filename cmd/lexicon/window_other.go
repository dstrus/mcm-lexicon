//go:build !windows || !window

// Without -tags window, or on macOS and Linux, MCM Lexicon opens in the
// browser and its console window stops it (window_windows.go has the
// window).

package main

const windowed = false

func runWindow(address, token string) {}
func focusWindow() bool               { return false }
func showError(msg string) bool       { return false }
