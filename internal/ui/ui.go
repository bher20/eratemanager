package ui

import (
    "embed"
    "io/fs"
    "net/http"
)

// content embeds all static assets for the web UI.
//
//go:embed static/*
var content embed.FS

// Handler returns an http.Handler that serves the embedded UI assets under /.
func Handler() http.Handler {
    sub, err := fs.Sub(content, "static")
    if err != nil {
        // This should never happen in a correctly built binary.
        panic(err)
    }
    return http.FileServer(http.FS(sub))
}
