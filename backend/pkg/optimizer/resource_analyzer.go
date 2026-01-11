package optimizer

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/k8s-service-optimizer/backend/internal/models"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// resourceAnalyzer handles resource usage analysis
type resourceAnalyzer struct {
	optimizer *OptimizerEngine
}

// newResourceAnalyzer creates a new resource analyzer
func newResourceAnalyzer(opt *OptimizerEngine) *resourceAnalyzer {
	return &resourceAnalyzer{
		optimizer: opt,
	}
}

// analyzeDeployment performs comprehensive analysis of a deployment
func (ra *resourceAnalyzer) analyzeDeployment(namespace, name string) (*analysisResult, error) {
	// Collect deployment metrics
	metrics, err := ra.collectDeploymentMetrics(namespace, name)
	if err != nil {
		return nil, fmt.Errorf("failed to collect deployment metrics: %w", err)
	}

	// Validate we have enough data
	if len(metrics.CPUTimeSeries) < ra.optimizer.config.MinimumDataPoints {
		return nil, fmt.Errorf("insufficient data points for analysis: got %d, need at least %d",
			len(metrics.CPUTimeSeries), ra.optimizer.config.MinimumDataPoints)
	}

	// Perform analysis
	result := &analysisResult{
		Deployment: *metrics,
		Timestamp:  time.Now(),
	}

	// Analyze CPU
	ra.analyzeCPU(result)

	// Analyze Memory
	ra.analyzeMemory(result)

	// Analyze HPA if it exists
	if metrics.HasHPA {
		ra.analyzeHPA(result)
	}

	// Calculate overall scores
	ra.calculateScores(result)

	return result, nil
}

// collectDeploymentMetrics collects all relevant metrics for a deployment
func (ra *resourceAnalyzer) collectDeploymentMetrics(namespace, name string) (*deploymentMetrics, error) {
	ctx := context.Background()

	// Get deployment info
	deployment, err := ra.optimizer.k8sClient.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	metrics := &deploymentMetrics{
		Namespace:       namespace,
		Deployment:      name,
		CurrentReplicas: *deployment.Spec.Replicas,
		Timestamp:       time.Now(),
	}

	// Extract resource requests and limits from deployment spec
	if len(deployment.Spec.Template.Spec.Containers) > 0 {
		container := deployment.Spec.Template.Spec.Containers[0]

		// CPU
		if cpuReq, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
			metrics.CPURequested = cpuReq.MilliValue()
		}
		if cpuLimit, ok := container.Resources.Limits[corev1.ResourceCPU]; ok {
			metrics.CPULimit = cpuLimit.MilliValue()
		}

		// Memory
		if memReq, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
			metrics.MemoryRequested = memReq.Value()
		}
		if memLimit, ok := container.Resources.Limits[corev1.ResourceMemory]; ok {
			metrics.MemoryLimit = memLimit.Value()
		}
	}

	// Get pods belonging to this deployment
	pods, err := ra.getDeploymentPods(deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment pods: %w", err)
	}

	// Collect metrics for each pod
	duration := ra.optimizer.config.AnalysisDuration
	var allCPUPoints []models.DataPoint
	var allMemoryPoints []models.DataPoint
	var restartCount int32

	for _, pod := range pods {
		// Get CPU time series
		cpuResource := fmt.Sprintf("pod/%s", pod.Name)
		cpuData, err := ra.optimizer.collector.GetTimeSeriesData(cpuResource, "cpu", duration)
		if err == nil {
			allCPUPoints = append(allCPUPoints, cpuData.Points...)
		}

		// Get Memory time series
		memResource := fmt.Sprintf("pod/%s", pod.Name)
		memData, err := ra.optimizer.collector.GetTimeSeriesData(memResource, "memory", duration)
		if err == nil {
			allMemoryPoints = append(allMemoryPoints, memData.Points...)
		}

		// Count restarts
		for _, containerStatus := range pod.Status.ContainerStatuses {
			restartCount += containerStatus.RestartCount
		}
	}

	metrics.RestartCount = restartCount
	metrics.CPUTimeSeries = allCPUPoints
	metrics.MemoryTimeSeries = allMemoryPoints

	// Calculate CPU statistics
	if len(allCPUPoints) > 0 {
		cpuValues := extractValues(allCPUPoints)
		sort.Float64s(cpuValues)

		metrics.CPUCurrent = int64(cpuValues[len(cpuValues)-1])
		metrics.CPUP50 = int64(calculatePercentile(cpuValues, 50))
		metrics.CPUP95 = int64(calculatePercentile(cpuValues, 95))
		metrics.CPUP99 = int64(calculatePercentile(cpuValues, 99))
		metrics.CPUAverage = int64(calculateAverage(cpuValues))
		metrics.CPUMax = int64(cpuValues[len(cpuValues)-1])
	}

	// Calculate Memory statistics
	if len(allMemoryPoints) > 0 {
		memValues := extractValues(allMemoryPoints)
		sort.Float64s(memValues)

		metrics.MemoryCurrent = int64(memValues[len(memValues)-1])
		metrics.MemoryP50 = int64(calculatePercentile(memValues, 50))
		metrics.MemoryP95 = int64(calculatePercentile(memValues, 95))
		metrics.MemoryP99 = int64(calculatePercentile(memValues, 99))
		metrics.MemoryAverage = int64(calculateAverage(memValues))
		metrics.MemoryMax = int64(memValues[len(memValues)-1])
	}

	// Check for HPA
	hpaList, err := ra.optimizer.k8sClient.Clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, hpa := range hpaList.Items {
			if hpa.Spec.ScaleTargetRef.Name == name {
				metrics.HasHPA = true
				metrics.MinReplicas = *hpa.Spec.MinReplicas
				metrics.MaxReplicas = hpa.Spec.MaxReplicas
				metrics.HPADesiredReplicas = hpa.Status.DesiredReplicas

				// Extract CPU target
				for _, metric := range hpa.Spec.Metrics {
					if metric.Resource != nil && metric.Resource.Name == corev1.ResourceCPU {
						if metric.Resource.Target.AverageUtilization != nil {
							metrics.HPATargetCPU = *metric.Resource.Target.AverageUtilization
						}
					}
				}

				// Extract current CPU
				for _, current := range hpa.Status.CurrentMetrics {
					if current.Resource != nil && current.Resource.Name == corev1.ResourceCPU {
						if current.Resource.Current.AverageUtilization != nil {
							metrics.HPACurrentCPU = *current.Resource.Current.AverageUtilization
						}
					}
				}

				// Get replica time series
				hpaResource := fmt.Sprintf("hpa/%s", hpa.Name)
				replicaData, err := ra.optimizer.collector.GetTimeSeriesData(hpaResource, "current_replicas", duration)
				if err == nil {
					metrics.ReplicaTimeSeries = replicaData.Points
					metrics.ScalingEvents = ra.countScalingEvents(replicaData.Points)
				}

				break
			}
		}
	}

	return metrics, nil
}

// getDeploymentPods gets all pods belonging to a deployment
func (ra *resourceAnalyzer) getDeploymentPods(deployment *appsv1.Deployment) ([]corev1.Pod, error) {
	ctx := context.Background()

	// Build label selector from deployment selector
	selector := metav1.FormatLabelSelector(deployment.Spec.Selector)

	podList, err := ra.optimizer.k8sClient.Clientset.CoreV1().Pods(deployment.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

// analyzeCPU performs CPU usage analysis
func (ra *resourceAnalyzer) analyzeCPU(result *analysisResult) {
	metrics := &result.Deployment

	// Calculate utilization (P95 usage vs requested)
	if metrics.CPURequested > 0 {
		result.CPUUtilization = float64(metrics.CPUP95) / float64(metrics.CPURequested)
	}

	// Check for over-provisioning (P95 usage < 50% of requested)
	if result.CPUUtilization < ra.optimizer.config.CPUOverProvisionedThreshold {
		result.CPUOverProvisioned = true
	}

	// Check for under-provisioning (P95 usage > 80% of limit)
	if metrics.CPULimit > 0 {
		utilizationVsLimit := float64(metrics.CPUP95) / float64(metrics.CPULimit)
		if utilizationVsLimit > ra.optimizer.config.CPUUnderProvisionedThreshold {
			result.CPUUnderProvisioned = true
		}
	}

	// Calculate variance (stability metric)
	if len(metrics.CPUTimeSeries) > 0 {
		values := extractValues(metrics.CPUTimeSeries)
		result.CPUVariance = calculateVariance(values)
	}

	// Calculate efficiency score (0-100)
	result.CPUEfficiency = ra.calculateResourceEfficiency(result.CPUUtilization, result.CPUVariance)
}

// analyzeMemory performs memory usage analysis
func (ra *resourceAnalyzer) analyzeMemory(result *analysisResult) {
	metrics := &result.Deployment

	// Calculate utilization (P95 usage vs requested)
	if metrics.MemoryRequested > 0 {
		result.MemoryUtilization = float64(metrics.MemoryP95) / float64(metrics.MemoryRequested)
	}

	// Check for over-provisioning (P95 usage < 50% of requested)
	if result.MemoryUtilization < ra.optimizer.config.MemoryOverProvisionedThreshold {
		result.MemoryOverProvisioned = true
	}

	// Check for under-provisioning (P95 usage > 80% of limit)
	if metrics.MemoryLimit > 0 {
		utilizationVsLimit := float64(metrics.MemoryP95) / float64(metrics.MemoryLimit)
		if utilizationVsLimit > ra.optimizer.config.MemoryUnderProvisionedThreshold {
			result.MemoryUnderProvisioned = true
		}
	}

	// Calculate variance (stability metric)
	if len(metrics.MemoryTimeSeries) > 0 {
		values := extractValues(metrics.MemoryTimeSeries)
		result.MemoryVariance = calculateVariance(values)
	}

	// Calculate efficiency score (0-100)
	result.MemoryEfficiency = ra.calculateResourceEfficiency(result.MemoryUtilization, result.MemoryVariance)
}

// analyzeHPA performs HPA configuration analysis
func (ra *resourceAnalyzer) analyzeHPA(result *analysisResult) {
	metrics := &result.Deployment

	// Calculate scaling frequency (events per day)
	if len(metrics.ReplicaTimeSeries) > 1 {
		durationDays := time.Since(metrics.ReplicaTimeSeries[0].Timestamp).Hours() / 24
		if durationDays > 0 {
			result.HPAScalingFrequency = float64(metrics.ScalingEvents) / durationDays
		}
	}

	// Calculate scaling amplitude (how much replicas change)
	if len(metrics.ReplicaTimeSeries) > 0 {
		values := extractValues(metrics.ReplicaTimeSeries)
		sort.Float64s(values)
		minReplicas := values[0]
		maxReplicas := values[len(values)-1]
		result.HPAScalingAmplitude = maxReplicas - minReplicas
	}

	// Check if frequently hitting max replicas
	if len(metrics.ReplicaTimeSeries) > 0 {
		atMaxCount := 0
		for _, point := range metrics.ReplicaTimeSeries {
			if int32(point.Value) >= metrics.MaxReplicas {
				atMaxCount++
			}
		}
		atMaxPercent := float64(atMaxCount) / float64(len(metrics.ReplicaTimeSeries))
		if atMaxPercent > 0.1 { // More than 10% of the time at max
			result.HPAHitCeiling = true
		}
	}

	// Check if consistently at minimum replicas
	if len(metrics.ReplicaTimeSeries) > 0 {
		atMinCount := 0
		for _, point := range metrics.ReplicaTimeSeries {
			if int32(point.Value) <= metrics.MinReplicas {
				atMinCount++
			}
		}
		atMinPercent := float64(atMinCount) / float64(len(metrics.ReplicaTimeSeries))
		if atMinPercent > 0.8 { // More than 80% of the time at min
			result.HPAIdleAtMinimum = true
		}
	}

	// Check if target CPU is too aggressive or too conservative
	if metrics.HPATargetCPU > 0 && metrics.HPACurrentCPU > 0 {
		diff := math.Abs(float64(metrics.HPACurrentCPU - metrics.HPATargetCPU))
		if diff > 20 { // More than 20% difference
			result.HPANeedsOptimization = true
		}
	}

	// Check if scaling too frequently (more than once per hour)
	if result.HPAScalingFrequency > 24 {
		result.HPANeedsOptimization = true
	}
}

// calculateScores calculates various efficiency scores
func (ra *resourceAnalyzer) calculateScores(result *analysisResult) {
	metrics := &result.Deployment

	// Resource Utilization Score (50% weight)
	// Optimal is 70-90% utilization
	cpuUtilScore := ra.calculateUtilizationScore(result.CPUUtilization)
	memUtilScore := ra.calculateUtilizationScore(result.MemoryUtilization)
	result.ResourceUtilizationScore = (cpuUtilScore + memUtilScore) / 2.0

	// Stability Score (30% weight)
	// Based on restart count, variance, and scaling patterns
	stabilityScore := 100.0

	// Penalize for restarts (each restart reduces score)
	restartPenalty := float64(metrics.RestartCount) * 5.0
	stabilityScore -= restartPenalty

	// Penalize for high variance (unstable resource usage)
	if result.CPUVariance > 1000 {
		stabilityScore -= 10.0
	}
	if result.MemoryVariance > 1000000000 { // 1GB variance
		stabilityScore -= 10.0
	}

	// Penalize for frequent scaling
	if result.HPAScalingFrequency > 10 {
		stabilityScore -= 20.0
	}

	result.StabilityScore = math.Max(0, stabilityScore)

	// Cost Efficiency Score (20% weight)
	// Based on waste (over-provisioning)
	costScore := 100.0

	if result.CPUOverProvisioned {
		// Calculate waste percentage
		waste := 1.0 - result.CPUUtilization
		costScore -= waste * 50 // Up to 50 points penalty
	}

	if result.MemoryOverProvisioned {
		waste := 1.0 - result.MemoryUtilization
		costScore -= waste * 50 // Up to 50 points penalty
	}

	result.CostEfficiencyScore = math.Max(0, costScore)

	// Overall Score (weighted average)
	result.OverallScore = (result.ResourceUtilizationScore * 0.5) +
		(result.StabilityScore * 0.3) +
		(result.CostEfficiencyScore * 0.2)
}

// calculateResourceEfficiency calculates efficiency score for a resource (0-100)
func (ra *resourceAnalyzer) calculateResourceEfficiency(utilization, variance float64) float64 {
	// Start with utilization score
	score := ra.calculateUtilizationScore(utilization)

	// Adjust for variance (high variance reduces efficiency)
	if variance > 1000 {
		score *= 0.9 // 10% penalty for high variance
	}

	return score
}

// calculateUtilizationScore calculates score based on utilization (0-100)
func (ra *resourceAnalyzer) calculateUtilizationScore(utilization float64) float64 {
	optMin := ra.optimizer.config.OptimalUtilizationMin
	optMax := ra.optimizer.config.OptimalUtilizationMax

	if utilization >= optMin && utilization <= optMax {
		// Perfect range
		return 100.0
	} else if utilization < optMin {
		// Under-utilized (over-provisioned)
		// Linear decrease from 100 at optMin to 0 at 0%
		return 100.0 * (utilization / optMin)
	} else {
		// Over-utilized (under-provisioned)
		// Linear decrease from 100 at optMax to 0 at 200%
		excess := utilization - optMax
		maxExcess := 2.0 - optMax // Max expected is 200%
		return math.Max(0, 100.0*(1.0-(excess/maxExcess)))
	}
}

// countScalingEvents counts the number of scaling events in time series data
func (ra *resourceAnalyzer) countScalingEvents(points []models.DataPoint) int {
	if len(points) < 2 {
		return 0
	}

	count := 0
	prevValue := points[0].Value

	for i := 1; i < len(points); i++ {
		if points[i].Value != prevValue {
			count++
			prevValue = points[i].Value
		}
	}

	return count
}

// Helper functions

// extractValues extracts float64 values from data points
func extractValues(points []models.DataPoint) []float64 {
	values := make([]float64, len(points))
	for i, point := range points {
		values[i] = point.Value
	}
	return values
}

// calculatePercentile calculates the percentile value from a sorted slice
func calculatePercentile(sortedValues []float64, percentile float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}

	if len(sortedValues) == 1 {
		return sortedValues[0]
	}

	index := (percentile / 100.0) * float64(len(sortedValues)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sortedValues) {
		return sortedValues[len(sortedValues)-1]
	}

	weight := index - float64(lower)
	return sortedValues[lower]*(1-weight) + sortedValues[upper]*weight
}

// calculateAverage calculates the average of values
func calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// calculateVariance calculates the variance of values
func calculateVariance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	avg := calculateAverage(values)
	sumSquaredDiff := 0.0

	for _, v := range values {
		diff := v - avg
		sumSquaredDiff += diff * diff
	}

	return sumSquaredDiff / float64(len(values))
}

// formatResourceQuantity formats a resource quantity from int64 value
func formatResourceQuantity(value int64, resourceType string) string {
	if resourceType == "cpu" {
		// Convert millicores to string (e.g., 100m, 1000m = 1)
		if value >= 1000 {
			return fmt.Sprintf("%d", value/1000)
		}
		return fmt.Sprintf("%dm", value)
	} else if resourceType == "memory" {
		// Convert bytes to Mi or Gi
		if value >= 1024*1024*1024 {
			return fmt.Sprintf("%dGi", value/(1024*1024*1024))
		}
		return fmt.Sprintf("%dMi", value/(1024*1024))
	}
	return fmt.Sprintf("%d", value)
}

// parseResourceQuantity parses a resource quantity string to int64
func parseResourceQuantity(value string, resourceType string) int64 {
	quantity, err := resource.ParseQuantity(value)
	if err != nil {
		return 0
	}

	if resourceType == "cpu" {
		return quantity.MilliValue()
	} else if resourceType == "memory" {
		return quantity.Value()
	}

	return 0
}

// convertBytesToGBHours converts bytes to GB for cost calculation
func convertBytesToGB(bytes int64) float64 {
	return float64(bytes) / (1024 * 1024 * 1024)
}

// convertMillicoresToVCPU converts millicores to vCPU for cost calculation
func convertMillicoresToVCPU(millicores int64) float64 {
	return float64(millicores) / 1000.0
}

// convertToMap converts a struct to a map for recommendation config
func convertResourceConfigToMap(config resourceConfig) map[string]interface{} {
	result := make(map[string]interface{})
	if config.CPURequest != "" {
		result["cpu_request"] = config.CPURequest
	}
	if config.CPULimit != "" {
		result["cpu_limit"] = config.CPULimit
	}
	if config.MemoryRequest != "" {
		result["memory_request"] = config.MemoryRequest
	}
	if config.MemoryLimit != "" {
		result["memory_limit"] = config.MemoryLimit
	}
	return result
}

// convertHPAConfigToMap converts HPA config to map
func convertHPAConfigToMap(config hpaConfig) map[string]interface{} {
	return map[string]interface{}{
		"min_replicas": config.MinReplicas,
		"max_replicas": config.MaxReplicas,
		"target_cpu":   config.TargetCPU,
	}
}

// convertScalingConfigToMap converts scaling config to map
func convertScalingConfigToMap(config scalingConfig) map[string]interface{} {
	return map[string]interface{}{
		"replicas": config.Replicas,
	}
}

// sanitizeDeploymentName ensures deployment name is valid for resource naming
func sanitizeDeploymentName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, "_", "-"))
}
