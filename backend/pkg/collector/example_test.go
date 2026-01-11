package collector_test

import (
	"fmt"
	"log"
	"time"

	"github.com/k8s-service-optimizer/backend/internal/k8s"
	"github.com/k8s-service-optimizer/backend/pkg/collector"
)

// Example demonstrates basic usage of the metrics collector
func Example() {
	// Create K8s client
	k8sClient, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("Failed to create k8s client: %v", err)
	}

	// Create metrics collector with default config
	mc := collector.New(k8sClient)

	// Configure namespaces to monitor
	mc.SetNamespaces([]string{"default", "kube-system"})

	// Start collection
	if err := mc.Start(); err != nil {
		log.Fatalf("Failed to start collector: %v", err)
	}
	defer mc.Stop()

	// Wait for some metrics to be collected
	time.Sleep(30 * time.Second)

	// Query current metrics
	pods, err := mc.CollectPodMetrics("default")
	if err != nil {
		log.Printf("Error collecting pod metrics: %v", err)
	} else {
		fmt.Printf("Collected metrics for %d pods\n", len(pods))
	}

	nodes, err := mc.CollectNodeMetrics()
	if err != nil {
		log.Printf("Error collecting node metrics: %v", err)
	} else {
		fmt.Printf("Collected metrics for %d nodes\n", len(nodes))
	}

	// Get time series data for a specific pod
	ts, err := mc.GetTimeSeriesData("pod/my-app-abc123", "cpu", 5*time.Minute)
	if err != nil {
		log.Printf("Error getting time series: %v", err)
	} else {
		fmt.Printf("Retrieved %d data points for CPU\n", len(ts.Points))
	}

	// Get percentiles for resource usage
	p50, p95, p99, err := mc.GetResourcePercentiles("pod/my-app-abc123", "cpu", 5*time.Minute)
	if err != nil {
		log.Printf("Error calculating percentiles: %v", err)
	} else {
		fmt.Printf("CPU Percentiles - P50: %.2f, P95: %.2f, P99: %.2f\n", p50, p95, p99)
	}
}

// ExampleCollector_customConfig demonstrates using custom configuration
func ExampleCollector_customConfig() {
	k8sClient, _ := k8s.NewClient()

	// Create custom configuration
	config := collector.Config{
		CollectionInterval: 10 * time.Second, // Collect every 10 seconds
		RetentionPeriod:    12 * time.Hour,   // Keep 12 hours of data
		CleanupInterval:    30 * time.Minute, // Clean up every 30 minutes
	}

	// Create collector with custom config
	mc := collector.NewWithConfig(k8sClient, config)

	mc.Start()
	defer mc.Stop()

	// Use the collector...
}

// ExampleCollector_monitoring demonstrates continuous monitoring
func ExampleCollector_monitoring() {
	k8sClient, _ := k8s.NewClient()
	mc := collector.New(k8sClient)

	mc.Start()
	defer mc.Stop()

	// Monitor metrics every minute
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for i := 0; i < 5; i++ {
		<-ticker.C

		// Get all stored metric keys
		keys := mc.GetStoredMetricKeys()
		fmt.Printf("Monitoring %d metrics\n", len(keys))

		// Get store size
		size := mc.GetStoreSize()
		fmt.Printf("Store contains %d data points\n", size)
	}
}
