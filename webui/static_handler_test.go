package webui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestStaticAssetHandler_ServeHTTP(t *testing.T) {
	// Create a test filesystem
	testFS := fstest.MapFS{
		"index.html":         {Data: []byte("<html>Dashboard</html>")},
		"css/dashboard.css":  {Data: []byte("body { color: white; }")},
		"js/dashboard.js":    {Data: []byte("console.log('dashboard');")},
		"js/websocket.js":    {Data: []byte("class WebSocketClient {}")},
		"subdir/index.html":  {Data: []byte("<html>Subdir</html>")},
	}

	config := DefaultStaticAssetConfig()
	handler := NewStaticAssetHandlerWithFS(testFS, config)

	tests := []struct {
		name           string
		path           string
		wantStatus     int
		wantBodyContains string
		wantContentType  string
	}{
		{
			name:             "serve index.html at root",
			path:             "/static/",
			wantStatus:       http.StatusOK,
			wantBodyContains: "Dashboard",
			wantContentType:  "text/html",
		},
		{
			name:             "serve index.html explicitly",
			path:             "/static/index.html",
			wantStatus:       http.StatusOK,
			wantBodyContains: "Dashboard",
			wantContentType:  "text/html",
		},
		{
			name:             "serve CSS file",
			path:             "/static/css/dashboard.css",
			wantStatus:       http.StatusOK,
			wantBodyContains: "color: white",
			wantContentType:  "text/css",
		},
		{
			name:             "serve JS file",
			path:             "/static/js/dashboard.js",
			wantStatus:       http.StatusOK,
			wantBodyContains: "console.log",
			wantContentType:  "javascript", // Go's mime returns "text/javascript"
		},
		{
			name:             "serve subdirectory index",
			path:             "/static/subdir/",
			wantStatus:       http.StatusOK,
			wantBodyContains: "Subdir",
			wantContentType:  "text/html",
		},
		{
			name:             "not found",
			path:             "/static/nonexistent.html",
			wantStatus:       http.StatusNotFound,
			wantBodyContains: "404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()

			// Use StripPrefix like in real usage
			http.StripPrefix("/static", handler).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}

			body, _ := io.ReadAll(rr.Body)
			if !strings.Contains(string(body), tt.wantBodyContains) {
				t.Errorf("body = %q, want to contain %q", string(body), tt.wantBodyContains)
			}

			if tt.wantContentType != "" {
				contentType := rr.Header().Get("Content-Type")
				if !strings.Contains(contentType, tt.wantContentType) {
					t.Errorf("Content-Type = %q, want to contain %q", contentType, tt.wantContentType)
				}
			}
		})
	}
}

func TestStaticAssetHandler_MethodNotAllowed(t *testing.T) {
	testFS := fstest.MapFS{
		"index.html": {Data: []byte("<html></html>")},
	}

	handler := NewStaticAssetHandlerWithFS(testFS, DefaultStaticAssetConfig())

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/static/index.html", nil)
			rr := httptest.NewRecorder()

			http.StripPrefix("/static", handler).ServeHTTP(rr, req)

			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestStaticAssetHandler_HeadMethod(t *testing.T) {
	testFS := fstest.MapFS{
		"index.html": {Data: []byte("<html>Test</html>")},
	}

	handler := NewStaticAssetHandlerWithFS(testFS, DefaultStaticAssetConfig())

	req := httptest.NewRequest(http.MethodHead, "/static/index.html", nil)
	rr := httptest.NewRecorder()

	http.StripPrefix("/static", handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Content-Type = %q, want to contain text/html", contentType)
	}
}

func TestStaticAssetHandler_CustomNotFound(t *testing.T) {
	testFS := fstest.MapFS{}

	customNotFoundCalled := false
	config := DefaultStaticAssetConfig()
	config.NotFoundFunc = func(w http.ResponseWriter, r *http.Request) {
		customNotFoundCalled = true
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Custom 404"))
	}

	handler := NewStaticAssetHandlerWithFS(testFS, config)

	req := httptest.NewRequest(http.MethodGet, "/static/nonexistent.html", nil)
	rr := httptest.NewRecorder()

	http.StripPrefix("/static", handler).ServeHTTP(rr, req)

	if !customNotFoundCalled {
		t.Error("custom not found handler was not called")
	}

	body, _ := io.ReadAll(rr.Body)
	if !strings.Contains(string(body), "Custom 404") {
		t.Errorf("body = %q, want to contain Custom 404", string(body))
	}
}

func TestStaticAssetHandler_PathTraversal(t *testing.T) {
	testFS := fstest.MapFS{
		"index.html": {Data: []byte("<html></html>")},
	}

	handler := NewStaticAssetHandlerWithFS(testFS, DefaultStaticAssetConfig())

	// Test various path traversal attempts
	paths := []string{
		"/static/../etc/passwd",
		"/static/../../etc/passwd",
		"/static/./../../etc/passwd",
		"/static/%2e%2e/etc/passwd",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rr := httptest.NewRecorder()

			http.StripPrefix("/static", handler).ServeHTTP(rr, req)

			// Should return 404, not serve any file outside the FS
			if rr.Code != http.StatusNotFound {
				t.Errorf("path %q: status = %d, want %d", path, rr.Code, http.StatusNotFound)
			}
		})
	}
}

func TestStaticAssetHandler_DetectContentType(t *testing.T) {
	handler := &StaticAssetHandler{}

	tests := []struct {
		path     string
		wantType string
	}{
		{"/index.html", "text/html"},
		{"/style.css", "text/css"},
		{"/app.js", "javascript"},           // Go's mime returns "text/javascript"
		{"/data.json", "application/json"},
		{"/image.png", "image/png"},
		{"/image.jpg", "image/jpeg"},
		{"/image.svg", "image/svg+xml"},
		{"/icon.ico", "image"},              // Go's mime returns "image/vnd.microsoft.icon"
		{"/font.woff", "font/woff"},
		{"/font.woff2", "font/woff2"},
		{"/unknown.xyz", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := handler.detectContentType(tt.path)
			if !strings.Contains(got, tt.wantType) {
				t.Errorf("detectContentType(%q) = %q, want to contain %q", tt.path, got, tt.wantType)
			}
		})
	}
}

func TestDefaultStaticAssetConfig(t *testing.T) {
	config := DefaultStaticAssetConfig()

	if config.Prefix != "/static" {
		t.Errorf("Prefix = %q, want /static", config.Prefix)
	}

	if config.IndexFile != "index.html" {
		t.Errorf("IndexFile = %q, want index.html", config.IndexFile)
	}

	if !config.EnableCache {
		t.Error("EnableCache = false, want true")
	}

	if config.CacheMaxAge != 3600 {
		t.Errorf("CacheMaxAge = %d, want 3600", config.CacheMaxAge)
	}
}

func TestStaticAssetHandler_ServeDashboard(t *testing.T) {
	// This test uses the actual embedded FS
	config := DefaultStaticAssetConfig()
	handler := NewStaticAssetHandler(config)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rr := httptest.NewRecorder()

	handler.ServeDashboard().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Content-Type = %q, want to contain text/html", contentType)
	}

	body, _ := io.ReadAll(rr.Body)
	if !strings.Contains(string(body), "CanvusLocalLLM Dashboard") {
		t.Errorf("body should contain dashboard title")
	}
}
