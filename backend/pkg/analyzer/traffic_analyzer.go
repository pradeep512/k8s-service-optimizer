package analyzer

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/k8s-service-optimizer/backend/internal/models"
)

// AnalyzeTrafficPatterns analyzes traffic patterns for a service
func (a *analyzer) AnalyzeTrafficPatterns(namespace, service string, duration time.Duration) (*models.TrafficAnalysis, error) {
	// Get pod metrics for the service
	// Since we're working with pod metrics, we need to find pods belonging to the service
	// Pod naming convention: service-name-xxxxx
	resource := fmt.Sprintf("pod/%s", service)

	// Get CPU time series to estimate request rate
	cpuData, err := a.client.GetTimeSeriesData(resource, "cpu", duration)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU data: %w", err)
	}

	// If no data found, try to find pods with service prefix
	if len(cpuData.Points) == 0 {
		// Try to find any pod that starts with the service name
		resource = fmt.Sprintf("pod/%s-", service)
		cpuData, err = a.client.GetTimeSeriesData(resource, "cpu", duration)
		if err != nil {
			return nil, fmt.Errorf("failed to get CPU data for service pods: %w", err)
		}
	}

	if len(cpuData.Points) < a.config.MinDataPoints {
		return &models.TrafficAnalysis{
			Service:     service,
			Namespace:   namespace,
			RequestRate: 0,
			ErrorRate:   0,
			P50Latency:  0,
			P95Latency:  0,
			P99Latency:  0,
			Anomalies:   []models.Anomaly{},
			Timestamp:   time.Now(),
		}, nil
	}

	// Calculate percentiles for CPU (will be used for latency estimation)
	p50, p95, p99, err := a.calculatePercentiles(cpuData.Points)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate percentiles: %w", err)
	}

	// Estimate request rate from CPU usage
	// Assumption: Higher CPU = more requests
	// Average CPU in millicores / 10 = requests per second (rough estimate)
	avgCPU := a.calculateAverage(cpuData.Points)
	requestRate := avgCPU / 10.0

	// Estimate error rate from pod restart patterns
	// We'll detect anomalies in CPU which might indicate crashes/restarts
	errorRate := a.estimateErrorRate(cpuData.Points)

	// Estimate latency from CPU saturation
	// Higher CPU utilization = higher latency
	// P50 CPU / 10 = P50 latency in ms (rough estimate)
	p50Latency := p50 / 10.0
	p95Latency := p95 / 10.0
	p99Latency := p99 / 10.0

	// Detect anomalies in the traffic pattern
	anomalies, err := a.DetectAnomalies(resource, "cpu", duration)
	if err != nil {
		// Don't fail if anomaly detection fails
		anomalies = []models.Anomaly{}
	}

	return &models.TrafficAnalysis{
		Service:     service,
		Namespace:   namespace,
		RequestRate: math.Max(0, requestRate),
		ErrorRate:   errorRate,
		P50Latency:  math.Max(0, p50Latency),
		P95Latency:  math.Max(0, p95Latency),
		P99Latency:  math.Max(0, p99Latency),
		Anomalies:   anomalies,
		Timestamp:   time.Now(),
	}, nil
}

// estimateErrorRate estimates error rate from CPU patterns
func (a *analyzer) estimateErrorRate(points []models.DataPoint) float64 {
	if len(points) < 2 {
		return 0
	}

	// Count sudden drops in CPU (might indicate pod crashes)
	dropCount := 0
	for i := 1; i < len(points); i++ {
		if points[i-1].Value > 0 {
			changeRatio := points[i].Value / points[i-1].Value
			// If CPU drops by more than 80%, might be a crash
			if changeRatio < 0.2 {
				dropCount++
			}
		}
	}

	// Error rate as percentage of data points with drops
	errorRate := float64(dropCount) / float64(len(points))
	return math.Min(errorRate, 1.0) // Cap at 100%
}

// calculatePercentiles calculates P50, P95, P99 from data points
func (a *analyzer) calculatePercentiles(points []models.DataPoint) (p50, p95, p99 float64, err error) {
	if len(points) == 0 {
		return 0, 0, 0, fmt.Errorf("no data points")
	}

	values := make([]float64, len(points))
	for i, p := range points {
		values[i] = p.Value
	}

	p50 = percentile(values, 50)
	p95 = percentile(values, 95)
	p99 = percentile(values, 99)

	return p50, p95, p99, nil
}

// calculateAverage calculates the average value from data points
func (a *analyzer) calculateAverage(points []models.DataPoint) float64 {
	if len(points) == 0 {
		return 0
	}

	sum := 0.0
	for _, p := range points {
		sum += p.Value
	}

	return sum / float64(len(points))
}

// detectTrafficPattern detects the traffic pattern from data points
func (a *analyzer) detectTrafficPattern(points []models.DataPoint) trafficPattern {
	if len(points) < a.config.MinDataPoints {
		return PatternSteady
	}

	// Calculate trend
	trend := a.calculateTrend(points)

	// Detect pattern based on slope and variance
	avg := a.calculateAverage(points)
	variance := a.calculateVariance(points, avg)
	stdDev := math.Sqrt(variance)

	// High variance indicates spiking
	if stdDev > avg*0.5 {
		return PatternSpiking
	}

	// Check trend direction
	if trend.Slope > avg*0.1 {
		return PatternIncreasing
	} else if trend.Slope < -avg*0.1 {
		return PatternDeclining
	}

	// Check for periodicity
	if a.detectPeriodicity(points) {
		return PatternPeriodic
	}

	return PatternSteady
}

// calculateVariance calculates variance of data points
func (a *analyzer) calculateVariance(points []models.DataPoint, mean float64) float64 {
	if len(points) == 0 {
		return 0
	}

	sum := 0.0
	for _, p := range points {
		diff := p.Value - mean
		sum += diff * diff
	}

	return sum / float64(len(points))
}

// detectPeriodicity detects if data shows periodic patterns
func (a *analyzer) detectPeriodicity(points []models.DataPoint) bool {
	if len(points) < 20 {
		return false
	}

	// Simple autocorrelation check
	// Check if values repeat with similar patterns
	avg := a.calculateAverage(points)

	// Check for correlation at different lags
	maxCorrelation := 0.0
	for lag := 1; lag < len(points)/4; lag++ {
		correlation := 0.0
		count := 0
		for i := lag; i < len(points); i++ {
			correlation += (points[i].Value - avg) * (points[i-lag].Value - avg)
			count++
		}
		if count > 0 {
			correlation /= float64(count)
			if correlation > maxCorrelation {
				maxCorrelation = correlation
			}
		}
	}

	variance := a.calculateVariance(points, avg)

	// If correlation is strong relative to variance, it's periodic
	return variance > 0 && maxCorrelation > variance*0.5
}

// findPodsByService finds all pods belonging to a service
func (a *analyzer) findPodsByService(namespace, service string, duration time.Duration) ([]string, error) {
	// This is a helper to find pods that match the service name
	// In practice, we would query the metrics store for all pods with the service prefix
	// For now, we return a simple pattern
	return []string{fmt.Sprintf("pod/%s", service)}, nil
}

// aggregatePodMetrics aggregates metrics from multiple pods
func (a *analyzer) aggregatePodMetrics(podResources []string, metric string, duration time.Duration) ([]models.DataPoint, error) {
	if len(podResources) == 0 {
		return []models.DataPoint{}, nil
	}

	// For simplicity, we'll use the first pod's metrics
	// In a full implementation, we would aggregate across all pods
	data, err := a.client.GetTimeSeriesData(podResources[0], metric, duration)
	if err != nil {
		return []models.DataPoint{}, err
	}

	return data.Points, nil
}

// Helper function to check if a resource name matches a service
func matchesService(resource, service string) bool {
	// Resource format: "pod/service-name-xxxxx"
	parts := strings.Split(resource, "/")
	if len(parts) != 2 {
		return false
	}

	podName := parts[1]
	return strings.HasPrefix(podName, service+"-")
}
