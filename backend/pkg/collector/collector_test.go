package collector

import (
	"testing"
	"time"

	"github.com/k8s-service-optimizer/backend/internal/models"
)

// TestMetricsStore tests the basic functionality of the metrics store
func TestMetricsStore(t *testing.T) {
	store := newMetricsStore(24 * time.Hour)

	// Store some test data
	now := time.Now()
	store.Store("pod/test-pod", "cpu", 100.0, now)
	store.Store("pod/test-pod", "cpu", 200.0, now.Add(1*time.Second))
	store.Store("pod/test-pod", "cpu", 300.0, now.Add(2*time.Second))
	store.Store("pod/test-pod", "memory", 1024.0, now)

	// Test GetTimeSeriesData
	tsData, err := store.GetTimeSeriesData("pod/test-pod", "cpu", 1*time.Hour)
	if err != nil {
		t.Fatalf("GetTimeSeriesData failed: %v", err)
	}

	if tsData.Resource != "pod/test-pod" {
		t.Errorf("Expected resource 'pod/test-pod', got '%s'", tsData.Resource)
	}

	if tsData.Metric != "cpu" {
		t.Errorf("Expected metric 'cpu', got '%s'", tsData.Metric)
	}

	if len(tsData.Points) != 3 {
		t.Errorf("Expected 3 points, got %d", len(tsData.Points))
	}

	// Test GetResourcePercentiles
	p50, p95, p99, err := store.GetResourcePercentiles("pod/test-pod", "cpu", 1*time.Hour)
	if err != nil {
		t.Fatalf("GetResourcePercentiles failed: %v", err)
	}

	if p50 != 200.0 {
		t.Errorf("Expected P50 = 200.0, got %f", p50)
	}

	if p95 < 200.0 || p95 > 300.0 {
		t.Errorf("Expected P95 between 200-300, got %f", p95)
	}

	if p99 < 200.0 || p99 > 300.0 {
		t.Errorf("Expected P99 between 200-300, got %f", p99)
	}

	// Test Size
	size := store.Size()
	if size != 4 {
		t.Errorf("Expected size 4, got %d", size)
	}
}

// TestMetricsStoreCleanup tests the cleanup functionality
func TestMetricsStoreCleanup(t *testing.T) {
	store := newMetricsStore(1 * time.Hour)

	now := time.Now()

	// Store old data (2 hours ago)
	store.Store("pod/old-pod", "cpu", 100.0, now.Add(-2*time.Hour))
	store.Store("pod/old-pod", "cpu", 150.0, now.Add(-90*time.Minute))

	// Store recent data
	store.Store("pod/new-pod", "cpu", 200.0, now.Add(-30*time.Minute))
	store.Store("pod/new-pod", "cpu", 250.0, now)

	// Initial size should be 4
	if store.Size() != 4 {
		t.Errorf("Expected initial size 4, got %d", store.Size())
	}

	// Run cleanup
	removed := store.Cleanup()

	// Should remove 2 old points
	if removed != 2 {
		t.Errorf("Expected 2 points removed, got %d", removed)
	}

	// New size should be 2
	if store.Size() != 2 {
		t.Errorf("Expected size 2 after cleanup, got %d", store.Size())
	}

	// Old pod should be completely removed
	_, err := store.GetTimeSeriesData("pod/old-pod", "cpu", 24*time.Hour)
	if err != nil {
		t.Errorf("Expected no error for missing data, got: %v", err)
	}
}

// TestCalculatePercentile tests the percentile calculation function
func TestCalculatePercentile(t *testing.T) {
	tests := []struct {
		name       string
		values     []float64
		percentile float64
		expected   float64
	}{
		{
			name:       "single value",
			values:     []float64{100},
			percentile: 50,
			expected:   100,
		},
		{
			name:       "two values P50",
			values:     []float64{100, 200},
			percentile: 50,
			expected:   150,
		},
		{
			name:       "five values P50",
			values:     []float64{100, 200, 300, 400, 500},
			percentile: 50,
			expected:   300,
		},
		{
			name:       "five values P95",
			values:     []float64{100, 200, 300, 400, 500},
			percentile: 95,
			expected:   480,
		},
		{
			name:       "five values P99",
			values:     []float64{100, 200, 300, 400, 500},
			percentile: 99,
			expected:   496,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculatePercentile(tt.values, tt.percentile)
			// Use a small tolerance for floating point comparison
			tolerance := 0.0001
			if result < tt.expected-tolerance || result > tt.expected+tolerance {
				t.Errorf("calculatePercentile(%v, %f) = %f, expected %f",
					tt.values, tt.percentile, result, tt.expected)
			}
		})
	}
}

// TestMetricsStoreConcurrency tests thread-safety of the store
func TestMetricsStoreConcurrency(t *testing.T) {
	store := newMetricsStore(24 * time.Hour)

	// Run concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				store.Store("pod/test", "cpu", float64(j), time.Now())
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify data was stored
	size := store.Size()
	if size != 1000 {
		t.Errorf("Expected 1000 points, got %d", size)
	}
}

// TestConfigDefaults tests the default configuration
func TestConfigDefaults(t *testing.T) {
	config := DefaultConfig()

	if config.CollectionInterval != 15*time.Second {
		t.Errorf("Expected collection interval 15s, got %v", config.CollectionInterval)
	}

	if config.RetentionPeriod != 24*time.Hour {
		t.Errorf("Expected retention period 24h, got %v", config.RetentionPeriod)
	}

	if config.CleanupInterval != 1*time.Hour {
		t.Errorf("Expected cleanup interval 1h, got %v", config.CleanupInterval)
	}
}

// TestMetricsStoreEmptyQuery tests querying non-existent data
func TestMetricsStoreEmptyQuery(t *testing.T) {
	store := newMetricsStore(24 * time.Hour)

	// Query non-existent resource
	tsData, err := store.GetTimeSeriesData("pod/nonexistent", "cpu", 1*time.Hour)
	if err != nil {
		t.Errorf("Expected no error for non-existent data, got: %v", err)
	}

	if len(tsData.Points) != 0 {
		t.Errorf("Expected 0 points for non-existent data, got %d", len(tsData.Points))
	}

	// Query percentiles for non-existent resource
	_, _, _, err = store.GetResourcePercentiles("pod/nonexistent", "cpu", 1*time.Hour)
	if err == nil {
		t.Error("Expected error for percentiles of non-existent data")
	}
}

// TestStoreBatch tests batch storage functionality
func TestStoreBatch(t *testing.T) {
	store := newMetricsStore(24 * time.Hour)

	now := time.Now()
	entries := []metricsEntry{
		{
			Key: metricKey{Resource: "pod/test1", Metric: "cpu"},
			Points: []models.DataPoint{
				{Timestamp: now, Value: 100},
				{Timestamp: now.Add(1 * time.Second), Value: 200},
			},
		},
		{
			Key: metricKey{Resource: "pod/test2", Metric: "memory"},
			Points: []models.DataPoint{
				{Timestamp: now, Value: 1024},
			},
		},
	}

	store.StoreBatch(entries)

	if store.Size() != 3 {
		t.Errorf("Expected 3 points after batch store, got %d", store.Size())
	}

	// Verify data can be retrieved
	tsData, err := store.GetTimeSeriesData("pod/test1", "cpu", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to get time series data: %v", err)
	}

	if len(tsData.Points) != 2 {
		t.Errorf("Expected 2 points for pod/test1 cpu, got %d", len(tsData.Points))
	}
}
