// Package static provides embedded static assets for the web UI.
// This file uses Go's embed directive to bundle static files into the binary.
package static

import (
	"embed"
	"io/fs"
)

// StaticFS contains all embedded static assets for the web UI.
// This includes:
// - index.html (dashboard)
// - css/dashboard.css (dark theme styling)
// - js/websocket.js (WebSocket client)
// - js/dashboard.js (main dashboard application)
//
//go:embed index.html css js
var StaticFS embed.FS

// GetFS returns the embedded filesystem.
// This can be used with http.FileServer for serving static assets.
func GetFS() fs.FS {
	return StaticFS
}

// GetSubFS returns a sub-filesystem rooted at the given directory.
// Returns the root FS if an error occurs.
func GetSubFS(dir string) fs.FS {
	subFS, err := fs.Sub(StaticFS, dir)
	if err != nil {
		return StaticFS
	}
	return subFS
}

// MustGetSubFS returns a sub-filesystem rooted at the given directory.
// Panics if the directory doesn't exist.
func MustGetSubFS(dir string) fs.FS {
	subFS, err := fs.Sub(StaticFS, dir)
	if err != nil {
		panic("static: failed to get sub-filesystem: " + err.Error())
	}
	return subFS
}

// ReadFile reads a file from the embedded filesystem.
func ReadFile(name string) ([]byte, error) {
	return StaticFS.ReadFile(name)
}

// ReadFileString reads a file from the embedded filesystem and returns it as a string.
func ReadFileString(name string) (string, error) {
	data, err := StaticFS.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
