# Analyzer Usage Examples

This document provides practical examples of using the Traffic & Cost Analyzer.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Traffic Analysis](#traffic-analysis)
3. [Cost Analysis](#cost-analysis)
4. [Anomaly Detection](#anomaly-detection)
5. [Resource Prediction](#resource-prediction)
6. [Complete Analysis Pipeline](#complete-analysis-pipeline)
7. [Custom Configuration](#custom-configuration)

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/k8s-service-optimizer/backend/internal/k8s"
    "github.com/k8s-service-optimizer/backend/pkg/collector"
    "github.com/k8s-service-optimizer/backend/pkg/analyzer"
)

func main() {
    // 1. Create K8s client
    k8sClient, err := k8s.NewClient()
    if err != nil {
        log.Fatalf("Failed to create k8s client: %v", err)
    }

    // 2. Create and start metrics collector
    mc := collector.New(k8sClient)
    mc.SetNamespaces([]string{"default", "k8s-optimizer"})

    if err := mc.Start(); err != nil {
        log.Fatalf("Failed to start collector: %v", err)
    }
    defer mc.Stop()

    // Wait for initial data collection
    fmt.Println("Collecting initial metrics...")
    time.Sleep(30 * time.Second)

    // 3. Create analyzer
    an := analyzer.New(mc)

    // 4. Analyze a service
    analyzeService(an, "default", "nginx")
}

func analyzeService(an analyzer.Analyzer, namespace, service string) {
    fmt.Printf("\n=== Analyzing Service: %s/%s ===\n\n", namespace, service)

    // Traffic analysis
    traffic, _ := an.AnalyzeTrafficPatterns(namespace, service, 24*time.Hour)
    fmt.Printf("Traffic Metrics:\n")
    fmt.Printf("  Request Rate: %.2f req/s\n", traffic.RequestRate)
    fmt.Printf("  Error Rate: %.2f%%\n", traffic.ErrorRate*100)
    fmt.Printf("  P95 Latency: %.2fms\n", traffic.P95Latency)

    // Cost analysis
    cost, _ := an.CalculateServiceCost(namespace, service)
    fmt.Printf("\nCost Metrics:\n")
    fmt.Printf("  Monthly Cost: $%.2f\n", cost.TotalCost)
    fmt.Printf("  Wasted: $%.2f (%.1f%% efficiency)\n",
        cost.WastedCost, cost.EfficiencyScore)

    // Anomaly detection
    anomalies, _ := an.DetectAnomalies(fmt.Sprintf("pod/%s", service), "cpu", 24*time.Hour)
    fmt.Printf("\nAnomalies: %d detected\n", len(anomalies))

    // Resource prediction
    prediction, _ := an.PredictResourceNeeds(namespace, service, 72)
    fmt.Printf("\nPrediction (72h):\n")
    fmt.Printf("  CPU: %dm (%.1f%% confidence)\n",
        prediction.PredictedCPU, prediction.Confidence*100)
}
```

## Traffic Analysis

### Example 1: Basic Traffic Analysis

```go
// Analyze traffic for the last 24 hours
traffic, err := analyzer.AnalyzeTrafficPatterns("default", "api-server", 24*time.Hour)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Service: %s\n", traffic.Service)
fmt.Printf("Request Rate: %.2f req/s\n", traffic.RequestRate)
fmt.Printf("Error Rate: %.2f%%\n", traffic.ErrorRate*100)
fmt.Printf("Latency Percentiles:\n")
fmt.Printf("  P50: %.2fms\n", traffic.P50Latency)
fmt.Printf("  P95: %.2fms\n", traffic.P95Latency)
fmt.Printf("  P99: %.2fms\n", traffic.P99Latency)
fmt.Printf("Anomalies: %d\n", len(traffic.Anomalies))
```

**Output:**
```
Service: api-server
Request Rate: 45.30 req/s
Error Rate: 0.05%
Latency Percentiles:
  P50: 12.50ms
  P95: 28.70ms
  P99: 45.20ms
Anomalies: 2
```

### Example 2: Comparing Multiple Time Windows

```go
func compareTrafficWindows(an analyzer.Analyzer, namespace, service string) {
    windows := []struct {
        duration time.Duration
        label    string
    }{
        {1 * time.Hour, "Last Hour"},
        {6 * time.Hour, "Last 6 Hours"},
        {24 * time.Hour, "Last Day"},
        {168 * time.Hour, "Last Week"},
    }

    for _, w := range windows {
        traffic, err := an.AnalyzeTrafficPatterns(namespace, service, w.duration)
        if err != nil {
            continue
        }

        fmt.Printf("%s:\n", w.label)
        fmt.Printf("  Requests: %.2f req/s\n", traffic.RequestRate)
        fmt.Printf("  Errors: %.2f%%\n", traffic.ErrorRate*100)
        fmt.Printf("  P95 Latency: %.2fms\n\n", traffic.P95Latency)
    }
}
```

### Example 3: Analyzing Multiple Services

```go
func analyzeAllServices(an analyzer.Analyzer, namespace string, services []string) {
    for _, service := range services {
        traffic, err := an.AnalyzeTrafficPatterns(namespace, service, 24*time.Hour)
        if err != nil {
            log.Printf("Failed to analyze %s: %v", service, err)
            continue
        }

        fmt.Printf("%-20s | Req: %6.2f/s | Err: %5.2f%% | P95: %6.2fms\n",
            service,
            traffic.RequestRate,
            traffic.ErrorRate*100,
            traffic.P95Latency,
        )
    }
}
```

## Cost Analysis

### Example 1: Service Cost Breakdown

```go
cost, err := analyzer.CalculateServiceCost("production", "payment-service")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("=== Cost Breakdown: %s ===\n", cost.Service)
fmt.Printf("CPU Cost:        $%8.2f/month\n", cost.CPUCost)
fmt.Printf("Memory Cost:     $%8.2f/month\n", cost.MemoryCost)
fmt.Printf("Total Cost:      $%8.2f/month\n", cost.TotalCost)
fmt.Printf("Wasted Cost:     $%8.2f/month\n", cost.WastedCost)
fmt.Printf("Efficiency:      %8.1f%%\n", cost.EfficiencyScore)
fmt.Printf("\nPotential Savings: $%.2f/month\n", cost.WastedCost)
```

**Output:**
```
=== Cost Breakdown: payment-service ===
CPU Cost:        $   21.60/month
Memory Cost:     $    8.64/month
Total Cost:      $   30.24/month
Wasted Cost:     $   12.10/month
Efficiency:          60.0%

Potential Savings: $12.10/month
```

### Example 2: Namespace Cost Summary

```go
func calculateNamespaceCost(an analyzer.Analyzer, namespace string, services []string) {
    totalCost := 0.0
    totalWaste := 0.0

    fmt.Printf("=== Namespace: %s ===\n\n", namespace)
    fmt.Printf("%-20s | %10s | %10s | %10s\n",
        "Service", "Total", "Wasted", "Efficiency")
    fmt.Println(strings.Repeat("-", 60))

    for _, service := range services {
        cost, err := an.CalculateServiceCost(namespace, service)
        if err != nil {
            continue
        }

        fmt.Printf("%-20s | $%8.2f | $%8.2f | %8.1f%%\n",
            service,
            cost.TotalCost,
            cost.WastedCost,
            cost.EfficiencyScore,
        )

        totalCost += cost.TotalCost
        totalWaste += cost.WastedCost
    }

    fmt.Println(strings.Repeat("-", 60))
    fmt.Printf("%-20s | $%8.2f | $%8.2f | %8.1f%%\n",
        "TOTAL",
        totalCost,
        totalWaste,
        (1-totalWaste/totalCost)*100,
    )
}
```

### Example 3: Cost Optimization Recommendations

```go
func generateCostRecommendations(an analyzer.Analyzer, namespace string, services []string) {
    recommendations := []struct {
        service    string
        wastedCost float64
        efficiency float64
    }{}

    for _, service := range services {
        cost, err := an.CalculateServiceCost(namespace, service)
        if err != nil {
            continue
        }

        // Only recommend if waste is significant
        if cost.WastedCost > 5.0 && cost.EfficiencyScore < 70.0 {
            recommendations = append(recommendations, struct {
                service    string
                wastedCost float64
                efficiency float64
            }{
                service:    service,
                wastedCost: cost.WastedCost,
                efficiency: cost.EfficiencyScore,
            })
        }
    }

    // Sort by wasted cost (descending)
    sort.Slice(recommendations, func(i, j int) bool {
        return recommendations[i].wastedCost > recommendations[j].wastedCost
    })

    fmt.Println("=== Cost Optimization Recommendations ===\n")
    for i, rec := range recommendations {
        fmt.Printf("%d. %s\n", i+1, rec.service)
        fmt.Printf("   Wasted: $%.2f/month (%.1f%% efficiency)\n",
            rec.wastedCost, rec.efficiency)
        fmt.Printf("   Action: Reduce resource requests to match P95 usage\n\n")
    }
}
```

## Anomaly Detection

### Example 1: Detecting CPU Anomalies

```go
anomalies, err := analyzer.DetectAnomalies("pod/web-server", "cpu", 24*time.Hour)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Detected %d anomalies:\n\n", len(anomalies))

for i, a := range anomalies {
    fmt.Printf("%d. [%s] %s\n", i+1, a.Severity, a.Type)
    fmt.Printf("   Time: %s\n", a.DetectedAt.Format(time.RFC3339))
    fmt.Printf("   Description: %s\n", a.Description)
    fmt.Printf("   Value: %.2f (expected: %.2f)\n\n", a.Value, a.Expected)
}
```

**Output:**
```
Detected 3 anomalies:

1. [high] spike
   Time: 2026-01-11T10:15:00Z
   Description: Sudden spike: value increased from 120.00 to 450.00 (3.8x)
   Value: 450.00 (expected: 120.00)

2. [medium] drop
   Time: 2026-01-11T11:30:00Z
   Description: Sudden drop: value decreased from 380.00 to 95.00 (0.3x)
   Value: 95.00 (expected: 380.00)

3. [low] oscillation
   Time: 2026-01-11T12:00:00Z
   Description: Rapid oscillation detected: 65.2% direction changes
   Value: 45.30 (expected: 150.00)
```

### Example 2: Real-time Anomaly Monitoring

```go
func monitorAnomalies(an analyzer.Analyzer, resource, metric string) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    fmt.Printf("Monitoring anomalies for %s/%s...\n\n", resource, metric)

    for range ticker.C {
        anomalies, err := an.DetectAnomalies(resource, metric, 10*time.Minute)
        if err != nil {
            log.Printf("Error: %v", err)
            continue
        }

        if len(anomalies) > 0 {
            for _, a := range anomalies {
                if a.Severity == "critical" || a.Severity == "high" {
                    fmt.Printf("[ALERT] %s anomaly: %s\n", a.Severity, a.Description)
                }
            }
        }
    }
}
```

### Example 3: Anomaly Summary Report

```go
func generateAnomalyReport(an analyzer.Analyzer, resources []string, duration time.Duration) {
    severityCounts := map[string]int{
        "critical": 0,
        "high":     0,
        "medium":   0,
        "low":      0,
    }

    typeCounts := map[string]int{
        "spike":       0,
        "drop":        0,
        "drift":       0,
        "oscillation": 0,
    }

    for _, resource := range resources {
        anomalies, err := an.DetectAnomalies(resource, "cpu", duration)
        if err != nil {
            continue
        }

        for _, a := range anomalies {
            severityCounts[a.Severity]++
            typeCounts[a.Type]++
        }
    }

    fmt.Println("=== Anomaly Summary Report ===\n")
    fmt.Println("By Severity:")
    fmt.Printf("  Critical: %d\n", severityCounts["critical"])
    fmt.Printf("  High:     %d\n", severityCounts["high"])
    fmt.Printf("  Medium:   %d\n", severityCounts["medium"])
    fmt.Printf("  Low:      %d\n\n", severityCounts["low"])

    fmt.Println("By Type:")
    fmt.Printf("  Spikes:       %d\n", typeCounts["spike"])
    fmt.Printf("  Drops:        %d\n", typeCounts["drop"])
    fmt.Printf("  Drifts:       %d\n", typeCounts["drift"])
    fmt.Printf("  Oscillations: %d\n", typeCounts["oscillation"])
}
```

## Resource Prediction

### Example 1: 72-Hour Prediction

```go
prediction, err := analyzer.PredictResourceNeeds("production", "api-gateway", 72)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("=== Resource Prediction: %s ===\n", prediction.Service)
fmt.Printf("Forecast: %d hours ahead\n\n", prediction.Hours)

fmt.Printf("Predicted CPU:     %dm\n", prediction.PredictedCPU)
fmt.Printf("Predicted Memory:  %dMB\n", prediction.PredictedMemory/(1024*1024))
fmt.Printf("Confidence:        %.1f%%\n\n", prediction.Confidence*100)

// Interpretation
if prediction.Confidence > 0.8 {
    fmt.Println("High confidence - prediction is reliable")
} else if prediction.Confidence > 0.5 {
    fmt.Println("Medium confidence - use with caution")
} else {
    fmt.Println("Low confidence - more data needed")
}
```

### Example 2: Capacity Planning

```go
func planCapacity(an analyzer.Analyzer, namespace, service string) {
    predictions := []int{24, 72, 168} // 1 day, 3 days, 1 week

    fmt.Printf("=== Capacity Planning: %s ===\n\n", service)

    for _, hours := range predictions {
        pred, err := an.PredictResourceNeeds(namespace, service, hours)
        if err != nil {
            continue
        }

        fmt.Printf("%s ahead:\n", formatDuration(hours))
        fmt.Printf("  CPU: %dm (confidence: %.1f%%)\n",
            pred.PredictedCPU, pred.Confidence*100)
        fmt.Printf("  Memory: %dMB (confidence: %.1f%%)\n\n",
            pred.PredictedMemory/(1024*1024), pred.Confidence*100)
    }
}

func formatDuration(hours int) string {
    if hours < 24 {
        return fmt.Sprintf("%d hours", hours)
    }
    days := hours / 24
    if days == 1 {
        return "1 day"
    }
    return fmt.Sprintf("%d days", days)
}
```

## Complete Analysis Pipeline

### Example: Full Service Analysis

```go
func fullServiceAnalysis(an analyzer.Analyzer, namespace, service string) {
    fmt.Printf("\n╔════════════════════════════════════════╗\n")
    fmt.Printf("║  Full Analysis: %s/%-15s  ║\n", namespace, service)
    fmt.Printf("╚════════════════════════════════════════╝\n\n")

    // 1. Traffic Analysis
    fmt.Println("📊 TRAFFIC ANALYSIS")
    traffic, err := an.AnalyzeTrafficPatterns(namespace, service, 24*time.Hour)
    if err == nil {
        fmt.Printf("  Request Rate: %.2f req/s\n", traffic.RequestRate)
        fmt.Printf("  Error Rate: %.2f%%\n", traffic.ErrorRate*100)
        fmt.Printf("  P95 Latency: %.2fms\n", traffic.P95Latency)
    }

    // 2. Cost Analysis
    fmt.Println("\n💰 COST ANALYSIS")
    cost, err := an.CalculateServiceCost(namespace, service)
    if err == nil {
        fmt.Printf("  Monthly Cost: $%.2f\n", cost.TotalCost)
        fmt.Printf("  Wasted: $%.2f/month\n", cost.WastedCost)
        fmt.Printf("  Efficiency: %.1f%%\n", cost.EfficiencyScore)
    }

    // 3. Waste Analysis
    fmt.Println("\n♻️  WASTE ANALYSIS")
    waste, err := an.CalculateWaste(namespace, service)
    if err == nil {
        fmt.Printf("  Over-provisioning: %.1f%%\n", waste)
        if waste > 30 {
            fmt.Println("  ⚠️  High waste - consider reducing resources")
        }
    }

    // 4. Anomaly Detection
    fmt.Println("\n🔍 ANOMALY DETECTION")
    anomalies, err := an.DetectAnomalies(fmt.Sprintf("pod/%s", service), "cpu", 24*time.Hour)
    if err == nil {
        fmt.Printf("  Total Anomalies: %d\n", len(anomalies))

        critical := 0
        high := 0
        for _, a := range anomalies {
            if a.Severity == "critical" {
                critical++
            } else if a.Severity == "high" {
                high++
            }
        }

        if critical > 0 {
            fmt.Printf("  ❗ Critical: %d\n", critical)
        }
        if high > 0 {
            fmt.Printf("  ⚠️  High: %d\n", high)
        }
    }

    // 5. Resource Prediction
    fmt.Println("\n🔮 RESOURCE PREDICTION (72h)")
    prediction, err := an.PredictResourceNeeds(namespace, service, 72)
    if err == nil {
        fmt.Printf("  Predicted CPU: %dm\n", prediction.PredictedCPU)
        fmt.Printf("  Predicted Memory: %dMB\n", prediction.PredictedMemory/(1024*1024))
        fmt.Printf("  Confidence: %.1f%%\n", prediction.Confidence*100)
    }

    fmt.Println("\n" + strings.Repeat("─", 44))
}
```

## Custom Configuration

### Example: High-Sensitivity Anomaly Detection

```go
// More sensitive to anomalies
config := analyzer.Config{
    CPUCostPerVCPUHour:  0.03,
    MemoryCostPerGBHour: 0.004,
    AnomalyThreshold:    2.0,   // Lower threshold (more sensitive)
    SpikeThreshold:      1.5,   // Detect smaller spikes
    DropThreshold:       0.7,   // Detect smaller drops
    MinDataPoints:       5,     // Require fewer points
    TrendHistoryDays:    7,
}

an := analyzer.NewWithConfig(mc, config)
```

### Example: Cost Analysis with Custom Pricing

```go
// AWS-like pricing
config := analyzer.Config{
    CPUCostPerVCPUHour:  0.0416,  // t3.medium pricing
    MemoryCostPerGBHour: 0.0052,
    AnomalyThreshold:    3.0,
    SpikeThreshold:      2.0,
    DropThreshold:       0.5,
    MinDataPoints:       10,
    TrendHistoryDays:    14,      // More history for better trends
}

an := analyzer.NewWithConfig(mc, config)
```

## Error Handling

### Example: Robust Analysis with Fallbacks

```go
func robustAnalysis(an analyzer.Analyzer, namespace, service string) {
    // Try traffic analysis
    traffic, err := an.AnalyzeTrafficPatterns(namespace, service, 24*time.Hour)
    if err != nil {
        log.Printf("Warning: Could not analyze traffic for %s: %v", service, err)
        // Continue with other analyses
    } else {
        displayTrafficMetrics(traffic)
    }

    // Try cost analysis
    cost, err := an.CalculateServiceCost(namespace, service)
    if err != nil {
        log.Printf("Warning: Could not calculate cost for %s: %v", service, err)
    } else {
        displayCostMetrics(cost)
    }

    // Try prediction with fallback durations
    for _, hours := range []int{72, 48, 24} {
        prediction, err := an.PredictResourceNeeds(namespace, service, hours)
        if err == nil && prediction.Confidence > 0.3 {
            displayPrediction(prediction)
            break
        }
    }
}
```

## Best Practices

1. **Wait for Data Collection**: Allow time for metrics to be collected before analysis
2. **Handle Errors Gracefully**: Not all services will have sufficient data
3. **Use Appropriate Time Windows**: Longer windows for trends, shorter for real-time analysis
4. **Validate Confidence**: Check prediction confidence before making decisions
5. **Regular Analysis**: Run analysis periodically to track changes over time
6. **Adjust Configuration**: Tune thresholds based on your environment

## See Also

- [README.md](README.md) - Complete documentation
- [analyzer_test.go](analyzer_test.go) - Unit tests
- [example_test.go](example_test.go) - More examples
