package analyzer

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/k8s-service-optimizer/backend/internal/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CalculateServiceCost calculates the cost for a specific service
func (a *analyzer) CalculateServiceCost(namespace, service string) (*models.CostBreakdown, error) {
	// Get resource requests and usage for the service
	// We need to find all pods for this service
	resource := fmt.Sprintf("pod/%s", service)

	// Calculate time window (use 24 hours for cost calculation)
	duration := 24 * time.Hour

	// Get CPU usage data
	cpuData, err := a.client.GetTimeSeriesData(resource, "cpu", duration)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU data: %w", err)
	}

	// Get memory usage data
	memData, err := a.client.GetTimeSeriesData(resource, "memory", duration)
	if err != nil {
		return nil, fmt.Errorf("failed to get memory data: %w", err)
	}

	// If no data, return zero cost
	if len(cpuData.Points) == 0 && len(memData.Points) == 0 {
		return &models.CostBreakdown{
			Service:         service,
			Namespace:       namespace,
			CPUCost:         0,
			MemoryCost:      0,
			TotalCost:       0,
			WastedCost:      0,
			EfficiencyScore: 100,
			Timestamp:       time.Now(),
		}, nil
	}

	// Calculate P95 usage (what's actually needed)
	cpuP95 := 0.0
	memP95 := 0.0

	if len(cpuData.Points) > 0 {
		_, cpuP95, _, _ = a.calculatePercentiles(cpuData.Points)
	}

	if len(memData.Points) > 0 {
		_, memP95, _, _ = a.calculatePercentiles(memData.Points)
	}

	// Get requested resources (what we're paying for)
	// For now, we'll estimate based on max usage + buffer
	// In a real implementation, we'd query the pod spec
	cpuRequested := cpuP95 * 1.3  // 30% buffer
	memRequested := memP95 * 1.3  // 30% buffer

	// If we have very low usage, set a minimum request
	if cpuRequested < 100 {
		cpuRequested = 100 // 100m minimum
	}
	if memRequested < 128*1024*1024 {
		memRequested = 128 * 1024 * 1024 // 128MB minimum
	}

	// Calculate costs
	// Monthly cost formula:
	// CPU: (cpu_millicores / 1000) × CPUCostPerVCPUHour × 24 × 30
	// Memory: (memory_bytes / (1024^3)) × MemoryCostPerGBHour × 24 × 30

	hoursPerMonth := 24.0 * 30.0

	// CPU cost
	cpuVCores := cpuRequested / 1000.0
	cpuCost := cpuVCores * a.config.CPUCostPerVCPUHour * hoursPerMonth

	// Memory cost (convert bytes to GB)
	memGB := memRequested / (1024.0 * 1024.0 * 1024.0)
	memCost := memGB * a.config.MemoryCostPerGBHour * hoursPerMonth

	totalCost := cpuCost + memCost

	// Calculate waste (requested - P95 usage)
	cpuWaste := math.Max(0, cpuRequested-cpuP95)
	memWaste := math.Max(0, memRequested-memP95)

	// Calculate wasted cost
	cpuWasteVCores := cpuWaste / 1000.0
	cpuWasteCost := cpuWasteVCores * a.config.CPUCostPerVCPUHour * hoursPerMonth

	memWasteGB := memWaste / (1024.0 * 1024.0 * 1024.0)
	memWasteCost := memWasteGB * a.config.MemoryCostPerGBHour * hoursPerMonth

	wastedCost := cpuWasteCost + memWasteCost

	// Calculate efficiency score (0-100)
	// Efficiency = 100 - (waste_percentage)
	efficiencyScore := 100.0
	if totalCost > 0 {
		wastePercentage := (wastedCost / totalCost) * 100.0
		efficiencyScore = math.Max(0, 100.0-wastePercentage)
	}

	return &models.CostBreakdown{
		Service:         service,
		Namespace:       namespace,
		CPUCost:         roundTo2Decimals(cpuCost),
		MemoryCost:      roundTo2Decimals(memCost),
		TotalCost:       roundTo2Decimals(totalCost),
		WastedCost:      roundTo2Decimals(wastedCost),
		EfficiencyScore: roundTo2Decimals(efficiencyScore),
		Timestamp:       time.Now(),
	}, nil
}

// GetCostTrends gets cost trends over time
func (a *analyzer) GetCostTrends(namespace string, duration time.Duration) ([]models.CostBreakdown, error) {
	// For cost trends, we calculate cost at multiple time points
	// We'll sample at intervals throughout the duration

	// Sample every 6 hours
	sampleInterval := 6 * time.Hour
	numSamples := int(duration / sampleInterval)

	if numSamples < 1 {
		numSamples = 1
	}

	trends := make([]models.CostBreakdown, 0, numSamples)

	// For now, we'll return a single current cost
	// In a full implementation, we would calculate historical costs
	// by using time-windowed queries

	// This is a simplified version - we'd need to track service names
	// For now, return empty if we can't determine services
	return trends, nil
}

// CalculateWaste calculates wasted resources (over-provisioning)
func (a *analyzer) CalculateWaste(namespace, service string) (float64, error) {
	resource := fmt.Sprintf("pod/%s", service)
	duration := 24 * time.Hour

	// Get CPU usage data
	cpuData, err := a.client.GetTimeSeriesData(resource, "cpu", duration)
	if err != nil {
		return 0, fmt.Errorf("failed to get CPU data: %w", err)
	}

	// Get memory usage data
	memData, err := a.client.GetTimeSeriesData(resource, "memory", duration)
	if err != nil {
		return 0, fmt.Errorf("failed to get memory data: %w", err)
	}

	if len(cpuData.Points) == 0 && len(memData.Points) == 0 {
		return 0, nil
	}

	// Calculate P95 usage
	cpuP95 := 0.0
	memP95 := 0.0

	if len(cpuData.Points) > 0 {
		_, cpuP95, _, _ = a.calculatePercentiles(cpuData.Points)
	}

	if len(memData.Points) > 0 {
		_, memP95, _, _ = a.calculatePercentiles(memData.Points)
	}

	// Estimate requested resources (30% buffer above P95)
	cpuRequested := cpuP95 * 1.3
	memRequested := memP95 * 1.3

	// Calculate waste percentage
	totalRequested := cpuRequested + memRequested
	totalUsed := cpuP95 + memP95

	if totalRequested == 0 {
		return 0, nil
	}

	wastePercentage := ((totalRequested - totalUsed) / totalRequested) * 100.0

	return roundTo2Decimals(math.Max(0, wastePercentage)), nil
}

// roundTo2Decimals rounds a float to 2 decimal places
func roundTo2Decimals(value float64) float64 {
	return math.Round(value*100) / 100
}

// getServicePods retrieves all pods for a service (helper function)
func (a *analyzer) getServicePods(namespace, service string) ([]string, error) {
	// In a full implementation, we would query the k8s API for pods with matching labels
	// For now, we use naming convention
	return []string{service}, nil
}

// getResourceRequests gets the resource requests for a pod
func (a *analyzer) getResourceRequests(namespace, podName string) (cpuMillis int64, memBytes int64, err error) {
	// This would query the k8s API to get pod spec
	// For now, return estimated values based on usage
	return 0, 0, fmt.Errorf("not implemented")
}

// calculateCostForResources calculates cost for given resource amounts
func (a *analyzer) calculateCostForResources(cpuMillis, memBytes int64) (cpuCost, memCost, totalCost float64) {
	hoursPerMonth := 24.0 * 30.0

	// CPU cost
	cpuVCores := float64(cpuMillis) / 1000.0
	cpuCost = cpuVCores * a.config.CPUCostPerVCPUHour * hoursPerMonth

	// Memory cost
	memGB := float64(memBytes) / (1024.0 * 1024.0 * 1024.0)
	memCost = memGB * a.config.MemoryCostPerGBHour * hoursPerMonth

	totalCost = cpuCost + memCost

	return roundTo2Decimals(cpuCost), roundTo2Decimals(memCost), roundTo2Decimals(totalCost)
}

// Helper to extract service name from deployment or pod name
func extractServiceName(resourceName string) string {
	// Remove pod hash suffix
	parts := strings.Split(resourceName, "-")
	if len(parts) > 2 {
		// Return everything except the last 2 parts (which are usually hash)
		return strings.Join(parts[:len(parts)-2], "-")
	}
	return resourceName
}

// Dummy context for future k8s API calls
var _ = context.Background()
var _ = metav1.ListOptions{}
