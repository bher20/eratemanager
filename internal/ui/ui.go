package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// content embeds all static assets for the web UI.
//
//go:embed static/*
var content embed.FS

// Handler returns an http.Handler that serves the embedded UI assets under /.
// For the React SPA, it falls back to index.html for client-side routing.
func Handler() http.Handler {
	sub, err := fs.Sub(content, "static")
	if err != nil {
		// This should never happen in a correctly built binary.
		panic(err)
	}

	// Check if React app exists
	reactFS, reactErr := fs.Sub(sub, "react-app")
	if reactErr == nil {
		// Serve React app with SPA fallback
		return &spaHandler{fs: http.FS(reactFS)}
	}

	// Fallback to legacy static files
	return http.FileServer(http.FS(sub))
}

// spaHandler serves a Single Page Application with fallback to index.html
type spaHandler struct {
	fs http.FileSystem
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean the path
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
	}
	upath = path.Clean(upath)

	// Try to open the file
	f, err := h.fs.Open(upath)
	if err != nil {
		// If file not found and it's not an asset, serve index.html for SPA routing
		if !strings.Contains(upath, ".") {
			// Serve index.html for client-side routing
			r.URL.Path = "/index.html"
			http.FileServer(h.fs).ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
		return
	}
	f.Close()

	// File exists, serve it
	http.FileServer(h.fs).ServeHTTP(w, r)
}
