package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/k8s-service-optimizer/backend/internal/k8s"
	"github.com/k8s-service-optimizer/backend/pkg/analyzer"
	"github.com/k8s-service-optimizer/backend/pkg/collector"
	"github.com/k8s-service-optimizer/backend/pkg/optimizer"
)

// Server represents the API server
type Server struct {
	collector  collector.MetricsCollector
	optimizer  optimizer.Optimizer
	analyzer   analyzer.Analyzer
	k8sClient  *k8s.Client
	httpServer *http.Server
	wsHub      *WebSocketHub
	config     *Config
	startTime  time.Time
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewServer creates a new API server
func NewServer(k8sClient *k8s.Client, collector collector.MetricsCollector, optimizer optimizer.Optimizer, analyzer analyzer.Analyzer) *Server {
	return NewServerWithConfig(k8sClient, collector, optimizer, analyzer, &Config{
		Port:           "8080",
		EnableCORS:     true,
		LogLevel:       "info",
		UpdateInterval: 5 * time.Second,
	})
}

// NewServerWithConfig creates a new API server with custom configuration
func NewServerWithConfig(k8sClient *k8s.Client, collector collector.MetricsCollector, optimizer optimizer.Optimizer, analyzer analyzer.Analyzer, config *Config) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		collector: collector,
		optimizer: optimizer,
		analyzer:  analyzer,
		k8sClient: k8sClient,
		wsHub:     NewWebSocketHub(),
		config:    config,
		startTime: time.Now(),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start starts the API server
func (s *Server) Start() error {
	// Start the WebSocket hub
	go s.wsHub.Run()
	log.Println("WebSocket hub started")

	// Start the periodic update broadcaster
	go s.startUpdateBroadcaster()
	log.Println("Update broadcaster started")

	// Setup routes
	router := s.setupRoutes()

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%s", s.config.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting API server on port %s", s.config.Port)

	// Start server (blocking)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down API server...")

	// Cancel the context to stop background goroutines
	s.cancel()

	// Shutdown HTTP server
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown server: %w", err)
		}
	}

	log.Println("API server stopped")
	return nil
}

// startUpdateBroadcaster starts the periodic WebSocket update broadcaster
func (s *Server) startUpdateBroadcaster() {
	ticker := time.NewTicker(s.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			log.Println("Update broadcaster stopped")
			return

		case <-ticker.C:
			// Only broadcast if there are connected clients
			if s.wsHub.GetClientCount() == 0 {
				continue
			}

			// Broadcast metrics update
			s.broadcastMetricsUpdate()

			// Broadcast recommendations update
			s.broadcastRecommendationsUpdate()

			// Broadcast anomalies update (if any)
			s.broadcastAnomaliesUpdate()
		}
	}
}

// broadcastMetricsUpdate broadcasts metrics updates to all WebSocket clients
func (s *Server) broadcastMetricsUpdate() {
	// Get node metrics
	nodeMetrics, err := s.collector.CollectNodeMetrics()
	if err != nil {
		log.Printf("Warning: failed to collect node metrics for broadcast: %v", err)
		return
	}

	// Broadcast node metrics
	s.wsHub.Broadcast("metrics_update", map[string]interface{}{
		"type":    "nodes",
		"metrics": nodeMetrics,
	})
}

// broadcastRecommendationsUpdate broadcasts new recommendations to all WebSocket clients
func (s *Server) broadcastRecommendationsUpdate() {
	recommendations, err := s.optimizer.GetAllRecommendations()
	if err != nil {
		log.Printf("Warning: failed to get recommendations for broadcast: %v", err)
		return
	}

	// Only broadcast if there are recommendations
	if len(recommendations) > 0 {
		s.wsHub.Broadcast("recommendation_new", map[string]interface{}{
			"count":           len(recommendations),
			"recommendations": recommendations,
		})
	}
}

// broadcastAnomaliesUpdate broadcasts detected anomalies to all WebSocket clients
func (s *Server) broadcastAnomaliesUpdate() {
	// This is a simplified implementation
	// In a real system, you would track which anomalies have already been broadcast
	// and only send new ones

	// For now, we skip this to avoid spamming clients
	// Anomalies can be fetched via the REST API endpoint when needed
}

// BroadcastStatusUpdate broadcasts a status update to all WebSocket clients
func (s *Server) BroadcastStatusUpdate(status string, message string) {
	s.wsHub.Broadcast("status_update", map[string]interface{}{
		"status":  status,
		"message": message,
	})
}

// GetConfig returns the server configuration
func (s *Server) GetConfig() *Config {
	return s.config
}

// GetUptime returns the server uptime
func (s *Server) GetUptime() time.Duration {
	return time.Since(s.startTime)
}

// IsRunning returns whether the server is running
func (s *Server) IsRunning() bool {
	return s.httpServer != nil
}
