# Metrics Collector

The metrics collector package provides functionality for collecting and storing Kubernetes metrics in an in-memory time-series database.

## Features

- **Automatic Collection**: Collects pod, node, and HPA metrics every 15 seconds (configurable)
- **In-Memory Storage**: Stores up to 24 hours of metrics data (configurable)
- **Thread-Safe**: Uses sync.RWMutex for concurrent access
- **Time-Series Queries**: Query metrics for any time duration
- **Percentile Calculations**: Calculate P50, P95, P99 for resource usage
- **Automatic Cleanup**: Removes data older than retention period

## Usage

### Basic Usage

```go
import (
    "github.com/k8s-service-optimizer/backend/internal/k8s"
    "github.com/k8s-service-optimizer/backend/pkg/collector"
)

// Create K8s client
k8sClient, _ := k8s.NewClient()

// Create metrics collector
mc := collector.New(k8sClient)

// Start collection
mc.Start()
defer mc.Stop()

// Query metrics
pods, _ := mc.CollectPodMetrics("default")
nodes, _ := mc.CollectNodeMetrics()
hpas, _ := mc.CollectHPAMetrics("default")
```

### Custom Configuration

```go
config := collector.Config{
    CollectionInterval: 10 * time.Second, // Collect every 10 seconds
    RetentionPeriod:    12 * time.Hour,   // Keep 12 hours of data
    CleanupInterval:    30 * time.Minute, // Clean up every 30 minutes
}

mc := collector.NewWithConfig(k8sClient, config)
```

### Time-Series Queries

```go
// Get CPU metrics for the last 5 minutes
ts, _ := mc.GetTimeSeriesData("pod/echo-demo-xyz", "cpu", 5*time.Minute)

for _, point := range ts.Points {
    fmt.Printf("Time: %v, Value: %.2f\n", point.Timestamp, point.Value)
}
```

### Percentile Calculations

```go
// Calculate percentiles for memory usage over the last hour
p50, p95, p99, _ := mc.GetResourcePercentiles("pod/echo-demo-xyz", "memory", 1*time.Hour)

fmt.Printf("Memory P50: %.2f, P95: %.2f, P99: %.2f\n", p50, p95, p99)
```

### Monitor Multiple Namespaces

```go
mc := collector.New(k8sClient)

// Set namespaces to monitor
mc.SetNamespaces([]string{"default", "production", "staging"})

mc.Start()
```

## Architecture

### Components

1. **Collector** (`collector.go`): Main orchestrator that manages collection lifecycle
2. **K8s Collector** (`k8s_collector.go`): Handles actual metric collection from Kubernetes API
3. **Metrics Store** (`metrics_store.go`): Thread-safe in-memory time-series storage
4. **Types** (`types.go`): Interface definitions and data structures

### Data Flow

```
┌─────────────┐
│  Collector  │ (Main orchestrator)
└──────┬──────┘
       │
       ├─── Collection Loop (every 15s)
       │    ├─> K8s Collector → Pod Metrics
       │    ├─> K8s Collector → Node Metrics
       │    └─> K8s Collector → HPA Metrics
       │
       ├─── Metrics Store (Thread-safe storage)
       │    └─> Time-series data with 24h retention
       │
       └─── Cleanup Loop (every 1h)
            └─> Remove data older than 24h
```

## Resource Naming Convention

Resources are identified using the following naming pattern:

- Pods: `pod/<pod-name>`
- Nodes: `node/<node-name>`
- HPAs: `hpa/<hpa-name>`

Metrics include:

- For Pods/Nodes: `cpu`, `memory`
- For HPAs: `current_replicas`, `desired_replicas`, `target_cpu`, `current_cpu`

## Thread Safety

All public methods are thread-safe and can be called concurrently. The metrics store uses `sync.RWMutex` to ensure safe concurrent access:

- **Read operations**: Use read locks (allow concurrent reads)
- **Write operations**: Use write locks (exclusive access)

## Performance Considerations

### Memory Usage

With default settings (15s interval, 24h retention):
- **Data points per metric**: ~5,760 points (24h * 60min * 60s / 15s)
- **Storage per point**: ~24 bytes (timestamp + float64)
- **Per metric overhead**: ~138 KB

For a cluster with:
- 100 pods × 2 metrics = 200 metric series
- 10 nodes × 2 metrics = 20 metric series
- 20 HPAs × 4 metrics = 80 metric series

**Total metrics**: 300 series
**Estimated memory**: ~42 MB

### CPU Usage

- Collection runs every 15 seconds
- Cleanup runs every 1 hour
- Kubernetes API calls are minimal (3 per namespace + 1 for nodes)

## Testing

Run tests with:

```bash
go test ./pkg/collector/
```

Run tests with verbose output:

```bash
go test -v ./pkg/collector/
```

## Integration Points

### Used By

- API Server (queries metrics for dashboards)
- Optimizer Engine (analyzes metrics for recommendations)
- Analyzer Service (calculates resource efficiency)

### Dependencies

- `backend/internal/k8s/client.go` - Kubernetes client wrapper
- `backend/internal/models/types.go` - Data models

## Configuration

Default configuration values:

| Parameter | Default | Description |
|-----------|---------|-------------|
| CollectionInterval | 15s | How often to collect metrics |
| RetentionPeriod | 24h | How long to keep metrics |
| CleanupInterval | 1h | How often to cleanup old data |

## Error Handling

All methods return errors following Go conventions:

- **Collection errors**: Logged but don't stop the collector
- **Storage errors**: Returned to caller for handling
- **Query errors**: Returned when data not found or invalid parameters

## Future Enhancements

Potential improvements (currently out of scope):

- [ ] Persistent storage backend
- [ ] Metric aggregation across namespaces
- [ ] Custom metric collection
- [ ] Prometheus integration
- [ ] Metric export functionality
- [ ] Configurable aggregation functions
