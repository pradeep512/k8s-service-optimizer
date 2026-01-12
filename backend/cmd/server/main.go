package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/k8s-service-optimizer/backend/internal/k8s"
	"github.com/k8s-service-optimizer/backend/pkg/analyzer"
	"github.com/k8s-service-optimizer/backend/pkg/api"
	"github.com/k8s-service-optimizer/backend/pkg/collector"
	"github.com/k8s-service-optimizer/backend/pkg/optimizer"
)

func main() {
	log.Println("Starting k8s-service-optimizer API server...")

	// Load configuration from environment variables
	config := loadConfig()

	// Create Kubernetes client
	log.Println("Connecting to Kubernetes cluster...")
	k8sClient, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}
	log.Println("Successfully connected to Kubernetes cluster")

	// Create metrics collector
	log.Println("Initializing metrics collector...")
	mc := collector.New(k8sClient)

	// Set namespaces to monitor (from env or default)
	namespaces := getNamespaces()
	mc.SetNamespaces(namespaces)
	log.Printf("Monitoring namespaces: %v", namespaces)

	// Start the collector
	if err := mc.Start(); err != nil {
		log.Fatalf("Failed to start metrics collector: %v", err)
	}
	defer mc.Stop()
	log.Println("Metrics collector started")

	// Create optimizer
	log.Println("Initializing optimizer engine...")
	opt := optimizer.New(k8sClient, mc)
	log.Println("Optimizer engine initialized")

	// Create analyzer
	log.Println("Initializing analyzer...")
	an := analyzer.New(mc)
	log.Println("Analyzer initialized")

	// Create API server
	log.Println("Initializing API server...")
	srv := api.NewServerWithConfig(k8sClient, mc, opt, an, config)

	// Channel to listen for interrupt signals
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("API server listening on port %s", config.Port)
		log.Println("Press Ctrl+C to stop the server")
		serverErrors <- srv.Start()
	}()

	// Wait for interrupt signal or server error
	select {
	case err := <-serverErrors:
		if err != nil {
			log.Fatalf("Server error: %v", err)
		}

	case <-sigint:
		log.Println("\nReceived interrupt signal, shutting down gracefully...")

		// Create a deadline for shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		// Shutdown the API server
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		// Stop the metrics collector
		mc.Stop()

		log.Println("Shutdown complete")
	}
}

// loadConfig loads configuration from environment variables
func loadConfig() *api.Config {
	port := getEnv("PORT", "8080")
	logLevel := getEnv("LOG_LEVEL", "info")
	updateInterval := getEnvDuration("UPDATE_INTERVAL", 5*time.Second)

	config := &api.Config{
		Port:           port,
		EnableCORS:     true,
		LogLevel:       logLevel,
		UpdateInterval: updateInterval,
	}

	log.Printf("Configuration loaded: port=%s, log_level=%s, update_interval=%s",
		config.Port, config.LogLevel, config.UpdateInterval)

	return config
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvDuration gets a duration from environment variable with a default value
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		log.Printf("Warning: invalid duration for %s: %s, using default: %s", key, value, defaultValue)
	}
	return defaultValue
}

// getNamespaces gets the list of namespaces to monitor from environment
func getNamespaces() []string {
	namespacesEnv := getEnv("NAMESPACES", "default")

	// Split by comma if multiple namespaces
	var namespaces []string
	current := ""
	for _, char := range namespacesEnv {
		if char == ',' {
			if current != "" {
				namespaces = append(namespaces, current)
				current = ""
			}
		} else if char != ' ' {
			current += string(char)
		}
	}
	if current != "" {
		namespaces = append(namespaces, current)
	}

	if len(namespaces) == 0 {
		namespaces = []string{"default"}
	}

	return namespaces
}
