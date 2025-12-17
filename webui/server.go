// Package webui provides the web-based user interface for CanvusLocalLLM.
// This file contains the WebUIServer organism that wires together all web UI components.
package webui

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go_backend/metrics"
	"go_backend/webui/static"

	"go.uber.org/zap"
)

// AuthProvider is an interface for authentication middleware.
// This interface is implemented by auth.AuthMiddleware and allows
// the server to be decoupled from the auth package to avoid import cycles.
type AuthProvider interface {
	// Middleware wraps an http.Handler with authentication
	Middleware(next http.Handler) http.Handler
	// MiddlewareFunc wraps an http.HandlerFunc with authentication
	MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc
	// LoginHandler returns a handler for the login page
	LoginHandler() http.HandlerFunc
	// LogoutHandler returns a handler for logout
	LogoutHandler() http.HandlerFunc
}

// WebUIServer is the main HTTP server organism for the dashboard.
// It wires together:
//   - StaticAssetHandler for serving embedded static files
//   - AuthProvider for session-based authentication (optional)
//   - LoggingMiddleware for request logging
//   - DashboardAPI for REST API endpoints
//   - WebSocketBroadcaster for real-time updates
//
// Methods:
//   - NewServer() creates a configured server instance
//   - Start() begins listening on the configured port
//   - Shutdown() gracefully shuts down the server
type WebUIServer struct {
	httpServer    *http.Server
	mux           *http.ServeMux
	config        ServerConfig
	logger        *zap.Logger
	authProvider  AuthProvider
	loggingMw     *LoggingMiddleware
	dashboardAPI  *DashboardAPI
	wsBroadcaster *WebSocketBroadcaster
	staticHandler *StaticAssetHandler
}

// ServerConfig configures the WebUIServer.
type ServerConfig struct {
	// Port to listen on (default: 3000)
	Port int

	// Host to bind to (default: "localhost")
	Host string

	// ReadTimeout for HTTP requests (default: 30s)
	ReadTimeout time.Duration

	// WriteTimeout for HTTP responses (default: 30s)
	WriteTimeout time.Duration

	// IdleTimeout for keep-alive connections (default: 120s)
	IdleTimeout time.Duration

	// ShutdownTimeout for graceful shutdown (default: 30s)
	ShutdownTimeout time.Duration

	// StaticConfig for static asset handler
	StaticConfig StaticAssetConfig

	// LogSkipPaths are paths to skip logging
	LogSkipPaths []string

	// VersionInfo for API responses
	VersionInfo VersionInfo
}

// DefaultServerConfig returns a ServerConfig with sensible defaults.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Port:            3000,
		Host:            "localhost",
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		StaticConfig:    DefaultStaticAssetConfig(),
		LogSkipPaths:    []string{"/health", "/api/status"},
		VersionInfo: VersionInfo{
			Version: "1.0.0",
		},
	}
}

// NewServer creates a new WebUIServer with the given configuration.
// It wires together all the middleware and handlers.
// The authProvider is optional and can be nil for unauthenticated servers.
func NewServer(
	config ServerConfig,
	metricsStore metrics.MetricsCollector,
	gpuCollector *metrics.GPUCollector,
	authProvider AuthProvider,
	logger *zap.Logger,
) (*WebUIServer, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Create the mux
	mux := http.NewServeMux()

	// Create static asset handler
	staticHandler := NewStaticAssetHandler(config.StaticConfig)

	// Create logging middleware
	loggingConfig := LoggingMiddlewareConfig{
		SkipPaths: config.LogSkipPaths,
	}
	loggingMw := NewLoggingMiddlewareWithConfig(loggingConfig)

	// Create dashboard API
	apiConfig := DashboardAPIConfig{
		DefaultLimit: 20,
		MaxLimit:     100,
		VersionInfo:  config.VersionInfo,
	}
	dashboardAPI := NewDashboardAPI(metricsStore, gpuCollector, apiConfig)

	// Create WebSocket broadcaster
	wsBroadcaster := NewWebSocketBroadcaster()

	server := &WebUIServer{
		mux:           mux,
		config:        config,
		logger:        logger,
		authProvider:  authProvider,
		loggingMw:     loggingMw,
		dashboardAPI:  dashboardAPI,
		wsBroadcaster: wsBroadcaster,
		staticHandler: staticHandler,
	}

	// Setup routes
	server.setupRoutes()

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	server.httpServer = &http.Server{
		Addr:         addr,
		Handler:      server.rootHandler(),
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	logger.Info("WebUI server created",
		zap.String("addr", addr),
		zap.Bool("auth_enabled", authProvider != nil),
	)

	return server, nil
}

// setupRoutes configures all the HTTP routes.
func (s *WebUIServer) setupRoutes() {
	// Health check endpoint (no auth required)
	s.mux.HandleFunc("/health", s.handleHealth)

	// Static assets
	s.staticHandler.RegisterRoutes(s.mux)

	// Dashboard (serves index.html)
	s.mux.HandleFunc("/dashboard", s.staticHandler.ServeDashboard())
	s.mux.HandleFunc("/dashboard/", s.staticHandler.ServeDashboard())

	// API endpoints
	s.dashboardAPI.RegisterRoutes(s.mux)

	// WebSocket endpoint
	s.mux.HandleFunc("/ws", s.wsBroadcaster.HandleConnection)

	// Auth routes (if enabled)
	if s.authProvider != nil {
		s.mux.HandleFunc("/login", s.authProvider.LoginHandler())
		s.mux.HandleFunc("/logout", s.authProvider.LogoutHandler())
	}

	// Root redirect
	s.mux.HandleFunc("/", s.handleRoot)
}

// rootHandler wraps the mux with middleware.
func (s *WebUIServer) rootHandler() http.Handler {
	var handler http.Handler = s.mux

	// Apply logging middleware
	handler = s.loggingMw.Handler(handler)

	return handler
}

// handleRoot handles requests to the root path.
func (s *WebUIServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	// Only handle exact root path
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Redirect to dashboard or login
	if s.authProvider != nil {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
	} else {
		http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
	}
}

// handleHealth handles health check requests.
func (s *WebUIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// Start begins listening for HTTP requests.
// It starts the WebSocket broadcaster and the HTTP server.
// This method blocks until the server is shut down.
func (s *WebUIServer) Start(ctx context.Context) error {
	// Start WebSocket broadcaster
	go s.wsBroadcaster.Start(ctx)

	s.logger.Info("WebUI server starting",
		zap.String("addr", s.httpServer.Addr),
	)

	// Start HTTP server
	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server error: %w", err)
	}

	return nil
}

// StartTLS begins listening for HTTPS requests.
func (s *WebUIServer) StartTLS(ctx context.Context, certFile, keyFile string) error {
	// Start WebSocket broadcaster
	go s.wsBroadcaster.Start(ctx)

	s.logger.Info("WebUI server starting with TLS",
		zap.String("addr", s.httpServer.Addr),
	)

	err := s.httpServer.ListenAndServeTLS(certFile, keyFile)
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("https server error: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server.
func (s *WebUIServer) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down WebUI server")

	// Create a timeout context if not provided
	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
	defer cancel()

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown error: %w", err)
	}

	s.logger.Info("WebUI server stopped")
	return nil
}

// GetBroadcaster returns the WebSocket broadcaster for sending messages.
func (s *WebUIServer) GetBroadcaster() *WebSocketBroadcaster {
	return s.wsBroadcaster
}

// GetDashboardAPI returns the dashboard API for direct access.
func (s *WebUIServer) GetDashboardAPI() *DashboardAPI {
	return s.dashboardAPI
}

// Addr returns the server's address.
func (s *WebUIServer) Addr() string {
	return s.httpServer.Addr
}

// ProtectHandler wraps a handler with auth middleware if enabled.
func (s *WebUIServer) ProtectHandler(handler http.Handler) http.Handler {
	if s.authProvider != nil {
		return s.authProvider.Middleware(handler)
	}
	return handler
}

// ProtectHandlerFunc wraps a handler function with auth middleware if enabled.
func (s *WebUIServer) ProtectHandlerFunc(handler http.HandlerFunc) http.HandlerFunc {
	if s.authProvider != nil {
		return s.authProvider.MiddlewareFunc(handler)
	}
	return handler
}

// ServeEmbeddedFile serves a specific file from the embedded filesystem.
func (s *WebUIServer) ServeEmbeddedFile(w http.ResponseWriter, name string) {
	data, err := static.ReadFile(name)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Detect content type
	contentType := s.staticHandler.detectContentType(name)
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// HasAuth returns whether authentication is enabled.
func (s *WebUIServer) HasAuth() bool {
	return s.authProvider != nil
}
