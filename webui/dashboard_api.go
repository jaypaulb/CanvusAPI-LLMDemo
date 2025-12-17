// Package webui provides the DashboardAPI organism for REST API handlers.
// This file contains handlers for the dashboard API endpoints that serve
// metrics, status, and task information to the web UI.
package webui

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"go_backend/metrics"
)

// DashboardAPI is an organism that provides REST API handlers for the dashboard.
// It composes the MetricsStore for data access and provides JSON responses
// for the frontend dashboard.
//
// Endpoints:
// - GET /api/status    - System health status
// - GET /api/canvases  - All canvas statuses
// - GET /api/tasks     - Recent task records (with limit param)
// - GET /api/metrics   - Task processing metrics
// - GET /api/gpu       - GPU metrics (with optional history param)
type DashboardAPI struct {
	store        metrics.MetricsCollector
	gpuCollector *metrics.GPUCollector
	defaultLimit int
	maxLimit     int
	versionInfo  VersionInfo
}

// VersionInfo contains version metadata for the status endpoint.
type VersionInfo struct {
	Version   string `json:"version"`
	BuildDate string `json:"build_date,omitempty"`
	GitCommit string `json:"git_commit,omitempty"`
}

// DashboardAPIConfig configures the DashboardAPI behavior.
type DashboardAPIConfig struct {
	// DefaultLimit is the default number of items to return in list endpoints
	DefaultLimit int

	// MaxLimit is the maximum number of items that can be requested
	MaxLimit int

	// VersionInfo contains application version metadata
	VersionInfo VersionInfo
}

// DefaultDashboardAPIConfig returns a default configuration.
func DefaultDashboardAPIConfig() DashboardAPIConfig {
	return DashboardAPIConfig{
		DefaultLimit: 20,
		MaxLimit:     100,
		VersionInfo: VersionInfo{
			Version: "0.0.0",
		},
	}
}

// NewDashboardAPI creates a new DashboardAPI with the specified configuration.
// The store parameter provides access to metrics data.
// The gpuCollector parameter is optional and enables GPU metrics endpoints.
func NewDashboardAPI(store metrics.MetricsCollector, gpuCollector *metrics.GPUCollector, config DashboardAPIConfig) *DashboardAPI {
	if config.DefaultLimit < 1 {
		config.DefaultLimit = 20
	}
	if config.MaxLimit < 1 {
		config.MaxLimit = 100
	}

	return &DashboardAPI{
		store:        store,
		gpuCollector: gpuCollector,
		defaultLimit: config.DefaultLimit,
		maxLimit:     config.MaxLimit,
		versionInfo:  config.VersionInfo,
	}
}

// StatusResponse represents the JSON response for /api/status.
type StatusResponse struct {
	Health     string    `json:"health"`
	Version    string    `json:"version"`
	BuildDate  string    `json:"build_date,omitempty"`
	GitCommit  string    `json:"git_commit,omitempty"`
	Uptime     string    `json:"uptime"`
	UptimeSecs float64   `json:"uptime_secs"`
	LastCheck  time.Time `json:"last_check"`
	GPUAvail   bool      `json:"gpu_available"`
}

// HandleStatus handles GET /api/status requests.
func (api *DashboardAPI) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	status := api.store.GetSystemStatus()

	gpuAvail := false
	if api.gpuCollector != nil {
		gpuAvail = api.gpuCollector.IsAvailable()
	}

	response := StatusResponse{
		Health:     status.Health,
		Version:    api.versionInfo.Version,
		BuildDate:  api.versionInfo.BuildDate,
		GitCommit:  api.versionInfo.GitCommit,
		Uptime:     formatDuration(status.Uptime),
		UptimeSecs: status.Uptime.Seconds(),
		LastCheck:  status.LastCheck,
		GPUAvail:   gpuAvail,
	}

	api.writeJSON(w, http.StatusOK, response)
}

// CanvasesResponse represents the JSON response for /api/canvases.
type CanvasesResponse struct {
	Canvases []metrics.CanvasStatus `json:"canvases"`
	Count    int                    `json:"count"`
}

// HandleCanvases handles GET /api/canvases requests.
func (api *DashboardAPI) HandleCanvases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	canvases := api.store.GetAllCanvasStatuses()

	response := CanvasesResponse{
		Canvases: canvases,
		Count:    len(canvases),
	}

	api.writeJSON(w, http.StatusOK, response)
}

// TasksResponse represents the JSON response for /api/tasks.
type TasksResponse struct {
	Tasks []metrics.TaskRecord `json:"tasks"`
	Count int                  `json:"count"`
	Limit int                  `json:"limit"`
}

// HandleTasks handles GET /api/tasks requests.
// Query parameters:
// - limit: number of tasks to return (default: 20, max: 100)
func (api *DashboardAPI) HandleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	limit := api.defaultLimit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	if limit > api.maxLimit {
		limit = api.maxLimit
	}

	tasks := api.store.GetRecentTasks(limit)

	response := TasksResponse{
		Tasks: tasks,
		Count: len(tasks),
		Limit: limit,
	}

	api.writeJSON(w, http.StatusOK, response)
}

// MetricsResponse represents the JSON response for /api/metrics.
type MetricsResponse struct {
	TotalProcessed int64                               `json:"total_processed"`
	TotalSuccess   int64                               `json:"total_success"`
	TotalErrors    int64                               `json:"total_errors"`
	SuccessRate    float64                             `json:"success_rate"`
	ByType         map[string]*metrics.TaskTypeMetrics `json:"by_type"`
}

// HandleMetrics handles GET /api/metrics requests.
func (api *DashboardAPI) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	taskMetrics := api.store.GetTaskMetrics()

	var successRate float64
	if taskMetrics.TotalProcessed > 0 {
		successRate = float64(taskMetrics.TotalSuccess) / float64(taskMetrics.TotalProcessed) * 100
	}

	response := MetricsResponse{
		TotalProcessed: taskMetrics.TotalProcessed,
		TotalSuccess:   taskMetrics.TotalSuccess,
		TotalErrors:    taskMetrics.TotalErrors,
		SuccessRate:    successRate,
		ByType:         taskMetrics.ByType,
	}

	api.writeJSON(w, http.StatusOK, response)
}

// GPUResponse represents the JSON response for /api/gpu.
type GPUResponse struct {
	Available   bool                 `json:"available"`
	Current     *metrics.GPUMetrics  `json:"current,omitempty"`
	History     []metrics.GPUMetrics `json:"history,omitempty"`
	HistorySize int                  `json:"history_size,omitempty"`
	Error       string               `json:"error,omitempty"`
}

// HandleGPU handles GET /api/gpu requests.
// Query parameters:
// - history: number of historical samples to include (default: 0)
func (api *DashboardAPI) HandleGPU(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if api.gpuCollector == nil {
		response := GPUResponse{
			Available: false,
			Error:     "GPU monitoring not configured",
		}
		api.writeJSON(w, http.StatusOK, response)
		return
	}

	available := api.gpuCollector.IsAvailable()

	response := GPUResponse{
		Available: available,
	}

	if available {
		current := api.gpuCollector.GetCurrentMetrics()
		response.Current = &current

		// Check for history parameter
		if historyStr := r.URL.Query().Get("history"); historyStr != "" {
			if historyLimit, err := strconv.Atoi(historyStr); err == nil && historyLimit > 0 {
				history := api.gpuCollector.GetHistory(historyLimit)
				response.History = history
				response.HistorySize = len(history)
			}
		}
	} else {
		if err := api.gpuCollector.GetLastError(); err != nil {
			response.Error = err.Error()
		}
	}

	api.writeJSON(w, http.StatusOK, response)
}

// RegisterRoutes registers all API routes on the given ServeMux.
func (api *DashboardAPI) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/status", api.HandleStatus)
	mux.HandleFunc("/api/canvases", api.HandleCanvases)
	mux.HandleFunc("/api/tasks", api.HandleTasks)
	mux.HandleFunc("/api/metrics", api.HandleMetrics)
	mux.HandleFunc("/api/gpu", api.HandleGPU)
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// writeJSON writes a JSON response with the given status code.
func (api *DashboardAPI) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Best effort - headers already written
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// writeError writes an error response.
func (api *DashboardAPI) writeError(w http.ResponseWriter, status int, message string) {
	response := ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	}
	api.writeJSON(w, status, response)
}

// formatDuration formats a duration into a human-readable string.
// This is a local helper that formats durations for the API.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return d.Round(time.Second).String()
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return strconv.Itoa(hours) + "h" + strconv.Itoa(minutes) + "m" + strconv.Itoa(seconds) + "s"
	}

	return strconv.Itoa(minutes) + "m" + strconv.Itoa(seconds) + "s"
}
