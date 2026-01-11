package analyzer

import (
	"github.com/k8s-service-optimizer/backend/pkg/collector"
)

// New creates a new analyzer with default configuration
func New(client collector.MetricsCollector) Analyzer {
	return NewWithConfig(client, DefaultConfig())
}

// NewWithConfig creates a new analyzer with custom configuration
func NewWithConfig(client collector.MetricsCollector, config Config) Analyzer {
	return &analyzer{
		client: client,
		config: config,
	}
}
