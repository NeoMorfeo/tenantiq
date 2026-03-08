package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed dist/*
var distFS embed.FS

// Handler returns an http.Handler that serves the embedded SPA.
// API routes (/api, /openapi.json, /docs) are not handled here —
// they fall through to the next handler in the chain.
// All other routes serve the SPA with fallback to index.html
// so that client-side routing (TanStack Router) works.
func Handler() http.Handler {
	sub, _ := fs.Sub(distFS, "dist")
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the exact file (JS, CSS, images, etc.)
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		if _, err := fs.Stat(sub, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// File not found → serve index.html (SPA fallback).
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
