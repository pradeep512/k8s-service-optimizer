package analyzer_test

import (
	"fmt"
	"log"
	"time"

	"github.com/k8s-service-optimizer/backend/internal/k8s"
	"github.com/k8s-service-optimizer/backend/pkg/analyzer"
	"github.com/k8s-service-optimizer/backend/pkg/collector"
)

// Example demonstrates basic usage of the analyzer
func Example() {
	// Create K8s client
	k8sClient, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("Failed to create k8s client: %v", err)
	}

	// Create metrics collector
	mc := collector.New(k8sClient)
	if err := mc.Start(); err != nil {
		log.Fatalf("Failed to start collector: %v", err)
	}
	defer mc.Stop()

	// Wait for some data to be collected
	time.Sleep(5 * time.Second)

	// Create analyzer
	an := analyzer.New(mc)

	// Analyze traffic patterns
	traffic, err := an.AnalyzeTrafficPatterns("default", "nginx", 1*time.Hour)
	if err != nil {
		log.Printf("Failed to analyze traffic: %v", err)
	} else {
		fmt.Printf("Traffic Analysis for nginx:\n")
		fmt.Printf("  Request Rate: %.2f req/s\n", traffic.RequestRate)
		fmt.Printf("  Error Rate: %.2f%%\n", traffic.ErrorRate*100)
		fmt.Printf("  P95 Latency: %.2fms\n", traffic.P95Latency)
	}

	// Calculate service cost
	cost, err := an.CalculateServiceCost("default", "nginx")
	if err != nil {
		log.Printf("Failed to calculate cost: %v", err)
	} else {
		fmt.Printf("\nCost Analysis for nginx:\n")
		fmt.Printf("  CPU Cost: $%.2f/month\n", cost.CPUCost)
		fmt.Printf("  Memory Cost: $%.2f/month\n", cost.MemoryCost)
		fmt.Printf("  Total Cost: $%.2f/month\n", cost.TotalCost)
		fmt.Printf("  Wasted Cost: $%.2f/month\n", cost.WastedCost)
		fmt.Printf("  Efficiency: %.1f%%\n", cost.EfficiencyScore)
	}

	// Detect anomalies
	anomalies, err := an.DetectAnomalies("pod/nginx", "cpu", 1*time.Hour)
	if err != nil {
		log.Printf("Failed to detect anomalies: %v", err)
	} else {
		fmt.Printf("\nAnomalies detected: %d\n", len(anomalies))
		for i, a := range anomalies {
			if i >= 3 {
				break // Show only first 3
			}
			fmt.Printf("  [%s] %s: %s\n", a.Severity, a.Type, a.Description)
		}
	}

	// Predict resource needs
	prediction, err := an.PredictResourceNeeds("default", "nginx", 72)
	if err != nil {
		log.Printf("Failed to predict resources: %v", err)
	} else {
		fmt.Printf("\nResource Prediction (72 hours):\n")
		fmt.Printf("  Predicted CPU: %dm\n", prediction.PredictedCPU)
		fmt.Printf("  Predicted Memory: %dMB\n", prediction.PredictedMemory/(1024*1024))
		fmt.Printf("  Confidence: %.1f%%\n", prediction.Confidence*100)
	}

	// Calculate waste
	waste, err := an.CalculateWaste("default", "nginx")
	if err != nil {
		log.Printf("Failed to calculate waste: %v", err)
	} else {
		fmt.Printf("\nResource Waste:\n")
		fmt.Printf("  Over-provisioning: %.1f%%\n", waste)
	}
}

// ExampleAnalyzer_AnalyzeTrafficPatterns demonstrates traffic pattern analysis
func ExampleAnalyzer_AnalyzeTrafficPatterns() {
	k8sClient, _ := k8s.NewClient()
	mc := collector.New(k8sClient)
	mc.Start()
	defer mc.Stop()

	an := analyzer.New(mc)

	// Wait for data collection
	time.Sleep(2 * time.Second)

	// Analyze traffic for a service over the last 24 hours
	traffic, err := an.AnalyzeTrafficPatterns("k8s-optimizer", "echo-demo", 24*time.Hour)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Request Rate: %.2f req/s\n", traffic.RequestRate)
	fmt.Printf("Error Rate: %.2f%%\n", traffic.ErrorRate*100)
	fmt.Printf("P95 Latency: %.2fms\n", traffic.P95Latency)
}

// ExampleAnalyzer_CalculateServiceCost demonstrates cost calculation
func ExampleAnalyzer_CalculateServiceCost() {
	k8sClient, _ := k8s.NewClient()
	mc := collector.New(k8sClient)
	mc.Start()
	defer mc.Stop()

	an := analyzer.New(mc)

	// Wait for data collection
	time.Sleep(2 * time.Second)

	// Calculate cost for a service
	cost, err := an.CalculateServiceCost("k8s-optimizer", "echo-demo")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Monthly Cost: $%.2f\n", cost.TotalCost)
	fmt.Printf("Wasted Cost: $%.2f\n", cost.WastedCost)
	fmt.Printf("Efficiency Score: %.1f%%\n", cost.EfficiencyScore)
}

// ExampleAnalyzer_DetectAnomalies demonstrates anomaly detection
func ExampleAnalyzer_DetectAnomalies() {
	k8sClient, _ := k8s.NewClient()
	mc := collector.New(k8sClient)
	mc.Start()
	defer mc.Stop()

	an := analyzer.New(mc)

	// Wait for data collection
	time.Sleep(2 * time.Second)

	// Detect CPU anomalies
	anomalies, err := an.DetectAnomalies("pod/echo-demo-xxx", "cpu", 24*time.Hour)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	for _, a := range anomalies {
		fmt.Printf("Anomaly: %s - %s (%s)\n", a.Type, a.Description, a.Severity)
	}
}

// ExampleAnalyzer_PredictResourceNeeds demonstrates resource prediction
func ExampleAnalyzer_PredictResourceNeeds() {
	k8sClient, _ := k8s.NewClient()
	mc := collector.New(k8sClient)
	mc.Start()
	defer mc.Stop()

	an := analyzer.New(mc)

	// Wait for data collection
	time.Sleep(2 * time.Second)

	// Predict resource needs for next 72 hours
	prediction, err := an.PredictResourceNeeds("k8s-optimizer", "echo-demo", 72)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Predicted CPU (72h): %dm\n", prediction.PredictedCPU)
	fmt.Printf("Predicted Memory (72h): %dMB\n", prediction.PredictedMemory/(1024*1024))
	fmt.Printf("Confidence: %.1f%%\n", prediction.Confidence*100)
}

// ExampleAnalyzer_CalculateWaste demonstrates waste calculation
func ExampleAnalyzer_CalculateWaste() {
	k8sClient, _ := k8s.NewClient()
	mc := collector.New(k8sClient)
	mc.Start()
	defer mc.Stop()

	an := analyzer.New(mc)

	// Wait for data collection
	time.Sleep(2 * time.Second)

	// Calculate wasted resources
	waste, err := an.CalculateWaste("k8s-optimizer", "echo-demo")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Over-provisioning: %.1f%%\n", waste)
}

// ExampleConfig demonstrates custom configuration
func ExampleConfig() {
	k8sClient, _ := k8s.NewClient()
	mc := collector.New(k8sClient)

	// Create custom configuration
	config := analyzer.Config{
		CPUCostPerVCPUHour:  0.04,   // Higher CPU cost
		MemoryCostPerGBHour: 0.005,  // Higher memory cost
		AnomalyThreshold:    2.5,    // More sensitive anomaly detection
		SpikeThreshold:      1.8,    // Lower spike threshold
		DropThreshold:       0.6,    // Higher drop threshold
		MinDataPoints:       5,      // Fewer points needed
		TrendHistoryDays:    14,     // Use 2 weeks of history
	}

	// Create analyzer with custom config
	an := analyzer.NewWithConfig(mc, config)

	fmt.Printf("Analyzer created with custom config\n")
	fmt.Printf("CPU Cost: $%.3f/vCPU-hour\n", config.CPUCostPerVCPUHour)
	fmt.Printf("Memory Cost: $%.4f/GB-hour\n", config.MemoryCostPerGBHour)

	_ = an // Use analyzer
}
