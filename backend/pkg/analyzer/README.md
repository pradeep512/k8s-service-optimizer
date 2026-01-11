# Traffic & Cost Analyzer

The Traffic & Cost Analyzer component analyzes traffic patterns, detects anomalies, calculates service costs, and predicts future resource needs for Kubernetes services.

## Features

### 1. Traffic Pattern Analysis
- Estimates request rate from CPU usage patterns
- Calculates error rates from pod restart patterns
- Estimates latency percentiles (P50, P95, P99)
- Detects traffic patterns: steady, spiking, periodic, declining, increasing

### 2. Cost Calculation
- Calculates service costs based on resource usage
- Standard cloud pricing:
  - CPU: $0.03 per vCPU-hour (1000 millicores = 1 vCPU)
  - Memory: $0.004 per GB-hour (1024 MB = 1 GB)
- Provides monthly cost projections
- Calculates wasted resources and costs
- Generates efficiency scores (0-100)

### 3. Anomaly Detection
- **Z-Score Method**: Detects values >3 standard deviations from mean
- **Spike Detection**: Sudden increases (>2x normal)
- **Drop Detection**: Sudden decreases (<0.5x normal)
- **Drift Detection**: Gradual sustained changes
- **Oscillation Detection**: Rapid fluctuations
- Severity levels: critical, high, medium, low

### 4. Trend Analysis & Prediction
- Uses simple linear regression for trend analysis
- Projects resource needs forward (configurable hours)
- Calculates confidence based on R² value
- Provides prediction range (best/worst case)
- Supports seasonal pattern detection

### 5. Waste Calculation
- Identifies over-provisioned resources
- Calculates waste as: Requested - P95 Usage
- Reports waste percentage
- Estimates wasted cost

## Installation

```go
import "github.com/k8s-service-optimizer/backend/pkg/analyzer"
```

## Usage

### Basic Setup

```go
package main

import (
    "fmt"
    "time"

    "github.com/k8s-service-optimizer/backend/internal/k8s"
    "github.com/k8s-service-optimizer/backend/pkg/collector"
    "github.com/k8s-service-optimizer/backend/pkg/analyzer"
)

func main() {
    // Create K8s client
    k8sClient, _ := k8s.NewClient()

    // Create metrics collector
    mc := collector.New(k8sClient)
    mc.Start()
    defer mc.Stop()

    // Create analyzer with default config
    an := analyzer.New(mc)

    // Use analyzer...
}
```

### Analyze Traffic Patterns

```go
// Analyze traffic for a service over the last 24 hours
traffic, err := an.AnalyzeTrafficPatterns("default", "nginx", 24*time.Hour)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Request Rate: %.2f req/s\n", traffic.RequestRate)
fmt.Printf("Error Rate: %.2f%%\n", traffic.ErrorRate*100)
fmt.Printf("P50 Latency: %.2fms\n", traffic.P50Latency)
fmt.Printf("P95 Latency: %.2fms\n", traffic.P95Latency)
fmt.Printf("P99 Latency: %.2fms\n", traffic.P99Latency)
fmt.Printf("Anomalies detected: %d\n", len(traffic.Anomalies))
```

### Calculate Service Cost

```go
// Calculate monthly cost for a service
cost, err := an.CalculateServiceCost("default", "nginx")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("CPU Cost: $%.2f/month\n", cost.CPUCost)
fmt.Printf("Memory Cost: $%.2f/month\n", cost.MemoryCost)
fmt.Printf("Total Cost: $%.2f/month\n", cost.TotalCost)
fmt.Printf("Wasted Cost: $%.2f/month\n", cost.WastedCost)
fmt.Printf("Efficiency Score: %.1f%%\n", cost.EfficiencyScore)
```

### Detect Anomalies

```go
// Detect anomalies in CPU metrics
anomalies, err := an.DetectAnomalies("pod/nginx-xxx", "cpu", 24*time.Hour)
if err != nil {
    log.Fatal(err)
}

for _, a := range anomalies {
    fmt.Printf("[%s] %s: %s\n", a.Severity, a.Type, a.Description)
    fmt.Printf("  Value: %.2f, Expected: %.2f\n", a.Value, a.Expected)
}
```

### Predict Resource Needs

```go
// Predict resource needs for next 72 hours
prediction, err := an.PredictResourceNeeds("default", "nginx", 72)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Predicted CPU (72h): %dm\n", prediction.PredictedCPU)
fmt.Printf("Predicted Memory (72h): %dMB\n", prediction.PredictedMemory/(1024*1024))
fmt.Printf("Confidence: %.1f%%\n", prediction.Confidence*100)
```

### Calculate Waste

```go
// Calculate wasted resources
waste, err := an.CalculateWaste("default", "nginx")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Over-provisioning: %.1f%%\n", waste)
```

## Configuration

### Custom Configuration

```go
config := analyzer.Config{
    CPUCostPerVCPUHour:  0.04,   // $0.04 per vCPU-hour
    MemoryCostPerGBHour: 0.005,  // $0.005 per GB-hour
    AnomalyThreshold:    2.5,    // 2.5 standard deviations
    SpikeThreshold:      1.8,    // 1.8x multiplier for spikes
    DropThreshold:       0.6,    // 0.6x multiplier for drops
    MinDataPoints:       5,      // Minimum 5 data points
    TrendHistoryDays:    14,     // Use 14 days of history
}

an := analyzer.NewWithConfig(mc, config)
```

### Default Configuration

| Parameter | Default | Description |
|-----------|---------|-------------|
| `CPUCostPerVCPUHour` | 0.03 | Cost per vCPU-hour in dollars |
| `MemoryCostPerGBHour` | 0.004 | Cost per GB-hour in dollars |
| `AnomalyThreshold` | 3.0 | Z-score threshold for anomaly detection |
| `SpikeThreshold` | 2.0 | Multiplier for spike detection |
| `DropThreshold` | 0.5 | Multiplier for drop detection |
| `MinDataPoints` | 10 | Minimum data points for analysis |
| `TrendHistoryDays` | 7 | Days of history for trend analysis |

## Cost Calculation Details

### Monthly Cost Formula

**CPU Cost:**
```
monthly_cpu_cost = (cpu_millicores / 1000) × CPUCostPerVCPUHour × 24 × 30
```

**Memory Cost:**
```
monthly_mem_cost = (memory_bytes / (1024³)) × MemoryCostPerGBHour × 24 × 30
```

**Total Cost:**
```
total_cost = monthly_cpu_cost + monthly_mem_cost
```

### Waste Calculation

```
requested_resources = P95_usage × 1.3  (30% buffer)
waste = requested_resources - P95_usage
waste_percentage = (waste / requested_resources) × 100
wasted_cost = waste × unit_price × hours
```

### Efficiency Score

```
efficiency_score = 100 - waste_percentage
```

## Anomaly Detection Algorithms

### 1. Z-Score Method
Detects values that are more than N standard deviations from the mean:
```
z_score = |value - mean| / std_dev
if z_score > threshold: anomaly detected
```

### 2. Spike Detection
Detects sudden increases:
```
ratio = current_value / previous_value
if ratio > spike_threshold: spike detected
```

### 3. Drop Detection
Detects sudden decreases:
```
ratio = current_value / previous_value
if ratio < drop_threshold: drop detected
```

### 4. Drift Detection
Detects gradual sustained changes:
```
first_half_mean = mean(first 50% of data)
second_half_mean = mean(last 50% of data)
change = |second_half_mean - first_half_mean| / first_half_mean
if change > 0.3: drift detected
```

### 5. Oscillation Detection
Detects rapid fluctuations:
```
direction_changes = count of sign changes in derivatives
change_rate = direction_changes / total_points
if change_rate > 0.5 && std_dev > mean × 0.2: oscillation detected
```

## Trend Analysis & Prediction

### Linear Regression

The analyzer uses simple linear regression to calculate trends:

```
y = mx + b

where:
  m (slope) = (n×ΣXY - ΣX×ΣY) / (n×ΣX² - (ΣX)²)
  b (intercept) = (ΣY - m×ΣX) / n
```

### R² (Coefficient of Determination)

Confidence is based on R²:

```
R² = 1 - (SS_residual / SS_total)

where:
  SS_total = Σ(y - mean_y)²
  SS_residual = Σ(y - predicted_y)²
```

R² ranges from 0 to 1:
- 0.9-1.0: Excellent fit (high confidence)
- 0.7-0.9: Good fit (medium confidence)
- <0.7: Poor fit (low confidence)

### Prediction Formula

```
future_value = current_value + (slope × hours_ahead)
```

## Traffic Simulation

Since kind clusters don't have service mesh, traffic metrics are simulated from pod metrics:

| Metric | Simulation Method |
|--------|------------------|
| Request Rate | CPU usage / 10 |
| Error Rate | Pod restart count / total samples |
| Latency P50/P95/P99 | CPU P50/P95/P99 / 10 |

## Example Cost Breakdown

For a service with:
- 3 replicas
- 200m CPU request per replica
- 256MB memory request per replica

**Calculations:**
```
Total CPU: 600m (3 × 200m)
Total Memory: 768MB (3 × 256MB)

CPU Cost: (600/1000) × $0.03 × 24 × 30 = $12.96/month
Memory Cost: (768/1024) × $0.004 × 24 × 30 = $2.21/month
Total Cost: $15.17/month

If P95 usage is 400m CPU and 512MB:
Waste: 200m CPU + 256MB
Wasted Cost: ~$5.06/month
Efficiency Score: 66.6%
```

## Error Handling

The analyzer handles edge cases gracefully:
- Returns zero values for services with no data
- Requires minimum data points for meaningful analysis
- Validates input parameters
- Provides detailed error messages

## Performance Considerations

- In-memory calculations (no database queries)
- Time complexity: O(n) for most operations
- Efficient percentile calculation with sorting
- Minimal memory footprint

## Testing

Run tests with:
```bash
go test -v ./pkg/analyzer/...
```

See `analyzer_test.go` for comprehensive unit tests.

## Integration Points

**Uses:**
- Metrics Collector (`pkg/collector`) - for historical data
- K8s Client (`internal/k8s`) - for service information
- Models (`internal/models`) - for data structures

**Used By:**
- API Server - for cost/traffic endpoints
- Dashboard - for visualizations
- Optimizer - for optimization recommendations

## Files

- `analyzer.go` - Main analyzer implementation
- `types.go` - Interface definitions and types
- `traffic_analyzer.go` - Traffic pattern analysis
- `cost_analyzer.go` - Cost calculation
- `anomaly_detector.go` - Anomaly detection algorithms
- `trends.go` - Trend analysis and prediction
- `analyzer_test.go` - Unit tests
- `example_test.go` - Usage examples

## License

Part of the k8s-service-optimizer project.
