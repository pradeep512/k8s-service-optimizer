package collector

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/k8s-service-optimizer/backend/internal/models"
)

// metricsStore provides thread-safe in-memory storage for time-series metrics
type metricsStore struct {
	mu              sync.RWMutex
	data            map[metricKey][]models.DataPoint
	retentionPeriod time.Duration
}

// newMetricsStore creates a new metrics store
func newMetricsStore(retentionPeriod time.Duration) *metricsStore {
	return &metricsStore{
		data:            make(map[metricKey][]models.DataPoint),
		retentionPeriod: retentionPeriod,
	}
}

// Store adds a metric data point to the store
func (s *metricsStore) Store(resource, metric string, value float64, timestamp time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := metricKey{
		Resource: resource,
		Metric:   metric,
	}

	point := models.DataPoint{
		Timestamp: timestamp,
		Value:     value,
	}

	s.data[key] = append(s.data[key], point)
}

// StoreBatch adds multiple metric data points to the store
func (s *metricsStore) StoreBatch(entries []metricsEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, entry := range entries {
		s.data[entry.Key] = append(s.data[entry.Key], entry.Points...)
	}
}

// GetTimeSeriesData retrieves time-series data for a resource/metric within a duration
func (s *metricsStore) GetTimeSeriesData(resource, metric string, duration time.Duration) (models.TimeSeriesData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := metricKey{
		Resource: resource,
		Metric:   metric,
	}

	allPoints, exists := s.data[key]
	if !exists || len(allPoints) == 0 {
		return models.TimeSeriesData{
			Resource: resource,
			Metric:   metric,
			Points:   []models.DataPoint{},
		}, nil
	}

	// Filter points within the duration
	cutoff := time.Now().Add(-duration)
	var filteredPoints []models.DataPoint

	for _, point := range allPoints {
		if point.Timestamp.After(cutoff) {
			filteredPoints = append(filteredPoints, point)
		}
	}

	// Sort by timestamp
	sort.Slice(filteredPoints, func(i, j int) bool {
		return filteredPoints[i].Timestamp.Before(filteredPoints[j].Timestamp)
	})

	return models.TimeSeriesData{
		Resource: resource,
		Metric:   metric,
		Points:   filteredPoints,
	}, nil
}

// GetResourcePercentiles calculates percentiles for a resource metric
func (s *metricsStore) GetResourcePercentiles(resource, metric string, duration time.Duration) (p50, p95, p99 float64, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := metricKey{
		Resource: resource,
		Metric:   metric,
	}

	allPoints, exists := s.data[key]
	if !exists || len(allPoints) == 0 {
		return 0, 0, 0, fmt.Errorf("no data found for resource %s metric %s", resource, metric)
	}

	// Filter points within the duration
	cutoff := time.Now().Add(-duration)
	var values []float64

	for _, point := range allPoints {
		if point.Timestamp.After(cutoff) {
			values = append(values, point.Value)
		}
	}

	if len(values) == 0 {
		return 0, 0, 0, fmt.Errorf("no data found for resource %s metric %s within duration %v", resource, metric, duration)
	}

	// Sort values for percentile calculation
	sort.Float64s(values)

	p50 = calculatePercentile(values, 50)
	p95 = calculatePercentile(values, 95)
	p99 = calculatePercentile(values, 99)

	return p50, p95, p99, nil
}

// calculatePercentile calculates the percentile value from a sorted slice
func calculatePercentile(sortedValues []float64, percentile float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}

	if len(sortedValues) == 1 {
		return sortedValues[0]
	}

	// Calculate the index for the percentile
	index := (percentile / 100.0) * float64(len(sortedValues)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sortedValues) {
		return sortedValues[len(sortedValues)-1]
	}

	// Linear interpolation between two closest values
	weight := index - float64(lower)
	return sortedValues[lower]*(1-weight) + sortedValues[upper]*weight
}

// Cleanup removes data older than the retention period
func (s *metricsStore) Cleanup() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-s.retentionPeriod)
	removedCount := 0

	for key, points := range s.data {
		var kept []models.DataPoint

		for _, point := range points {
			if point.Timestamp.After(cutoff) {
				kept = append(kept, point)
			} else {
				removedCount++
			}
		}

		if len(kept) == 0 {
			// Remove the entire key if no points remain
			delete(s.data, key)
		} else {
			s.data[key] = kept
		}
	}

	return removedCount
}

// Size returns the total number of data points in the store
func (s *metricsStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := 0
	for _, points := range s.data {
		total += len(points)
	}
	return total
}

// Keys returns all metric keys in the store
func (s *metricsStore) Keys() []metricKey {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]metricKey, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}
	return keys
}
