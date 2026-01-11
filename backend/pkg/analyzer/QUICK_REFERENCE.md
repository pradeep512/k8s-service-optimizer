# Analyzer Quick Reference

## Installation

```go
import "github.com/k8s-service-optimizer/backend/pkg/analyzer"
```

## Quick Start

```go
// Setup
k8sClient, _ := k8s.NewClient()
mc := collector.New(k8sClient)
mc.Start()
defer mc.Stop()
an := analyzer.New(mc)

// Wait for data
time.Sleep(30 * time.Second)
```

## Core Methods

### Traffic Analysis
```go
traffic, err := an.AnalyzeTrafficPatterns(namespace, service, duration)
// Returns: RequestRate, ErrorRate, P50/P95/P99 Latency, Anomalies
```

### Cost Calculation
```go
cost, err := an.CalculateServiceCost(namespace, service)
// Returns: CPUCost, MemoryCost, TotalCost, WastedCost, EfficiencyScore
```

### Anomaly Detection
```go
anomalies, err := an.DetectAnomalies(resource, metric, duration)
// Returns: []Anomaly (Type, Severity, Description, Value, Expected)
```

### Resource Prediction
```go
prediction, err := an.PredictResourceNeeds(namespace, service, hours)
// Returns: PredictedCPU, PredictedMemory, Confidence
```

### Waste Calculation
```go
waste, err := an.CalculateWaste(namespace, service)
// Returns: Over-provisioning percentage (0-100)
```

### Cost Trends
```go
trends, err := an.GetCostTrends(namespace, duration)
// Returns: []CostBreakdown over time
```

## Configuration

### Default Config
```go
an := analyzer.New(mc)  // Uses defaults
```

### Custom Config
```go
config := analyzer.Config{
    CPUCostPerVCPUHour:  0.03,   // CPU price
    MemoryCostPerGBHour: 0.004,  // Memory price
    AnomalyThreshold:    3.0,    // Z-score threshold
    SpikeThreshold:      2.0,    // Spike multiplier
    DropThreshold:       0.5,    // Drop multiplier
    MinDataPoints:       10,     // Min points for analysis
    TrendHistoryDays:    7,      // Trend history window
}
an := analyzer.NewWithConfig(mc, config)
```

## Data Structures

### TrafficAnalysis
```go
type TrafficAnalysis struct {
    Service     string
    Namespace   string
    RequestRate float64      // req/s
    ErrorRate   float64      // 0.0-1.0
    P50Latency  float64      // milliseconds
    P95Latency  float64
    P99Latency  float64
    Anomalies   []Anomaly
    Timestamp   time.Time
}
```

### CostBreakdown
```go
type CostBreakdown struct {
    Service         string
    Namespace       string
    CPUCost         float64  // $/month
    MemoryCost      float64  // $/month
    TotalCost       float64  // $/month
    WastedCost      float64  // $/month
    EfficiencyScore float64  // 0-100
    Timestamp       time.Time
}
```

### Anomaly
```go
type Anomaly struct {
    Type        string   // spike, drop, drift, oscillation
    Severity    string   // critical, high, medium, low
    Description string
    DetectedAt  time.Time
    Value       float64
    Expected    float64
}
```

### ResourcePrediction
```go
type ResourcePrediction struct {
    Service         string
    Namespace       string
    Hours           int
    PredictedCPU    int64    // millicores
    PredictedMemory int64    // bytes
    Confidence      float64  // 0.0-1.0
    Timestamp       time.Time
}
```

## Common Patterns

### Full Service Analysis
```go
func analyzeService(an analyzer.Analyzer, ns, svc string) {
    // Traffic
    traffic, _ := an.AnalyzeTrafficPatterns(ns, svc, 24*time.Hour)
    fmt.Printf("Requests: %.2f/s\n", traffic.RequestRate)

    // Cost
    cost, _ := an.CalculateServiceCost(ns, svc)
    fmt.Printf("Cost: $%.2f/month\n", cost.TotalCost)

    // Anomalies
    anomalies, _ := an.DetectAnomalies("pod/"+svc, "cpu", 24*time.Hour)
    fmt.Printf("Anomalies: %d\n", len(anomalies))

    // Prediction
    pred, _ := an.PredictResourceNeeds(ns, svc, 72)
    fmt.Printf("Predicted CPU: %dm\n", pred.PredictedCPU)

    // Waste
    waste, _ := an.CalculateWaste(ns, svc)
    fmt.Printf("Waste: %.1f%%\n", waste)
}
```

### Cost Report
```go
func costReport(an analyzer.Analyzer, ns string, services []string) {
    total := 0.0
    for _, svc := range services {
        cost, _ := an.CalculateServiceCost(ns, svc)
        total += cost.TotalCost
        fmt.Printf("%-20s $%8.2f\n", svc, cost.TotalCost)
    }
    fmt.Printf("%-20s $%8.2f\n", "TOTAL", total)
}
```

### Anomaly Monitor
```go
func monitorAnomalies(an analyzer.Analyzer, resource string) {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        anomalies, _ := an.DetectAnomalies(resource, "cpu", 10*time.Minute)
        for _, a := range anomalies {
            if a.Severity == "critical" || a.Severity == "high" {
                log.Printf("[%s] %s", a.Severity, a.Description)
            }
        }
    }
}
```

## Cost Formulas

### Monthly Cost
```
CPU:    (millicores / 1000) × $0.03 × 24 × 30
Memory: (bytes / 1024³) × $0.004 × 24 × 30
Total:  CPU + Memory
```

### Waste
```
Requested = P95_usage × 1.3
Waste = Requested - P95_usage
Waste% = (Waste / Requested) × 100
```

### Efficiency
```
Efficiency = 100 - Waste%
```

## Anomaly Types

| Type | Threshold | Description |
|------|-----------|-------------|
| spike | >2x | Sudden increase |
| drop | <0.5x | Sudden decrease |
| drift | >30% change | Gradual shift in baseline |
| oscillation | >50% direction changes | Rapid fluctuation |

## Severities

| Severity | Z-Score | Spike Ratio |
|----------|---------|-------------|
| critical | >5.0 | >5x |
| high | >4.0 | >3x |
| medium | >3.0 | >2x |
| low | >2.5 | >1.5x |

## Traffic Metrics (Simulated)

| Metric | Formula |
|--------|---------|
| Request Rate | CPU (m) / 10 |
| Error Rate | Restarts / Samples |
| P50 Latency | P50_CPU / 10 (ms) |
| P95 Latency | P95_CPU / 10 (ms) |
| P99 Latency | P99_CPU / 10 (ms) |

## Prediction

### Linear Regression
```
y = mx + b
Future = Current + (Slope × Hours)
```

### Confidence (R²)
- 0.9-1.0: Excellent (high confidence)
- 0.7-0.9: Good (medium confidence)
- <0.7: Poor (low confidence)

## Error Handling

```go
// Always check errors
traffic, err := an.AnalyzeTrafficPatterns(ns, svc, duration)
if err != nil {
    log.Printf("Warning: %v", err)
    // Use defaults or skip
}

// Check for sufficient data
if len(traffic.Anomalies) == 0 && traffic.RequestRate == 0 {
    log.Println("Insufficient data - wait longer")
}

// Validate confidence
if prediction.Confidence < 0.5 {
    log.Println("Low confidence - use with caution")
}
```

## Best Practices

1. **Wait for Data**: Allow 30+ seconds for initial collection
2. **Check Confidence**: Validate prediction confidence before decisions
3. **Handle Errors**: Not all services will have sufficient data
4. **Use Appropriate Windows**:
   - Traffic analysis: 1-24 hours
   - Cost calculation: 24 hours
   - Predictions: 7 days history
   - Anomalies: 1-24 hours
5. **Tune Thresholds**: Adjust based on environment
6. **Regular Analysis**: Run periodically for trends

## Common Issues

### No Data
- Wait longer for collection (30+ seconds)
- Check if collector is running
- Verify namespace/service names

### Low Confidence
- Need more historical data
- Increase TrendHistoryDays
- Use shorter prediction horizons

### Too Many Anomalies
- Increase AnomalyThreshold
- Adjust Spike/Drop thresholds
- Filter by severity

### Costs Seem Wrong
- Verify pricing configuration
- Check if using P95 vs average
- Consider replica count

## Examples

See [EXAMPLES.md](EXAMPLES.md) for detailed examples.

## Testing

```bash
# Run tests
go test ./pkg/analyzer/...

# With coverage
go test ./pkg/analyzer/... -cover

# Verbose
go test ./pkg/analyzer/... -v
```

## Documentation

- [README.md](README.md) - Full documentation
- [EXAMPLES.md](EXAMPLES.md) - Detailed examples
- [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md) - Implementation details
