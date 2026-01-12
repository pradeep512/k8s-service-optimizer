package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestRespondWithSuccess tests the respondWithSuccess function
func TestRespondWithSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"message": "test"}

	respondWithSuccess(w, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected content type application/json, got %s", contentType)
	}
}

// TestRespondWithError tests the respondWithError function
func TestRespondWithError(t *testing.T) {
	w := httptest.NewRecorder()

	respondWithError(w, http.StatusBadRequest, "TEST_ERROR", "Test error message")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected content type application/json, got %s", contentType)
	}
}

// TestParseTimeSeriesQueryParams tests the parseTimeSeriesQueryParams function
func TestParseTimeSeriesQueryParams(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/metrics/timeseries?resource=node/worker-1&metric=cpu&duration=1h", nil)

	params, err := parseTimeSeriesQueryParams(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if params.Resource != "node/worker-1" {
		t.Errorf("Expected resource 'node/worker-1', got '%s'", params.Resource)
	}

	if params.Metric != "cpu" {
		t.Errorf("Expected metric 'cpu', got '%s'", params.Metric)
	}

	if params.Duration != 1*time.Hour {
		t.Errorf("Expected duration 1h, got %v", params.Duration)
	}
}

// TestParseAnomalyQueryParams tests the parseAnomalyQueryParams function
func TestParseAnomalyQueryParams(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/anomalies?resource=pod/test&duration=24h", nil)

	params, err := parseAnomalyQueryParams(req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if params.Resource != "pod/test" {
		t.Errorf("Expected resource 'pod/test', got '%s'", params.Resource)
	}

	if params.Duration != 24*time.Hour {
		t.Errorf("Expected duration 24h, got %v", params.Duration)
	}
}

// TestWebSocketHubCreation tests creating a WebSocket hub
func TestWebSocketHubCreation(t *testing.T) {
	hub := NewWebSocketHub()

	if hub == nil {
		t.Fatal("Expected hub to be created, got nil")
	}

	if hub.clients == nil {
		t.Error("Expected clients map to be initialized")
	}

	if hub.broadcast == nil {
		t.Error("Expected broadcast channel to be initialized")
	}

	if hub.register == nil {
		t.Error("Expected register channel to be initialized")
	}

	if hub.unregister == nil {
		t.Error("Expected unregister channel to be initialized")
	}
}

// TestGetClientCount tests getting the WebSocket client count
func TestGetClientCount(t *testing.T) {
	hub := NewWebSocketHub()

	count := hub.GetClientCount()
	if count != 0 {
		t.Errorf("Expected 0 clients, got %d", count)
	}
}
