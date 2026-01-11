package collector

import (
	"time"

	"github.com/k8s-service-optimizer/backend/internal/models"
)

// MetricsCollector defines the interface for collecting metrics
type MetricsCollector interface {
	// Start begins the metrics collection loop
	Start() error

	// Stop stops the metrics collection
	Stop()

	// CollectPodMetrics collects current pod metrics for a namespace
	CollectPodMetrics(namespace string) ([]models.PodMetrics, error)

	// CollectNodeMetrics collects current node metrics
	CollectNodeMetrics() ([]models.NodeMetrics, error)

	// CollectHPAMetrics collects HPA metrics for a namespace
	CollectHPAMetrics(namespace string) ([]models.HPAMetrics, error)

	// GetTimeSeriesData retrieves time-series data for a resource/metric
	GetTimeSeriesData(resource, metric string, duration time.Duration) (models.TimeSeriesData, error)

	// GetResourcePercentiles calculates percentiles for a resource metric
	GetResourcePercentiles(resource, metric string, duration time.Duration) (p50, p95, p99 float64, err error)
}

// Config holds collector configuration
type Config struct {
	// CollectionInterval is how often to collect metrics
	CollectionInterval time.Duration

	// RetentionPeriod is how long to keep metrics in memory
	RetentionPeriod time.Duration

	// CleanupInterval is how often to run cleanup of old data
	CleanupInterval time.Duration
}

// DefaultConfig returns default collector configuration
func DefaultConfig() Config {
	return Config{
		CollectionInterval: 15 * time.Second,
		RetentionPeriod:    24 * time.Hour,
		CleanupInterval:    1 * time.Hour,
	}
}

// metricKey represents a unique identifier for a metric
type metricKey struct {
	Resource string // e.g., "pod/echo-demo-xxx", "node/worker-1"
	Metric   string // e.g., "cpu", "memory"
}

// metricsEntry stores time-series data for a specific metric
type metricsEntry struct {
	Key    metricKey
	Points []models.DataPoint
}
