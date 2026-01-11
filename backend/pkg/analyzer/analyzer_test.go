package analyzer

import (
	"testing"
	"time"

	"github.com/k8s-service-optimizer/backend/internal/models"
)

// mockCollector is a mock implementation of MetricsCollector for testing
type mockCollector struct {
	timeSeriesData map[string]models.TimeSeriesData
	percentiles    map[string]percentileData
}

type percentileData struct {
	p50 float64
	p95 float64
	p99 float64
}

func newMockCollector() *mockCollector {
	return &mockCollector{
		timeSeriesData: make(map[string]models.TimeSeriesData),
		percentiles:    make(map[string]percentileData),
	}
}

func (m *mockCollector) Start() error                                      { return nil }
func (m *mockCollector) Stop()                                             {}
func (m *mockCollector) CollectPodMetrics(namespace string) ([]models.PodMetrics, error) {
	return []models.PodMetrics{}, nil
}
func (m *mockCollector) CollectNodeMetrics() ([]models.NodeMetrics, error) {
	return []models.NodeMetrics{}, nil
}
func (m *mockCollector) CollectHPAMetrics(namespace string) ([]models.HPAMetrics, error) {
	return []models.HPAMetrics{}, nil
}

func (m *mockCollector) GetTimeSeriesData(resource, metric string, duration time.Duration) (models.TimeSeriesData, error) {
	key := resource + "/" + metric
	if data, ok := m.timeSeriesData[key]; ok {
		return data, nil
	}
	return models.TimeSeriesData{
		Resource: resource,
		Metric:   metric,
		Points:   []models.DataPoint{},
	}, nil
}

func (m *mockCollector) GetResourcePercentiles(resource, metric string, duration time.Duration) (p50, p95, p99 float64, err error) {
	key := resource + "/" + metric
	if data, ok := m.percentiles[key]; ok {
		return data.p50, data.p95, data.p99, nil
	}
	return 0, 0, 0, nil
}

func (m *mockCollector) addTimeSeriesData(resource, metric string, points []models.DataPoint) {
	key := resource + "/" + metric
	m.timeSeriesData[key] = models.TimeSeriesData{
		Resource: resource,
		Metric:   metric,
		Points:   points,
	}
}

// TestNew verifies analyzer creation
func TestNew(t *testing.T) {
	mc := newMockCollector()
	an := New(mc)

	if an == nil {
		t.Fatal("Expected non-nil analyzer")
	}
}

// TestNewWithConfig verifies analyzer creation with custom config
func TestNewWithConfig(t *testing.T) {
	mc := newMockCollector()
	config := Config{
		CPUCostPerVCPUHour:  0.05,
		MemoryCostPerGBHour: 0.006,
		AnomalyThreshold:    2.5,
		SpikeThreshold:      1.5,
		DropThreshold:       0.6,
		MinDataPoints:       5,
		TrendHistoryDays:    10,
	}

	an := NewWithConfig(mc, config)
	if an == nil {
		t.Fatal("Expected non-nil analyzer")
	}

	// Verify it's using custom config
	analyzer := an.(*analyzer)
	if analyzer.config.CPUCostPerVCPUHour != 0.05 {
		t.Errorf("Expected CPU cost 0.05, got %f", analyzer.config.CPUCostPerVCPUHour)
	}
}

// TestDefaultConfig verifies default configuration values
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.CPUCostPerVCPUHour != 0.03 {
		t.Errorf("Expected CPU cost 0.03, got %f", config.CPUCostPerVCPUHour)
	}

	if config.MemoryCostPerGBHour != 0.004 {
		t.Errorf("Expected memory cost 0.004, got %f", config.MemoryCostPerGBHour)
	}

	if config.AnomalyThreshold != 3.0 {
		t.Errorf("Expected anomaly threshold 3.0, got %f", config.AnomalyThreshold)
	}

	if config.MinDataPoints != 10 {
		t.Errorf("Expected min data points 10, got %d", config.MinDataPoints)
	}
}

// TestAnalyzeTrafficPatterns tests traffic pattern analysis
func TestAnalyzeTrafficPatterns(t *testing.T) {
	mc := newMockCollector()
	an := New(mc)

	// Add mock CPU data
	now := time.Now()
	points := []models.DataPoint{
		{Timestamp: now.Add(-10 * time.Minute), Value: 100},
		{Timestamp: now.Add(-9 * time.Minute), Value: 110},
		{Timestamp: now.Add(-8 * time.Minute), Value: 105},
		{Timestamp: now.Add(-7 * time.Minute), Value: 115},
		{Timestamp: now.Add(-6 * time.Minute), Value: 108},
		{Timestamp: now.Add(-5 * time.Minute), Value: 112},
		{Timestamp: now.Add(-4 * time.Minute), Value: 106},
		{Timestamp: now.Add(-3 * time.Minute), Value: 111},
		{Timestamp: now.Add(-2 * time.Minute), Value: 109},
		{Timestamp: now.Add(-1 * time.Minute), Value: 113},
		{Timestamp: now, Value: 107},
	}
	mc.addTimeSeriesData("pod/nginx", "cpu", points)

	traffic, err := an.AnalyzeTrafficPatterns("default", "nginx", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to analyze traffic: %v", err)
	}

	if traffic.Service != "nginx" {
		t.Errorf("Expected service 'nginx', got '%s'", traffic.Service)
	}

	if traffic.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", traffic.Namespace)
	}

	// Request rate should be approximately avg CPU / 10
	if traffic.RequestRate < 0 {
		t.Errorf("Expected positive request rate, got %f", traffic.RequestRate)
	}

	// Error rate should be between 0 and 1
	if traffic.ErrorRate < 0 || traffic.ErrorRate > 1 {
		t.Errorf("Expected error rate between 0 and 1, got %f", traffic.ErrorRate)
	}
}

// TestAnalyzeTrafficPatternsNoData tests with no data
func TestAnalyzeTrafficPatternsNoData(t *testing.T) {
	mc := newMockCollector()
	an := New(mc)

	traffic, err := an.AnalyzeTrafficPatterns("default", "nginx", 1*time.Hour)
	if err != nil {
		t.Fatalf("Expected no error with no data, got: %v", err)
	}

	if traffic.RequestRate != 0 {
		t.Errorf("Expected 0 request rate with no data, got %f", traffic.RequestRate)
	}

	if traffic.ErrorRate != 0 {
		t.Errorf("Expected 0 error rate with no data, got %f", traffic.ErrorRate)
	}
}

// TestCalculateServiceCost tests cost calculation
func TestCalculateServiceCost(t *testing.T) {
	mc := newMockCollector()
	an := New(mc)

	// Add mock data: 200m CPU, 256MB memory
	now := time.Now()
	cpuPoints := []models.DataPoint{
		{Timestamp: now.Add(-5 * time.Minute), Value: 180},
		{Timestamp: now.Add(-4 * time.Minute), Value: 190},
		{Timestamp: now.Add(-3 * time.Minute), Value: 200},
		{Timestamp: now.Add(-2 * time.Minute), Value: 195},
		{Timestamp: now.Add(-1 * time.Minute), Value: 205},
		{Timestamp: now, Value: 198},
	}
	memPoints := []models.DataPoint{
		{Timestamp: now.Add(-5 * time.Minute), Value: 250 * 1024 * 1024},
		{Timestamp: now.Add(-4 * time.Minute), Value: 255 * 1024 * 1024},
		{Timestamp: now.Add(-3 * time.Minute), Value: 260 * 1024 * 1024},
		{Timestamp: now.Add(-2 * time.Minute), Value: 258 * 1024 * 1024},
		{Timestamp: now.Add(-1 * time.Minute), Value: 262 * 1024 * 1024},
		{Timestamp: now, Value: 256 * 1024 * 1024},
	}
	mc.addTimeSeriesData("pod/nginx", "cpu", cpuPoints)
	mc.addTimeSeriesData("pod/nginx", "memory", memPoints)

	cost, err := an.CalculateServiceCost("default", "nginx")
	if err != nil {
		t.Fatalf("Failed to calculate cost: %v", err)
	}

	if cost.Service != "nginx" {
		t.Errorf("Expected service 'nginx', got '%s'", cost.Service)
	}

	if cost.TotalCost <= 0 {
		t.Errorf("Expected positive total cost, got %f", cost.TotalCost)
	}

	if cost.CPUCost <= 0 {
		t.Errorf("Expected positive CPU cost, got %f", cost.CPUCost)
	}

	if cost.MemoryCost <= 0 {
		t.Errorf("Expected positive memory cost, got %f", cost.MemoryCost)
	}

	if cost.EfficiencyScore < 0 || cost.EfficiencyScore > 100 {
		t.Errorf("Expected efficiency score between 0 and 100, got %f", cost.EfficiencyScore)
	}
}

// TestCalculateServiceCostNoData tests cost calculation with no data
func TestCalculateServiceCostNoData(t *testing.T) {
	mc := newMockCollector()
	an := New(mc)

	cost, err := an.CalculateServiceCost("default", "nginx")
	if err != nil {
		t.Fatalf("Expected no error with no data, got: %v", err)
	}

	if cost.TotalCost != 0 {
		t.Errorf("Expected 0 total cost with no data, got %f", cost.TotalCost)
	}

	if cost.EfficiencyScore != 100 {
		t.Errorf("Expected 100%% efficiency with no data, got %f", cost.EfficiencyScore)
	}
}

// TestDetectAnomalies tests anomaly detection
func TestDetectAnomalies(t *testing.T) {
	mc := newMockCollector()
	an := New(mc)

	// Create data with a clear spike
	now := time.Now()
	points := []models.DataPoint{
		{Timestamp: now.Add(-10 * time.Minute), Value: 100},
		{Timestamp: now.Add(-9 * time.Minute), Value: 105},
		{Timestamp: now.Add(-8 * time.Minute), Value: 102},
		{Timestamp: now.Add(-7 * time.Minute), Value: 98},
		{Timestamp: now.Add(-6 * time.Minute), Value: 103},
		{Timestamp: now.Add(-5 * time.Minute), Value: 500}, // SPIKE
		{Timestamp: now.Add(-4 * time.Minute), Value: 101},
		{Timestamp: now.Add(-3 * time.Minute), Value: 99},
		{Timestamp: now.Add(-2 * time.Minute), Value: 104},
		{Timestamp: now.Add(-1 * time.Minute), Value: 100},
		{Timestamp: now, Value: 102},
	}
	mc.addTimeSeriesData("pod/nginx", "cpu", points)

	anomalies, err := an.DetectAnomalies("pod/nginx", "cpu", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to detect anomalies: %v", err)
	}

	// Should detect at least the spike
	if len(anomalies) == 0 {
		t.Error("Expected to detect anomalies, found none")
	}

	// Check that spike was detected
	foundSpike := false
	for _, a := range anomalies {
		if a.Type == string(AnomalySpike) {
			foundSpike = true
			break
		}
	}

	if !foundSpike {
		t.Error("Expected to detect spike anomaly")
	}
}

// TestDetectAnomaliesNoData tests anomaly detection with no data
func TestDetectAnomaliesNoData(t *testing.T) {
	mc := newMockCollector()
	an := New(mc)

	anomalies, err := an.DetectAnomalies("pod/nginx", "cpu", 1*time.Hour)
	if err != nil {
		t.Fatalf("Expected no error with no data, got: %v", err)
	}

	if len(anomalies) != 0 {
		t.Errorf("Expected no anomalies with no data, got %d", len(anomalies))
	}
}

// TestPredictResourceNeeds tests resource prediction
func TestPredictResourceNeeds(t *testing.T) {
	mc := newMockCollector()
	an := New(mc)

	// Add trending data (increasing trend)
	now := time.Now()
	cpuPoints := []models.DataPoint{
		{Timestamp: now.Add(-10 * time.Hour), Value: 100},
		{Timestamp: now.Add(-9 * time.Hour), Value: 110},
		{Timestamp: now.Add(-8 * time.Hour), Value: 120},
		{Timestamp: now.Add(-7 * time.Hour), Value: 130},
		{Timestamp: now.Add(-6 * time.Hour), Value: 140},
		{Timestamp: now.Add(-5 * time.Hour), Value: 150},
		{Timestamp: now.Add(-4 * time.Hour), Value: 160},
		{Timestamp: now.Add(-3 * time.Hour), Value: 170},
		{Timestamp: now.Add(-2 * time.Hour), Value: 180},
		{Timestamp: now.Add(-1 * time.Hour), Value: 190},
		{Timestamp: now, Value: 200},
	}
	memPoints := []models.DataPoint{
		{Timestamp: now.Add(-10 * time.Hour), Value: 200 * 1024 * 1024},
		{Timestamp: now.Add(-5 * time.Hour), Value: 210 * 1024 * 1024},
		{Timestamp: now, Value: 220 * 1024 * 1024},
	}
	mc.addTimeSeriesData("pod/nginx", "cpu", cpuPoints)
	mc.addTimeSeriesData("pod/nginx", "memory", memPoints)

	prediction, err := an.PredictResourceNeeds("default", "nginx", 24)
	if err != nil {
		t.Fatalf("Failed to predict resources: %v", err)
	}

	if prediction.Service != "nginx" {
		t.Errorf("Expected service 'nginx', got '%s'", prediction.Service)
	}

	if prediction.Hours != 24 {
		t.Errorf("Expected hours 24, got %d", prediction.Hours)
	}

	// With increasing trend, prediction should be higher than current
	if prediction.PredictedCPU < 200 {
		t.Errorf("Expected predicted CPU >= 200, got %d", prediction.PredictedCPU)
	}

	if prediction.Confidence < 0 || prediction.Confidence > 1 {
		t.Errorf("Expected confidence between 0 and 1, got %f", prediction.Confidence)
	}
}

// TestCalculateWaste tests waste calculation
func TestCalculateWaste(t *testing.T) {
	mc := newMockCollector()
	an := New(mc)

	// Add data where usage is consistently low (lots of waste)
	now := time.Now()
	cpuPoints := []models.DataPoint{
		{Timestamp: now.Add(-5 * time.Minute), Value: 50},
		{Timestamp: now.Add(-4 * time.Minute), Value: 55},
		{Timestamp: now.Add(-3 * time.Minute), Value: 52},
		{Timestamp: now.Add(-2 * time.Minute), Value: 48},
		{Timestamp: now.Add(-1 * time.Minute), Value: 53},
		{Timestamp: now, Value: 51},
	}
	memPoints := []models.DataPoint{
		{Timestamp: now.Add(-5 * time.Minute), Value: 100 * 1024 * 1024},
		{Timestamp: now.Add(-4 * time.Minute), Value: 105 * 1024 * 1024},
		{Timestamp: now.Add(-3 * time.Minute), Value: 102 * 1024 * 1024},
		{Timestamp: now.Add(-2 * time.Minute), Value: 98 * 1024 * 1024},
		{Timestamp: now.Add(-1 * time.Minute), Value: 103 * 1024 * 1024},
		{Timestamp: now, Value: 100 * 1024 * 1024},
	}
	mc.addTimeSeriesData("pod/nginx", "cpu", cpuPoints)
	mc.addTimeSeriesData("pod/nginx", "memory", memPoints)

	waste, err := an.CalculateWaste("default", "nginx")
	if err != nil {
		t.Fatalf("Failed to calculate waste: %v", err)
	}

	// Should have some waste due to 30% buffer
	if waste < 0 || waste > 100 {
		t.Errorf("Expected waste between 0 and 100, got %f", waste)
	}
}

// TestCalculateTrend tests trend calculation
func TestCalculateTrend(t *testing.T) {
	mc := newMockCollector()
	an := New(mc).(*analyzer)

	// Create linear increasing trend
	now := time.Now()
	points := []models.DataPoint{
		{Timestamp: now.Add(-4 * time.Hour), Value: 100},
		{Timestamp: now.Add(-3 * time.Hour), Value: 110},
		{Timestamp: now.Add(-2 * time.Hour), Value: 120},
		{Timestamp: now.Add(-1 * time.Hour), Value: 130},
		{Timestamp: now, Value: 140},
	}

	trend := an.calculateTrend(points)

	// Slope should be positive (increasing)
	if trend.Slope < 0 {
		t.Errorf("Expected positive slope for increasing trend, got %f", trend.Slope)
	}

	// R² should be high (close to 1) for perfect linear trend
	if trend.RSquared < 0.9 {
		t.Errorf("Expected high R² for linear trend, got %f", trend.RSquared)
	}
}

// TestCalculatePercentiles tests percentile calculation
func TestCalculatePercentiles(t *testing.T) {
	mc := newMockCollector()
	an := New(mc).(*analyzer)

	points := []models.DataPoint{
		{Value: 10}, {Value: 20}, {Value: 30}, {Value: 40}, {Value: 50},
		{Value: 60}, {Value: 70}, {Value: 80}, {Value: 90}, {Value: 100},
	}

	p50, p95, p99, err := an.calculatePercentiles(points)
	if err != nil {
		t.Fatalf("Failed to calculate percentiles: %v", err)
	}

	// P50 should be around 50-55
	if p50 < 45 || p50 > 60 {
		t.Errorf("Expected P50 around 50-55, got %f", p50)
	}

	// P95 should be around 95
	if p95 < 90 || p95 > 100 {
		t.Errorf("Expected P95 around 95, got %f", p95)
	}

	// P99 should be around 99
	if p99 < 95 || p99 > 100 {
		t.Errorf("Expected P99 around 99, got %f", p99)
	}
}

// TestRoundTo2Decimals tests decimal rounding
func TestRoundTo2Decimals(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{1.234, 1.23},
		{1.235, 1.24},
		{1.999, 2.00},
		{0.001, 0.00},
		{10.456, 10.46},
	}

	for _, tt := range tests {
		result := roundTo2Decimals(tt.input)
		if result != tt.expected {
			t.Errorf("roundTo2Decimals(%f) = %f, expected %f", tt.input, result, tt.expected)
		}
	}
}
