package optimizer_test

import (
	"fmt"
	"log"

	"github.com/k8s-service-optimizer/backend/internal/k8s"
	"github.com/k8s-service-optimizer/backend/pkg/collector"
	"github.com/k8s-service-optimizer/backend/pkg/optimizer"
)

// ExampleNew demonstrates basic usage of the optimizer
func ExampleNew() {
	// Create Kubernetes client
	k8sClient, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("Failed to create k8s client: %v", err)
	}

	// Create metrics collector
	mc := collector.New(k8sClient)
	mc.SetNamespaces([]string{"default", "k8s-optimizer"})

	// Start collecting metrics
	err = mc.Start()
	if err != nil {
		log.Fatalf("Failed to start collector: %v", err)
	}
	defer mc.Stop()

	// Create optimizer with default config
	opt := optimizer.New(k8sClient, mc)

	// Analyze a deployment
	analysis, err := opt.AnalyzeDeployment("k8s-optimizer", "echo-demo")
	if err != nil {
		log.Fatalf("Failed to analyze deployment: %v", err)
	}

	fmt.Printf("Deployment: %s/%s\n", analysis.Namespace, analysis.Deployment)
	fmt.Printf("Health Score: %.2f/100\n", analysis.HealthScore)
	fmt.Printf("CPU Efficiency: %.2f/100\n", analysis.CPUUsage.Efficiency)
	fmt.Printf("Memory Efficiency: %.2f/100\n", analysis.MemoryUsage.Efficiency)

	// Generate recommendations
	recommendations, err := opt.GenerateRecommendations(analysis)
	if err != nil {
		log.Fatalf("Failed to generate recommendations: %v", err)
	}

	fmt.Printf("\nGenerated %d recommendations:\n", len(recommendations))
	for i, rec := range recommendations {
		fmt.Printf("%d. [%s] %s\n", i+1, rec.Priority, rec.Description)
		fmt.Printf("   Type: %s\n", rec.Type)
		fmt.Printf("   Estimated Savings: $%.2f/month\n", rec.EstimatedSavings)
		fmt.Printf("   Impact: %s\n", rec.Impact)
	}

	// Calculate efficiency score
	score, err := opt.CalculateEfficiencyScore("k8s-optimizer", "echo-demo")
	if err != nil {
		log.Fatalf("Failed to calculate efficiency score: %v", err)
	}

	fmt.Printf("\nOverall Efficiency Score: %.2f/100\n", score)
}

// ExampleOptimizerEngine_AnalyzeDeployment demonstrates deployment analysis
func ExampleOptimizerEngine_AnalyzeDeployment() {
	k8sClient, _ := k8s.NewClient()
	mc := collector.New(k8sClient)
	mc.Start()
	defer mc.Stop()

	opt := optimizer.New(k8sClient, mc)

	// Analyze a specific deployment
	analysis, err := opt.AnalyzeDeployment("default", "my-app")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	// Print CPU analysis
	fmt.Printf("CPU Analysis:\n")
	fmt.Printf("  Requested: %dm\n", analysis.CPUUsage.Requested)
	fmt.Printf("  P95 Usage: %dm\n", analysis.CPUUsage.P95)
	fmt.Printf("  Utilization: %.1f%%\n", analysis.CPUUsage.Utilization)
	fmt.Printf("  Efficiency: %.1f/100\n", analysis.CPUUsage.Efficiency)

	// Print Memory analysis
	fmt.Printf("Memory Analysis:\n")
	fmt.Printf("  Requested: %d bytes\n", analysis.MemoryUsage.Requested)
	fmt.Printf("  P95 Usage: %d bytes\n", analysis.MemoryUsage.P95)
	fmt.Printf("  Utilization: %.1f%%\n", analysis.MemoryUsage.Utilization)
	fmt.Printf("  Efficiency: %.1f/100\n", analysis.MemoryUsage.Efficiency)
}

// ExampleOptimizerEngine_GenerateRecommendations demonstrates recommendation generation
func ExampleOptimizerEngine_GenerateRecommendations() {
	k8sClient, _ := k8s.NewClient()
	mc := collector.New(k8sClient)
	mc.Start()
	defer mc.Stop()

	opt := optimizer.New(k8sClient, mc)

	// First analyze
	analysis, err := opt.AnalyzeDeployment("default", "my-app")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	// Then generate recommendations
	recommendations, err := opt.GenerateRecommendations(analysis)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	for _, rec := range recommendations {
		fmt.Printf("Recommendation: %s\n", rec.Description)
		fmt.Printf("  Priority: %s\n", rec.Priority)
		fmt.Printf("  Savings: $%.2f/month\n", rec.EstimatedSavings)
	}
}

// ExampleOptimizerEngine_GetAllRecommendations demonstrates retrieving all recommendations
func ExampleOptimizerEngine_GetAllRecommendations() {
	k8sClient, _ := k8s.NewClient()
	mc := collector.New(k8sClient)
	mc.Start()
	defer mc.Stop()

	opt := optimizer.New(k8sClient, mc)

	// Generate some recommendations first
	namespaces := []string{"default", "production"}
	_, _ = opt.GenerateAllRecommendations(namespaces)

	// Get all active recommendations
	allRecs, err := opt.GetAllRecommendations()
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Total active recommendations: %d\n", len(allRecs))

	// Get statistics
	stats := opt.GetRecommendationStats()
	fmt.Printf("High priority: %d\n", stats["high_priority"])
	fmt.Printf("Medium priority: %d\n", stats["medium_priority"])
	fmt.Printf("Low priority: %d\n", stats["low_priority"])
	fmt.Printf("Total potential savings: $%.2f/month\n", stats["total_savings"])
}

// ExampleNewWithConfig demonstrates using custom configuration
func ExampleNewWithConfig() {
	k8sClient, _ := k8s.NewClient()
	mc := collector.New(k8sClient)
	mc.Start()
	defer mc.Stop()

	// Create custom configuration
	config := optimizer.DefaultConfig()
	config.CPUOverProvisionedThreshold = 0.6    // 60% threshold instead of 50%
	config.MemoryOverProvisionedThreshold = 0.6 // 60% threshold instead of 50%
	config.OptimalUtilizationMin = 0.6          // 60% instead of 70%
	config.OptimalUtilizationMax = 0.85         // 85% instead of 90%

	// Create optimizer with custom config
	opt := optimizer.NewWithConfig(k8sClient, mc, config)

	// Use the optimizer
	score, err := opt.CalculateEfficiencyScore("default", "my-app")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Efficiency Score (custom config): %.2f/100\n", score)
}
