package models

import "time"

// PodMetrics represents resource usage metrics for a pod
type PodMetrics struct {
	Name      string
	Namespace string
	CPU       int64 // millicores
	Memory    int64 // bytes
	Timestamp time.Time
}

// NodeMetrics represents resource usage metrics for a node
type NodeMetrics struct {
	Name      string
	CPU       int64 // millicores
	Memory    int64 // bytes
	Timestamp time.Time
}

// HPAMetrics represents HPA status and metrics
type HPAMetrics struct {
	Name             string
	Namespace        string
	CurrentReplicas  int32
	DesiredReplicas  int32
	MinReplicas      int32
	MaxReplicas      int32
	TargetCPU        int32
	CurrentCPU       int32
	Timestamp        time.Time
}

// TimeSeriesData represents a time series of metric values
type TimeSeriesData struct {
	Resource string
	Metric   string
	Points   []DataPoint
}

// DataPoint represents a single metric data point
type DataPoint struct {
	Timestamp time.Time
	Value     float64
}

// Analysis represents the analysis result for a deployment
type Analysis struct {
	Namespace   string
	Deployment  string
	CPUUsage    ResourceAnalysis
	MemoryUsage ResourceAnalysis
	Replicas    ReplicaAnalysis
	HealthScore float64
	Timestamp   time.Time
}

// ResourceAnalysis represents analysis of CPU or memory usage
type ResourceAnalysis struct {
	Requested     int64
	Current       int64
	P50           int64
	P95           int64
	P99           int64
	Average       int64
	Max           int64
	Utilization   float64 // percentage
	Efficiency    float64 // 0-100 score
}

// ReplicaAnalysis represents analysis of replica usage
type ReplicaAnalysis struct {
	Current     int32
	Min         int32
	Max         int32
	Recommended int32
}

// Recommendation represents an optimization recommendation
type Recommendation struct {
	ID              string
	Type            string // "resource", "hpa", "scaling"
	Namespace       string
	Deployment      string
	Priority        string // "high", "medium", "low"
	Description     string
	CurrentConfig   interface{}
	RecommendedConfig interface{}
	EstimatedSavings float64
	Impact          string
	CreatedAt       time.Time
}

// TrafficAnalysis represents traffic pattern analysis
type TrafficAnalysis struct {
	Service       string
	Namespace     string
	RequestRate   float64
	ErrorRate     float64
	P50Latency    float64
	P95Latency    float64
	P99Latency    float64
	Anomalies     []Anomaly
	Timestamp     time.Time
}

// Anomaly represents a detected anomaly
type Anomaly struct {
	Type        string
	Severity    string
	Description string
	DetectedAt  time.Time
	Value       float64
	Expected    float64
}

// CostBreakdown represents cost analysis for a service
type CostBreakdown struct {
	Service         string
	Namespace       string
	CPUCost         float64
	MemoryCost      float64
	TotalCost       float64
	WastedCost      float64
	EfficiencyScore float64
	Timestamp       time.Time
}

// ResourcePrediction represents predicted resource needs
type ResourcePrediction struct {
	Service       string
	Namespace     string
	Hours         int
	PredictedCPU  int64
	PredictedMemory int64
	Confidence    float64
	Timestamp     time.Time
}

// ClusterOverview represents overall cluster status
type ClusterOverview struct {
	TotalNodes      int
	HealthyNodes    int
	TotalPods       int
	HealthyPods     int
	CPUCapacity     int64
	CPUUsage        int64
	MemoryCapacity  int64
	MemoryUsage     int64
	Namespaces      []string
	Timestamp       time.Time
}

// ServiceDetail represents detailed information about a service
type ServiceDetail struct {
	Name        string
	Namespace   string
	Type        string
	Replicas    int32
	HealthScore float64
	CPUUsage    ResourceAnalysis
	MemoryUsage ResourceAnalysis
	Traffic     TrafficAnalysis
	Cost        CostBreakdown
	Pods        []PodInfo
	Timestamp   time.Time
}

// PodInfo represents basic pod information
type PodInfo struct {
	Name      string
	Status    string
	Restarts  int32
	Age       time.Duration
	Node      string
	CPUUsage  int64
	MemoryUsage int64
}
