package analyzer

import (
	"fmt"
	"math"
	"time"

	"github.com/k8s-service-optimizer/backend/internal/models"
)

// PredictResourceNeeds predicts future resource requirements
func (a *analyzer) PredictResourceNeeds(namespace, service string, hours int) (*models.ResourcePrediction, error) {
	resource := fmt.Sprintf("pod/%s", service)

	// Use trend history from config
	duration := time.Duration(a.config.TrendHistoryDays) * 24 * time.Hour

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

	if len(cpuData.Points) < a.config.MinDataPoints && len(memData.Points) < a.config.MinDataPoints {
		return &models.ResourcePrediction{
			Service:         service,
			Namespace:       namespace,
			Hours:           hours,
			PredictedCPU:    0,
			PredictedMemory: 0,
			Confidence:      0,
			Timestamp:       time.Now(),
		}, nil
	}

	// Calculate trends for CPU
	cpuTrend := a.calculateTrend(cpuData.Points)
	memTrend := a.calculateTrend(memData.Points)

	// Get current values (latest or average)
	currentCPU := 0.0
	currentMem := 0.0

	if len(cpuData.Points) > 0 {
		currentCPU = cpuData.Points[len(cpuData.Points)-1].Value
	}
	if len(memData.Points) > 0 {
		currentMem = memData.Points[len(memData.Points)-1].Value
	}

	// Predict future values using linear regression
	// Future value = current_value + (slope × time_delta)
	timeDeltaHours := float64(hours)

	predictedCPU := currentCPU + (cpuTrend.Slope * timeDeltaHours)
	predictedMem := currentMem + (memTrend.Slope * timeDeltaHours)

	// Ensure predictions are non-negative
	predictedCPU = math.Max(0, predictedCPU)
	predictedMem = math.Max(0, predictedMem)

	// Calculate confidence based on R² (average of CPU and memory)
	confidence := (cpuTrend.RSquared + memTrend.RSquared) / 2.0

	// Apply safety factor for low confidence
	if confidence < 0.5 {
		// If confidence is low, add safety buffer
		predictedCPU *= 1.2
		predictedMem *= 1.2
	}

	return &models.ResourcePrediction{
		Service:         service,
		Namespace:       namespace,
		Hours:           hours,
		PredictedCPU:    int64(predictedCPU),
		PredictedMemory: int64(predictedMem),
		Confidence:      roundTo2Decimals(confidence),
		Timestamp:       time.Now(),
	}, nil
}

// calculateTrend calculates trend using simple linear regression
func (a *analyzer) calculateTrend(points []models.DataPoint) trendData {
	if len(points) < 2 {
		return trendData{
			Slope:      0,
			Intercept:  0,
			RSquared:   0,
			Prediction: 0,
		}
	}

	// Convert timestamps to hours since first point
	n := float64(len(points))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	firstTime := points[0].Timestamp

	for _, point := range points {
		x := point.Timestamp.Sub(firstTime).Hours()
		y := point.Value

		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	// Calculate slope and intercept
	// slope = (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	// intercept = (sumY - slope*sumX) / n

	denominator := n*sumXX - sumX*sumX
	slope := 0.0
	intercept := 0.0

	if denominator != 0 {
		slope = (n*sumXY - sumX*sumY) / denominator
		intercept = (sumY - slope*sumX) / n
	}

	// Calculate R² (coefficient of determination)
	meanY := sumY / n
	ssTotal := 0.0
	ssResidual := 0.0

	for _, point := range points {
		x := point.Timestamp.Sub(firstTime).Hours()
		y := point.Value
		predicted := slope*x + intercept

		ssTotal += (y - meanY) * (y - meanY)
		ssResidual += (y - predicted) * (y - predicted)
	}

	rSquared := 0.0
	if ssTotal > 0 {
		rSquared = 1.0 - (ssResidual / ssTotal)
		// R² should be between 0 and 1
		rSquared = math.Max(0, math.Min(1, rSquared))
	}

	// Calculate prediction for next point
	lastX := points[len(points)-1].Timestamp.Sub(firstTime).Hours()
	prediction := slope*(lastX+1) + intercept

	return trendData{
		Slope:      slope,
		Intercept:  intercept,
		RSquared:   rSquared,
		Prediction: prediction,
	}
}

// predictValue predicts a value at a future time based on trend
func (a *analyzer) predictValue(trend trendData, currentValue float64, hoursAhead float64) float64 {
	// Use trend slope to project forward
	predicted := currentValue + (trend.Slope * hoursAhead)
	return math.Max(0, predicted)
}

// calculateConfidence calculates prediction confidence
func (a *analyzer) calculateConfidence(points []models.DataPoint, trend trendData) float64 {
	if len(points) < a.config.MinDataPoints {
		return 0.0
	}

	// Use R² as base confidence
	confidence := trend.RSquared

	// Adjust based on data recency and volume
	dataVolumeBonus := math.Min(0.1, float64(len(points))/1000.0)
	confidence += dataVolumeBonus

	// Cap at 1.0
	return math.Min(1.0, confidence)
}

// forecastRange provides best/worst case predictions
func (a *analyzer) forecastRange(points []models.DataPoint, trend trendData, hoursAhead float64) (best, worst float64) {
	if len(points) == 0 {
		return 0, 0
	}

	// Calculate standard error
	stdDev := math.Sqrt(a.calculateVariance(points, a.calculateAverage(points)))

	currentValue := points[len(points)-1].Value
	predicted := a.predictValue(trend, currentValue, hoursAhead)

	// Best case: prediction - std dev
	// Worst case: prediction + 2*std dev (to be safe)
	best = math.Max(0, predicted-stdDev)
	worst = predicted + 2*stdDev

	return best, worst
}

// detectSeasonality detects seasonal patterns in data
func (a *analyzer) detectSeasonality(points []models.DataPoint) (hasSeason bool, period time.Duration) {
	if len(points) < 50 {
		return false, 0
	}

	// Simple autocorrelation to detect periodicity
	mean := a.calculateAverage(points)
	variance := a.calculateVariance(points, mean)

	if variance == 0 {
		return false, 0
	}

	maxCorrelation := 0.0
	bestLag := 0

	// Check for daily and weekly patterns
	maxLag := min(len(points)/2, 168) // Up to 1 week if hourly data

	for lag := 12; lag < maxLag; lag++ { // Start at 12 hours
		correlation := 0.0
		count := 0

		for i := lag; i < len(points); i++ {
			correlation += (points[i].Value - mean) * (points[i-lag].Value - mean)
			count++
		}

		if count > 0 {
			correlation = correlation / (float64(count) * variance)

			if correlation > maxCorrelation {
				maxCorrelation = correlation
				bestLag = lag
			}
		}
	}

	// If correlation is strong, we have seasonality
	if maxCorrelation > 0.6 && bestLag > 0 {
		// Estimate period based on lag
		if len(points) > 1 {
			avgInterval := points[1].Timestamp.Sub(points[0].Timestamp)
			period = time.Duration(bestLag) * avgInterval
			return true, period
		}
	}

	return false, 0
}

// exponentialSmoothing applies exponential smoothing for prediction
func (a *analyzer) exponentialSmoothing(points []models.DataPoint, alpha float64) []float64 {
	if len(points) == 0 {
		return []float64{}
	}

	smoothed := make([]float64, len(points))
	smoothed[0] = points[0].Value

	for i := 1; i < len(points); i++ {
		smoothed[i] = alpha*points[i].Value + (1-alpha)*smoothed[i-1]
	}

	return smoothed
}

// predictWithSeasonality predicts considering seasonal patterns
func (a *analyzer) predictWithSeasonality(points []models.DataPoint, hoursAhead int) (prediction float64, confidence float64) {
	if len(points) < a.config.MinDataPoints {
		return 0, 0
	}

	hasSeason, period := a.detectSeasonality(points)

	if !hasSeason {
		// Fall back to linear trend
		trend := a.calculateTrend(points)
		currentValue := points[len(points)-1].Value
		prediction = a.predictValue(trend, currentValue, float64(hoursAhead))
		confidence = a.calculateConfidence(points, trend)
		return prediction, confidence
	}

	// With seasonality, use historical pattern
	periodHours := int(period.Hours())
	if periodHours == 0 {
		periodHours = 24 // Default to daily
	}

	// Find comparable historical point
	offsetInPeriod := hoursAhead % periodHours
	historicalIndex := len(points) - periodHours + offsetInPeriod

	if historicalIndex >= 0 && historicalIndex < len(points) {
		prediction = points[historicalIndex].Value
		confidence = 0.7 // Moderate confidence for seasonal prediction
	} else {
		// Fall back to average
		prediction = a.calculateAverage(points)
		confidence = 0.5
	}

	return prediction, confidence
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// percentile calculates the percentile value from a slice of values
func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Make a copy and sort
	sorted := make([]float64, len(values))
	copy(sorted, values)

	// Simple bubble sort for small arrays
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	if len(sorted) == 1 {
		return sorted[0]
	}

	// Calculate index
	index := (p / 100.0) * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	// Linear interpolation
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}
