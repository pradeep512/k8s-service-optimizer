package optimizer

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/k8s-service-optimizer/backend/internal/models"
)

// recommendationGenerator handles generation of optimization recommendations
type recommendationGenerator struct {
	optimizer *OptimizerEngine
}

// newRecommendationGenerator creates a new recommendation generator
func newRecommendationGenerator(opt *OptimizerEngine) *recommendationGenerator {
	return &recommendationGenerator{
		optimizer: opt,
	}
}

// generateRecommendations generates all applicable recommendations for an analysis
func (rg *recommendationGenerator) generateRecommendations(analysis *analysisResult) ([]models.Recommendation, error) {
	var recommendations []models.Recommendation

	// Generate resource recommendations (CPU/Memory right-sizing)
	resourceRecs := rg.generateResourceRecommendations(analysis)
	recommendations = append(recommendations, resourceRecs...)

	// Generate HPA recommendations if HPA exists
	if analysis.Deployment.HasHPA {
		hpaRecs := rg.generateHPARecommendations(analysis)
		recommendations = append(recommendations, hpaRecs...)
	}

	// Generate scaling recommendations
	scalingRecs := rg.generateScalingRecommendations(analysis)
	recommendations = append(recommendations, scalingRecs...)

	return recommendations, nil
}

// generateResourceRecommendations generates CPU and memory right-sizing recommendations
func (rg *recommendationGenerator) generateResourceRecommendations(analysis *analysisResult) []models.Recommendation {
	var recommendations []models.Recommendation
	metrics := &analysis.Deployment

	// Check if we need CPU adjustment
	if analysis.CPUOverProvisioned || analysis.CPUUnderProvisioned {
		rec := rg.generateCPURecommendation(analysis)
		if rec != nil {
			recommendations = append(recommendations, *rec)
		}
	}

	// Check if we need memory adjustment
	if analysis.MemoryOverProvisioned || analysis.MemoryUnderProvisioned {
		rec := rg.generateMemoryRecommendation(analysis)
		if rec != nil {
			recommendations = append(recommendations, *rec)
		}
	}

	// Generate combined resource recommendation if both need adjustment
	if (analysis.CPUOverProvisioned || analysis.CPUUnderProvisioned) &&
		(analysis.MemoryOverProvisioned || analysis.MemoryUnderProvisioned) {
		rec := rg.generateCombinedResourceRecommendation(analysis)
		if rec != nil {
			recommendations = append(recommendations, *rec)
		}
	}

	// If no specific issues but efficiency is low, suggest optimization
	if len(recommendations) == 0 && analysis.OverallScore < 70 {
		if metrics.CPURequested > 0 && metrics.MemoryRequested > 0 {
			rec := rg.generateGeneralOptimizationRecommendation(analysis)
			if rec != nil {
				recommendations = append(recommendations, *rec)
			}
		}
	}

	return recommendations
}

// generateCPURecommendation generates a CPU-specific recommendation
func (rg *recommendationGenerator) generateCPURecommendation(analysis *analysisResult) *models.Recommendation {
	metrics := &analysis.Deployment

	if metrics.CPURequested == 0 {
		return nil
	}

	// Calculate recommended CPU
	var recommendedCPU int64
	var description string

	if analysis.CPUOverProvisioned {
		// Reduce CPU: P95 usage * 1.2 (20% buffer)
		recommendedCPU = int64(float64(metrics.CPUP95) * rg.optimizer.config.OverProvisionedBuffer)
		description = fmt.Sprintf("Reduce CPU request from %s to %s (P95 usage: %s, utilization: %.1f%%)",
			formatResourceQuantity(metrics.CPURequested, "cpu"),
			formatResourceQuantity(recommendedCPU, "cpu"),
			formatResourceQuantity(metrics.CPUP95, "cpu"),
			analysis.CPUUtilization*100)
	} else if analysis.CPUUnderProvisioned {
		// Increase CPU: P95 usage * 1.5 (50% buffer)
		recommendedCPU = int64(float64(metrics.CPUP95) * rg.optimizer.config.UnderProvisionedBuffer)
		description = fmt.Sprintf("Increase CPU request from %s to %s (P95 usage: %s, utilization: %.1f%%)",
			formatResourceQuantity(metrics.CPURequested, "cpu"),
			formatResourceQuantity(recommendedCPU, "cpu"),
			formatResourceQuantity(metrics.CPUP95, "cpu"),
			analysis.CPUUtilization*100)
	} else {
		return nil
	}

	// Calculate savings
	currentCost := rg.calculateCPUCost(metrics.CPURequested)
	recommendedCost := rg.calculateCPUCost(recommendedCPU)
	savings := currentCost - recommendedCost

	// Determine priority
	priority := rg.optimizer.scorer.getPriorityLevel(analysis, savings)

	// Build current and recommended configs
	currentConfig := resourceConfig{
		CPURequest: formatResourceQuantity(metrics.CPURequested, "cpu"),
		CPULimit:   formatResourceQuantity(metrics.CPULimit, "cpu"),
	}

	recommendedConfig := resourceConfig{
		CPURequest: formatResourceQuantity(recommendedCPU, "cpu"),
		CPULimit:   formatResourceQuantity(recommendedCPU*2, "cpu"), // Limit = 2x request
	}

	impact := rg.optimizer.scorer.formatImpactMessage(RecommendationTypeResource, analysis, savings)

	return &models.Recommendation{
		ID:                uuid.New().String(),
		Type:              string(RecommendationTypeResource),
		Namespace:         metrics.Namespace,
		Deployment:        metrics.Deployment,
		Priority:          string(priority),
		Description:       description,
		CurrentConfig:     convertResourceConfigToMap(currentConfig),
		RecommendedConfig: convertResourceConfigToMap(recommendedConfig),
		EstimatedSavings:  savings,
		Impact:            impact,
		CreatedAt:         time.Now(),
	}
}

// generateMemoryRecommendation generates a memory-specific recommendation
func (rg *recommendationGenerator) generateMemoryRecommendation(analysis *analysisResult) *models.Recommendation {
	metrics := &analysis.Deployment

	if metrics.MemoryRequested == 0 {
		return nil
	}

	// Calculate recommended Memory
	var recommendedMemory int64
	var description string

	if analysis.MemoryOverProvisioned {
		// Reduce Memory: P95 usage * 1.2 (20% buffer)
		recommendedMemory = int64(float64(metrics.MemoryP95) * rg.optimizer.config.OverProvisionedBuffer)
		description = fmt.Sprintf("Reduce memory request from %s to %s (P95 usage: %s, utilization: %.1f%%)",
			formatResourceQuantity(metrics.MemoryRequested, "memory"),
			formatResourceQuantity(recommendedMemory, "memory"),
			formatResourceQuantity(metrics.MemoryP95, "memory"),
			analysis.MemoryUtilization*100)
	} else if analysis.MemoryUnderProvisioned {
		// Increase Memory: P95 usage * 1.5 (50% buffer)
		recommendedMemory = int64(float64(metrics.MemoryP95) * rg.optimizer.config.UnderProvisionedBuffer)
		description = fmt.Sprintf("Increase memory request from %s to %s (P95 usage: %s, utilization: %.1f%%)",
			formatResourceQuantity(metrics.MemoryRequested, "memory"),
			formatResourceQuantity(recommendedMemory, "memory"),
			formatResourceQuantity(metrics.MemoryP95, "memory"),
			analysis.MemoryUtilization*100)
	} else {
		return nil
	}

	// Calculate savings
	currentCost := rg.calculateMemoryCost(metrics.MemoryRequested)
	recommendedCost := rg.calculateMemoryCost(recommendedMemory)
	savings := currentCost - recommendedCost

	// Determine priority
	priority := rg.optimizer.scorer.getPriorityLevel(analysis, savings)

	// Build current and recommended configs
	currentConfig := resourceConfig{
		MemoryRequest: formatResourceQuantity(metrics.MemoryRequested, "memory"),
		MemoryLimit:   formatResourceQuantity(metrics.MemoryLimit, "memory"),
	}

	recommendedConfig := resourceConfig{
		MemoryRequest: formatResourceQuantity(recommendedMemory, "memory"),
		MemoryLimit:   formatResourceQuantity(recommendedMemory*2, "memory"), // Limit = 2x request
	}

	impact := rg.optimizer.scorer.formatImpactMessage(RecommendationTypeResource, analysis, savings)

	return &models.Recommendation{
		ID:                uuid.New().String(),
		Type:              string(RecommendationTypeResource),
		Namespace:         metrics.Namespace,
		Deployment:        metrics.Deployment,
		Priority:          string(priority),
		Description:       description,
		CurrentConfig:     convertResourceConfigToMap(currentConfig),
		RecommendedConfig: convertResourceConfigToMap(recommendedConfig),
		EstimatedSavings:  savings,
		Impact:            impact,
		CreatedAt:         time.Now(),
	}
}

// generateCombinedResourceRecommendation generates a combined CPU and memory recommendation
func (rg *recommendationGenerator) generateCombinedResourceRecommendation(analysis *analysisResult) *models.Recommendation {
	metrics := &analysis.Deployment

	if metrics.CPURequested == 0 || metrics.MemoryRequested == 0 {
		return nil
	}

	// Calculate recommended CPU
	var recommendedCPU int64
	if analysis.CPUOverProvisioned {
		recommendedCPU = int64(float64(metrics.CPUP95) * rg.optimizer.config.OverProvisionedBuffer)
	} else if analysis.CPUUnderProvisioned {
		recommendedCPU = int64(float64(metrics.CPUP95) * rg.optimizer.config.UnderProvisionedBuffer)
	} else {
		recommendedCPU = metrics.CPURequested
	}

	// Calculate recommended Memory
	var recommendedMemory int64
	if analysis.MemoryOverProvisioned {
		recommendedMemory = int64(float64(metrics.MemoryP95) * rg.optimizer.config.OverProvisionedBuffer)
	} else if analysis.MemoryUnderProvisioned {
		recommendedMemory = int64(float64(metrics.MemoryP95) * rg.optimizer.config.UnderProvisionedBuffer)
	} else {
		recommendedMemory = metrics.MemoryRequested
	}

	// Build description
	description := fmt.Sprintf("Optimize resources: CPU %s→%s (%.1f%% util), Memory %s→%s (%.1f%% util)",
		formatResourceQuantity(metrics.CPURequested, "cpu"),
		formatResourceQuantity(recommendedCPU, "cpu"),
		analysis.CPUUtilization*100,
		formatResourceQuantity(metrics.MemoryRequested, "memory"),
		formatResourceQuantity(recommendedMemory, "memory"),
		analysis.MemoryUtilization*100)

	// Calculate total savings
	cpuSavings := rg.calculateCPUCost(metrics.CPURequested) - rg.calculateCPUCost(recommendedCPU)
	memorySavings := rg.calculateMemoryCost(metrics.MemoryRequested) - rg.calculateMemoryCost(recommendedMemory)
	totalSavings := cpuSavings + memorySavings

	// Determine priority
	priority := rg.optimizer.scorer.getPriorityLevel(analysis, totalSavings)

	// Build configs
	currentConfig := resourceConfig{
		CPURequest:    formatResourceQuantity(metrics.CPURequested, "cpu"),
		CPULimit:      formatResourceQuantity(metrics.CPULimit, "cpu"),
		MemoryRequest: formatResourceQuantity(metrics.MemoryRequested, "memory"),
		MemoryLimit:   formatResourceQuantity(metrics.MemoryLimit, "memory"),
	}

	recommendedConfig := resourceConfig{
		CPURequest:    formatResourceQuantity(recommendedCPU, "cpu"),
		CPULimit:      formatResourceQuantity(recommendedCPU*2, "cpu"),
		MemoryRequest: formatResourceQuantity(recommendedMemory, "memory"),
		MemoryLimit:   formatResourceQuantity(recommendedMemory*2, "memory"),
	}

	impact := rg.optimizer.scorer.formatImpactMessage(RecommendationTypeResource, analysis, totalSavings)

	return &models.Recommendation{
		ID:                uuid.New().String(),
		Type:              string(RecommendationTypeResource),
		Namespace:         metrics.Namespace,
		Deployment:        metrics.Deployment,
		Priority:          string(priority),
		Description:       description,
		CurrentConfig:     convertResourceConfigToMap(currentConfig),
		RecommendedConfig: convertResourceConfigToMap(recommendedConfig),
		EstimatedSavings:  totalSavings,
		Impact:            impact,
		CreatedAt:         time.Now(),
	}
}

// generateHPARecommendations generates HPA optimization recommendations
func (rg *recommendationGenerator) generateHPARecommendations(analysis *analysisResult) []models.Recommendation {
	var recommendations []models.Recommendation
	metrics := &analysis.Deployment

	if !metrics.HasHPA {
		return recommendations
	}

	// Check if min replicas should be adjusted
	if analysis.HPAIdleAtMinimum {
		rec := rg.generateMinReplicasRecommendation(analysis)
		if rec != nil {
			recommendations = append(recommendations, *rec)
		}
	}

	// Check if max replicas should be adjusted
	if analysis.HPAHitCeiling {
		rec := rg.generateMaxReplicasRecommendation(analysis)
		if rec != nil {
			recommendations = append(recommendations, *rec)
		}
	}

	// Check if target CPU should be adjusted
	if analysis.HPANeedsOptimization {
		rec := rg.generateTargetCPURecommendation(analysis)
		if rec != nil {
			recommendations = append(recommendations, *rec)
		}
	}

	// Generate combined HPA optimization if multiple issues
	if len(recommendations) > 1 {
		rec := rg.generateCombinedHPARecommendation(analysis)
		if rec != nil {
			recommendations = append(recommendations, *rec)
		}
	}

	return recommendations
}

// generateMinReplicasRecommendation generates recommendation to adjust min replicas
func (rg *recommendationGenerator) generateMinReplicasRecommendation(analysis *analysisResult) *models.Recommendation {
	metrics := &analysis.Deployment

	// If consistently at minimum, consider reducing min replicas
	recommendedMin := metrics.MinReplicas
	if metrics.MinReplicas > 1 {
		recommendedMin = metrics.MinReplicas - 1
	}

	if recommendedMin == metrics.MinReplicas {
		return nil
	}

	description := fmt.Sprintf("Reduce HPA min replicas from %d to %d (consistently idle at minimum)",
		metrics.MinReplicas, recommendedMin)

	// Calculate potential savings
	savings := rg.calculateReplicaCostSavings(metrics, 1)

	currentConfig := hpaConfig{
		MinReplicas: metrics.MinReplicas,
		MaxReplicas: metrics.MaxReplicas,
		TargetCPU:   metrics.HPATargetCPU,
	}

	recommendedConfig := hpaConfig{
		MinReplicas: recommendedMin,
		MaxReplicas: metrics.MaxReplicas,
		TargetCPU:   metrics.HPATargetCPU,
	}

	priority := rg.optimizer.scorer.getPriorityLevel(analysis, savings)
	impact := rg.optimizer.scorer.formatImpactMessage(RecommendationTypeHPA, analysis, savings)

	return &models.Recommendation{
		ID:                uuid.New().String(),
		Type:              string(RecommendationTypeHPA),
		Namespace:         metrics.Namespace,
		Deployment:        metrics.Deployment,
		Priority:          string(priority),
		Description:       description,
		CurrentConfig:     convertHPAConfigToMap(currentConfig),
		RecommendedConfig: convertHPAConfigToMap(recommendedConfig),
		EstimatedSavings:  savings,
		Impact:            impact,
		CreatedAt:         time.Now(),
	}
}

// generateMaxReplicasRecommendation generates recommendation to adjust max replicas
func (rg *recommendationGenerator) generateMaxReplicasRecommendation(analysis *analysisResult) *models.Recommendation {
	metrics := &analysis.Deployment

	// If frequently hitting ceiling, increase max replicas
	recommendedMax := metrics.MaxReplicas + 2

	description := fmt.Sprintf("Increase HPA max replicas from %d to %d (frequently hitting ceiling)",
		metrics.MaxReplicas, recommendedMax)

	// No savings, actually cost increase, but prevents performance issues
	savings := 0.0

	currentConfig := hpaConfig{
		MinReplicas: metrics.MinReplicas,
		MaxReplicas: metrics.MaxReplicas,
		TargetCPU:   metrics.HPATargetCPU,
	}

	recommendedConfig := hpaConfig{
		MinReplicas: metrics.MinReplicas,
		MaxReplicas: recommendedMax,
		TargetCPU:   metrics.HPATargetCPU,
	}

	priority := rg.optimizer.scorer.getPriorityLevel(analysis, savings)
	impact := "Medium risk - increasing capacity to handle peak load without performance degradation"

	return &models.Recommendation{
		ID:                uuid.New().String(),
		Type:              string(RecommendationTypeHPA),
		Namespace:         metrics.Namespace,
		Deployment:        metrics.Deployment,
		Priority:          string(priority),
		Description:       description,
		CurrentConfig:     convertHPAConfigToMap(currentConfig),
		RecommendedConfig: convertHPAConfigToMap(recommendedConfig),
		EstimatedSavings:  savings,
		Impact:            impact,
		CreatedAt:         time.Now(),
	}
}

// generateTargetCPURecommendation generates recommendation to adjust target CPU
func (rg *recommendationGenerator) generateTargetCPURecommendation(analysis *analysisResult) *models.Recommendation {
	metrics := &analysis.Deployment

	if metrics.HPATargetCPU == 0 || metrics.HPACurrentCPU == 0 {
		return nil
	}

	// Adjust target based on current vs target
	var recommendedTarget int32
	var description string

	if metrics.HPACurrentCPU > metrics.HPATargetCPU {
		// Current is higher than target, target might be too aggressive
		recommendedTarget = (metrics.HPATargetCPU + metrics.HPACurrentCPU) / 2
		description = fmt.Sprintf("Adjust HPA target CPU from %d%% to %d%% (current avg: %d%%)",
			metrics.HPATargetCPU, recommendedTarget, metrics.HPACurrentCPU)
	} else {
		// Current is lower than target, target might be too conservative
		recommendedTarget = metrics.HPACurrentCPU + 10
		if recommendedTarget > 80 {
			recommendedTarget = 80
		}
		description = fmt.Sprintf("Adjust HPA target CPU from %d%% to %d%% (current avg: %d%%)",
			metrics.HPATargetCPU, recommendedTarget, metrics.HPACurrentCPU)
	}

	currentConfig := hpaConfig{
		MinReplicas: metrics.MinReplicas,
		MaxReplicas: metrics.MaxReplicas,
		TargetCPU:   metrics.HPATargetCPU,
	}

	recommendedConfig := hpaConfig{
		MinReplicas: metrics.MinReplicas,
		MaxReplicas: metrics.MaxReplicas,
		TargetCPU:   recommendedTarget,
	}

	savings := 0.0
	priority := PriorityMedium
	impact := rg.optimizer.scorer.formatImpactMessage(RecommendationTypeHPA, analysis, savings)

	return &models.Recommendation{
		ID:                uuid.New().String(),
		Type:              string(RecommendationTypeHPA),
		Namespace:         metrics.Namespace,
		Deployment:        metrics.Deployment,
		Priority:          string(priority),
		Description:       description,
		CurrentConfig:     convertHPAConfigToMap(currentConfig),
		RecommendedConfig: convertHPAConfigToMap(recommendedConfig),
		EstimatedSavings:  savings,
		Impact:            impact,
		CreatedAt:         time.Now(),
	}
}

// generateCombinedHPARecommendation generates a comprehensive HPA optimization recommendation
func (rg *recommendationGenerator) generateCombinedHPARecommendation(analysis *analysisResult) *models.Recommendation {
	metrics := &analysis.Deployment

	// Determine all recommended changes
	recommendedMin := metrics.MinReplicas
	recommendedMax := metrics.MaxReplicas
	recommendedTarget := metrics.HPATargetCPU

	if analysis.HPAIdleAtMinimum && metrics.MinReplicas > 1 {
		recommendedMin = metrics.MinReplicas - 1
	}

	if analysis.HPAHitCeiling {
		recommendedMax = metrics.MaxReplicas + 2
	}

	if analysis.HPANeedsOptimization && metrics.HPACurrentCPU > 0 {
		if metrics.HPACurrentCPU > metrics.HPATargetCPU {
			recommendedTarget = (metrics.HPATargetCPU + metrics.HPACurrentCPU) / 2
		} else {
			recommendedTarget = metrics.HPACurrentCPU + 10
			if recommendedTarget > 80 {
				recommendedTarget = 80
			}
		}
	}

	description := fmt.Sprintf("Optimize HPA configuration: min replicas %d→%d, max replicas %d→%d, target CPU %d%%→%d%%",
		metrics.MinReplicas, recommendedMin,
		metrics.MaxReplicas, recommendedMax,
		metrics.HPATargetCPU, recommendedTarget)

	currentConfig := hpaConfig{
		MinReplicas: metrics.MinReplicas,
		MaxReplicas: metrics.MaxReplicas,
		TargetCPU:   metrics.HPATargetCPU,
	}

	recommendedConfig := hpaConfig{
		MinReplicas: recommendedMin,
		MaxReplicas: recommendedMax,
		TargetCPU:   recommendedTarget,
	}

	// Calculate savings from reducing min replicas
	savings := 0.0
	if recommendedMin < metrics.MinReplicas {
		savings = rg.calculateReplicaCostSavings(metrics, int(metrics.MinReplicas-recommendedMin))
	}

	priority := rg.optimizer.scorer.getPriorityLevel(analysis, savings)
	impact := rg.optimizer.scorer.formatImpactMessage(RecommendationTypeHPA, analysis, savings)

	return &models.Recommendation{
		ID:                uuid.New().String(),
		Type:              string(RecommendationTypeHPA),
		Namespace:         metrics.Namespace,
		Deployment:        metrics.Deployment,
		Priority:          string(priority),
		Description:       description,
		CurrentConfig:     convertHPAConfigToMap(currentConfig),
		RecommendedConfig: convertHPAConfigToMap(recommendedConfig),
		EstimatedSavings:  savings,
		Impact:            impact,
		CreatedAt:         time.Now(),
	}
}

// generateScalingRecommendations generates replica scaling recommendations
func (rg *recommendationGenerator) generateScalingRecommendations(analysis *analysisResult) []models.Recommendation {
	var recommendations []models.Recommendation
	metrics := &analysis.Deployment

	// Don't generate scaling recommendations if HPA is managing replicas
	if metrics.HasHPA {
		return recommendations
	}

	// Check if we should scale up or down
	if analysis.CPUUtilization > 0.8 || analysis.MemoryUtilization > 0.8 {
		// High utilization - recommend scaling up
		rec := rg.generateScaleUpRecommendation(analysis)
		if rec != nil {
			recommendations = append(recommendations, *rec)
		}
	} else if analysis.CPUUtilization < 0.5 && analysis.MemoryUtilization < 0.5 && metrics.CurrentReplicas > 1 {
		// Low utilization - recommend scaling down
		rec := rg.generateScaleDownRecommendation(analysis)
		if rec != nil {
			recommendations = append(recommendations, *rec)
		}
	}

	return recommendations
}

// generateScaleUpRecommendation generates recommendation to scale up replicas
func (rg *recommendationGenerator) generateScaleUpRecommendation(analysis *analysisResult) *models.Recommendation {
	metrics := &analysis.Deployment

	recommendedReplicas := metrics.CurrentReplicas + 1

	description := fmt.Sprintf("Scale up from %d to %d replicas (high utilization: CPU %.1f%%, Memory %.1f%%)",
		metrics.CurrentReplicas, recommendedReplicas,
		analysis.CPUUtilization*100, analysis.MemoryUtilization*100)

	currentConfig := scalingConfig{
		Replicas: metrics.CurrentReplicas,
	}

	recommendedConfig := scalingConfig{
		Replicas: recommendedReplicas,
	}

	priority := PriorityHigh
	impact := "Medium risk - adding capacity to handle current load"

	return &models.Recommendation{
		ID:                uuid.New().String(),
		Type:              string(RecommendationTypeScaling),
		Namespace:         metrics.Namespace,
		Deployment:        metrics.Deployment,
		Priority:          string(priority),
		Description:       description,
		CurrentConfig:     convertScalingConfigToMap(currentConfig),
		RecommendedConfig: convertScalingConfigToMap(recommendedConfig),
		EstimatedSavings:  0.0, // Scaling up costs money
		Impact:            impact,
		CreatedAt:         time.Now(),
	}
}

// generateScaleDownRecommendation generates recommendation to scale down replicas
func (rg *recommendationGenerator) generateScaleDownRecommendation(analysis *analysisResult) *models.Recommendation {
	metrics := &analysis.Deployment

	recommendedReplicas := metrics.CurrentReplicas - 1
	if recommendedReplicas < 1 {
		recommendedReplicas = 1
	}

	description := fmt.Sprintf("Scale down from %d to %d replicas (low utilization: CPU %.1f%%, Memory %.1f%%)",
		metrics.CurrentReplicas, recommendedReplicas,
		analysis.CPUUtilization*100, analysis.MemoryUtilization*100)

	// Calculate savings
	savings := rg.calculateReplicaCostSavings(metrics, 1)

	currentConfig := scalingConfig{
		Replicas: metrics.CurrentReplicas,
	}

	recommendedConfig := scalingConfig{
		Replicas: recommendedReplicas,
	}

	priority := rg.optimizer.scorer.getPriorityLevel(analysis, savings)
	impact := rg.optimizer.scorer.formatImpactMessage(RecommendationTypeScaling, analysis, savings)

	return &models.Recommendation{
		ID:                uuid.New().String(),
		Type:              string(RecommendationTypeScaling),
		Namespace:         metrics.Namespace,
		Deployment:        metrics.Deployment,
		Priority:          string(priority),
		Description:       description,
		CurrentConfig:     convertScalingConfigToMap(currentConfig),
		RecommendedConfig: convertScalingConfigToMap(recommendedConfig),
		EstimatedSavings:  savings,
		Impact:            impact,
		CreatedAt:         time.Now(),
	}
}

// generateGeneralOptimizationRecommendation generates a general optimization recommendation
func (rg *recommendationGenerator) generateGeneralOptimizationRecommendation(analysis *analysisResult) *models.Recommendation {
	metrics := &analysis.Deployment

	description := fmt.Sprintf("Consider optimizing resources for better efficiency (current score: %.1f/100)",
		analysis.OverallScore)

	// Calculate optimal resources based on P95
	recommendedCPU := int64(float64(metrics.CPUP95) * 1.2)
	recommendedMemory := int64(float64(metrics.MemoryP95) * 1.2)

	currentConfig := resourceConfig{
		CPURequest:    formatResourceQuantity(metrics.CPURequested, "cpu"),
		CPULimit:      formatResourceQuantity(metrics.CPULimit, "cpu"),
		MemoryRequest: formatResourceQuantity(metrics.MemoryRequested, "memory"),
		MemoryLimit:   formatResourceQuantity(metrics.MemoryLimit, "memory"),
	}

	recommendedConfig := resourceConfig{
		CPURequest:    formatResourceQuantity(recommendedCPU, "cpu"),
		CPULimit:      formatResourceQuantity(recommendedCPU*2, "cpu"),
		MemoryRequest: formatResourceQuantity(recommendedMemory, "memory"),
		MemoryLimit:   formatResourceQuantity(recommendedMemory*2, "memory"),
	}

	cpuSavings := rg.calculateCPUCost(metrics.CPURequested) - rg.calculateCPUCost(recommendedCPU)
	memorySavings := rg.calculateMemoryCost(metrics.MemoryRequested) - rg.calculateMemoryCost(recommendedMemory)
	totalSavings := cpuSavings + memorySavings

	priority := PriorityLow
	impact := rg.optimizer.scorer.formatImpactMessage(RecommendationTypeResource, analysis, totalSavings)

	return &models.Recommendation{
		ID:                uuid.New().String(),
		Type:              string(RecommendationTypeResource),
		Namespace:         metrics.Namespace,
		Deployment:        metrics.Deployment,
		Priority:          string(priority),
		Description:       description,
		CurrentConfig:     convertResourceConfigToMap(currentConfig),
		RecommendedConfig: convertResourceConfigToMap(recommendedConfig),
		EstimatedSavings:  totalSavings,
		Impact:            impact,
		CreatedAt:         time.Now(),
	}
}

// Cost calculation helper methods

// calculateCPUCost calculates monthly cost for CPU (in millicores)
func (rg *recommendationGenerator) calculateCPUCost(millicores int64) float64 {
	vcpus := convertMillicoresToVCPU(millicores)
	hourlyRate := vcpus * rg.optimizer.config.CPUCostPerVCPUHour
	return hourlyRate * 24 * 30 // Monthly cost
}

// calculateMemoryCost calculates monthly cost for memory (in bytes)
func (rg *recommendationGenerator) calculateMemoryCost(bytes int64) float64 {
	gb := convertBytesToGB(bytes)
	hourlyRate := gb * rg.optimizer.config.MemoryCostPerGBHour
	return hourlyRate * 24 * 30 // Monthly cost
}

// calculateReplicaCostSavings calculates savings from reducing replicas
func (rg *recommendationGenerator) calculateReplicaCostSavings(metrics *deploymentMetrics, replicaReduction int) float64 {
	cpuCostPerReplica := rg.calculateCPUCost(metrics.CPURequested)
	memoryCostPerReplica := rg.calculateMemoryCost(metrics.MemoryRequested)
	costPerReplica := cpuCostPerReplica + memoryCostPerReplica

	return costPerReplica * float64(replicaReduction)
}
