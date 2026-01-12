# Dashboard Enhancement Implementation Plan

## Executive Summary

This plan addresses the missing dashboard components to create a fully functional, real-time Kubernetes optimization dashboard. The current implementation shows basic cluster metrics but lacks:
- **C2**: Resource Utilization charts showing CPU/Memory trends
- **C3**: Service Details panel with deep-dive metrics
- **C4**: Live HPA scaling visualization
- **C5**: Recent Recommendations display

Based on load testing results showing successful HPA scaling (2→10 replicas @ 401% CPU), we'll implement real-time visualizations to showcase these system reactions.

---

## Current State Analysis

### What Works
- ✅ Basic cluster overview (nodes, pods, CPU%, memory%)
- ✅ Services list with table view
- ✅ Backend API returning correct data
- ✅ 30-second auto-refresh
- ✅ Nginx proxy correctly routing API calls
- ✅ HPA successfully scaling workloads

### What's Missing
- ❌ Resource utilization charts (placeholder text only)
- ❌ Service detail panel (not clickable)
- ❌ HPA scaling visualization
- ❌ Recent recommendations list
- ❌ Real-time WebSocket updates
- ❌ Pod-level metrics display

### Current Files
- Dashboard pages:
  - `dashboard/src/pages/Dashboard.tsx` - Main overview (lines 118-126: placeholder)
  - `dashboard/src/pages/Services.tsx` - Services list (lines 142-147: placeholder)
- API layer:
  - `dashboard/src/services/api.ts` - API client
  - `dashboard/src/services/types.ts` - TypeScript interfaces
- Backend:
  - `backend/pkg/api/router.go` - Route definitions
  - `backend/pkg/api/handlers.go` - Request handlers
  - `backend/internal/models/types.go` - Data models

---

## Available Backend Endpoints

From `backend/pkg/api/router.go`:

```go
// Cluster endpoints
GET  /api/v1/cluster/overview     // ClusterOverview
GET  /api/v1/services              // []ServiceSummary
GET  /api/v1/recommendations       // []Recommendation

// Metrics endpoints
GET  /api/v1/metrics/nodes         // []NodeMetrics
GET  /api/v1/metrics/pods/:namespace  // []PodMetrics

// HPA endpoints
GET  /api/v1/hpa/:namespace        // []HPAMetrics (inferred)

// WebSocket
WS   /ws                           // Real-time updates
```

### Response Shapes (from backend Go structs)

**ClusterOverview** (`backend/internal/models/types.go:141`):
```go
TotalNodes, HealthyNodes int
TotalPods, HealthyPods int
CPUCapacity, CPUUsage int64     // millicores
MemoryCapacity, MemoryUsage int64  // bytes
```

**ServiceSummary** (from `backend/pkg/api/handlers.go`):
```go
name string
namespace string
type string  // "ClusterIP", "NodePort"
clusterIP string
ports int
age string  // Duration as string
```

**PodMetrics** (`backend/internal/models/types.go:6`):
```go
Name string
Namespace string
CPU int64       // millicores
Memory int64    // bytes
Timestamp time.Time
```

**HPAMetrics** (`backend/internal/models/types.go:22`):
```go
Name string
Namespace string
CurrentReplicas int32
DesiredReplicas int32
MinReplicas int32
MaxReplicas int32
TargetCPU int32
CurrentCPU int32
Timestamp time.Time
```

---

## Implementation Plan

### Phase 1: Data Layer Enhancements (1-2 hours)

#### 1.1 Add Missing API Methods

**File**: `dashboard/src/services/api.ts`

Add methods for HPA and pod metrics:

```typescript
// Add to OptimizerAPI class

async getHPAMetrics(namespace?: string): Promise<HPAMetrics[]> {
  const ns = namespace || 'k8s-optimizer';
  return this.fetchJSON<HPAMetrics[]>(`/hpa/${ns}`);
}

async getPodMetrics(namespace?: string): Promise<PodMetrics[]> {
  const ns = namespace || 'k8s-optimizer';
  return this.fetchJSON<PodMetrics[]>(`/metrics/pods/${ns}`);
}

async getNodeMetrics(): Promise<NodeMetrics[]> {
  return this.fetchJSON<NodeMetrics[]>('/metrics/nodes');
}
```

#### 1.2 Update Type Definitions

**File**: `dashboard/src/services/types.ts`

Add missing interfaces:

```typescript
export interface HPAMetrics {
  name: string;
  namespace: string;
  currentReplicas: number;
  desiredReplicas: number;
  minReplicas: number;
  maxReplicas: number;
  targetCPU: number;
  currentCPU: number;
  timestamp: string;
}

export interface PodMetrics {
  name: string;
  namespace: string;
  cpu: number;      // millicores
  memory: number;   // bytes
  timestamp: string;
}

export interface NodeMetrics {
  name: string;
  cpu: number;      // millicores
  memory: number;   // bytes
  timestamp: string;
}

export interface TimeSeriesPoint {
  timestamp: Date;
  value: number;
}

export interface ResourceChart {
  cpu: TimeSeriesPoint[];
  memory: TimeSeriesPoint[];
}
```

#### 1.3 Add Backend HPA Endpoint (if missing)

**File**: `backend/pkg/api/router.go`

Check if HPA route exists. If not, add:

```go
r.Get("/hpa/{namespace}", h.HandleGetHPAMetrics)
```

**File**: `backend/pkg/api/handlers.go`

Add handler:

```go
func (h *Handlers) HandleGetHPAMetrics(w http.ResponseWriter, r *http.Request) {
    namespace := chi.URLParam(r, "namespace")

    hpaMetrics, err := h.collector.CollectHPAMetrics(namespace)
    if err != nil {
        h.respondError(w, http.StatusInternalServerError, "Failed to collect HPA metrics", err)
        return
    }

    h.respondJSON(w, http.StatusOK, hpaMetrics)
}
```

---

### Phase 2: Resource Utilization Charts (C2) (2-3 hours)

#### 2.1 Install Charting Library

```bash
cd dashboard
npm install recharts
```

Recharts is lightweight, React-friendly, and doesn't require complex configuration.

#### 2.2 Create Chart Components

**File**: `dashboard/src/components/Charts/ResourceChart.tsx` (new file)

```typescript
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';

interface ResourceChartProps {
  title: string;
  data: Array<{ time: string; cpu: number; memory: number }>;
  unit: string;
}

export default function ResourceChart({ title, data, unit }: ResourceChartProps) {
  return (
    <div className="rounded-lg bg-white p-6 shadow">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">{title}</h3>
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="time" />
          <YAxis label={{ value: unit, angle: -90, position: 'insideLeft' }} />
          <Tooltip />
          <Legend />
          <Line type="monotone" dataKey="cpu" stroke="#8b5cf6" name="CPU" strokeWidth={2} />
          <Line type="monotone" dataKey="memory" stroke="#3b82f6" name="Memory" strokeWidth={2} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
```

#### 2.3 Create HPA Scaling Chart

**File**: `dashboard/src/components/Charts/HPAChart.tsx` (new file)

```typescript
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { HPAMetrics } from '../../services/types';

interface HPAChartProps {
  hpaData: HPAMetrics[];
}

export default function HPAChart({ hpaData }: HPAChartProps) {
  const chartData = hpaData.map(hpa => ({
    name: hpa.name,
    current: hpa.currentReplicas,
    desired: hpa.desiredReplicas,
    min: hpa.minReplicas,
    max: hpa.maxReplicas,
    cpuPercent: hpa.currentCPU,
    targetCPU: hpa.targetCPU
  }));

  return (
    <div className="rounded-lg bg-white p-6 shadow">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">HPA Scaling Status</h3>
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={chartData}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="name" />
          <YAxis yAxisId="left" label={{ value: 'Replicas', angle: -90, position: 'insideLeft' }} />
          <YAxis yAxisId="right" orientation="right" label={{ value: 'CPU %', angle: 90, position: 'insideRight' }} />
          <Tooltip />
          <Legend />
          <Line yAxisId="left" type="monotone" dataKey="current" stroke="#10b981" name="Current Replicas" strokeWidth={2} />
          <Line yAxisId="left" type="monotone" dataKey="desired" stroke="#f59e0b" name="Desired Replicas" strokeWidth={2} />
          <Line yAxisId="left" type="monotone" dataKey="min" stroke="#6b7280" name="Min" strokeDasharray="5 5" />
          <Line yAxisId="left" type="monotone" dataKey="max" stroke="#6b7280" name="Max" strokeDasharray="5 5" />
          <Line yAxisId="right" type="monotone" dataKey="cpuPercent" stroke="#ef4444" name="CPU %" strokeWidth={2} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
```

#### 2.4 Update Dashboard Page

**File**: `dashboard/src/pages/Dashboard.tsx`

Replace placeholder (lines 118-136) with:

```typescript
import ResourceChart from '../components/Charts/ResourceChart';
import HPAChart from '../components/Charts/HPAChart';
import { useEffect, useState } from 'react';

// Inside Dashboard component:
const [hpaMetrics, setHpaMetrics] = useState<HPAMetrics[]>([]);
const [podMetrics, setPodMetrics] = useState<PodMetrics[]>([]);
const [nodeMetrics, setNodeMetrics] = useState<NodeMetrics[]>([]);

useEffect(() => {
  const fetchMetrics = async () => {
    try {
      const [hpa, pods, nodes] = await Promise.all([
        api.getHPAMetrics(),
        api.getPodMetrics(),
        api.getNodeMetrics()
      ]);
      setHpaMetrics(hpa);
      setPodMetrics(pods);
      setNodeMetrics(nodes);
    } catch (err) {
      console.error('Error fetching metrics:', err);
    }
  };

  fetchMetrics();
  const interval = setInterval(fetchMetrics, 15000); // 15s refresh for charts
  return () => clearInterval(interval);
}, []);

// Transform node metrics for chart
const resourceData = nodeMetrics.map(node => ({
  time: new Date(node.timestamp).toLocaleTimeString(),
  cpu: node.cpu / 1000, // Convert millicores to cores
  memory: node.memory / 1024 / 1024 / 1024 // Convert bytes to GB
}));

// In render:
<div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
  <ResourceChart
    title="Cluster Resource Usage"
    data={resourceData}
    unit="Cores / GB"
  />
  <HPAChart hpaData={hpaMetrics} />
</div>
```

---

### Phase 3: Service Details Panel (C3) (2-3 hours)

#### 3.1 Create Service Detail Component

**File**: `dashboard/src/components/ServiceDetail.tsx` (new file)

```typescript
import { useEffect, useState } from 'react';
import { api } from '../services/api';
import type { Service, PodMetrics, HPAMetrics } from '../services/types';

interface ServiceDetailProps {
  service: Service;
  onClose: () => void;
}

export default function ServiceDetail({ service, onClose }: ServiceDetailProps) {
  const [pods, setPods] = useState<PodMetrics[]>([]);
  const [hpa, setHpa] = useState<HPAMetrics | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchDetails = async () => {
      try {
        setLoading(true);
        const [podMetrics, hpaMetrics] = await Promise.all([
          api.getPodMetrics(service.namespace),
          api.getHPAMetrics(service.namespace)
        ]);

        // Filter pods for this service
        const servicePods = podMetrics.filter(p =>
          p.name.startsWith(service.name)
        );
        setPods(servicePods);

        // Find HPA for this service
        const serviceHpa = hpaMetrics.find(h =>
          h.name.includes(service.name)
        );
        setHpa(serviceHpa || null);
      } catch (err) {
        console.error('Error fetching service details:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchDetails();
    const interval = setInterval(fetchDetails, 15000);
    return () => clearInterval(interval);
  }, [service]);

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto" onClick={onClose}>
      <div className="flex min-h-screen items-center justify-center p-4">
        <div className="fixed inset-0 bg-black bg-opacity-50" />

        <div
          className="relative w-full max-w-4xl rounded-lg bg-white shadow-xl"
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div className="flex items-center justify-between border-b p-6">
            <div>
              <h2 className="text-2xl font-bold text-gray-900">{service.name}</h2>
              <p className="text-sm text-gray-600">{service.namespace}</p>
            </div>
            <button
              onClick={onClose}
              className="rounded-lg p-2 hover:bg-gray-100"
            >
              <span className="text-2xl">&times;</span>
            </button>
          </div>

          {/* Content */}
          <div className="p-6 space-y-6">
            {loading ? (
              <div className="text-center py-8">Loading details...</div>
            ) : (
              <>
                {/* HPA Status */}
                {hpa && (
                  <div className="rounded-lg bg-blue-50 p-4">
                    <h3 className="font-semibold text-gray-900 mb-3">
                      Horizontal Pod Autoscaler
                    </h3>
                    <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
                      <div>
                        <div className="text-xs text-gray-600">Current Replicas</div>
                        <div className="text-2xl font-bold text-blue-600">
                          {hpa.currentReplicas}
                        </div>
                      </div>
                      <div>
                        <div className="text-xs text-gray-600">Desired Replicas</div>
                        <div className="text-2xl font-bold text-blue-600">
                          {hpa.desiredReplicas}
                        </div>
                      </div>
                      <div>
                        <div className="text-xs text-gray-600">CPU Usage</div>
                        <div className={`text-2xl font-bold ${
                          hpa.currentCPU > hpa.targetCPU ? 'text-red-600' : 'text-green-600'
                        }`}>
                          {hpa.currentCPU}%
                        </div>
                        <div className="text-xs text-gray-600">Target: {hpa.targetCPU}%</div>
                      </div>
                      <div>
                        <div className="text-xs text-gray-600">Range</div>
                        <div className="text-lg font-semibold text-gray-700">
                          {hpa.minReplicas} - {hpa.maxReplicas}
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                {/* Pod List */}
                <div>
                  <h3 className="font-semibold text-gray-900 mb-3">
                    Pods ({pods.length})
                  </h3>
                  <div className="space-y-2">
                    {pods.map((pod) => (
                      <div
                        key={pod.name}
                        className="flex items-center justify-between rounded-lg border p-3"
                      >
                        <div className="flex-1">
                          <div className="font-medium text-sm">{pod.name}</div>
                          <div className="text-xs text-gray-600">
                            Last updated: {new Date(pod.timestamp).toLocaleTimeString()}
                          </div>
                        </div>
                        <div className="flex space-x-4 text-sm">
                          <div>
                            <span className="text-gray-600">CPU:</span>{' '}
                            <span className="font-semibold">{pod.cpu}m</span>
                          </div>
                          <div>
                            <span className="text-gray-600">Memory:</span>{' '}
                            <span className="font-semibold">
                              {(pod.memory / 1024 / 1024).toFixed(0)}Mi
                            </span>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
```

#### 3.2 Update Services Page

**File**: `dashboard/src/pages/Services.tsx`

Add state and click handler:

```typescript
import ServiceDetail from '../components/ServiceDetail';

// Add state
const [selectedService, setSelectedService] = useState<Service | null>(null);

// Update table row (line 91):
<tr
  key={`${service.namespace}-${service.name}`}
  className="hover:bg-gray-50 cursor-pointer"
  onClick={() => setSelectedService(service)}
>

// Add at end of component (after table):
{selectedService && (
  <ServiceDetail
    service={selectedService}
    onClose={() => setSelectedService(null)}
  />
)}

// Remove placeholder section (lines 142-147)
```

---

### Phase 4: Recent Recommendations (C4) (1 hour)

#### 4.1 Create Recommendations Component

**File**: `dashboard/src/components/RecentRecommendations.tsx` (new file)

```typescript
import { useEffect, useState } from 'react';
import { api } from '../services/api';
import type { Recommendation } from '../services/types';

export default function RecentRecommendations() {
  const [recommendations, setRecommendations] = useState<Recommendation[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchRecommendations = async () => {
      try {
        const data = await api.getRecommendations();
        // Show only the 5 most recent
        setRecommendations(data.slice(0, 5));
      } catch (err) {
        console.error('Error fetching recommendations:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchRecommendations();
    const interval = setInterval(fetchRecommendations, 30000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return <div className="text-center py-4">Loading recommendations...</div>;
  }

  if (recommendations.length === 0) {
    return (
      <div className="text-center py-8 text-gray-600">
        No recommendations yet. System is analyzing your workloads.
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {recommendations.map((rec) => (
        <div
          key={rec.id}
          className={`rounded-lg border-l-4 p-4 ${
            rec.priority === 'high'
              ? 'border-red-500 bg-red-50'
              : rec.priority === 'medium'
              ? 'border-yellow-500 bg-yellow-50'
              : 'border-blue-500 bg-blue-50'
          }`}
        >
          <div className="flex items-start justify-between">
            <div className="flex-1">
              <div className="flex items-center space-x-2">
                <span className={`inline-flex rounded px-2 py-1 text-xs font-semibold ${
                  rec.priority === 'high'
                    ? 'bg-red-100 text-red-800'
                    : rec.priority === 'medium'
                    ? 'bg-yellow-100 text-yellow-800'
                    : 'bg-blue-100 text-blue-800'
                }`}>
                  {rec.priority}
                </span>
                <span className="text-xs text-gray-600">{rec.type}</span>
              </div>
              <p className="mt-2 text-sm text-gray-900">{rec.description}</p>
              <div className="mt-2 flex items-center space-x-4 text-xs text-gray-600">
                <span>{rec.namespace}/{rec.deployment}</span>
                {rec.estimatedSavings > 0 && (
                  <span className="text-green-600 font-semibold">
                    Save: ${rec.estimatedSavings.toFixed(2)}/month
                  </span>
                )}
              </div>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
```

#### 4.2 Add to Dashboard

**File**: `dashboard/src/pages/Dashboard.tsx`

Replace recommendations placeholder (lines 128-136):

```typescript
import RecentRecommendations from '../components/RecentRecommendations';

// In render:
<div className="rounded-lg bg-white p-6 shadow">
  <h2 className="text-lg font-semibold text-gray-900 mb-4">
    Recent Recommendations
  </h2>
  <RecentRecommendations />
</div>
```

---

### Phase 5: Real-Time WebSocket Updates (Optional, 2 hours)

#### 5.1 Create WebSocket Hook

**File**: `dashboard/src/hooks/useWebSocket.ts`

```typescript
import { useEffect, useRef, useState } from 'react';

interface WebSocketMessage {
  type: string;
  data: any;
}

export function useWebSocket(url: string) {
  const ws = useRef<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null);

  useEffect(() => {
    // Construct WebSocket URL
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}${url}`;

    ws.current = new WebSocket(wsUrl);

    ws.current.onopen = () => {
      console.log('WebSocket connected');
      setIsConnected(true);
    };

    ws.current.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        setLastMessage(message);
      } catch (err) {
        console.error('Error parsing WebSocket message:', err);
      }
    };

    ws.current.onclose = () => {
      console.log('WebSocket disconnected');
      setIsConnected(false);
    };

    ws.current.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    return () => {
      if (ws.current) {
        ws.current.close();
      }
    };
  }, [url]);

  return { isConnected, lastMessage };
}
```

#### 5.2 Use WebSocket in Dashboard

```typescript
import { useWebSocket } from '../hooks/useWebSocket';

// In Dashboard component:
const { isConnected, lastMessage } = useWebSocket('/ws');

// React to updates
useEffect(() => {
  if (lastMessage) {
    console.log('WebSocket update:', lastMessage);
    // Refresh specific data based on message type
    if (lastMessage.type === 'hpa_update') {
      // Refresh HPA data
    } else if (lastMessage.type === 'recommendation_new') {
      // Refresh recommendations
    }
  }
}, [lastMessage]);

// Show connection indicator in header
<div className="flex items-center space-x-2">
  <div className={`h-2 w-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`} />
  <span className="text-xs text-gray-600">
    {isConnected ? 'Live' : 'Disconnected'}
  </span>
</div>
```

---

## Testing Plan

### Manual Testing

1. **Resource Charts**:
   ```bash
   # Generate load to see HPA scaling
   kubectl -n k8s-optimizer run load-gen --rm -it --image=busybox -- sh -c '
     for i in $(seq 1 20); do
       while true; do wget -q -O- http://echo-demo.k8s-optimizer.svc.cluster.local >/dev/null; done &
     done
     wait'

   # Watch dashboard at http://localhost:3000
   # Verify charts update showing:
   # - CPU% increasing from 2% → 256% → 401%
   # - Replicas scaling 2 → 4 → 8 → 10
   # - Chart updates every 15 seconds
   ```

2. **Service Details**:
   ```bash
   # Navigate to Services page
   # Click on "echo-demo" service
   # Verify modal shows:
   # - HPA status with current/desired replicas
   # - CPU usage vs target
   # - List of all pods with resource usage
   # - Auto-refresh every 15s
   ```

3. **Recommendations**:
   ```bash
   # Check if recommendations appear
   curl http://localhost:8080/api/v1/recommendations | jq .

   # Verify dashboard shows recent recommendations
   # with priority badges and estimated savings
   ```

### Automated Testing

**File**: `dashboard/src/components/__tests__/ResourceChart.test.tsx`

```typescript
import { render } from '@testing-library/react';
import ResourceChart from '../Charts/ResourceChart';

test('renders resource chart with data', () => {
  const data = [
    { time: '10:00', cpu: 1.5, memory: 2.3 },
    { time: '10:01', cpu: 2.1, memory: 2.5 }
  ];

  const { getByText } = render(
    <ResourceChart title="Test Chart" data={data} unit="Cores" />
  );

  expect(getByText('Test Chart')).toBeInTheDocument();
});
```

---

## Deployment Checklist

### Build and Deploy

```bash
# 1. Install dependencies
cd dashboard
npm install recharts

# 2. Build dashboard
npm run build

# 3. Rebuild Docker image
cd ..
docker build -t k8s-optimizer-dashboard:latest -f dashboard/Dockerfile dashboard/

# 4. Load into kind
kind load docker-image k8s-optimizer-dashboard:latest --name k8s-optimizer

# 5. Restart pods
kubectl -n k8s-optimizer delete pods -l app=optimizer-dashboard

# 6. Wait for rollout
kubectl -n k8s-optimizer rollout status deployment optimizer-dashboard

# 7. Test
curl http://localhost:3000
```

### Verify Backend Routes

```bash
# Check all endpoints return data
curl http://localhost:8080/api/v1/cluster/overview | jq .
curl http://localhost:8080/api/v1/services | jq .
curl http://localhost:8080/api/v1/metrics/pods/k8s-optimizer | jq .
curl http://localhost:8080/api/v1/recommendations | jq .

# Test HPA endpoint (if added)
curl http://localhost:8080/api/v1/hpa/k8s-optimizer | jq .
```

---

## Expected Results

After implementation, the dashboard will:

1. **Show real-time HPA scaling** matching the load test results:
   - CPU: 2% → 256% → 401% (chart line)
   - Replicas: 2 → 4 → 8 → 10 (bar chart)
   - Visual indication when scaling up/down

2. **Display resource trends** for the entire cluster:
   - Node CPU/Memory usage over time
   - Pod-level metrics aggregated
   - 15-second refresh rate

3. **Provide service deep-dive** on click:
   - All pods for that service
   - HPA configuration and current status
   - Real-time resource usage per pod

4. **List actionable recommendations**:
   - Priority-based sorting
   - Cost savings estimates
   - One-click drill-down (future enhancement)

---

## Future Enhancements (Out of Scope)

- Historical data (24h+ retention)
- Custom time range selection
- Comparison charts (before/after optimization)
- Alert configuration
- Export to CSV/PDF
- Dark mode
- Mobile responsive layouts

---

## Success Criteria

✅ Resource utilization charts display live data
✅ HPA scaling visualization matches kubectl observations
✅ Service detail modal opens on row click
✅ Recommendations list shows priorities and savings
✅ All charts auto-refresh every 15-30 seconds
✅ No console errors in browser
✅ Backend returns < 500ms for all endpoints
✅ Dashboard loads in < 2 seconds

---

## Timeline Estimate

| Phase | Task | Time | Cumulative |
|-------|------|------|------------|
| 1 | Data layer (API methods, types) | 1-2h | 2h |
| 2 | Resource charts (install recharts, components) | 2-3h | 5h |
| 3 | Service details panel | 2-3h | 8h |
| 4 | Recommendations list | 1h | 9h |
| 5 | WebSocket (optional) | 2h | 11h |
| Testing | Manual + automated | 2h | 13h |
| **Total** | **Complete implementation** | **~13 hours** | |

---

## Notes

- The plan leverages existing backend APIs to minimize backend changes
- If HPA endpoint is missing in backend, it can be added in 30 minutes
- Charts use simulated time-series by polling current values every 15s
- For true time-series, backend would need historical data retention (future enhancement)
- All TypeScript types match Go struct field casing (PascalCase in Go, camelCase in TS with mappers)

---

**Plan Status**: Ready for implementation
**Last Updated**: 2026-01-12
**Dependencies**: recharts (npm package)
**Backend Changes Required**: Minimal (1 endpoint if HPA route missing)
