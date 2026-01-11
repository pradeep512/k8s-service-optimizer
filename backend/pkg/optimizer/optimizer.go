package optimizer

import (
	"context"
	"fmt"
	"sync"

	"github.com/k8s-service-optimizer/backend/internal/k8s"
	"github.com/k8s-service-optimizer/backend/internal/models"
	"github.com/k8s-service-optimizer/backend/pkg/collector"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Optimizer defines the interface for resource optimization
type Optimizer interface {
	// AnalyzeDeployment analyzes a specific deployment
	AnalyzeDeployment(namespace, name string) (*models.Analysis, error)

	// GenerateRecommendations generates optimization recommendations
	GenerateRecommendations(analysis *models.Analysis) ([]models.Recommendation, error)

	// CalculateEfficiencyScore calculates efficiency score for a deployment
	CalculateEfficiencyScore(namespace, name string) (float64, error)

	// EstimateCostSavings estimates cost savings from a recommendation
	EstimateCostSavings(recommendation *models.Recommendation) (float64, error)

	// ApplyRecommendation applies an optimization recommendation
	ApplyRecommendation(recommendationID string) error

	// GetAllRecommendations gets all active recommendations
	GetAllRecommendations() ([]models.Recommendation, error)
}

// OptimizerEngine implements the Optimizer interface
type OptimizerEngine struct {
	k8sClient         *k8s.Client
	collector         collector.MetricsCollector
	config            Config
	analyzer          *resourceAnalyzer
	recommendationGen *recommendationGenerator
	scorer            *scorer

	// In-memory storage for recommendations
	recommendations   map[string]models.Recommendation
	recommendationsMu sync.RWMutex

	// Cache for analysis results
	analysisCache   map[string]*analysisResult
	analysisCacheMu sync.RWMutex
}

// New creates a new optimizer with default configuration
func New(k8sClient *k8s.Client, collector collector.MetricsCollector) *OptimizerEngine {
	return NewWithConfig(k8sClient, collector, DefaultConfig())
}

// NewWithConfig creates a new optimizer with custom configuration
func NewWithConfig(k8sClient *k8s.Client, collector collector.MetricsCollector, config Config) *OptimizerEngine {
	opt := &OptimizerEngine{
		k8sClient:       k8sClient,
		collector:       collector,
		config:          config,
		recommendations: make(map[string]models.Recommendation),
		analysisCache:   make(map[string]*analysisResult),
	}

	// Initialize components
	opt.analyzer = newResourceAnalyzer(opt)
	opt.recommendationGen = newRecommendationGenerator(opt)
	opt.scorer = newScorer(opt)

	return opt
}

// AnalyzeDeployment analyzes a specific deployment
func (opt *OptimizerEngine) AnalyzeDeployment(namespace, name string) (*models.Analysis, error) {
	// Perform internal analysis
	internalAnalysis, err := opt.analyzer.analyzeDeployment(namespace, name)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze deployment: %w", err)
	}

	// Cache the internal analysis
	cacheKey := fmt.Sprintf("%s/%s", namespace, name)
	opt.analysisCacheMu.Lock()
	opt.analysisCache[cacheKey] = internalAnalysis
	opt.analysisCacheMu.Unlock()

	// Convert to public Analysis model
	analysis := opt.convertToPublicAnalysis(internalAnalysis)

	return analysis, nil
}

// GenerateRecommendations generates optimization recommendations
func (opt *OptimizerEngine) GenerateRecommendations(analysis *models.Analysis) ([]models.Recommendation, error) {
	// Get the internal analysis from cache
	cacheKey := fmt.Sprintf("%s/%s", analysis.Namespace, analysis.Deployment)
	opt.analysisCacheMu.RLock()
	internalAnalysis, exists := opt.analysisCache[cacheKey]
	opt.analysisCacheMu.RUnlock()

	if !exists {
		// If not in cache, re-analyze
		_, err := opt.AnalyzeDeployment(analysis.Namespace, analysis.Deployment)
		if err != nil {
			return nil, fmt.Errorf("failed to re-analyze deployment: %w", err)
		}

		opt.analysisCacheMu.RLock()
		internalAnalysis = opt.analysisCache[cacheKey]
		opt.analysisCacheMu.RUnlock()
	}

	// Generate recommendations
	recommendations, err := opt.recommendationGen.generateRecommendations(internalAnalysis)
	if err != nil {
		return nil, fmt.Errorf("failed to generate recommendations: %w", err)
	}

	// Store recommendations in memory
	opt.recommendationsMu.Lock()
	for _, rec := range recommendations {
		opt.recommendations[rec.ID] = rec
	}
	opt.recommendationsMu.Unlock()

	return recommendations, nil
}

// CalculateEfficiencyScore calculates efficiency score for a deployment
func (opt *OptimizerEngine) CalculateEfficiencyScore(namespace, name string) (float64, error) {
	// Analyze deployment
	internalAnalysis, err := opt.analyzer.analyzeDeployment(namespace, name)
	if err != nil {
		return 0, fmt.Errorf("failed to analyze deployment: %w", err)
	}

	// Calculate efficiency score
	score := opt.scorer.calculateEfficiencyScore(internalAnalysis)

	return score, nil
}

// EstimateCostSavings estimates cost savings from a recommendation
func (opt *OptimizerEngine) EstimateCostSavings(recommendation *models.Recommendation) (float64, error) {
	// The savings are already calculated and stored in the recommendation
	// This method provides a way to recalculate or verify them

	if recommendation == nil {
		return 0, fmt.Errorf("recommendation is nil")
	}

	// Return the pre-calculated savings
	return recommendation.EstimatedSavings, nil
}

// ApplyRecommendation applies an optimization recommendation
func (opt *OptimizerEngine) ApplyRecommendation(recommendationID string) error {
	// Get the recommendation
	opt.recommendationsMu.RLock()
	_, exists := opt.recommendations[recommendationID]
	opt.recommendationsMu.RUnlock()

	if !exists {
		return fmt.Errorf("recommendation not found: %s", recommendationID)
	}

	// Note: Actual application of recommendations is out of scope per requirements
	// This method would implement the actual Kubernetes resource updates
	return fmt.Errorf("applying recommendations is not implemented (out of scope)")
}

// GetAllRecommendations gets all active recommendations
func (opt *OptimizerEngine) GetAllRecommendations() ([]models.Recommendation, error) {
	opt.recommendationsMu.RLock()
	defer opt.recommendationsMu.RUnlock()

	recommendations := make([]models.Recommendation, 0, len(opt.recommendations))
	for _, rec := range opt.recommendations {
		recommendations = append(recommendations, rec)
	}

	return recommendations, nil
}

// GetRecommendationByID gets a specific recommendation by ID
func (opt *OptimizerEngine) GetRecommendationByID(id string) (*models.Recommendation, error) {
	opt.recommendationsMu.RLock()
	defer opt.recommendationsMu.RUnlock()

	rec, exists := opt.recommendations[id]
	if !exists {
		return nil, fmt.Errorf("recommendation not found: %s", id)
	}

	return &rec, nil
}

// GetRecommendationsForDeployment gets all recommendations for a specific deployment
func (opt *OptimizerEngine) GetRecommendationsForDeployment(namespace, name string) ([]models.Recommendation, error) {
	opt.recommendationsMu.RLock()
	defer opt.recommendationsMu.RUnlock()

	var recommendations []models.Recommendation
	for _, rec := range opt.recommendations {
		if rec.Namespace == namespace && rec.Deployment == name {
			recommendations = append(recommendations, rec)
		}
	}

	return recommendations, nil
}

// ClearRecommendations clears all recommendations from memory
func (opt *OptimizerEngine) ClearRecommendations() {
	opt.recommendationsMu.Lock()
	defer opt.recommendationsMu.Unlock()

	opt.recommendations = make(map[string]models.Recommendation)
}

// ClearCache clears the analysis cache
func (opt *OptimizerEngine) ClearCache() {
	opt.analysisCacheMu.Lock()
	defer opt.analysisCacheMu.Unlock()

	opt.analysisCache = make(map[string]*analysisResult)
}

// GetConfig returns the current optimizer configuration
func (opt *OptimizerEngine) GetConfig() Config {
	return opt.config
}

// convertToPublicAnalysis converts internal analysisResult to public models.Analysis
func (opt *OptimizerEngine) convertToPublicAnalysis(internal *analysisResult) *models.Analysis {
	metrics := &internal.Deployment

	return &models.Analysis{
		Namespace:  metrics.Namespace,
		Deployment: metrics.Deployment,
		CPUUsage: models.ResourceAnalysis{
			Requested:   metrics.CPURequested,
			Current:     metrics.CPUCurrent,
			P50:         metrics.CPUP50,
			P95:         metrics.CPUP95,
			P99:         metrics.CPUP99,
			Average:     metrics.CPUAverage,
			Max:         metrics.CPUMax,
			Utilization: internal.CPUUtilization * 100, // Convert to percentage
			Efficiency:  internal.CPUEfficiency,
		},
		MemoryUsage: models.ResourceAnalysis{
			Requested:   metrics.MemoryRequested,
			Current:     metrics.MemoryCurrent,
			P50:         metrics.MemoryP50,
			P95:         metrics.MemoryP95,
			P99:         metrics.MemoryP99,
			Average:     metrics.MemoryAverage,
			Max:         metrics.MemoryMax,
			Utilization: internal.MemoryUtilization * 100, // Convert to percentage
			Efficiency:  internal.MemoryEfficiency,
		},
		Replicas: models.ReplicaAnalysis{
			Current:     metrics.CurrentReplicas,
			Min:         metrics.MinReplicas,
			Max:         metrics.MaxReplicas,
			Recommended: opt.calculateRecommendedReplicas(internal),
		},
		HealthScore: opt.scorer.calculateHealthScore(internal),
		Timestamp:   internal.Timestamp,
	}
}

// calculateRecommendedReplicas calculates recommended replica count
func (opt *OptimizerEngine) calculateRecommendedReplicas(analysis *analysisResult) int32 {
	metrics := &analysis.Deployment

	// If HPA is enabled, use HPA's desired replicas
	if metrics.HasHPA {
		return metrics.HPADesiredReplicas
	}

	// Otherwise, calculate based on utilization
	if analysis.CPUUtilization > 0.8 || analysis.MemoryUtilization > 0.8 {
		// High utilization - recommend scaling up
		return metrics.CurrentReplicas + 1
	} else if analysis.CPUUtilization < 0.5 && analysis.MemoryUtilization < 0.5 && metrics.CurrentReplicas > 1 {
		// Low utilization - recommend scaling down
		return metrics.CurrentReplicas - 1
	}

	// Current is fine
	return metrics.CurrentReplicas
}

// AnalyzeAllDeployments analyzes all deployments in given namespaces
func (opt *OptimizerEngine) AnalyzeAllDeployments(namespaces []string) ([]models.Analysis, error) {
	var allAnalyses []models.Analysis

	for _, namespace := range namespaces {
		// List all deployments in namespace
		ctx := context.Background()
		deploymentList, err := opt.k8sClient.Clientset.AppsV1().Deployments(namespace).List(
			ctx,
			metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list deployments in namespace %s: %w", namespace, err)
		}

		// Analyze each deployment
		for _, deployment := range deploymentList.Items {
			analysis, err := opt.AnalyzeDeployment(namespace, deployment.Name)
			if err != nil {
				// Log error but continue with other deployments
				fmt.Printf("Warning: failed to analyze deployment %s/%s: %v\n", namespace, deployment.Name, err)
				continue
			}
			allAnalyses = append(allAnalyses, *analysis)
		}
	}

	return allAnalyses, nil
}

// GenerateAllRecommendations generates recommendations for all deployments
func (opt *OptimizerEngine) GenerateAllRecommendations(namespaces []string) ([]models.Recommendation, error) {
	// First analyze all deployments
	analyses, err := opt.AnalyzeAllDeployments(namespaces)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze deployments: %w", err)
	}

	// Generate recommendations for each analysis
	var allRecommendations []models.Recommendation
	for _, analysis := range analyses {
		recs, err := opt.GenerateRecommendations(&analysis)
		if err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to generate recommendations for %s/%s: %v\n",
				analysis.Namespace, analysis.Deployment, err)
			continue
		}
		allRecommendations = append(allRecommendations, recs...)
	}

	return allRecommendations, nil
}

// GetTotalPotentialSavings calculates total potential savings from all recommendations
func (opt *OptimizerEngine) GetTotalPotentialSavings() (float64, error) {
	opt.recommendationsMu.RLock()
	defer opt.recommendationsMu.RUnlock()

	totalSavings := 0.0
	for _, rec := range opt.recommendations {
		totalSavings += rec.EstimatedSavings
	}

	return totalSavings, nil
}

// GetRecommendationStats returns statistics about recommendations
func (opt *OptimizerEngine) GetRecommendationStats() map[string]interface{} {
	opt.recommendationsMu.RLock()
	defer opt.recommendationsMu.RUnlock()

	stats := map[string]interface{}{
		"total":           len(opt.recommendations),
		"high_priority":   0,
		"medium_priority": 0,
		"low_priority":    0,
		"by_type": map[string]int{
			"resource": 0,
			"hpa":      0,
			"scaling":  0,
		},
	}

	totalSavings := 0.0

	for _, rec := range opt.recommendations {
		// Count by priority
		switch rec.Priority {
		case "high":
			stats["high_priority"] = stats["high_priority"].(int) + 1
		case "medium":
			stats["medium_priority"] = stats["medium_priority"].(int) + 1
		case "low":
			stats["low_priority"] = stats["low_priority"].(int) + 1
		}

		// Count by type
		if byType, ok := stats["by_type"].(map[string]int); ok {
			if count, exists := byType[rec.Type]; exists {
				byType[rec.Type] = count + 1
			}
		}

		// Sum savings
		totalSavings += rec.EstimatedSavings
	}

	stats["total_savings"] = totalSavings

	return stats
}
