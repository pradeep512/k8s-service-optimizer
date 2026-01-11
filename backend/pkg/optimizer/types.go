package optimizer

import (
	"time"

	"github.com/k8s-service-optimizer/backend/internal/models"
)

// Config holds optimizer configuration
type Config struct {
	// AnalysisDuration is the time window to analyze metrics (default: 7 days)
	AnalysisDuration time.Duration

	// CPUOverProvisionedThreshold is the threshold for detecting over-provisioned CPU (default: 0.5 = 50%)
	CPUOverProvisionedThreshold float64

	// MemoryOverProvisionedThreshold is the threshold for detecting over-provisioned memory (default: 0.5 = 50%)
	MemoryOverProvisionedThreshold float64

	// CPUUnderProvisionedThreshold is the threshold for detecting under-provisioned CPU (default: 0.8 = 80%)
	CPUUnderProvisionedThreshold float64

	// MemoryUnderProvisionedThreshold is the threshold for detecting under-provisioned memory (default: 0.8 = 80%)
	MemoryUnderProvisionedThreshold float64

	// OverProvisionedBuffer is the buffer to add to recommendations for over-provisioned resources (default: 1.2 = 20% buffer)
	OverProvisionedBuffer float64

	// UnderProvisionedBuffer is the buffer to add to recommendations for under-provisioned resources (default: 1.5 = 50% buffer)
	UnderProvisionedBuffer float64

	// CPUCostPerVCPUHour is the cost per vCPU-hour for cost estimation (default: $0.03)
	CPUCostPerVCPUHour float64

	// MemoryCostPerGBHour is the cost per GB-hour for cost estimation (default: $0.004)
	MemoryCostPerGBHour float64

	// MinimumDataPoints is the minimum number of data points required for analysis (default: 10)
	MinimumDataPoints int

	// OptimalUtilizationMin is the minimum optimal resource utilization (default: 0.7 = 70%)
	OptimalUtilizationMin float64

	// OptimalUtilizationMax is the maximum optimal resource utilization (default: 0.9 = 90%)
	OptimalUtilizationMax float64
}

// DefaultConfig returns the default optimizer configuration
func DefaultConfig() Config {
	return Config{
		AnalysisDuration:                7 * 24 * time.Hour, // 7 days
		CPUOverProvisionedThreshold:     0.5,
		MemoryOverProvisionedThreshold:  0.5,
		CPUUnderProvisionedThreshold:    0.8,
		MemoryUnderProvisionedThreshold: 0.8,
		OverProvisionedBuffer:           1.2, // 20% buffer
		UnderProvisionedBuffer:          1.5, // 50% buffer
		CPUCostPerVCPUHour:              0.03,
		MemoryCostPerGBHour:             0.004,
		MinimumDataPoints:               10,
		OptimalUtilizationMin:           0.7,
		OptimalUtilizationMax:           0.9,
	}
}

// deploymentMetrics holds aggregated metrics for a deployment
type deploymentMetrics struct {
	Namespace  string
	Deployment string

	// CPU metrics (in millicores)
	CPURequested int64
	CPULimit     int64
	CPUCurrent   int64
	CPUP50       int64
	CPUP95       int64
	CPUP99       int64
	CPUAverage   int64
	CPUMax       int64

	// Memory metrics (in bytes)
	MemoryRequested int64
	MemoryLimit     int64
	MemoryCurrent   int64
	MemoryP50       int64
	MemoryP95       int64
	MemoryP99       int64
	MemoryAverage   int64
	MemoryMax       int64

	// Replica information
	CurrentReplicas int32
	MinReplicas     int32
	MaxReplicas     int32

	// HPA information (if exists)
	HasHPA             bool
	HPATargetCPU       int32
	HPACurrentCPU      int32
	HPADesiredReplicas int32

	// Stability metrics
	RestartCount  int32
	ScalingEvents int

	// Time series data for variance calculation
	CPUTimeSeries     []models.DataPoint
	MemoryTimeSeries  []models.DataPoint
	ReplicaTimeSeries []models.DataPoint

	Timestamp time.Time
}

// analysisResult holds the results of resource analysis
type analysisResult struct {
	Deployment deploymentMetrics

	// CPU analysis
	CPUUtilization      float64
	CPUEfficiency       float64
	CPUVariance         float64
	CPUOverProvisioned  bool
	CPUUnderProvisioned bool

	// Memory analysis
	MemoryUtilization      float64
	MemoryEfficiency       float64
	MemoryVariance         float64
	MemoryOverProvisioned  bool
	MemoryUnderProvisioned bool

	// HPA analysis
	HPANeedsOptimization bool
	HPAScalingFrequency  float64
	HPAScalingAmplitude  float64
	HPAHitCeiling        bool
	HPAIdleAtMinimum     bool

	// Overall scores
	ResourceUtilizationScore float64
	StabilityScore           float64
	CostEfficiencyScore      float64
	OverallScore             float64

	Timestamp time.Time
}

// resourceConfig represents the current or recommended resource configuration
type resourceConfig struct {
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
}

// hpaConfig represents the current or recommended HPA configuration
type hpaConfig struct {
	MinReplicas int32
	MaxReplicas int32
	TargetCPU   int32
}

// scalingConfig represents scaling-related configuration
type scalingConfig struct {
	Replicas int32
}

// recommendationPriority determines the priority level of a recommendation
type recommendationPriority string

const (
	PriorityHigh   recommendationPriority = "high"
	PriorityMedium recommendationPriority = "medium"
	PriorityLow    recommendationPriority = "low"
)

// recommendationType defines the type of recommendation
type recommendationType string

const (
	RecommendationTypeResource recommendationType = "resource"
	RecommendationTypeHPA      recommendationType = "hpa"
	RecommendationTypeScaling  recommendationType = "scaling"
)
