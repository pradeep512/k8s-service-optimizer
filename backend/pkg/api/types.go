package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// Config holds the server configuration
type Config struct {
	Port           string
	EnableCORS     bool
	LogLevel       string
	UpdateInterval time.Duration // For WebSocket updates (e.g., 5s)
}

// APIResponse is the standard response wrapper for all API endpoints
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError represents an error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status string `json:"status"`
}

// ReadyResponse represents the readiness check response
type ReadyResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// StatusResponse represents the system status
type StatusResponse struct {
	Version      string    `json:"version"`
	Uptime       string    `json:"uptime"`
	CollectorRunning bool  `json:"collector_running"`
	Timestamp    time.Time `json:"timestamp"`
}

// TimeSeriesQueryParams represents query parameters for time series data
type TimeSeriesQueryParams struct {
	Resource string        `json:"resource"`
	Metric   string        `json:"metric"`
	Duration time.Duration `json:"duration"`
}

// AnomalyQueryParams represents query parameters for anomaly detection
type AnomalyQueryParams struct {
	Resource string        `json:"resource"`
	Duration time.Duration `json:"duration"`
}

// ApplyRecommendationResponse represents the response for applying a recommendation
type ApplyRecommendationResponse struct {
	Status string `json:"status"`
	ID     string `json:"id"`
	Message string `json:"message,omitempty"`
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// Helper functions for consistent response handling

// respondWithSuccess sends a successful response with data
func respondWithSuccess(w http.ResponseWriter, data interface{}) {
	response := APIResponse{
		Success: true,
		Data:    data,
	}
	respondWithJSON(w, http.StatusOK, response)
}

// respondWithError sends an error response
func respondWithError(w http.ResponseWriter, statusCode int, code string, message string) {
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	}
	respondWithJSON(w, statusCode, response)
}

// respondWithJSON sends a JSON response
func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// If encoding fails, log it but we've already written the status code
		// so we can't change the response at this point
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// parseTimeSeriesQueryParams extracts time series query parameters from the request
func parseTimeSeriesQueryParams(r *http.Request) (*TimeSeriesQueryParams, error) {
	resource := r.URL.Query().Get("resource")
	metric := r.URL.Query().Get("metric")
	durationStr := r.URL.Query().Get("duration")

	// Default duration is 1 hour
	duration := 1 * time.Hour
	if durationStr != "" {
		parsedDuration, err := time.ParseDuration(durationStr)
		if err != nil {
			return nil, err
		}
		duration = parsedDuration
	}

	return &TimeSeriesQueryParams{
		Resource: resource,
		Metric:   metric,
		Duration: duration,
	}, nil
}

// parseAnomalyQueryParams extracts anomaly query parameters from the request
func parseAnomalyQueryParams(r *http.Request) (*AnomalyQueryParams, error) {
	resource := r.URL.Query().Get("resource")
	durationStr := r.URL.Query().Get("duration")

	// Default duration is 24 hours
	duration := 24 * time.Hour
	if durationStr != "" {
		parsedDuration, err := time.ParseDuration(durationStr)
		if err != nil {
			return nil, err
		}
		duration = parsedDuration
	}

	return &AnomalyQueryParams{
		Resource: resource,
		Duration: duration,
	}, nil
}
