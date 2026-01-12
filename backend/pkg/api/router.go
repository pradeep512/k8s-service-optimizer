package api

import (
	"github.com/gorilla/mux"
)

// setupRoutes configures all the routes for the API server
func (s *Server) setupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Apply middleware in order:
	// 1. Recovery (catch panics)
	// 2. Logging (log requests)
	// 3. CORS (handle CORS)
	// 4. Request ID (add request ID)
	r.Use(recoveryMiddleware)
	r.Use(loggingMiddleware)
	if s.config.EnableCORS {
		r.Use(corsMiddleware)
	}
	r.Use(requestIDMiddleware)

	// Health endpoints (no /api prefix)
	r.HandleFunc("/health", s.handleHealth).Methods("GET")
	r.HandleFunc("/ready", s.handleReady).Methods("GET")

	// WebSocket endpoint (no /api prefix)
	r.HandleFunc("/ws/updates", s.handleWebSocket)

	// API v1 routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Status
	api.HandleFunc("/status", s.handleStatus).Methods("GET")

	// Cluster & Services
	api.HandleFunc("/cluster/overview", s.handleClusterOverview).Methods("GET")
	api.HandleFunc("/services", s.handleListServices).Methods("GET")
	api.HandleFunc("/services/{namespace}/{name}", s.handleServiceDetail).Methods("GET")

	// Deployments (for dashboard service metrics)
	api.HandleFunc("/deployments", s.handleListDeployments).Methods("GET")
	api.HandleFunc("/deployments/{namespace}/{name}", s.handleDeploymentDetail).Methods("GET")

	// Metrics
	api.HandleFunc("/metrics/nodes", s.handleNodeMetrics).Methods("GET")
	api.HandleFunc("/metrics/pods/{namespace}", s.handlePodMetrics).Methods("GET")
	api.HandleFunc("/metrics/timeseries", s.handleTimeSeries).Methods("GET")
	api.HandleFunc("/hpa/{namespace}", s.handleHPAMetrics).Methods("GET")

	// Optimization
	api.HandleFunc("/recommendations", s.handleRecommendations).Methods("GET")
	api.HandleFunc("/recommendations/{id}", s.handleRecommendationByID).Methods("GET")
	api.HandleFunc("/recommendations/{id}/apply", s.handleApplyRecommendation).Methods("POST")

	// Analysis
	api.HandleFunc("/analysis/{namespace}/{service}", s.handleAnalysis).Methods("GET")
	api.HandleFunc("/traffic/{namespace}/{service}", s.handleTraffic).Methods("GET")
	api.HandleFunc("/cost/{namespace}/{service}", s.handleCost).Methods("GET")
	api.HandleFunc("/anomalies", s.handleAnomalies).Methods("GET")

	return r
}
