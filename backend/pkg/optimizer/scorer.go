package optimizer

import (
	"fmt"
	"math"
)

// scorer handles efficiency scoring calculations
type scorer struct {
	optimizer *OptimizerEngine
}

// newScorer creates a new scorer
func newScorer(opt *OptimizerEngine) *scorer {
	return &scorer{
		optimizer: opt,
	}
}

// calculateEfficiencyScore calculates the overall efficiency score for a deployment (0-100)
func (s *scorer) calculateEfficiencyScore(analysis *analysisResult) float64 {
	// Score is already calculated in the analysis result
	return analysis.OverallScore
}

// calculateResourceUtilizationScore calculates resource utilization score (0-100)
// This component accounts for 50% of the overall score
func (s *scorer) calculateResourceUtilizationScore(analysis *analysisResult) float64 {
	// Average of CPU and memory utilization scores
	cpuScore := s.calculateSingleResourceUtilizationScore(analysis.CPUUtilization)
	memoryScore := s.calculateSingleResourceUtilizationScore(analysis.MemoryUtilization)

	return (cpuScore + memoryScore) / 2.0
}

// calculateSingleResourceUtilizationScore calculates utilization score for a single resource
func (s *scorer) calculateSingleResourceUtilizationScore(utilization float64) float64 {
	optMin := s.optimizer.config.OptimalUtilizationMin
	optMax := s.optimizer.config.OptimalUtilizationMax

	if utilization >= optMin && utilization <= optMax {
		// Optimal range - full score
		return 100.0
	} else if utilization < optMin {
		// Under-utilized (over-provisioned)
		// Linear decrease from 100 at optMin to 0 at 0%
		if utilization <= 0 {
			return 0.0
		}
		return 100.0 * (utilization / optMin)
	} else {
		// Over-utilized (under-provisioned)
		// Linear decrease from 100 at optMax to 0 at 200%
		excess := utilization - optMax
		maxExcess := 2.0 - optMax
		if excess >= maxExcess {
			return 0.0
		}
		return math.Max(0, 100.0*(1.0-(excess/maxExcess)))
	}
}

// calculateStabilityScore calculates stability score (0-100)
// This component accounts for 30% of the overall score
func (s *scorer) calculateStabilityScore(analysis *analysisResult) float64 {
	metrics := &analysis.Deployment
	score := 100.0

	// Factor 1: Restart count (0 restarts = perfect)
	// Each restart reduces the score
	restartPenalty := float64(metrics.RestartCount) * 5.0
	score -= restartPenalty

	// Factor 2: Resource usage stability (low variance is good)
	variancePenalty := s.calculateVariancePenalty(analysis.CPUVariance, analysis.MemoryVariance)
	score -= variancePenalty

	// Factor 3: Scaling stability (for HPA-enabled deployments)
	if metrics.HasHPA {
		scalingPenalty := s.calculateScalingPenalty(analysis.HPAScalingFrequency, analysis.HPAScalingAmplitude)
		score -= scalingPenalty
	}

	// Ensure score is between 0 and 100
	return math.Max(0, math.Min(100, score))
}

// calculateVariancePenalty calculates penalty for resource usage variance
func (s *scorer) calculateVariancePenalty(cpuVariance, memoryVariance float64) float64 {
	penalty := 0.0

	// CPU variance penalty (up to 10 points)
	// High variance (> 1000 millicores squared) indicates instability
	if cpuVariance > 1000 {
		penalty += math.Min(10, cpuVariance/1000)
	}

	// Memory variance penalty (up to 10 points)
	// High variance (> 1GB squared) indicates instability
	if memoryVariance > 1000000000 { // 1GB in bytes
		penalty += math.Min(10, memoryVariance/1000000000)
	}

	return penalty
}

// calculateScalingPenalty calculates penalty for HPA scaling instability
func (s *scorer) calculateScalingPenalty(frequency, amplitude float64) float64 {
	penalty := 0.0

	// Frequency penalty (up to 20 points)
	// Scaling more than 10 times per day is considered unstable
	if frequency > 10 {
		penalty += math.Min(20, (frequency-10)*2)
	}

	// Amplitude penalty (up to 10 points)
	// Large swings in replica count indicate instability
	if amplitude > 5 {
		penalty += math.Min(10, (amplitude - 5))
	}

	return penalty
}

// calculateCostEfficiencyScore calculates cost efficiency score (0-100)
// This component accounts for 20% of the overall score
func (s *scorer) calculateCostEfficiencyScore(analysis *analysisResult) float64 {
	score := 100.0

	// Calculate waste for CPU
	if analysis.CPUOverProvisioned {
		cpuWaste := 1.0 - analysis.CPUUtilization
		// Each 10% of waste reduces score by 5 points (up to 50 points)
		score -= cpuWaste * 50
	}

	// Calculate waste for Memory
	if analysis.MemoryOverProvisioned {
		memoryWaste := 1.0 - analysis.MemoryUtilization
		// Each 10% of waste reduces score by 5 points (up to 50 points)
		score -= memoryWaste * 50
	}

	// Ensure score is between 0 and 100
	return math.Max(0, math.Min(100, score))
}

// calculateWastedResources calculates the amount of wasted resources
func (s *scorer) calculateWastedResources(analysis *analysisResult) (wastedCPU, wastedMemory int64) {
	metrics := &analysis.Deployment

	// Calculate wasted CPU (requested but not used)
	if analysis.CPUOverProvisioned && metrics.CPURequested > 0 {
		wastedCPU = metrics.CPURequested - metrics.CPUP95
		if wastedCPU < 0 {
			wastedCPU = 0
		}
	}

	// Calculate wasted Memory (requested but not used)
	if analysis.MemoryOverProvisioned && metrics.MemoryRequested > 0 {
		wastedMemory = metrics.MemoryRequested - metrics.MemoryP95
		if wastedMemory < 0 {
			wastedMemory = 0
		}
	}

	return wastedCPU, wastedMemory
}

// calculateWastedCost calculates the monthly cost of wasted resources
func (s *scorer) calculateWastedCost(wastedCPU, wastedMemory int64) float64 {
	// Convert to appropriate units
	wastedVCPU := convertMillicoresToVCPU(wastedCPU)
	wastedGB := convertBytesToGB(wastedMemory)

	// Calculate hourly cost
	hourlyCPUCost := wastedVCPU * s.optimizer.config.CPUCostPerVCPUHour
	hourlyMemoryCost := wastedGB * s.optimizer.config.MemoryCostPerGBHour

	// Calculate monthly cost (24 hours * 30 days)
	monthlyCost := (hourlyCPUCost + hourlyMemoryCost) * 24 * 30

	return monthlyCost
}

// scoreResourceRequest scores how well a resource request is configured
func (s *scorer) scoreResourceRequest(requested, actual int64) float64 {
	if requested == 0 {
		return 0.0
	}

	utilization := float64(actual) / float64(requested)
	return s.calculateSingleResourceUtilizationScore(utilization)
}

// calculateHealthScore calculates an overall health score for the deployment
// This is similar to efficiency score but focuses more on stability and correctness
func (s *scorer) calculateHealthScore(analysis *analysisResult) float64 {
	metrics := &analysis.Deployment
	score := 100.0

	// Deduct for critical issues
	if analysis.CPUUnderProvisioned {
		score -= 20.0 // Critical: CPU under-provisioned
	}
	if analysis.MemoryUnderProvisioned {
		score -= 20.0 // Critical: Memory under-provisioned
	}

	// Deduct for restarts
	if metrics.RestartCount > 0 {
		restartPenalty := math.Min(30, float64(metrics.RestartCount)*3)
		score -= restartPenalty
	}

	// Deduct for HPA issues
	if metrics.HasHPA {
		if analysis.HPAHitCeiling {
			score -= 15.0 // HPA hitting max replicas frequently
		}
		if analysis.HPANeedsOptimization {
			score -= 10.0 // HPA not well tuned
		}
	}

	// Minor deduction for over-provisioning (waste)
	if analysis.CPUOverProvisioned {
		score -= 5.0
	}
	if analysis.MemoryOverProvisioned {
		score -= 5.0
	}

	return math.Max(0, math.Min(100, score))
}

// getPriorityLevel determines the priority level based on severity
func (s *scorer) getPriorityLevel(analysis *analysisResult, potentialSavings float64) recommendationPriority {
	// High priority if:
	// 1. Under-provisioned (critical)
	// 2. High savings potential (> $50/month)
	// 3. Low health score (< 60)

	healthScore := s.calculateHealthScore(analysis)

	if analysis.CPUUnderProvisioned || analysis.MemoryUnderProvisioned {
		return PriorityHigh
	}

	if potentialSavings > 50 {
		return PriorityHigh
	}

	if healthScore < 60 {
		return PriorityHigh
	}

	// Medium priority if:
	// 1. Moderate savings (> $20/month)
	// 2. HPA needs optimization
	// 3. Medium health score (60-80)

	if potentialSavings > 20 {
		return PriorityMedium
	}

	if analysis.HPANeedsOptimization {
		return PriorityMedium
	}

	if healthScore < 80 {
		return PriorityMedium
	}

	// Low priority for minor optimizations
	return PriorityLow
}

// getRiskLevel determines the risk level of applying a recommendation
func (s *scorer) getRiskLevel(recType recommendationType, analysis *analysisResult) string {
	switch recType {
	case RecommendationTypeResource:
		// Reducing resources is lower risk if currently over-provisioned
		// Increasing resources is low risk
		if analysis.CPUUnderProvisioned || analysis.MemoryUnderProvisioned {
			return "Low risk - increasing resources to prevent issues"
		}
		if analysis.CPUOverProvisioned || analysis.MemoryOverProvisioned {
			return "Low risk - reducing over-provisioned resources"
		}
		return "Medium risk - adjusting resource allocation"

	case RecommendationTypeHPA:
		// HPA changes are generally medium risk
		if analysis.HPAHitCeiling {
			return "Low risk - increasing max replicas to handle load"
		}
		return "Medium risk - optimizing autoscaling configuration"

	case RecommendationTypeScaling:
		// Scaling changes are low risk
		return "Low risk - adjusting replica count for better efficiency"

	default:
		return "Unknown risk"
	}
}

// formatImpactMessage creates a detailed impact message for a recommendation
func (s *scorer) formatImpactMessage(recType recommendationType, analysis *analysisResult, savings float64) string {
	riskLevel := s.getRiskLevel(recType, analysis)

	if savings > 0 {
		return fmt.Sprintf("%s - estimated savings of $%.2f/month based on %d-day usage patterns",
			riskLevel, savings, int(s.optimizer.config.AnalysisDuration.Hours()/24))
	}

	return fmt.Sprintf("%s - based on %d-day usage patterns",
		riskLevel, int(s.optimizer.config.AnalysisDuration.Hours()/24))
}
