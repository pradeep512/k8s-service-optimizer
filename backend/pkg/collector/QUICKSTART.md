# Metrics Collector - Quick Start Guide

## 5-Minute Quick Start

### 1. Import the Package

```go
import (
    "github.com/k8s-service-optimizer/backend/internal/k8s"
    "github.com/k8s-service-optimizer/backend/pkg/collector"
)
```

### 2. Create and Start Collector

```go
// Create Kubernetes client
k8sClient, err := k8s.NewClient()
if err != nil {
    log.Fatalf("Failed to create k8s client: %v", err)
}

// Create collector with default config
mc := collector.New(k8sClient)

// Start collecting metrics
mc.Start()
defer mc.Stop()
```

### 3. Query Metrics

```go
// Get current pod metrics
pods, err := mc.CollectPodMetrics("default")
for _, pod := range pods {
    fmt.Printf("Pod: %s, CPU: %d m, Memory: %d MB\n",
        pod.Name, pod.CPU, pod.Memory/(1024*1024))
}

// Get current node metrics
nodes, err := mc.CollectNodeMetrics()
for _, node := range nodes {
    fmt.Printf("Node: %s, CPU: %d m, Memory: %d MB\n",
        node.Name, node.CPU, node.Memory/(1024*1024))
}

// Get HPA metrics
hpas, err := mc.CollectHPAMetrics("default")
for _, hpa := range hpas {
    fmt.Printf("HPA: %s, Replicas: %d/%d\n",
        hpa.Name, hpa.CurrentReplicas, hpa.DesiredReplicas)
}
```

### 4. Time-Series Analysis

```go
// Get CPU metrics for the last 5 minutes
ts, err := mc.GetTimeSeriesData("pod/my-app-xyz", "cpu", 5*time.Minute)
if err == nil {
    fmt.Printf("Retrieved %d data points\n", len(ts.Points))
    for _, point := range ts.Points {
        fmt.Printf("  %s: %.2f\n", point.Timestamp, point.Value)
    }
}

// Calculate percentiles for the last hour
p50, p95, p99, err := mc.GetResourcePercentiles("pod/my-app-xyz", "cpu", 1*time.Hour)
if err == nil {
    fmt.Printf("CPU Percentiles - P50: %.2f, P95: %.2f, P99: %.2f\n", p50, p95, p99)
}
```

## Common Patterns

### Pattern 1: Custom Configuration

```go
config := collector.Config{
    CollectionInterval: 10 * time.Second,  // More frequent
    RetentionPeriod:    12 * time.Hour,    // Less retention
    CleanupInterval:    30 * time.Minute,  // More frequent cleanup
}

mc := collector.NewWithConfig(k8sClient, config)
```

### Pattern 2: Monitor Multiple Namespaces

```go
mc := collector.New(k8sClient)
mc.SetNamespaces([]string{"default", "production", "staging"})
mc.Start()
```

### Pattern 3: Periodic Monitoring

```go
ticker := time.NewTicker(1 * time.Minute)
defer ticker.Stop()

for {
    select {
    case <-ticker.C:
        pods, _ := mc.CollectPodMetrics("default")
        fmt.Printf("Monitoring %d pods\n", len(pods))

        // Analyze metrics
        for _, pod := range pods {
            resource := fmt.Sprintf("pod/%s", pod.Name)
            p95, _, _, _ := mc.GetResourcePercentiles(resource, "cpu", 5*time.Minute)

            if p95 > 800 { // > 80% of 1 core
                fmt.Printf("⚠️  High CPU on %s: %.2f m\n", pod.Name, p95)
            }
        }
    }
}
```

### Pattern 4: Store Size Monitoring

```go
// Check store size periodically
size := mc.GetStoreSize()
keys := mc.GetStoredMetricKeys()

fmt.Printf("Store contains %d data points across %d metrics\n", size, len(keys))
```

## Resource Naming

Resources follow this naming convention:

| Resource Type | Name Format | Examples |
|--------------|-------------|----------|
| Pod | `pod/<name>` | `pod/echo-demo-abc123` |
| Node | `node/<name>` | `node/worker-1` |
| HPA | `hpa/<name>` | `hpa/my-app-hpa` |

## Available Metrics

| Resource | Metrics | Unit |
|----------|---------|------|
| Pods | `cpu`, `memory` | millicores, bytes |
| Nodes | `cpu`, `memory` | millicores, bytes |
| HPAs | `current_replicas`, `desired_replicas`, `target_cpu`, `current_cpu` | count, count, %, % |

## Common Queries

### Get CPU trend for a pod
```go
ts, _ := mc.GetTimeSeriesData("pod/my-app-xyz", "cpu", 15*time.Minute)
```

### Get memory percentiles for a node
```go
p50, p95, p99, _ := mc.GetResourcePercentiles("node/worker-1", "memory", 1*time.Hour)
```

### Get HPA scaling behavior
```go
ts, _ := mc.GetTimeSeriesData("hpa/my-app-hpa", "current_replicas", 30*time.Minute)
```

## Error Handling

```go
// Collection methods return errors
pods, err := mc.CollectPodMetrics("default")
if err != nil {
    log.Printf("Failed to collect pod metrics: %v", err)
    // Collector continues running, this is not fatal
}

// Query methods return errors
ts, err := mc.GetTimeSeriesData("pod/nonexistent", "cpu", 5*time.Minute)
if err != nil {
    // No data found - ts.Points will be empty slice
}

// Percentile methods return errors when no data exists
p50, p95, p99, err := mc.GetResourcePercentiles("pod/new-pod", "cpu", 1*time.Hour)
if err != nil {
    // Not enough data yet
}
```

## Lifecycle Management

```go
// Start the collector
if err := mc.Start(); err != nil {
    // Collector is already running
}

// Check if running
if mc.IsRunning() {
    // Do something
}

// Stop the collector (blocks until cleanup complete)
mc.Stop()

// Stop is idempotent - safe to call multiple times
mc.Stop()
```

## Demo Application

A complete working example is available:

```bash
# Build the demo
go build -o collector-demo ./cmd/collector-demo/

# Run it (requires a Kubernetes cluster)
./collector-demo
```

The demo shows:
- Real-time metrics collection
- Periodic reporting
- Time-series analysis
- Percentile calculations
- Graceful shutdown

## Testing

```bash
# Run tests
go test ./pkg/collector/

# Run with verbose output
go test -v ./pkg/collector/

# Run with coverage
go test -cover ./pkg/collector/

# Run specific test
go test -v ./pkg/collector/ -run TestMetricsStore
```

## Performance Tips

1. **Adjust collection interval** based on your needs:
   - High precision: 5-10 seconds
   - Normal: 15 seconds (default)
   - Low overhead: 30-60 seconds

2. **Adjust retention period** based on memory:
   - Short-term analysis: 6-12 hours
   - Standard: 24 hours (default)
   - Extended: 48 hours (requires more memory)

3. **Monitor namespaces selectively**:
   - Only monitor namespaces you need
   - Use `SetNamespaces()` to limit scope

4. **Use appropriate query durations**:
   - Recent trends: 5-15 minutes
   - Hourly patterns: 1-6 hours
   - Daily patterns: 12-24 hours

## Troubleshooting

### Issue: No data returned
**Solution**: Wait at least one collection interval (15s) after starting

### Issue: Percentiles return error
**Solution**: Ensure enough data points exist (needs at least 2 collection cycles)

### Issue: High memory usage
**Solution**: Reduce retention period or collection interval

### Issue: Missing metrics for some pods
**Solution**: Ensure metrics-server is running in your cluster

## Next Steps

1. Read the full [README.md](README.md) for detailed documentation
2. Check [IMPLEMENTATION.md](IMPLEMENTATION.md) for architecture details
3. Review [example_test.go](example_test.go) for more usage patterns
4. Run the demo application to see it in action

## Support

For issues or questions:
1. Check the test files for usage examples
2. Review the inline documentation in the code
3. See the README for architecture details
