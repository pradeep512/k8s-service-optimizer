# Dashboard Fixes Implementation Plan

**Date**: 2026-01-12
**Status**: Ready to Implement

---

## Issues Identified

### 1. Charts Show Only 3 Current Data Points
**Problem**: Resource charts and HPA charts only display the most recent 3 data points, not a historical time window.

**Root Cause**:
- Charts are mapping from node metrics array (which only has 3 nodes)
- No time-series data storage in frontend
- Backend doesn't provide historical time-series data

### 2. Slow Refresh Rate
**Problem**: Dashboard updates every 30 seconds, making real-time monitoring difficult.

**Current State**:
- Cluster overview: 30s refresh
- Metrics (HPA, nodes): 15s refresh

**Target**: 5-10 second refresh for near real-time updates

### 3. Services Table Shows No Data
**Problem**: Services list shows:
- Replicas: 0
- Health Score: 0
- CPU Usage: 0
- Memory Usage: 0
- Status: "unknown"

**Root Cause**: Backend returns Kubernetes Service objects (ClusterIP, ports), not Deployment metrics

### 4. Service Detail Modal Shows No Data
**Problem**: Clicking a service opens modal but shows no pods, metrics, or recommendations.

**Root Cause**: Backend endpoint `/api/v1/services/{namespace}/{name}` doesn't exist or returns insufficient data

---

## Solution Architecture

### Part 1: Time-Series Data for Charts

#### Frontend Changes

**File**: `dashboard/src/pages/Dashboard.tsx`

**Strategy**: Store rolling window of metrics data in component state

```typescript
// Add state for time-series data
const [metricsHistory, setMetricsHistory] = useState<{
  timestamps: string[];
  cpuData: number[];
  memoryData: number[];
  hpaHistory: Map<string, Array<{ timestamp: string; replicas: number; cpu: number }>>;
}>({
  timestamps: [],
  cpuData: [],
  memoryData: [],
  hpaHistory: new Map()
});

const MAX_DATA_POINTS = 60; // Keep last 60 data points (5 minutes at 5s intervals)

useEffect(() => {
  const fetchMetrics = async () => {
    const [hpa, nodes] = await Promise.all([
      api.getHPAMetrics(),
      api.getNodeMetrics()
    ]);

    const now = new Date().toLocaleTimeString();
    const totalCPU = nodes.reduce((sum, n) => sum + n.cpuUsage, 0) / 1000;
    const totalMemory = nodes.reduce((sum, n) => sum + n.memoryUsage, 0) / 1024 / 1024 / 1024;

    setMetricsHistory(prev => {
      // Add new data point
      const newTimestamps = [...prev.timestamps, now].slice(-MAX_DATA_POINTS);
      const newCpuData = [...prev.cpuData, totalCPU].slice(-MAX_DATA_POINTS);
      const newMemoryData = [...prev.memoryData, totalMemory].slice(-MAX_DATA_POINTS);

      // Update HPA history
      const newHpaHistory = new Map(prev.hpaHistory);
      hpa.forEach(h => {
        const history = newHpaHistory.get(h.name) || [];
        history.push({
          timestamp: now,
          replicas: h.currentReplicas,
          cpu: h.currentCPU
        });
        newHpaHistory.set(h.name, history.slice(-MAX_DATA_POINTS));
      });

      return {
        timestamps: newTimestamps,
        cpuData: newCpuData,
        memoryData: newMemoryData,
        hpaHistory: newHpaHistory
      };
    });
  };

  fetchMetrics();
  const interval = setInterval(fetchMetrics, 5000); // 5 second refresh
  return () => clearInterval(interval);
}, []);
```

**Update ResourceChart Component**:
```typescript
<ResourceChart
  title="Cluster Resource Usage"
  data={metricsHistory.timestamps.map((time, i) => ({
    time,
    cpu: metricsHistory.cpuData[i],
    memory: metricsHistory.memoryData[i]
  }))}
  unit="Cores / GB"
/>
```

**Update HPAChart Component**:
Need to change from bar chart to time-series line chart

**File**: `dashboard/src/components/Charts/HPAChart.tsx`

```typescript
export default function HPAChart({ hpaHistory }: { hpaHistory: Map<string, Array<{timestamp: string, replicas: number, cpu: number}>> }) {
  // For each HPA, show replica trend over time
  const hpaName = Array.from(hpaHistory.keys())[0]; // Show first HPA
  const data = hpaHistory.get(hpaName) || [];

  return (
    <div className="rounded-lg bg-white p-6 shadow">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">
        HPA Scaling Trend: {hpaName}
      </h3>
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="timestamp" />
          <YAxis yAxisId="left" label={{ value: 'Replicas', angle: -90, position: 'insideLeft' }} />
          <YAxis yAxisId="right" orientation="right" label={{ value: 'CPU %', angle: 90, position: 'insideRight' }} />
          <Tooltip />
          <Legend />
          <Line yAxisId="left" type="monotone" dataKey="replicas" stroke="#10b981" name="Replicas" strokeWidth={2} />
          <Line yAxisId="right" type="monotone" dataKey="cpu" stroke="#ef4444" name="CPU %" strokeWidth={2} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
```

---

### Part 2: Faster Refresh Rates

**Changes Needed**:

1. **Dashboard.tsx** - Cluster overview: 30s → 10s
2. **Dashboard.tsx** - Metrics: 15s → 5s
3. **Services.tsx** - Services list: 30s → 10s

**Rationale**: 5-10 second intervals provide near real-time feedback without overwhelming the backend.

---

### Part 3: Fix Services Data

#### Backend Changes

**Problem**: `/api/v1/services` endpoint returns K8s Service objects, not Deployment metrics.

**Solution**: Create new endpoint that aggregates deployment data

**File**: `backend/pkg/api/router.go`

Add new route:
```go
api.HandleFunc("/deployments", s.handleListDeployments).Methods("GET")
api.HandleFunc("/deployments/{namespace}/{name}", s.handleDeploymentDetail).Methods("GET")
```

**File**: `backend/pkg/api/handlers.go`

Create new handler:
```go
func (s *Server) handleListDeployments(w http.ResponseWriter, r *http.Request) {
    ctx := context.Background()

    // Get all namespaces
    namespaces, err := s.k8sClient.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "K8S_ERROR", fmt.Sprintf("Failed to list namespaces: %v", err))
        return
    }

    var allDeployments []map[string]interface{}

    for _, ns := range namespaces.Items {
        deployments, err := s.k8sClient.Clientset.AppsV1().Deployments(ns.Name).List(ctx, metav1.ListOptions{})
        if err != nil {
            continue
        }

        for _, deploy := range deployments.Items {
            // Get pod metrics for this deployment
            podMetrics, _ := s.collector.CollectPodMetrics(ns.Name)

            var totalCPU, totalMemory int64
            var healthyPods int32

            // Match pods to deployment by label selector
            labelSelector := labels.SelectorFromSet(deploy.Spec.Selector.MatchLabels)
            pods, _ := s.k8sClient.Clientset.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{
                LabelSelector: labelSelector.String(),
            })

            for _, pod := range pods.Items {
                if pod.Status.Phase == "Running" {
                    healthyPods++
                }
                // Find metrics for this pod
                for _, pm := range podMetrics {
                    if pm.Name == pod.Name {
                        totalCPU += pm.CPU
                        totalMemory += pm.Memory
                    }
                }
            }

            replicas := int32(0)
            if deploy.Spec.Replicas != nil {
                replicas = *deploy.Spec.Replicas
            }

            healthScore := 0.0
            if replicas > 0 {
                healthScore = (float64(healthyPods) / float64(replicas)) * 100
            }

            status := "healthy"
            if healthScore < 80 {
                status = "warning"
            }
            if healthScore < 50 {
                status = "critical"
            }
            if replicas == 0 {
                status = "unknown"
            }

            deploymentInfo := map[string]interface{}{
                "name":        deploy.Name,
                "namespace":   deploy.Namespace,
                "replicas":    replicas,
                "healthScore": healthScore,
                "cpuUsage":    float64(totalCPU) / 1000.0, // millicores to cores
                "memoryUsage": totalMemory,                 // bytes
                "status":      status,
                "age":         time.Since(deploy.CreationTimestamp.Time).String(),
            }
            allDeployments = append(allDeployments, deploymentInfo)
        }
    }

    respondWithSuccess(w, allDeployments)
}
```

#### Frontend Changes

**File**: `dashboard/src/services/api.ts`

Add new method:
```typescript
async getDeployments(): Promise<Service[]> {
  const data = await this.fetchJSON<BackendDeployment[]>('/deployments');
  return data.map(mapDeployment);
}
```

Add interface and mapper:
```typescript
interface BackendDeployment {
  name: string;
  namespace: string;
  replicas: number;
  healthScore: number;
  cpuUsage: number;
  memoryUsage: number;
  status: string;
}

function mapDeployment(deploy: BackendDeployment): Service {
  return {
    name: deploy.name,
    namespace: deploy.namespace,
    replicas: deploy.replicas,
    healthScore: deploy.healthScore,
    cpuUsage: deploy.cpuUsage,
    memoryUsage: deploy.memoryUsage,
    status: deploy.status,
  };
}
```

**File**: `dashboard/src/pages/Services.tsx`

Update to use new endpoint:
```typescript
const data = await api.getDeployments(); // Changed from getServices()
```

---

### Part 4: Fix Service Detail Modal

#### Backend Changes

**File**: `backend/pkg/api/handlers.go`

```go
func (s *Server) handleDeploymentDetail(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    namespace := vars["namespace"]
    name := vars["name"]

    ctx := context.Background()

    // Get deployment
    deployment, err := s.k8sClient.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
    if err != nil {
        respondWithError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Deployment not found: %v", err))
        return
    }

    // Get pods for this deployment
    labelSelector := labels.SelectorFromSet(deployment.Spec.Selector.MatchLabels)
    podList, err := s.k8sClient.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
        LabelSelector: labelSelector.String(),
    })

    // Get pod metrics
    podMetrics, _ := s.collector.CollectPodMetrics(namespace)
    metricsMap := make(map[string]models.PodMetrics)
    for _, pm := range podMetrics {
        metricsMap[pm.Name] = pm
    }

    // Build pod list with metrics
    var pods []map[string]interface{}
    var totalCPU, totalMemory int64

    for _, pod := range podList.Items {
        metrics := metricsMap[pod.Name]
        totalCPU += metrics.CPU
        totalMemory += metrics.Memory

        podInfo := map[string]interface{}{
            "name":         pod.Name,
            "status":       string(pod.Status.Phase),
            "cpuUsage":     float64(metrics.CPU) / 1000.0,
            "memoryUsage":  metrics.Memory,
            "restartCount": sumRestartCounts(pod.Status.ContainerStatuses),
            "age":          time.Since(pod.CreationTimestamp.Time).String(),
        }
        pods = append(pods, podInfo)
    }

    replicas := int32(0)
    if deployment.Spec.Replicas != nil {
        replicas = *deployment.Spec.Replicas
    }

    // Calculate health score
    runningPods := 0
    for _, pod := range podList.Items {
        if pod.Status.Phase == "Running" {
            runningPods++
        }
    }
    healthScore := 0.0
    if replicas > 0 {
        healthScore = (float64(runningPods) / float64(replicas)) * 100
    }

    // Get recommendations for this deployment
    recommendations, _ := s.optimizer.GenerateRecommendations(namespace, name)

    detail := map[string]interface{}{
        "name":        deployment.Name,
        "namespace":   deployment.Namespace,
        "replicas":    replicas,
        "healthScore": healthScore,
        "cpuUsage":    float64(totalCPU) / 1000.0,
        "memoryUsage": totalMemory,
        "status":      "Running",
        "pods":        pods,
        "metrics": map[string]interface{}{
            "avgCPU":     float64(totalCPU) / float64(len(pods)) / 1000.0,
            "maxCPU":     float64(totalCPU) / 1000.0,
            "p95CPU":     float64(totalCPU) / 1000.0,
            "avgMemory":  totalMemory / int64(len(pods)),
            "maxMemory":  totalMemory,
            "p95Memory":  totalMemory,
        },
        "recommendations": recommendations,
    }

    respondWithSuccess(w, detail)
}

func sumRestartCounts(statuses []corev1.ContainerStatus) int {
    total := 0
    for _, status := range statuses {
        total += int(status.RestartCount)
    }
    return total
}
```

#### Frontend Changes

**File**: `dashboard/src/services/api.ts`

Update service detail method:
```typescript
async getServiceDetail(namespace: string, name: string): Promise<ServiceDetail> {
  return this.fetchJSON<ServiceDetail>(`/deployments/${namespace}/${name}`);
}
```

---

## Implementation Order

### Phase 1: Fix Services Data (High Priority)
**Time**: 1-2 hours

1. Add `/deployments` endpoint to backend
2. Add `/deployments/{namespace}/{name}` endpoint to backend
3. Rebuild and deploy backend
4. Update frontend API client to use new endpoints
5. Test Services page and modal

### Phase 2: Implement Time-Series Charts (High Priority)
**Time**: 1-2 hours

1. Update Dashboard.tsx to store metrics history
2. Update ResourceChart to use historical data
3. Refactor HPAChart to show time-series trend
4. Reduce refresh intervals to 5-10 seconds
5. Test charts with live data

### Phase 3: Testing & Refinement
**Time**: 30 minutes

1. Generate load to trigger HPA scaling
2. Verify charts update in real-time
3. Verify service data displays correctly
4. Verify service detail modal shows all data
5. Monitor dashboard performance

---

## Expected Results

### Charts
- ✅ Resource usage chart shows last 60 data points (5 minutes of history)
- ✅ HPA chart shows replica scaling trend over time
- ✅ Charts update every 5 seconds
- ✅ Can observe HPA scaling events in real-time

### Services Page
- ✅ Shows actual replica counts
- ✅ Shows calculated health scores
- ✅ Shows aggregated CPU/memory usage
- ✅ Shows meaningful status (healthy/warning/critical)

### Service Detail Modal
- ✅ Lists all pods with current metrics
- ✅ Shows pod restart counts
- ✅ Displays service-specific recommendations
- ✅ Shows metrics summary (avg, max, P95)

---

## Files to Modify

### Backend
- `backend/pkg/api/router.go` - Add deployment endpoints
- `backend/pkg/api/handlers.go` - Implement deployment handlers

### Frontend
- `dashboard/src/pages/Dashboard.tsx` - Time-series state management
- `dashboard/src/pages/Services.tsx` - Use deployments endpoint
- `dashboard/src/components/Charts/ResourceChart.tsx` - No changes needed
- `dashboard/src/components/Charts/HPAChart.tsx` - Change to time-series chart
- `dashboard/src/services/api.ts` - Add deployment methods
- `dashboard/src/services/types.ts` - No changes needed

---

## Testing Checklist

- [ ] Backend builds without errors
- [ ] Backend `/deployments` endpoint returns data
- [ ] Backend `/deployments/{namespace}/{name}` endpoint returns detail
- [ ] Frontend builds without errors
- [ ] Dashboard charts show 60 data points after 5 minutes
- [ ] Charts update every 5 seconds
- [ ] Services table shows non-zero values
- [ ] Service modal opens and displays data
- [ ] Load test triggers HPA scaling visible in charts

---

**Status**: Ready for implementation
**Estimated Total Time**: 2-3 hours
