package analyzer

import (
	"fmt"
	"math"
	"time"

	"github.com/k8s-service-optimizer/backend/internal/models"
)

// DetectAnomalies detects anomalies in metrics
func (a *analyzer) DetectAnomalies(resource, metric string, duration time.Duration) ([]models.Anomaly, error) {
	// Get time series data
	data, err := a.client.GetTimeSeriesData(resource, metric, duration)
	if err != nil {
		return nil, fmt.Errorf("failed to get time series data: %w", err)
	}

	if len(data.Points) < a.config.MinDataPoints {
		return []models.Anomaly{}, nil
	}

	anomalies := []models.Anomaly{}

	// Calculate statistics
	mean := a.calculateAverage(data.Points)
	variance := a.calculateVariance(data.Points, mean)
	stdDev := math.Sqrt(variance)

	// Detect different types of anomalies
	anomalies = append(anomalies, a.detectZScoreAnomalies(data.Points, mean, stdDev)...)
	anomalies = append(anomalies, a.detectSpikeAnomalies(data.Points, mean)...)
	anomalies = append(anomalies, a.detectDropAnomalies(data.Points, mean)...)
	anomalies = append(anomalies, a.detectDriftAnomalies(data.Points, mean)...)
	anomalies = append(anomalies, a.detectOscillationAnomalies(data.Points, mean, stdDev)...)

	return anomalies, nil
}

// detectZScoreAnomalies detects anomalies using Z-score method
func (a *analyzer) detectZScoreAnomalies(points []models.DataPoint, mean, stdDev float64) []models.Anomaly {
	anomalies := []models.Anomaly{}

	if stdDev == 0 {
		return anomalies
	}

	for _, point := range points {
		zScore := math.Abs((point.Value - mean) / stdDev)

		if zScore > a.config.AnomalyThreshold {
			severity := a.determineSeverity(zScore)

			anomaly := models.Anomaly{
				Type:        string(AnomalySpike),
				Severity:    string(severity),
				Description: fmt.Sprintf("Value %.2f is %.2f standard deviations from mean %.2f", point.Value, zScore, mean),
				DetectedAt:  point.Timestamp,
				Value:       point.Value,
				Expected:    mean,
			}

			// Determine if it's a spike or drop
			if point.Value < mean {
				anomaly.Type = string(AnomalyDrop)
			}

			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

// detectSpikeAnomalies detects sudden spikes (>2x normal)
func (a *analyzer) detectSpikeAnomalies(points []models.DataPoint, mean float64) []models.Anomaly {
	anomalies := []models.Anomaly{}

	if mean == 0 {
		return anomalies
	}

	for i := 1; i < len(points); i++ {
		// Check if current value is significantly higher than previous
		if points[i-1].Value > 0 {
			ratio := points[i].Value / points[i-1].Value

			if ratio > a.config.SpikeThreshold {
				severity := a.determineSpikeDropSeverity(ratio)

				anomaly := models.Anomaly{
					Type:        string(AnomalySpike),
					Severity:    string(severity),
					Description: fmt.Sprintf("Sudden spike: value increased from %.2f to %.2f (%.1fx)", points[i-1].Value, points[i].Value, ratio),
					DetectedAt:  points[i].Timestamp,
					Value:       points[i].Value,
					Expected:    points[i-1].Value,
				}

				anomalies = append(anomalies, anomaly)
			}
		}
	}

	return anomalies
}

// detectDropAnomalies detects sudden drops (<0.5x normal)
func (a *analyzer) detectDropAnomalies(points []models.DataPoint, mean float64) []models.Anomaly {
	anomalies := []models.Anomaly{}

	for i := 1; i < len(points); i++ {
		// Check if current value is significantly lower than previous
		if points[i-1].Value > 0 {
			ratio := points[i].Value / points[i-1].Value

			if ratio < a.config.DropThreshold && ratio > 0 {
				severity := a.determineSpikeDropSeverity(1.0 / ratio)

				anomaly := models.Anomaly{
					Type:        string(AnomalyDrop),
					Severity:    string(severity),
					Description: fmt.Sprintf("Sudden drop: value decreased from %.2f to %.2f (%.1fx)", points[i-1].Value, points[i].Value, ratio),
					DetectedAt:  points[i].Timestamp,
					Value:       points[i].Value,
					Expected:    points[i-1].Value,
				}

				anomalies = append(anomalies, anomaly)
			}
		}
	}

	return anomalies
}

// detectDriftAnomalies detects gradual sustained changes
func (a *analyzer) detectDriftAnomalies(points []models.DataPoint, mean float64) []models.Anomaly {
	anomalies := []models.Anomaly{}

	if len(points) < 20 {
		return anomalies
	}

	// Split data into two halves and compare means
	mid := len(points) / 2
	firstHalf := points[:mid]
	secondHalf := points[mid:]

	firstMean := a.calculateAverage(firstHalf)
	secondMean := a.calculateAverage(secondHalf)

	if firstMean == 0 {
		return anomalies
	}

	change := math.Abs(secondMean-firstMean) / firstMean

	// If mean changed by more than 30% between halves, it's drift
	if change > 0.3 {
		severity := SeverityMedium
		if change > 0.5 {
			severity = SeverityHigh
		}

		direction := "increase"
		if secondMean < firstMean {
			direction = "decrease"
		}

		anomaly := models.Anomaly{
			Type:        string(AnomalyDrift),
			Severity:    string(severity),
			Description: fmt.Sprintf("Gradual drift detected: %.1f%% %s in baseline", change*100, direction),
			DetectedAt:  points[mid].Timestamp,
			Value:       secondMean,
			Expected:    firstMean,
		}

		anomalies = append(anomalies, anomaly)
	}

	return anomalies
}

// detectOscillationAnomalies detects rapid fluctuations
func (a *analyzer) detectOscillationAnomalies(points []models.DataPoint, mean, stdDev float64) []models.Anomaly {
	anomalies := []models.Anomaly{}

	if len(points) < 10 {
		return anomalies
	}

	// Count direction changes
	directionChanges := 0
	for i := 2; i < len(points); i++ {
		prev := points[i-1].Value - points[i-2].Value
		curr := points[i].Value - points[i-1].Value

		// Direction changed if signs are different
		if prev*curr < 0 {
			directionChanges++
		}
	}

	// If direction changes more than 50% of the time, it's oscillating
	changeRate := float64(directionChanges) / float64(len(points)-2)
	if changeRate > 0.5 && stdDev > mean*0.2 {
		anomaly := models.Anomaly{
			Type:        string(AnomalyOscillation),
			Severity:    string(SeverityMedium),
			Description: fmt.Sprintf("Rapid oscillation detected: %.1f%% direction changes", changeRate*100),
			DetectedAt:  points[len(points)-1].Timestamp,
			Value:       stdDev,
			Expected:    mean,
		}

		anomalies = append(anomalies, anomaly)
	}

	return anomalies
}

// determineSeverity determines anomaly severity based on Z-score
func (a *analyzer) determineSeverity(zScore float64) anomalySeverity {
	if zScore > 5.0 {
		return SeverityCritical
	} else if zScore > 4.0 {
		return SeverityHigh
	} else if zScore > 3.0 {
		return SeverityMedium
	}
	return SeverityLow
}

// determineSpikeDropSeverity determines severity based on spike/drop ratio
func (a *analyzer) determineSpikeDropSeverity(ratio float64) anomalySeverity {
	if ratio > 5.0 {
		return SeverityCritical
	} else if ratio > 3.0 {
		return SeverityHigh
	} else if ratio > 2.0 {
		return SeverityMedium
	}
	return SeverityLow
}

// detectMovingAverageAnomalies detects anomalies using moving average
func (a *analyzer) detectMovingAverageAnomalies(points []models.DataPoint, windowSize int) []models.Anomaly {
	anomalies := []models.Anomaly{}

	if len(points) < windowSize {
		return anomalies
	}

	for i := windowSize; i < len(points); i++ {
		// Calculate moving average of previous window
		window := points[i-windowSize : i]
		movingAvg := a.calculateAverage(window)

		// Check if current value deviates significantly from moving average
		if movingAvg > 0 {
			deviation := math.Abs(points[i].Value-movingAvg) / movingAvg

			if deviation > 0.5 { // 50% deviation
				severity := SeverityMedium
				if deviation > 1.0 {
					severity = SeverityHigh
				}

				anomaly := models.Anomaly{
					Type:        string(AnomalySpike),
					Severity:    string(severity),
					Description: fmt.Sprintf("Value %.2f deviates %.1f%% from moving average %.2f", points[i].Value, deviation*100, movingAvg),
					DetectedAt:  points[i].Timestamp,
					Value:       points[i].Value,
					Expected:    movingAvg,
				}

				if points[i].Value < movingAvg {
					anomaly.Type = string(AnomalyDrop)
				}

				anomalies = append(anomalies, anomaly)
			}
		}
	}

	return anomalies
}

// detectRateOfChangeAnomalies detects anomalies based on rate of change
func (a *analyzer) detectRateOfChangeAnomalies(points []models.DataPoint) []models.Anomaly {
	anomalies := []models.Anomaly{}

	if len(points) < 3 {
		return anomalies
	}

	// Calculate rate of change for each point
	rates := make([]float64, len(points)-1)
	for i := 1; i < len(points); i++ {
		timeDiff := points[i].Timestamp.Sub(points[i-1].Timestamp).Seconds()
		if timeDiff > 0 && points[i-1].Value > 0 {
			rates[i-1] = (points[i].Value - points[i-1].Value) / timeDiff
		}
	}

	if len(rates) == 0 {
		return anomalies
	}

	// Calculate mean and std dev of rates
	meanRate := 0.0
	for _, r := range rates {
		meanRate += r
	}
	meanRate /= float64(len(rates))

	variance := 0.0
	for _, r := range rates {
		diff := r - meanRate
		variance += diff * diff
	}
	variance /= float64(len(rates))
	stdDevRate := math.Sqrt(variance)

	// Detect anomalous rates of change
	if stdDevRate > 0 {
		for i, rate := range rates {
			zScore := math.Abs((rate - meanRate) / stdDevRate)
			if zScore > a.config.AnomalyThreshold {
				severity := a.determineSeverity(zScore)

				anomaly := models.Anomaly{
					Type:        string(AnomalySpike),
					Severity:    string(severity),
					Description: fmt.Sprintf("Unusual rate of change: %.2f (expected %.2f)", rate, meanRate),
					DetectedAt:  points[i+1].Timestamp,
					Value:       points[i+1].Value,
					Expected:    points[i].Value + meanRate*points[i+1].Timestamp.Sub(points[i].Timestamp).Seconds(),
				}

				anomalies = append(anomalies, anomaly)
			}
		}
	}

	return anomalies
}
