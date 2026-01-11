# Resource Optimizer Engine

The Resource Optimizer Engine analyzes Kubernetes deployment resource usage patterns and generates actionable optimization recommendations for right-sizing resources, optimizing HPA configuration, and improving efficiency.

## Features

- **Resource Analysis**: Analyzes CPU and memory usage patterns using P50, P95, P99 percentiles
- **Right-Sizing Recommendations**: Generates recommendations to optimize resource requests and limits
- **HPA Optimization**: Analyzes and optimizes Horizontal Pod Autoscaler configurations
- **Efficiency Scoring**: Calculates efficiency scores (0-100) based on utilization, stability, and cost
- **Cost Estimation**: Estimates monthly cost savings from applying recommendations
- **Priority Classification**: Classifies recommendations as high, medium, or low priority

## Installation

```go
import "github.com/k8s-service-optimizer/backend/pkg/optimizer"
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/k8s-service-optimizer/backend/internal/k8s"
    "github.com/k8s-service-optimizer/backend/pkg/collector"
    "github.com/k8s-service-optimizer/backend/pkg/optimizer"
)

func main() {
    // Create dependencies
    k8sClient, _ := k8s.NewClient()
    mc := collector.New(k8sClient)
    mc.Start()
    defer mc.Stop()

    // Create optimizer
    opt := optimizer.New(k8sClient, mc)

    // Analyze a deployment
    analysis, _ := opt.AnalyzeDeployment("default", "my-app")
    fmt.Printf("Health Score: %.2f\n", analysis.HealthScore)

    // Generate recommendations
    recs, _ := opt.GenerateRecommendations(analysis)
    for _, rec := range recs {
        fmt.Printf("%s: %s (Savings: $%.2f/mo)\n",
            rec.Priority, rec.Description, rec.EstimatedSavings)
    }
}
```

## Core Components

### 1. Optimizer Interface

The main interface providing optimization capabilities:

```go
type Optimizer interface {
    AnalyzeDeployment(namespace, name string) (*models.Analysis, error)
    GenerateRecommendations(analysis *models.Analysis) ([]models.Recommendation, error)
    CalculateEfficiencyScore(namespace, name string) (float64, error)
    EstimateCostSavings(recommendation *models.Recommendation) (float64, error)
    ApplyRecommendation(recommendationID string) error
    GetAllRecommendations() ([]models.Recommendation, error)
}
```

### 2. Resource Analyzer

Analyzes resource usage patterns:
- Collects deployment metrics from the metrics collector
- Calculates percentiles (P50, P95, P99) for CPU and memory
- Identifies over-provisioning and under-provisioning
- Analyzes HPA scaling patterns
- Calculates variance and stability metrics

### 3. Recommendation Generator

Generates optimization recommendations:
- **Resource Recommendations**: Right-sizing CPU and memory
- **HPA Recommendations**: Optimizing min/max replicas and target CPU
- **Scaling Recommendations**: Adjusting replica count

### 4. Scorer

Calculates efficiency scores:
- **Resource Utilization Score (50%)**: Optimal at 70-90% utilization
- **Stability Score (30%)**: Based on restarts, variance, and scaling frequency
- **Cost Efficiency Score (20%)**: Based on resource waste

## Configuration

Customize the optimizer behavior:

```go
config := optimizer.DefaultConfig()
config.AnalysisDuration = 14 * 24 * time.Hour  // 14 days instead of 7
config.CPUOverProvisionedThreshold = 0.6       // 60% threshold
config.OptimalUtilizationMin = 0.65            // 65% optimal min
config.OptimalUtilizationMax = 0.85            // 85% optimal max
config.CPUCostPerVCPUHour = 0.04               // Custom pricing

opt := optimizer.NewWithConfig(k8sClient, mc, config)
```

### Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `AnalysisDuration` | 7 days | Time window for metrics analysis |
| `CPUOverProvisionedThreshold` | 0.5 (50%) | Threshold for detecting over-provisioned CPU |
| `MemoryOverProvisionedThreshold` | 0.5 (50%) | Threshold for detecting over-provisioned memory |
| `CPUUnderProvisionedThreshold` | 0.8 (80%) | Threshold for detecting under-provisioned CPU |
| `MemoryUnderProvisionedThreshold` | 0.8 (80%) | Threshold for detecting under-provisioned memory |
| `OverProvisionedBuffer` | 1.2 (20% buffer) | Buffer for over-provisioned resources |
| `UnderProvisionedBuffer` | 1.5 (50% buffer) | Buffer for under-provisioned resources |
| `CPUCostPerVCPUHour` | $0.03 | Cost per vCPU-hour for estimation |
| `MemoryCostPerGBHour` | $0.004 | Cost per GB-hour for estimation |
| `MinimumDataPoints` | 10 | Minimum data points required for analysis |
| `OptimalUtilizationMin` | 0.7 (70%) | Minimum optimal utilization |
| `OptimalUtilizationMax` | 0.9 (90%) | Maximum optimal utilization |

## Analysis Algorithm

### Resource Analysis

1. **Collect Metrics**: Gather CPU/memory metrics for the past N days (default: 7)
2. **Calculate Statistics**:
   - P50, P95, P99 percentiles
   - Average and max values
   - Variance (for stability)
3. **Detect Issues**:
   - Over-provisioned: P95 usage < 50% of requested
   - Under-provisioned: P95 usage > 80% of limit
4. **Calculate Scores**:
   - Utilization score (optimal: 70-90%)
   - Efficiency score (utilization + stability)

### Right-Sizing Algorithm

**For Over-Provisioned Resources:**
- Recommended = P95 usage × 1.2 (20% buffer)
- Priority: Based on potential savings

**For Under-Provisioned Resources:**
- Recommended = P95 usage × 1.5 (50% buffer)
- Priority: High (performance risk)

### HPA Optimization

Analyzes:
- **Scaling Frequency**: Events per day
- **Scaling Amplitude**: Range of replica changes
- **Ceiling Hit Rate**: % of time at max replicas
- **Idle Rate**: % of time at min replicas
- **Target Accuracy**: Difference between target and actual CPU

Recommendations:
- Increase max replicas if hitting ceiling > 10% of time
- Decrease min replicas if idle > 80% of time
- Adjust target CPU if difference > 20%

## Efficiency Scoring

Overall efficiency score (0-100) is calculated as:

```
Score = (Utilization × 0.5) + (Stability × 0.3) + (CostEfficiency × 0.2)
```

### Utilization Score (50% weight)
- Perfect: 70-90% utilization = 100 points
- Under-utilized: Linear decrease from 70% to 0%
- Over-utilized: Linear decrease from 90% to 200%

### Stability Score (30% weight)
- Base: 100 points
- Penalties:
  - Each restart: -5 points
  - High variance: -10 points
  - Frequent scaling (>10/day): -20 points

### Cost Efficiency Score (20% weight)
- Base: 100 points
- Penalties:
  - CPU waste: -50 points per 100% waste
  - Memory waste: -50 points per 100% waste

## Recommendation Types

### 1. Resource Recommendations
- Adjust CPU requests and limits
- Adjust memory requests and limits
- Combined resource optimization

**Example:**
```json
{
  "type": "resource",
  "priority": "high",
  "description": "Reduce CPU request from 200m to 100m",
  "current_config": {
    "cpu_request": "200m",
    "cpu_limit": "400m"
  },
  "recommended_config": {
    "cpu_request": "100m",
    "cpu_limit": "200m"
  },
  "estimated_savings": 15.50
}
```

### 2. HPA Recommendations
- Adjust min/max replicas
- Adjust target CPU percentage
- Combined HPA optimization

**Example:**
```json
{
  "type": "hpa",
  "priority": "medium",
  "description": "Increase HPA max replicas from 5 to 7",
  "current_config": {
    "min_replicas": 2,
    "max_replicas": 5,
    "target_cpu": 70
  },
  "recommended_config": {
    "min_replicas": 2,
    "max_replicas": 7,
    "target_cpu": 70
  }
}
```

### 3. Scaling Recommendations
- Scale up replicas (high utilization)
- Scale down replicas (low utilization)

**Example:**
```json
{
  "type": "scaling",
  "priority": "high",
  "description": "Scale up from 2 to 3 replicas",
  "estimated_savings": 0.0
}
```

## Priority Levels

Recommendations are prioritized automatically:

- **High Priority**:
  - Under-provisioned resources (performance risk)
  - High savings potential (>$50/month)
  - Low health score (<60)

- **Medium Priority**:
  - Moderate savings ($20-50/month)
  - HPA needs optimization
  - Medium health score (60-80)

- **Low Priority**:
  - Minor optimizations (<$20/month)
  - Small efficiency improvements
  - High health score (>80)

## Cost Estimation

Monthly costs are estimated using:

```go
CPUCost = vCPUs × $0.03/hour × 24 hours × 30 days
MemoryCost = GB × $0.004/hour × 24 hours × 30 days
Savings = CurrentCost - RecommendedCost
```

Default pricing (configurable):
- CPU: $0.03 per vCPU-hour (~$21.60/vCPU/month)
- Memory: $0.004 per GB-hour (~$2.88/GB/month)

## Advanced Usage

### Analyze All Deployments

```go
namespaces := []string{"default", "production", "staging"}
analyses, err := opt.AnalyzeAllDeployments(namespaces)
```

### Generate All Recommendations

```go
recommendations, err := opt.GenerateAllRecommendations(namespaces)
```

### Get Recommendation Statistics

```go
stats := opt.GetRecommendationStats()
fmt.Printf("Total: %d\n", stats["total"])
fmt.Printf("High Priority: %d\n", stats["high_priority"])
fmt.Printf("Total Savings: $%.2f/mo\n", stats["total_savings"])
```

### Filter Recommendations

```go
// Get recommendations for specific deployment
recs, err := opt.GetRecommendationsForDeployment("default", "my-app")

// Get a specific recommendation by ID
rec, err := opt.GetRecommendationByID("uuid-here")
```

## Data Requirements

The optimizer requires sufficient historical data:
- **Minimum**: 10 data points (configurable)
- **Recommended**: 7 days of metrics at 15-second intervals
- **Optimal**: 14-30 days for seasonal patterns

## Error Handling

The optimizer returns errors for:
- Deployment not found
- Insufficient data points
- Collector not running
- Invalid configuration

```go
analysis, err := opt.AnalyzeDeployment("default", "my-app")
if err != nil {
    log.Printf("Analysis failed: %v", err)
    return
}
```

## Limitations

- Does not apply recommendations automatically (use `ApplyRecommendation` stub)
- In-memory storage only (recommendations not persisted)
- Single-cluster support (no multi-cluster)
- Rule-based algorithms (no machine learning)

## Best Practices

1. **Run collector for at least 24 hours** before generating recommendations
2. **Use longer analysis windows** (14+ days) for production workloads
3. **Review high-priority recommendations immediately** (performance risk)
4. **Apply recommendations during low-traffic periods**
5. **Monitor after applying** to validate improvements
6. **Adjust configuration** based on your workload patterns

## Integration

The optimizer integrates with:
- **Metrics Collector**: Source of historical metrics
- **K8s Client**: Deployment and HPA configuration
- **API Server**: Expose recommendations via REST API
- **Dashboard**: Visualize recommendations and scores

## Performance

- Analysis per deployment: ~100-500ms
- Recommendation generation: ~10-50ms per recommendation
- Memory usage: ~1MB per deployment analyzed
- Concurrent operations: Thread-safe with mutex locks

## License

Part of the k8s-service-optimizer project.
