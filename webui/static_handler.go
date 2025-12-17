// Package webui provides web UI handlers including static asset serving.
package webui

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"go_backend/webui/static"
)

// StaticAssetHandler is a molecule that serves embedded static assets.
// It handles proper MIME type detection and caching headers.
type StaticAssetHandler struct {
	fs           fs.FS
	prefix       string
	indexFile    string
	enableCache  bool
	cacheMaxAge  int
	notFoundFunc http.HandlerFunc
}

// StaticAssetConfig configures the StaticAssetHandler.
type StaticAssetConfig struct {
	// Prefix is the URL prefix for static assets (default: "/static")
	Prefix string

	// IndexFile is the default file to serve for directory requests (default: "index.html")
	IndexFile string

	// EnableCache enables cache headers (default: true)
	EnableCache bool

	// CacheMaxAge is the max-age in seconds for cache headers (default: 3600)
	CacheMaxAge int

	// NotFoundFunc is called when a file is not found (default: http.NotFound)
	NotFoundFunc http.HandlerFunc
}

// DefaultStaticAssetConfig returns a default configuration.
func DefaultStaticAssetConfig() StaticAssetConfig {
	return StaticAssetConfig{
		Prefix:      "/static",
		IndexFile:   "index.html",
		EnableCache: true,
		CacheMaxAge: 3600,
	}
}

// NewStaticAssetHandler creates a new static asset handler using the embedded filesystem.
func NewStaticAssetHandler(config StaticAssetConfig) *StaticAssetHandler {
	if config.Prefix == "" {
		config.Prefix = "/static"
	}
	if config.IndexFile == "" {
		config.IndexFile = "index.html"
	}
	if config.CacheMaxAge == 0 {
		config.CacheMaxAge = 3600
	}

	return &StaticAssetHandler{
		fs:           static.GetFS(),
		prefix:       config.Prefix,
		indexFile:    config.IndexFile,
		enableCache:  config.EnableCache,
		cacheMaxAge:  config.CacheMaxAge,
		notFoundFunc: config.NotFoundFunc,
	}
}

// NewStaticAssetHandlerWithFS creates a handler with a custom filesystem.
// Useful for testing or using external files.
func NewStaticAssetHandlerWithFS(fsys fs.FS, config StaticAssetConfig) *StaticAssetHandler {
	h := NewStaticAssetHandler(config)
	h.fs = fsys
	return h
}

// ServeHTTP implements http.Handler for serving static assets.
func (h *StaticAssetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only allow GET and HEAD methods
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the file path from the URL
	urlPath := r.URL.Path

	// Strip the prefix if present
	if h.prefix != "" && strings.HasPrefix(urlPath, h.prefix) {
		urlPath = strings.TrimPrefix(urlPath, h.prefix)
	}

	// Clean the path to prevent directory traversal
	urlPath = path.Clean("/" + urlPath)
	urlPath = strings.TrimPrefix(urlPath, "/")

	// If empty or root, serve index file
	if urlPath == "" || urlPath == "." {
		urlPath = h.indexFile
	}

	// Try to open the file
	file, err := h.fs.Open(urlPath)
	if err != nil {
		h.handleNotFound(w, r)
		return
	}
	defer file.Close()

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		h.handleNotFound(w, r)
		return
	}

	// If it's a directory, try to serve index file
	if stat.IsDir() {
		indexPath := path.Join(urlPath, h.indexFile)
		indexFile, err := h.fs.Open(indexPath)
		if err != nil {
			h.handleNotFound(w, r)
			return
		}
		defer indexFile.Close()

		stat, err = indexFile.Stat()
		if err != nil {
			h.handleNotFound(w, r)
			return
		}
		urlPath = indexPath
		file = indexFile
	}

	// Determine content type
	contentType := h.detectContentType(urlPath)
	w.Header().Set("Content-Type", contentType)

	// Set cache headers
	if h.enableCache {
		w.Header().Set("Cache-Control", "public, max-age="+string(rune(h.cacheMaxAge)))
	} else {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	}

	// Cast to ReadSeeker if possible (embed.FS files support this)
	if seeker, ok := file.(fs.File); ok {
		if rs, ok := seeker.(http.File); ok {
			http.ServeContent(w, r, stat.Name(), stat.ModTime(), rs)
			return
		}
	}

	// Fallback: read and write
	data, err := fs.ReadFile(h.fs, urlPath)
	if err != nil {
		h.handleNotFound(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// Handler returns an http.Handler for use with mux.Handle.
func (h *StaticAssetHandler) Handler() http.Handler {
	return h
}

// StripPrefix returns a handler that strips the prefix before serving.
// Useful when mounting under a specific path.
func (h *StaticAssetHandler) StripPrefix() http.Handler {
	return http.StripPrefix(h.prefix, h)
}

// detectContentType determines the MIME type based on file extension.
func (h *StaticAssetHandler) detectContentType(filePath string) string {
	ext := filepath.Ext(filePath)

	// Check standard mime types first
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}

	// Fallback for common web types
	switch strings.ToLower(ext) {
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	case ".otf":
		return "font/otf"
	case ".map":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

// handleNotFound handles file not found responses.
func (h *StaticAssetHandler) handleNotFound(w http.ResponseWriter, r *http.Request) {
	if h.notFoundFunc != nil {
		h.notFoundFunc(w, r)
		return
	}
	http.NotFound(w, r)
}

// RegisterRoutes registers the static handler on a ServeMux.
func (h *StaticAssetHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle(h.prefix+"/", h.StripPrefix())
}

// ServeDashboard creates a handler that serves the dashboard index.html.
// This is useful for SPA-style routing where all paths should serve the index.
func (h *StaticAssetHandler) ServeDashboard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := static.ReadFile("index.html")
		if err != nil {
			http.Error(w, "Dashboard not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if !h.enableCache {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}
}
