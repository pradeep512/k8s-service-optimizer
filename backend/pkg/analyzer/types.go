package analyzer

import (
	"time"

	"github.com/k8s-service-optimizer/backend/internal/models"
	"github.com/k8s-service-optimizer/backend/pkg/collector"
)

// Analyzer defines the interface for traffic and cost analysis
type Analyzer interface {
	// AnalyzeTrafficPatterns analyzes traffic patterns for a service
	AnalyzeTrafficPatterns(namespace, service string, duration time.Duration) (*models.TrafficAnalysis, error)

	// CalculateServiceCost calculates the cost for a specific service
	CalculateServiceCost(namespace, service string) (*models.CostBreakdown, error)

	// DetectAnomalies detects anomalies in metrics
	DetectAnomalies(resource, metric string, duration time.Duration) ([]models.Anomaly, error)

	// PredictResourceNeeds predicts future resource requirements
	PredictResourceNeeds(namespace, service string, hours int) (*models.ResourcePrediction, error)

	// GetCostTrends gets cost trends over time
	GetCostTrends(namespace string, duration time.Duration) ([]models.CostBreakdown, error)

	// CalculateWaste calculates wasted resources (over-provisioning)
	CalculateWaste(namespace, service string) (float64, error)
}

// Config holds analyzer configuration
type Config struct {
	// CPUCostPerVCPUHour is the cost per vCPU-hour (1000 millicores = 1 vCPU)
	CPUCostPerVCPUHour float64

	// MemoryCostPerGBHour is the cost per GB-hour (1024 MB = 1 GB)
	MemoryCostPerGBHour float64

	// AnomalyThreshold is the Z-score threshold for anomaly detection
	AnomalyThreshold float64

	// SpikeThreshold is the multiplier for spike detection
	SpikeThreshold float64

	// DropThreshold is the multiplier for drop detection
	DropThreshold float64

	// MinDataPoints is the minimum number of data points required for analysis
	MinDataPoints int

	// TrendHistoryDays is the number of days to use for trend analysis
	TrendHistoryDays int
}

// DefaultConfig returns default analyzer configuration
func DefaultConfig() Config {
	return Config{
		CPUCostPerVCPUHour:  0.03,   // $0.03 per vCPU-hour
		MemoryCostPerGBHour: 0.004,  // $0.004 per GB-hour
		AnomalyThreshold:    3.0,    // 3 standard deviations
		SpikeThreshold:      2.0,    // 2x normal
		DropThreshold:       0.5,    // 0.5x normal
		MinDataPoints:       10,     // Minimum points for meaningful analysis
		TrendHistoryDays:    7,      // 7 days of history
	}
}

// trafficPattern represents the detected traffic pattern
type trafficPattern string

const (
	PatternSteady    trafficPattern = "steady"
	PatternSpiking   trafficPattern = "spiking"
	PatternPeriodic  trafficPattern = "periodic"
	PatternDeclining trafficPattern = "declining"
	PatternIncreasing trafficPattern = "increasing"
)

// anomalyType represents the type of anomaly
type anomalyType string

const (
	AnomalySpike       anomalyType = "spike"
	AnomalyDrop        anomalyType = "drop"
	AnomalyDrift       anomalyType = "drift"
	AnomalyOscillation anomalyType = "oscillation"
	AnomalyErrorSpike  anomalyType = "error_spike"
)

// anomalySeverity represents the severity of an anomaly
type anomalySeverity string

const (
	SeverityCritical anomalySeverity = "critical"
	SeverityHigh     anomalySeverity = "high"
	SeverityMedium   anomalySeverity = "medium"
	SeverityLow      anomalySeverity = "low"
)

// trendData represents trend analysis results
type trendData struct {
	Slope      float64 // Rate of change
	Intercept  float64 // Y-intercept
	RSquared   float64 // Confidence measure
	Prediction float64 // Predicted value
}

// analyzer implements the Analyzer interface
type analyzer struct {
	client    collector.MetricsCollector
	config    Config
}
