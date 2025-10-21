//go:build !dev

package main

import "net/http"

const IsDev = false

// initDevMode is a stub function for release builds. It does nothing.
func initDevMode(mux *http.ServeMux, htmlFileDir string) {
	// This function is intentionally left empty.
	// The actual implementation for development mode is in development.go.
}
