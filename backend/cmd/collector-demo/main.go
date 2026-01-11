package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/k8s-service-optimizer/backend/internal/k8s"
	"github.com/k8s-service-optimizer/backend/pkg/collector"
)

func main() {
	log.Println("Starting Metrics Collector Demo...")

	// Create K8s client
	k8sClient, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("Failed to create k8s client: %v", err)
	}
	log.Println("✓ Connected to Kubernetes cluster")

	// Create metrics collector with custom config
	config := collector.Config{
		CollectionInterval: 15 * time.Second,
		RetentionPeriod:    24 * time.Hour,
		CleanupInterval:    1 * time.Hour,
	}

	mc := collector.NewWithConfig(k8sClient, config)

	// Monitor multiple namespaces
	mc.SetNamespaces([]string{"default", "kube-system"})

	// Start the collector
	if err := mc.Start(); err != nil {
		log.Fatalf("Failed to start collector: %v", err)
	}
	log.Println("✓ Metrics collector started")
	log.Printf("  - Collection interval: %v", config.CollectionInterval)
	log.Printf("  - Retention period: %v", config.RetentionPeriod)
	log.Printf("  - Cleanup interval: %v", config.CleanupInterval)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start monitoring goroutine
	go monitorMetrics(mc)

	// Wait for shutdown signal
	<-sigChan
	log.Println("\nReceived shutdown signal, stopping collector...")
	mc.Stop()
	log.Println("✓ Collector stopped gracefully")
}

func monitorMetrics(mc *collector.Collector) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	firstRun := true

	for {
		if !firstRun {
			<-ticker.C
		}
		firstRun = false

		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Printf("Metrics Report - %s\n", time.Now().Format(time.RFC3339))
		fmt.Println(strings.Repeat("=", 60))

		// Collect and display pod metrics
		pods, err := mc.CollectPodMetrics("default")
		if err != nil {
			log.Printf("Error collecting pod metrics: %v", err)
		} else {
			fmt.Printf("\n📦 Pods in 'default' namespace: %d\n", len(pods))
			for i, pod := range pods {
				if i < 5 { // Show first 5 pods
					fmt.Printf("  - %-30s CPU: %6d m, Memory: %8d MB\n",
						pod.Name, pod.CPU, pod.Memory/(1024*1024))
				}
			}
			if len(pods) > 5 {
				fmt.Printf("  ... and %d more pods\n", len(pods)-5)
			}
		}

		// Collect and display node metrics
		nodes, err := mc.CollectNodeMetrics()
		if err != nil {
			log.Printf("Error collecting node metrics: %v", err)
		} else {
			fmt.Printf("\n🖥️  Cluster Nodes: %d\n", len(nodes))
			for _, node := range nodes {
				fmt.Printf("  - %-30s CPU: %6d m, Memory: %8d MB\n",
					node.Name, node.CPU, node.Memory/(1024*1024))
			}
		}

		// Collect and display HPA metrics
		hpas, err := mc.CollectHPAMetrics("default")
		if err != nil {
			log.Printf("Error collecting HPA metrics: %v", err)
		} else if len(hpas) > 0 {
			fmt.Printf("\n📊 HPAs in 'default' namespace: %d\n", len(hpas))
			for _, hpa := range hpas {
				fmt.Printf("  - %-30s Replicas: %d/%d, CPU: %d%% (target: %d%%)\n",
					hpa.Name, hpa.CurrentReplicas, hpa.DesiredReplicas,
					hpa.CurrentCPU, hpa.TargetCPU)
			}
		}

		// Display store statistics
		fmt.Printf("\n💾 Metrics Store:\n")
		fmt.Printf("  - Total data points: %d\n", mc.GetStoreSize())
		fmt.Printf("  - Unique metrics: %d\n", len(mc.GetStoredMetricKeys()))

		// If we have some data, show percentile analysis for first pod
		if len(pods) > 0 {
			podResource := fmt.Sprintf("pod/%s", pods[0].Name)

			// Try to get CPU percentiles
			p50, p95, p99, err := mc.GetResourcePercentiles(podResource, "cpu", 5*time.Minute)
			if err == nil {
				fmt.Printf("\n📈 CPU Percentiles for %s (last 5 min):\n", pods[0].Name)
				fmt.Printf("  - P50: %.2f m\n", p50)
				fmt.Printf("  - P95: %.2f m\n", p95)
				fmt.Printf("  - P99: %.2f m\n", p99)
			}

			// Get time series data
			ts, err := mc.GetTimeSeriesData(podResource, "cpu", 5*time.Minute)
			if err == nil && len(ts.Points) > 0 {
				fmt.Printf("\n📉 Time Series Data Points: %d (last 5 min)\n", len(ts.Points))
				if len(ts.Points) > 0 {
					latest := ts.Points[len(ts.Points)-1]
					fmt.Printf("  - Latest: %.2f m at %s\n",
						latest.Value, latest.Timestamp.Format("15:04:05"))
				}
			}
		}
	}
}
